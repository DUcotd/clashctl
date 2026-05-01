package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "启动交互式配置向导",
	Long:  `通过交互式界面引导你完成 Mihomo 的配置生成、写入与启动。`,
	RunE:  withAppConfig(runInit),
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string, appCfg *core.AppConfig) error {
	wizard := ui.NewWizard(appCfg)
	finalModel, err := runTUI(wizard)
	if err != nil {
		return fmt.Errorf("向导运行出错: %w", err)
	}

	if m, ok := finalModel.(interface{ Completed() bool }); ok {
		if !m.Completed() {
			fmt.Println("已取消")
		}
	}

	return nil
}
