package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	"clashctl/internal/system"
)

var (
	updateDryRun    bool
	updateForce     bool
	updatePreRelease bool
)

var updateSelfCmd = &cobra.Command{
	Use:   "self",
	Short: "更新 clashctl 本身",
	Long:  `检查并更新 clashctl 到最新版本。`,
	RunE:  runUpdateSelf,
}

func init() {
	updateSelfCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "仅检查更新，不实际下载")
	updateSelfCmd.Flags().BoolVar(&updateForce, "force", false, "强制更新（即使已是最新版本）")
	updateSelfCmd.Flags().BoolVar(&updatePreRelease, "pre-release", false, "包含预发布版本")
	rootCmd.AddCommand(updateSelfCmd)
}

func runUpdateSelf(cmd *cobra.Command, args []string) error {
	fmt.Printf("🔍 当前版本: %s\n", core.AppVersion)

	// Get latest release
	release, err := getLatestClashctlRelease()
	if err != nil {
		return fmt.Errorf("获取最新版本失败: %w", err)
	}

	latestVersion := release.TagName
	fmt.Printf("📡 最新版本: %s\n", latestVersion)

	// Check if update needed
	if latestVersion == core.AppVersion && !updateForce {
		fmt.Println("✅ 已是最新版本")
		return nil
	}

	if updateDryRun {
		fmt.Println("ℹ️  运行模式：仅检查（dry-run）")
		fmt.Printf("   可更新到: %s\n", latestVersion)
		return nil
	}

	// Determine binary name
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	binaryName := fmt.Sprintf("clashctl-%s-%s", goos, goarch)

	// Find download URL
	downloadURL := ""
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("找不到适合 %s/%s 的二进制文件", goos, goarch)
	}

	fmt.Printf("⬇️  下载 %s...\n", binaryName)

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "clashctl-update-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Try with mirror
	mirrorURL := mihomo.GetGitHubMirrorURL(downloadURL)
	if err := system.DownloadFile(mirrorURL, tmpFile.Name()); err != nil {
		// Fallback to original
		if err := system.DownloadFile(downloadURL, tmpFile.Name()); err != nil {
			return fmt.Errorf("下载失败: %w", err)
		}
	}

	// Verify the binary works
	tmpFile.Close()
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("设置执行权限失败: %w", err)
	}

	// Get current binary path
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取当前程序路径失败: %w", err)
	}

	// Backup current binary
	backupPath := currentPath + ".bak"
	if err := os.Rename(currentPath, backupPath); err != nil {
		return fmt.Errorf("备份当前版本失败: %w", err)
	}

	// Move new binary
	if err := os.Rename(tmpFile.Name(), currentPath); err != nil {
		// Try to restore backup
		_ = os.Rename(backupPath, currentPath)
		return fmt.Errorf("安装新版本失败: %w", err)
	}

	// Clean up backup
	_ = os.Remove(backupPath)

	fmt.Printf("✅ 更新成功: %s → %s\n", core.AppVersion, latestVersion)

	return nil
}

// GitHubRelease for clashctl
type ClashctlRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestClashctlRelease() (*ClashctlRelease, error) {
	var release ClashctlRelease
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", "DUcotd/clashctl")

	if err := system.FetchJSON(url, 15*1e9, &release); err != nil {
		// Try mirror
		mirrorURL := mihomo.GetGitHubMirrorURL(url)
		if mirrorErr := system.FetchJSON(mirrorURL, 15*1e9, &release); mirrorErr != nil {
			return nil, err
		}
	}

	return &release, nil
}
