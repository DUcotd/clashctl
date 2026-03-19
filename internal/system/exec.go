// Package system provides command execution utilities.
package system

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// CommandRunner abstracts process execution.
type CommandRunner interface {
	CombinedOutput(name string, args ...string) (string, error)
	Run(name string, args ...string) error
}

// ExecRunner executes commands via os/exec.
type ExecRunner struct{}

func (ExecRunner) CombinedOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(output)), fmt.Errorf("%s %v 执行失败: %w", name, args, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (ExecRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

var defaultRunner CommandRunner = ExecRunner{}
var defaultRunnerMu sync.RWMutex

// SetCommandRunner swaps the default runner and returns a restore function.
func SetCommandRunner(runner CommandRunner) func() {
	if runner == nil {
		runner = ExecRunner{}
	}
	defaultRunnerMu.Lock()
	prev := defaultRunner
	defaultRunner = runner
	defaultRunnerMu.Unlock()
	return func() {
		defaultRunnerMu.Lock()
		defaultRunner = prev
		defaultRunnerMu.Unlock()
	}
}

// CommandExists checks if a command is available in PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// RunCommand executes a command and returns its combined output.
func RunCommand(name string, args ...string) (string, error) {
	defaultRunnerMu.RLock()
	runner := defaultRunner
	defaultRunnerMu.RUnlock()
	return runner.CombinedOutput(name, args...)
}

// RunCommandSilent runs a command and returns only the error (discards output).
func RunCommandSilent(name string, args ...string) error {
	defaultRunnerMu.RLock()
	runner := defaultRunner
	defaultRunnerMu.RUnlock()
	return runner.Run(name, args...)
}
