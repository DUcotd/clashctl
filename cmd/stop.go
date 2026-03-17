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
	if err := mihomo.StopService("clashctl-mihomo"); err == nil {
		fmt.Println("✅ 已通过 systemd 停止")
		return nil
	}

	fmt.Println("⚠️  systemd 服务未运行或不存在")
	return nil
}
