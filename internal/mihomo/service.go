// Package mihomo provides systemd service management for Mihomo.
package mihomo

import (
	"fmt"
	"os"
	"text/template"

	"clashctl/internal/system"
)

const serviceTemplate = `[Unit]
Description=Mihomo Proxy Service (managed by clashctl)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{.Binary}} -d {{.ConfigDir}}
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`

// ServiceConfig holds parameters for generating the systemd service file.
type ServiceConfig struct {
	Binary    string
	ConfigDir string
	ServiceName string
}

// GenerateServiceFile writes a systemd service file to the appropriate location.
func GenerateServiceFile(cfg ServiceConfig) error {
	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return fmt.Errorf("解析服务模板失败: %w", err)
	}

	path := fmt.Sprintf("/etc/systemd/system/%s.service", cfg.ServiceName)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建服务文件 %s 失败: %w", path, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, cfg); err != nil {
		return fmt.Errorf("写入服务文件失败: %w", err)
	}

	return nil
}

// ReloadSystemd runs systemctl daemon-reload.
func ReloadSystemd() error {
	return system.RunCommandSilent("systemctl", "daemon-reload")
}

// EnableService enables a systemd service.
func EnableService(serviceName string) error {
	_, err := system.RunCommand("systemctl", "enable", serviceName)
	return err
}

// StartService starts a systemd service.
func StartService(serviceName string) error {
	_, err := system.RunCommand("systemctl", "start", serviceName)
	return err
}

// StopService stops a systemd service.
func StopService(serviceName string) error {
	_, err := system.RunCommand("systemctl", "stop", serviceName)
	return err
}

// RestartService restarts a systemd service.
func RestartService(serviceName string) error {
	_, err := system.RunCommand("systemctl", "restart", serviceName)
	return err
}

// ServiceStatus checks if a systemd service is active.
func ServiceStatus(serviceName string) (bool, error) {
	output, err := system.RunCommand("systemctl", "is-active", serviceName)
	if err != nil {
		// "inactive" returns exit code 3, which is not an error for us
		if output == "inactive" || output == "unknown" {
			return false, nil
		}
		return false, err
	}
	return output == "active", nil
}

// SetupSystemd performs the full systemd setup: generate, reload, enable, (optionally) start.
func SetupSystemd(cfg ServiceConfig, autoStart bool) error {
	// Generate service file
	if err := GenerateServiceFile(cfg); err != nil {
		return err
	}

	// Reload systemd
	if err := ReloadSystemd(); err != nil {
		return fmt.Errorf("systemctl daemon-reload 失败: %w", err)
	}

	// Enable service
	if err := EnableService(cfg.ServiceName); err != nil {
		return fmt.Errorf("systemctl enable 失败: %w", err)
	}

	// Start if requested
	if autoStart {
		if err := StartService(cfg.ServiceName); err != nil {
			return fmt.Errorf("systemctl start 失败: %w", err)
		}
	}

	return nil
}
