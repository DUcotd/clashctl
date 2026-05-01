package cmd

import (
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// runTUI creates a Bubble Tea program with alt-screen and mouse support,
// installs signal handling for graceful shutdown, and runs the model.
func runTUI(model tea.Model) (tea.Model, error) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	go func() {
		<-sigCh
		p.Quit()
	}()
	return p.Run()
}
