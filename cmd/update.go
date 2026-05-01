package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	"clashctl/internal/releases"
	"clashctl/internal/system"
)

const (
	githubOwner = "DUcotd"
	githubRepo  = "clashctl"
)

// currentVer references the canonical version from core.
var currentVer = core.AppVersion

var (
	updateDryRun     bool
	updateForce      bool
	updatePreRelease bool
	updateJSON       bool
)

type updateRunReport struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version,omitempty"`
	BinaryName     string `json:"binary_name,omitempty"`
	DownloadURL    string `json:"download_url,omitempty"`
	DryRun         bool   `json:"dry_run"`
	Force          bool   `json:"force"`
	PreRelease     bool   `json:"pre_release"`
	UpToDate       bool   `json:"up_to_date"`
	Updated        bool   `json:"updated"`
	RequiresRoot   bool   `json:"requires_root,omitempty"`
	Action         string `json:"action"`
	Error          string `json:"error,omitempty"`
}

func (r *updateRunReport) SetError(msg string) { r.Error = msg }

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"self"},
	Short:   "检查并更新 clashctl",
	Long:    `检查 GitHub Releases 获取最新版本，如有更新则自动下载替换。`,
	RunE:    runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "仅检查更新，不实际下载")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "强制更新（即使已是最新版本）")
	updateCmd.Flags().BoolVar(&updatePreRelease, "pre-release", false, "包含预发布版本")
	updateCmd.Flags().BoolVar(&updateJSON, "json", false, "以 JSON 输出更新结果")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	report := &updateRunReport{
		CurrentVersion: currentVer,
		DryRun:         updateDryRun,
		Force:          updateForce,
		PreRelease:     updatePreRelease,
		Action:         "check",
	}
	if !updateJSON {
		fmt.Printf("🔍 当前版本: %s\n\n", currentVer)
		fmt.Println("正在检查更新...")
	}

	// Fetch latest release info
	release, err := releases.FetchLatestGitHubRelease(githubOwner, githubRepo, updatePreRelease, mihomo.GetGitHubMirrorURL)
	if err != nil {
		return finishReport(report, fmt.Errorf("检查更新失败: %w", err), updateJSON)
	}

	latestTag := release.TagName
	report.LatestVersion = latestTag
	if !updateJSON {
		fmt.Printf("   最新版本: %s\n", latestTag)
	}

	if latestTag == currentVer {
		report.UpToDate = true
		report.Action = "none"
		if !updateForce {
			if updateDryRun && !updateJSON {
				fmt.Println("\nℹ️  运行模式：仅检查（dry-run）")
			}
			if !updateJSON {
				fmt.Println("\n✅ 已是最新版本！")
			}
			return finishReport(report, nil, updateJSON)
		}
		report.Action = "update"
		if !updateJSON {
			fmt.Println("\nℹ️  当前版本已是最新版本，将按 --force 重新安装")
		}
	}

	if latestTag != currentVer && !updateJSON {
		fmt.Printf("\n🆕 发现新版本: %s → %s\n", currentVer, latestTag)
	}

	if updateDryRun {
		report.Action = "check"
		if !updateJSON {
			fmt.Println("\nℹ️  运行模式：仅检查（dry-run）")
			fmt.Printf("   可更新到: %s\n", latestTag)
		}
		return finishReport(report, nil, updateJSON)
	}

	// Find the right binary for current platform
	binaryName := fmt.Sprintf("clashctl-%s-%s", runtime.GOOS, runtime.GOARCH)
	report.BinaryName = binaryName
	releaseAsset, ok := releases.FindGitHubReleaseAsset(release, binaryName)
	if !ok {
		return finishReport(report, fmt.Errorf("未找到适用于 %s/%s 的二进制文件", runtime.GOOS, runtime.GOARCH), updateJSON)
	}
	downloadURL := releaseAsset.BrowserDownloadURL
	report.DownloadURL = downloadURL
	checksumAsset := system.NamedDownload{}
	assets := releases.NamedDownloads(release)
	checksumAsset, ok = system.FindChecksumAsset(assets, binaryName)
	if !ok {
		return finishReport(report, fmt.Errorf("发布缺少 %s 的校验文件", binaryName), updateJSON)
	}

	if !updateJSON {
		fmt.Printf("   下载地址: %s\n", downloadURL)
	}

	// Get current binary path
	selfPath, err := os.Executable()
	if err != nil {
		return finishReport(report, fmt.Errorf("无法获取当前程序路径: %w", err), updateJSON)
	}

	// Check write permission
	if !system.IsRoot() {
		report.RequiresRoot = true
		err := fmt.Errorf("权限不足")
		if !updateJSON {
			fmt.Println("\n⚠️  更新需要 root 权限")
			fmt.Println("请使用 sudo clashctl update")
		}
		return finishReport(report, err, updateJSON)
	}

	if !updateJSON {
		fmt.Println("\n正在下载更新...")
	}

	// Download new binary to a unique temp file next to the current executable
	tmpPath, err := system.CreateSiblingTempFile(selfPath, ".tmp-*")
	if err != nil {
		return finishReport(report, fmt.Errorf("创建更新临时文件失败: %w", err), updateJSON)
	}
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	asset := system.NamedDownload{Name: binaryName, URL: downloadURL}
	if err := releases.DownloadVerifiedGitHubAsset(asset, checksumAsset, mihomo.GetGitHubMirrorURL, tmpPath); err != nil {
		return finishReport(report, fmt.Errorf("下载失败: %w", err), updateJSON)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return finishReport(report, fmt.Errorf("设置权限失败: %w", err), updateJSON)
	}
	if err := validateDownloadedClashctlBinary(tmpPath); err != nil {
		return finishReport(report, fmt.Errorf("下载的 clashctl 二进制不可用: %w", err), updateJSON)
	}

	if err := replaceCurrentExecutable(tmpPath, selfPath); err != nil {
		return finishReport(report, err, updateJSON)
	}
	report.Updated = true
	report.Action = "update"

	if !updateJSON {
		fmt.Printf("\n✅ 更新完成！\n")
		fmt.Printf("   %s → %s\n", currentVer, latestTag)
		fmt.Println("\n运行 'clashctl --help' 查看新版本功能")
	}

	return finishReport(report, nil, updateJSON)
}

func hasAlias(cmd *cobra.Command, alias string) bool {
	return slices.Contains(cmd.Aliases, alias)
}

func replaceCurrentExecutable(srcPath, destPath string) error {
	return system.ReplaceFile(srcPath, destPath, system.ReplaceFileOptions{
		Validate: validateDownloadedClashctlBinary,
	})
}

func validateDownloadedClashctlBinary(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "version")
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("执行 version 超时")
	}
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("version 执行失败: %s", msg)
	}
	if !strings.Contains(string(output), "clashctl ") {
		return fmt.Errorf("version 输出异常: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
