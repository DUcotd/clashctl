package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"myproxy/internal/mihomo"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "重启 Mihomo 服务",
	RunE:  runRestart,
}

func init() {
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	fmt.Println("🔄 正在重启 Mihomo...")

	if err := mihomo.RestartService("myproxy-mihomo"); err != nil {
		return fmt.Errorf("重启失败: %w", err)
	}

	fmt.Println("✅ Mihomo 已重启")
	return nil
}
