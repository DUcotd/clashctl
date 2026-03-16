// Package mihomo provides process management for Mihomo.
package mihomo

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Process manages a Mihomo child process.
type Process struct {
	ConfigDir string
	cmd       *exec.Cmd
}

// NewProcess creates a new Process manager.
func NewProcess(configDir string) *Process {
	return &Process{ConfigDir: configDir}
}

// Start launches Mihomo as a background process.
func (p *Process) Start() error {
	binary, err := FindBinary()
	if err != nil {
		return err
	}

	p.cmd = exec.Command(binary, "-d", p.ConfigDir)
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = os.Stderr
	p.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("启动 Mihomo 失败: %w", err)
	}

	// Give it a moment to start up
	time.Sleep(500 * time.Millisecond)

	// Check if it actually started
	if !p.IsRunning() {
		return fmt.Errorf("Mihomo 进程启动后立即退出")
	}

	return nil
}

// Stop terminates the Mihomo process.
func (p *Process) Stop() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	// Try graceful shutdown first
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return p.cmd.Process.Kill()
	}

	// Wait up to 5 seconds for graceful exit
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return p.cmd.Process.Kill()
	}
}

// IsRunning checks if the Mihomo process is still alive.
func (p *Process) IsRunning() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	// Signal 0 checks if process exists without sending a signal
	err := p.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// FindBinary locates the mihomo binary in PATH.
func FindBinary() (string, error) {
	// Try "mihomo" first, then "clash" as fallback
	for _, name := range []string{"mihomo", "clash-meta", "clash"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("未找到 mihomo 可执行文件。请先安装 Mihomo 并确保其在 PATH 中")
}

// GetBinaryVersion returns the version string of the mihomo binary.
func GetBinaryVersion() (string, error) {
	binary, err := FindBinary()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(binary, "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("获取版本号失败: %w", err)
	}

	// Version is typically the first line
	version := string(output)
	if len(version) > 100 {
		version = version[:100]
	}

	return version, nil
}
