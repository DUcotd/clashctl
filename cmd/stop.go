package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
	"clashctl/internal/system"
)

var stopCmd = &cobra.Command{
	Use:    "stop",
	Short:  "停止 Mihomo 服务",
	Hidden: true,
	RunE:   legacyRunner("clashctl stop", "clashctl service stop", runStop),
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	// Check root for systemd operations
	if mihomo.HasSystemd() {
		if err := system.RequireRootForOperation("systemd 服务停止"); err != nil {
			return err
		}
	}

	fmt.Println("🛑 正在停止 Mihomo...")
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	// Try systemd first
	if mihomo.HasSystemd() {
		if err := mihomo.StopService(mihomo.DefaultServiceName); err == nil {
			fmt.Println("✅ 已通过 systemd 停止")
			return nil
		}
	}

	// Fallback: stop managed subprocesses for the current config dir.
	stopped, err := mihomo.StopManagedProcess(cfg.ConfigDir)
	if err != nil {
		return err
	}
	if stopped {
		fmt.Println("✅ 已停止 Mihomo 进程")
		return nil
	}

	fmt.Println("⚠️  Mihomo 未在运行")
	return nil
}
