package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
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

	// Kill any existing mihomo processes first to avoid port conflicts
	if killed := mihomo.KillExistingMihomo(); killed {
		fmt.Println("🧹 已清理旧进程")
	}

	configDir := core.DefaultConfigDir

	// Try systemd first
	if mihomo.HasSystemd() {
		binary, err := mihomo.FindBinary()
		if err == nil {
			svcCfg := mihomo.ServiceConfig{
				Binary:      binary,
				ConfigDir:   configDir,
				ServiceName: core.DefaultServiceName,
			}
			if err := mihomo.SetupSystemd(svcCfg, true); err == nil {
				fmt.Println("✅ 通过 systemd 启动成功")
				return nil
			}
			fmt.Printf("⚠️  systemd 启动失败: %v\n正在回退到子进程模式...\n", err)
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
