package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Mihomo 服务",
	Long:  `根据已有配置文件启动 Mihomo。优先使用 systemd，否则以子进程方式启动。`,
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 正在启动 Mihomo...")

	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	if stopped, err := mihomo.StopManagedProcess(cfg.ConfigDir); err != nil {
		fmt.Printf("⚠️  清理旧进程失败: %v\n", err)
	} else if stopped {
		fmt.Println("🧹 已清理旧进程")
	}

	configDir := cfg.ConfigDir

	// Pre-download geodata if missing (avoids mihomo blocking on first startup)
	if mihomo.NeedGeoData(configDir) {
		fmt.Println("📦 首次运行，正在预下载 GeoSite/GeoIP 数据...")
		if _, err := mihomo.EnsureGeoData(configDir); err != nil {
			fmt.Printf("⚠️  预下载失败: %v (Mihomo 会自动重试)\n", err)
		}
	}

	// Try systemd first when enabled in app config.
	if cfg.EnableSystemd && mihomo.HasSystemd() {
		binary, err := mihomo.FindBinary()
		if err == nil {
			svcCfg := mihomo.ServiceConfig{
				Binary:      binary,
				ConfigDir:   configDir,
				ServiceName: mihomo.DefaultServiceName,
			}
			if err := mihomo.SetupSystemd(svcCfg, cfg.AutoStart, true); err == nil {
				fmt.Println("✅ 通过 systemd 启动成功")
				return nil
			}
			fmt.Printf("⚠️  systemd 启动失败: %v\n正在回退到子进程模式...\n", err)
		}
	}

	if mihomo.HasSystemd() {
		if active, _ := mihomo.ServiceStatus(mihomo.DefaultServiceName); active {
			if err := mihomo.StopService(mihomo.DefaultServiceName); err != nil {
				fmt.Printf("⚠️  停止已有 systemd 服务失败: %v\n", err)
			}
		}
	}

	// Fallback: direct process
	proc := mihomo.NewProcess(configDir)
	if err := proc.Start(); err != nil {
		return fmt.Errorf("启动失败: %w", err)
	}

	fmt.Println("✅ Mihomo 已以子进程方式启动")
	return nil
}
