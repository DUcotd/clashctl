// Package mihomo provides automatic Mihomo installation.
package mihomo

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"clashctl/internal/system"
)

const (
	// InstallPath is the default location where clashctl installs mihomo.
	InstallPath = "/usr/local/bin/mihomo"
	// MihomoGitHubOwner is the GitHub repo owner for Mihomo releases.
	MihomoGitHubOwner = "MetaCubeX"
	// MihomoGitHubRepo is the GitHub repo name for Mihomo releases.
	MihomoGitHubRepo = "mihomo"
	// GitHubAPITimeout is the timeout for GitHub API requests.
	GitHubAPITimeout = 15 * time.Second
	// MaxDecompressedBinaryBytes bounds gzip expansion when unpacking release assets.
	MaxDecompressedBinaryBytes = 200 * 1024 * 1024
)

// GitHub mirror URLs for users in China
var githubMirrors = []string{
	"https://ghproxy.com/",
	"https://mirror.ghproxy.com/",
}

// GitHubRelease represents a GitHub release (minimal fields).
type MihomoRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// InstallResult describes the resolved Mihomo binary.
type InstallResult struct {
	Path       string
	Version    string
	ReleaseTag string
	Installed  bool
}

// GetGitHubMirrorURL returns a mirror URL if GitHub is not directly accessible.
// Returns the original URL if no mirror is needed or available.
func GetGitHubMirrorURL(originalURL string) string {
	// Check if user has set a custom mirror
	if customMirror := os.Getenv("CLASHCTL_GITHUB_MIRROR"); customMirror != "" {
		mirror := strings.TrimRight(customMirror, "/")
		if strings.HasPrefix(originalURL, "https://github.com/") || strings.HasPrefix(originalURL, "https://api.github.com/") {
			return mirror + "/" + strings.TrimPrefix(originalURL, "https://")
		}
	}

	// Try default mirrors if the original URL is not reachable
	for _, mirror := range githubMirrors {
		if strings.HasPrefix(originalURL, "https://github.com/") || strings.HasPrefix(originalURL, "https://api.github.com/") {
			return mirror + strings.TrimPrefix(originalURL, "https://")
		}
	}

	return originalURL
}

// EnsureMihomo checks if mihomo is available, and if not, downloads and installs it.
// Returns the path to the binary.
func EnsureMihomo() (*InstallResult, error) {
	// First check if already available
	if path, err := FindBinary(); err == nil {
		version, _ := GetBinaryVersion()
		return &InstallResult{
			Path:      path,
			Version:   version,
			Installed: false,
		}, nil
	}

	// Not found, need to install
	return InstallMihomo()
}

// InstallMihomo downloads the latest mihomo binary to InstallPath.
func InstallMihomo() (*InstallResult, error) {
	release, err := fetchLatestMihomoRelease()
	if err != nil {
		return nil, fmt.Errorf("获取 Mihomo 版本信息失败: %w", err)
	}

	// Find matching binary
	assetName, downloadURL, isGz := findMihomoAsset(release)
	if downloadURL == "" {
		return nil, fmt.Errorf("未找到适用于 %s/%s 的 Mihomo 二进制文件", runtime.GOOS, runtime.GOARCH)
	}
	checksumAsset, err := findReleaseChecksumAsset(release, assetName)
	if err != nil {
		return nil, err
	}

	tmpPath, err := system.CreateSiblingTempFile(installedBinaryPath, ".download-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时下载文件失败: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if isGz {
		if err := downloadAndDecompressGz(system.NamedDownload{Name: assetName, URL: downloadURL}, checksumAsset, tmpPath); err != nil {
			return nil, fmt.Errorf("下载 Mihomo 失败: %w", err)
		}
	} else {
		if err := downloadBinary(system.NamedDownload{Name: assetName, URL: downloadURL}, checksumAsset, tmpPath); err != nil {
			return nil, fmt.Errorf("下载 Mihomo 失败: %w", err)
		}
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return nil, fmt.Errorf("设置执行权限失败: %w", err)
	}

	version, err := validateBinary(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("下载的 Mihomo 二进制不可用: %w", err)
	}

	if err := activateBinary(tmpPath, installedBinaryPath); err != nil {
		return nil, err
	}

	return &InstallResult{
		Path:       installedBinaryPath,
		Version:    version,
		ReleaseTag: release.TagName,
		Installed:  true,
	}, nil
}

func activateBinary(srcPath, destPath string) error {
	return system.ReplaceFile(srcPath, destPath, system.ReplaceFileOptions{
		Validate: func(path string) error {
			_, err := validateBinary(path)
			if err != nil {
				return fmt.Errorf("安装后的 Mihomo 二进制校验失败: %w", err)
			}
			return nil
		},
	})
}

// fetchLatestMihomoRelease gets the latest release info from GitHub API.
func fetchLatestMihomoRelease() (*MihomoRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		MihomoGitHubOwner, MihomoGitHubRepo)

	var release MihomoRelease
	if err := system.FetchJSON(url, GitHubAPITimeout, &release); err != nil {
		// Try mirror URL if original fails
		mirrorURL := GetGitHubMirrorURL(url)
		if mirrorURL != url {
			if mirrorErr := system.FetchJSON(mirrorURL, GitHubAPITimeout, &release); mirrorErr == nil {
				return &release, nil
			}
		}
		return nil, fmt.Errorf("获取 GitHub Release 失败: %w", err)
	}

	return &release, nil
}

// findMihomoAsset finds the correct binary asset for the current platform.
// Returns the download URL and whether it's gzip-compressed.
func findMihomoAsset(release *MihomoRelease) (string, string, bool) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Prefer uncompressed binary first
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if !isPlatformMatch(name, goos, goarch) {
			continue
		}
		if strings.HasSuffix(name, ".gz") || strings.HasSuffix(name, ".zip") ||
			strings.HasSuffix(name, ".deb") || strings.HasSuffix(name, ".rpm") ||
			strings.HasSuffix(name, ".zst") {
			continue
		}
		return asset.Name, asset.BrowserDownloadURL, false
	}

	// Then try .gz (most common for mihomo releases)
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if !isPlatformMatch(name, goos, goarch) {
			continue
		}
		if strings.HasSuffix(name, ".gz") && !strings.Contains(name, "pkg.tar") {
			return asset.Name, asset.BrowserDownloadURL, true
		}
	}

	return "", "", false
}

// isPlatformMatch checks if an asset name matches the target OS and architecture.
// It uses word-boundary-aware matching to avoid "arm" matching "arm64".
func isPlatformMatch(name, goos, goarch string) bool {
	if !strings.Contains(name, goos) {
		return false
	}
	if goarch == "arm" {
		// "arm" must not be followed by "64" (avoid matching "arm64")
		idx := strings.Index(name, "arm")
		for idx >= 0 {
			end := idx + 3
			if end+2 <= len(name) && name[end:end+2] == "64" {
				// This is "arm64", keep searching
				idx = strings.Index(name[end:], "arm")
				if idx >= 0 {
					idx += end
				}
				continue
			}
			return true
		}
		return false
	}
	return strings.Contains(name, goarch)
}

func findReleaseChecksumAsset(release *MihomoRelease, assetName string) (system.NamedDownload, error) {
	assets := make([]system.NamedDownload, 0, len(release.Assets))
	for _, asset := range release.Assets {
		assets = append(assets, system.NamedDownload{Name: asset.Name, URL: asset.BrowserDownloadURL})
	}
	checksumAsset, ok := system.FindChecksumAsset(assets, assetName)
	if !ok {
		return system.NamedDownload{}, fmt.Errorf("发布缺少 %s 的校验文件", assetName)
	}
	return checksumAsset, nil
}

// downloadBinary downloads a verified file to destPath.
func downloadBinary(asset, checksumAsset system.NamedDownload, destPath string) error {
	if err := system.DownloadVerifiedFile(asset, checksumAsset, destPath); err != nil {
		if !system.AllowUntrustedMirrorDownloads() {
			return err
		}
		mirrorAsset := asset
		mirrorAsset.URL = GetGitHubMirrorURL(asset.URL)
		mirrorChecksum := checksumAsset
		mirrorChecksum.URL = GetGitHubMirrorURL(checksumAsset.URL)
		if mirrorAsset.URL != asset.URL {
			if mirrorErr := system.DownloadVerifiedFile(mirrorAsset, mirrorChecksum, destPath); mirrorErr == nil {
				return nil
			}
		}
		return err
	}
	return nil
}

// downloadAndDecompressGz downloads a verified gzip-compressed file and decompresses it to destPath.
func downloadAndDecompressGz(asset, checksumAsset system.NamedDownload, destPath string) error {
	gzPath := destPath + ".gz"
	defer os.Remove(gzPath)
	if err := downloadBinary(asset, checksumAsset, gzPath); err != nil {
		return err
	}
	return decompressGzipFileLimited(gzPath, destPath, MaxDecompressedBinaryBytes)
}

func decompressGzipFileLimited(gzPath, destPath string, maxBytes int64) error {
	if maxBytes <= 0 {
		return fmt.Errorf("无效的解压大小上限: %d", maxBytes)
	}

	in, err := os.Open(gzPath)
	if err != nil {
		return err
	}
	defer in.Close()

	gzReader, err := gzip.NewReader(in)
	if err != nil {
		return fmt.Errorf("创建 gzip reader 失败: %w", err)
	}
	defer gzReader.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, io.LimitReader(gzReader, maxBytes+1))
	if err != nil {
		return err
	}
	if written > maxBytes {
		return fmt.Errorf("解压后的文件超过大小上限: %d bytes (最大允许 %d bytes)", written, maxBytes)
	}
	return nil
}
