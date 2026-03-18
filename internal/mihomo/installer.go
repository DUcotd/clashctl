// Package mihomo provides automatic Mihomo installation.
package mihomo

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	// InstallPath is the default location where clashctl installs mihomo.
	InstallPath = "/usr/local/bin/mihomo"
	// MihomoGitHubOwner is the GitHub repo owner for Mihomo releases.
	MihomoGitHubOwner = "MetaCubeX"
	// MihomoGitHubRepo is the GitHub repo name for Mihomo releases.
	MihomoGitHubRepo = "mihomo"
)

// GitHubRelease represents a GitHub release (minimal fields).
type MihomoRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// EnsureMihomo checks if mihomo is available, and if not, downloads and installs it.
// Returns the path to the binary.
func EnsureMihomo() (string, error) {
	// First check if already available
	if path, err := FindBinary(); err == nil {
		return path, nil
	}

	// Not found, need to install
	return InstallMihomo()
}

// InstallMihomo downloads the latest mihomo binary to InstallPath.
func InstallMihomo() (string, error) {
	fmt.Println("📦 Mihomo 未安装，正在自动下载...")

	release, err := fetchLatestMihomoRelease()
	if err != nil {
		return "", fmt.Errorf("获取 Mihomo 版本信息失败: %w", err)
	}

	fmt.Printf("   最新版本: %s\n", release.TagName)

	// Find matching binary
	downloadURL, isGz := findMihomoAsset(release)
	if downloadURL == "" {
		return "", fmt.Errorf("未找到适用于 %s/%s 的 Mihomo 二进制文件", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("   下载中: %s\n", downloadURL)

	if isGz {
		if err := downloadAndDecompressGz(downloadURL, InstallPath); err != nil {
			return "", fmt.Errorf("下载 Mihomo 失败: %w", err)
		}
	} else {
		if err := downloadBinary(downloadURL, InstallPath); err != nil {
			return "", fmt.Errorf("下载 Mihomo 失败: %w", err)
		}
	}

	if err := os.Chmod(InstallPath, 0755); err != nil {
		return "", fmt.Errorf("设置执行权限失败: %w", err)
	}

	fmt.Printf("✅ Mihomo 已安装到: %s\n", InstallPath)

	// Verify
	version, _ := GetBinaryVersion()
	if version != "" {
		fmt.Printf("   版本: %s\n", version)
	}

	return InstallPath, nil
}

// fetchLatestMihomoRelease gets the latest release info from GitHub API.
func fetchLatestMihomoRelease() (*MihomoRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		MihomoGitHubOwner, MihomoGitHubRepo)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回 %d", resp.StatusCode)
	}

	var release MihomoRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
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
		if strings.Contains(name, goos) && strings.Contains(name, goarch) &&
			!strings.HasSuffix(name, ".gz") && !strings.HasSuffix(name, ".zip") &&
			!strings.HasSuffix(name, ".deb") && !strings.HasSuffix(name, ".rpm") &&
			!strings.HasSuffix(name, ".zst") {
			return asset.BrowserDownloadURL, false
		}
	}

	// Then try .gz (most common for mihomo releases)
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, goos) && strings.Contains(name, goarch) &&
			strings.HasSuffix(name, ".gz") && !strings.Contains(name, "pkg.tar") {
			return asset.BrowserDownloadURL, true
		}
	}

	return "", false
}

// downloadBinary downloads a file from url to destPath.
func downloadBinary(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// downloadAndDecompressGz downloads a gzip-compressed file and decompresses it to destPath.
func downloadAndDecompressGz(url, destPath string) error {
	resp, err := http.Get(url)
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
