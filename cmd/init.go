package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"clashctl/internal/ui"
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
	appCfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	// Set up signal handling for cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Create and run the Bubble Tea wizard
	wizard := ui.NewWizard(appCfg)
	p := tea.NewProgram(wizard, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Handle signals in a goroutine
	go func() {
		<-sigCh
		// Bubble Tea handles terminal cleanup, just exit gracefully
		p.Quit()
	}()

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("向导运行出错: %w", err)
	}

	// Check if user quit without completing
	if m, ok := finalModel.(interface{ Completed() bool }); ok {
		if !m.Completed() {
			fmt.Println("已取消")
		}
	}

	return nil
}
