package cmd

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootVisibleCommandsAreConsolidated(t *testing.T) {
	got := visibleCommandNames(rootCmd)

	for _, want := range []string{"init", "nodes", "service", "config", "doctor", "backup", "update", "version"} {
		if !slices.Contains(got, want) {
			t.Fatalf("visible root commands = %#v, missing %q", got, want)
		}
	}

	for _, hidden := range []string{"advanced", "restore", "start", "stop", "restart", "status", "install", "export", "import", "tui"} {
		if slices.Contains(got, hidden) {
			t.Fatalf("visible root commands should not include legacy %q: %#v", hidden, got)
		}
	}
}

func TestNodesCommandDefaultsToTUI(t *testing.T) {
	if nodesCmd.RunE == nil {
		t.Fatal("nodesCmd.RunE should launch the node manager TUI")
	}
}

func TestLegacyCommandsAreHidden(t *testing.T) {
	for _, cmd := range []*cobra.Command{advancedCmd, restoreCmd, startCmd, stopCmd, restartCmd, statusCmd, installCmd, exportCmd, importCmd, tuiCmd} {
		if !cmd.Hidden {
			t.Fatalf("expected %q to be hidden", cmd.Name())
		}
	}
}

func visibleCommandNames(cmd *cobra.Command) []string {
	var out []string
	for _, child := range cmd.Commands() {
		if child.Hidden {
			continue
		}
		out = append(out, child.Name())
	}
	return out
}
