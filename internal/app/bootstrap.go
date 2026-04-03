// Package app provides the application bootstrap logic.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	configfile "clashctl/internal/config"
	"clashctl/internal/system"

	"gopkg.in/yaml.v3"

	"clashctl/internal/core"
)

// MyAppDir returns the clashctl config directory (~/.config/clashctl/).
func MyAppDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户主目录: %w", err)
	}
	return filepath.Join(home, ".config", "clashctl"), nil
}

// ConfigPath returns the path to clashctl's own config file.
func ConfigPath() (string, error) {
	dir, err := MyAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// EnsureMyAppDir creates the clashctl config directory if it doesn't exist.
func EnsureMyAppDir() error {
	dir, err := MyAppDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

// Bootstrap ensures the basic clashctl environment is ready.
func Bootstrap() error {
	if err := EnsureMyAppDir(); err != nil {
		return fmt.Errorf("初始化 clashctl 目录失败: %w", err)
	}
	return nil
}

// LoadOrCreateAppConfig loads the existing AppConfig or returns a default one.
func LoadOrCreateAppConfig() (*core.AppConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return core.DefaultAppConfig(), nil
	}

	// Load from YAML file
	data, err := configfile.ReadConfigWithLimit(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := core.DefaultAppConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	if strings.TrimSpace(cfg.ControllerSecret) == "" {
		cfg.ControllerSecret = core.GenerateControllerSecret()
	}
	if err := ValidateManagedPaths(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveAppConfig saves the AppConfig to disk as YAML.
func SaveAppConfig(cfg *core.AppConfig) error {
	if err := ValidateManagedPaths(cfg); err != nil {
		return err
	}
	if err := EnsureMyAppDir(); err != nil {
		return err
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := system.WriteFileAtomic(path, data, 0600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}
