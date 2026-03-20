package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
	"clashctl/internal/ui"
)

var tuiCmd = &cobra.Command{
	Use:    "tui",
	Short:  "启动交互式管理界面",
	Hidden: true,
}

var tuiNodesCmd = &cobra.Command{
	Use:    "nodes",
	Short:  "直接进入节点测速与切换界面",
	Long:   `跳过 init 向导，直接进入代理组/节点管理 TUI，可测速并切换节点。`,
	Hidden: true,
	RunE:   legacyRunner("clashctl tui nodes", "clashctl nodes", runTUINodes),
}

func init() {
	tuiCmd.AddCommand(tuiNodesCmd)
	rootCmd.AddCommand(tuiCmd)
}

func runTUINodes(cmd *cobra.Command, args []string) error {
	appCfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	client := mihomo.NewClient("http://" + appCfg.ControllerAddr)
	if err := client.CheckConnection(); err != nil {
		return fmt.Errorf("Controller API 不可达: %w\n请先运行 'clashctl service start' 或完成 'clashctl init'", err)
	}

	// Set up signal handling for cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	manager := ui.NewNodeManager(appCfg)
	p := tea.NewProgram(manager, tea.WithAltScreen())

	// Handle signals in a goroutine
	go func() {
		<-sigCh
		// Bubble Tea handles terminal cleanup, just exit gracefully
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("节点管理界面运行出错: %w", err)
	}
	return nil
}
