// Package system provides command execution utilities.
package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// CommandExists checks if a command is available in PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// RunCommand executes a command and returns its combined output.
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(output)), fmt.Errorf("%s %v failed: %w", name, args, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// RunCommandSilent runs a command and returns only the error (discards output).
func RunCommandSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
