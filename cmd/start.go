package cmd

import (
	"fmt"
	"os"
	"time"

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

	runtime := mihomo.NewRuntimeManager()
	result, err := runtime.Start(cfg, mihomo.StartOptions{
		VerifyInventory: true,
		WaitRetries:     15,
		WaitInterval:    2 * time.Second,
	})
	printRuntimeStartResult(os.Stdout, result)
	if err != nil {
		return err
	}
	return nil
}
