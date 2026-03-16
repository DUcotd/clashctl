package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"myproxy/internal/mihomo"
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

	// Try systemd first
	active, _ := mihomo.ServiceStatus("myproxy-mihomo")
	if active {
		fmt.Println("⚠️  Mihomo 服务已在运行中")
		return nil
	}

	// Try systemctl start
	if err := mihomo.StartService("myproxy-mihomo"); err == nil {
		fmt.Println("✅ 通过 systemd 启动成功")
		return nil
	}

	// Fallback: direct process
	proc := mihomo.NewProcess("/etc/mihomo")
	if err := proc.Start(); err != nil {
		return fmt.Errorf("启动失败: %w", err)
	}

	fmt.Println("✅ Mihomo 已以子进程方式启动")
	return nil
}
