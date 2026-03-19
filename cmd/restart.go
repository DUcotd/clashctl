package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
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
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	// Try systemd first
	if mihomo.HasSystemd() {
		if err := mihomo.RestartService(mihomo.DefaultServiceName); err == nil {
			fmt.Println("✅ Mihomo 已重启")
			return nil
		} else {
			fmt.Printf("⚠️  systemd 重启失败: %v\n正在回退到进程模式...\n", err)
		}
	}

	// Fallback: stop existing managed process and start a new one.
	if _, err := mihomo.StopManagedProcess(cfg.ConfigDir); err != nil {
		return fmt.Errorf("重启失败: %w", err)
	}

	proc := mihomo.NewProcess(cfg.ConfigDir)
	if err := proc.Start(); err != nil {
		return fmt.Errorf("重启失败: %w", err)
	}

	fmt.Println("✅ Mihomo 已重启")
	return nil
}
