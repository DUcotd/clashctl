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

// Start launches Mihomo as a background daemon process.
// It redirects stdout/stderr to /dev/null and creates a new process group
// so the process survives when the parent exits.
func (p *Process) Start() error {
	binary, err := FindBinary()
	if err != nil {
		return err
	}

	p.cmd = exec.Command(binary, "-d", p.ConfigDir)

	// Redirect to /dev/null so process doesn't hold terminal.
	// NOTE: intentionally not closing devNull — the background process
	// needs these FDs open for its entire lifetime.
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		p.cmd.Stdout = nil
		p.cmd.Stderr = nil
	} else {
		p.cmd.Stdout = devNull
		p.cmd.Stderr = devNull
		// Let the OS clean up when the process exits
		_ = devNull
	}

	// Create new process group to detach from parent.
	// Note: Setsid is blocked in some container environments (CAP_SYS_ADMIN required),
	// so we only use Setpgid which works everywhere.
	p.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Detach stdin
	p.cmd.Stdin = nil

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("启动 Mihomo 失败: %w", err)
	}

	// Give it a moment to start up
	time.Sleep(500 * time.Millisecond)

	// Check if it actually started (use process.Signal(0) which still works)
	if p.cmd.Process == nil {
		return fmt.Errorf("Mihomo 进程启动后立即退出")
	}
	if err := p.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("Mihomo 进程启动后立即退出: %w", err)
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
// Note: After Setsid/detach, this only works if we still have the pid reference.
func (p *Process) IsRunning() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	// Signal 0 checks if process exists without sending a signal
	err := p.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// IsMihomoRunning checks if ANY mihomo process is running (system-wide).
func IsMihomoRunning() bool {
	// Check if the controller API is reachable
	client := NewClient("http://127.0.0.1:9090")
	return client.CheckConnection() == nil
}

// KillExistingMihomo kills any running mihomo processes to free the port.
// Returns true if processes were killed, false if none were found.
func KillExistingMihomo() bool {
	// Use pkill to find and kill mihomo processes
	cmd := exec.Command("pkill", "-9", "mihomo")
	err := cmd.Run()
	if err != nil {
		// pkill returns non-zero if no processes matched - that's fine
		return false
	}
	// Give processes time to die and release ports
	time.Sleep(1 * time.Second)
	return true
}

// FindBinary locates the mihomo binary in PATH or at the default install location.
func FindBinary() (string, error) {
	// Try "mihomo" first, then "clash-meta", then "clash"
	for _, name := range []string{"mihomo", "clash-meta", "clash"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	// Fall back to clashctl's default install path
	if _, err := os.Stat(InstallPath); err == nil {
		return InstallPath, nil
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
