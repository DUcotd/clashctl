package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"myproxy/internal/app"
	"myproxy/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "启动交互式配置向导",
	Long:  `通过交互式界面引导你完成 Mihomo 的配置生成、写入与启动。`,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// Ensure myproxy app directory exists
	if err := app.Bootstrap(); err != nil {
		return err
	}

	// Create and run the Bubble Tea wizard
	wizard := ui.NewWizard()
	p := tea.NewProgram(wizard, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("向导运行出错: %w", err)
	}

	// Check if user quit without completing
	if m, ok := finalModel.(ui.WizardModel); ok {
		if !m.Completed() {
			fmt.Println("已取消")
		}
	}

	return nil
}
