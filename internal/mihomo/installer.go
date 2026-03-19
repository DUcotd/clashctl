// Package mihomo provides automatic Mihomo installation.
package mihomo

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
)

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
	downloadURL, isGz := findMihomoAsset(release)
	if downloadURL == "" {
		return nil, fmt.Errorf("未找到适用于 %s/%s 的 Mihomo 二进制文件", runtime.GOOS, runtime.GOARCH)
	}

	tmpPath := installedBinaryPath + ".download"
	if err := os.RemoveAll(tmpPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("清理旧临时文件失败: %w", err)
	}

	if isGz {
		if err := downloadAndDecompressGz(downloadURL, tmpPath); err != nil {
			return nil, fmt.Errorf("下载 Mihomo 失败: %w", err)
		}
	} else {
		if err := downloadBinary(downloadURL, tmpPath); err != nil {
			return nil, fmt.Errorf("下载 Mihomo 失败: %w", err)
		}
	}
	defer os.Remove(tmpPath)

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
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %w", err)
	}

	backupPath := destPath + ".bak"
	if err := os.RemoveAll(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理旧备份失败: %w", err)
	}

	hadExisting := false
	if _, err := os.Stat(destPath); err == nil {
		hadExisting = true
		if err := os.Rename(destPath, backupPath); err != nil {
			return fmt.Errorf("备份现有 Mihomo 失败: %w", err)
		}
	}

	if err := os.Rename(srcPath, destPath); err != nil {
		if hadExisting {
			_ = os.Rename(backupPath, destPath)
		}
		return fmt.Errorf("写入 Mihomo 二进制失败: %w", err)
	}

	if _, err := validateBinary(destPath); err != nil {
		_ = os.Remove(destPath)
		if hadExisting {
			_ = os.Rename(backupPath, destPath)
		}
		return fmt.Errorf("安装后的 Mihomo 二进制校验失败: %w", err)
	}

	if hadExisting {
		_ = os.Remove(backupPath)
	}
	return nil
}

// fetchLatestMihomoRelease gets the latest release info from GitHub API.
func fetchLatestMihomoRelease() (*MihomoRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		MihomoGitHubOwner, MihomoGitHubRepo)

	var release MihomoRelease
	if err := system.FetchJSON(url, GitHubAPITimeout, &release); err != nil {
		return nil, fmt.Errorf("获取 GitHub Release 失败: %w", err)
	}

	return &release, nil
}

// findMihomoAsset finds the correct binary asset for the current platform.
// Returns the download URL and whether it's gzip-compressed.
func findMihomoAsset(release *MihomoRelease) (string, bool) {
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
		return asset.BrowserDownloadURL, false
	}

	// Then try .gz (most common for mihomo releases)
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if !isPlatformMatch(name, goos, goarch) {
			continue
		}
		if strings.HasSuffix(name, ".gz") && !strings.Contains(name, "pkg.tar") {
			return asset.BrowserDownloadURL, true
		}
	}

	return "", false
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

// downloadBinary downloads a file from url to destPath.
func downloadBinary(url, destPath string) error {
	return system.DownloadFile(url, destPath)
}

// downloadAndDecompressGz downloads a gzip-compressed file and decompresses it to destPath.
func downloadAndDecompressGz(url, destPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := system.NewHTTPClient(5*time.Minute, false).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("创建 gzip reader 失败: %w", err)
	}
	defer gzReader.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, gzReader)
	return err
}
