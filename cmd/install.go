package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
	"clashctl/internal/system"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "安装 Mihomo 内核",
	Long:  `自动下载并安装最新版本的 Mihomo 内核到 /usr/local/bin/mihomo。`,
	RunE:  runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Check root
	if err := system.RequireRoot(); err != nil {
		return err
	}

	// Check if already installed
	if binary, err := mihomo.FindBinary(); err == nil {
		version, _ := mihomo.GetBinaryVersion()
		printInstallStatus(os.Stdout, binary, version)
		return nil
	}

	// Download and install
	result, err := mihomo.InstallMihomo()
	if err != nil {
		return fmt.Errorf("安装失败: %w", err)
	}
	printInstallResult(os.Stdout, result)

	return nil
}
