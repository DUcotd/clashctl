package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Mihomo 服务",
	RunE:  runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	fmt.Println("🛑 正在停止 Mihomo...")

	// Try systemd first
	if mihomo.HasSystemd() {
		if err := mihomo.StopService("clashctl-mihomo"); err == nil {
			fmt.Println("✅ 已通过 systemd 停止")
			return nil
		}
	}

	// Fallback: kill processes directly
	if mihomo.KillExistingMihomo() {
		fmt.Println("✅ 已停止 Mihomo 进程")
		return nil
	}

	fmt.Println("⚠️  Mihomo 未在运行")
	return nil
}
