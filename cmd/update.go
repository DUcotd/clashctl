package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"clashctl/internal/system"
)

const (
	githubOwner = "DUcotd"
	githubRepo  = "TUI-Proxy"
	currentVer  = "v1.1.0"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "检查并更新 clashctl",
	Long:  `检查 GitHub Releases 获取最新版本，如有更新则自动下载替换。`,
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

// GitHubRelease represents a GitHub release response.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Printf("🔍 当前版本: %s\n\n", currentVer)
	fmt.Println("正在检查更新...")

	// Fetch latest release info
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("检查更新失败: %w", err)
	}

	latestTag := release.TagName
	fmt.Printf("   最新版本: %s\n", latestTag)

	if latestTag == currentVer {
		fmt.Println("\n✅ 已是最新版本！")
		return nil
	}

	fmt.Printf("\n🆕 发现新版本: %s → %s\n", currentVer, latestTag)

	// Find the right binary for current platform
	binaryName := fmt.Sprintf("clashctl-%s-%s", runtime.GOOS, runtime.GOARCH)
	downloadURL := ""

	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("未找到适用于 %s/%s 的二进制文件", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("   下载地址: %s\n", downloadURL)

	// Get current binary path
	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取当前程序路径: %w", err)
	}

	// Check write permission
	if !system.IsRoot() {
		fmt.Println("\n⚠️  更新需要 root 权限")
		fmt.Println("请使用 sudo clashctl update")
		return fmt.Errorf("权限不足")
	}

	fmt.Println("\n正在下载更新...")

	// Download new binary to temp file
	tmpPath := selfPath + ".tmp"
	if err := downloadFile(downloadURL, tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("下载失败: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("设置权限失败: %w", err)
	}

	// Replace current binary
	// Backup old one first
	backupPath := selfPath + ".bak"
	if err := os.Rename(selfPath, backupPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("备份旧版本失败: %w", err)
	}

	if err := os.Rename(tmpPath, selfPath); err != nil {
		// Try to restore backup
		os.Rename(backupPath, selfPath)
		return fmt.Errorf("替换文件失败: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	fmt.Printf("\n✅ 更新完成！\n")
	fmt.Printf("   %s → %s\n", currentVer, latestTag)
	fmt.Println("\n运行 'clashctl --help' 查看新版本功能")

	return nil
}

func fetchLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)

	client := &http.Client{Timeout: 10 * 1e9}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回 %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &release, nil
}

func downloadFile(url, destPath string) error {
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

// RunSelfUpdate runs update via exec (for use after download).
func RunSelfUpdate() error {
	cmd := exec.Command(os.Args[0], "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
