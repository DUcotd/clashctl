// Package app provides the application bootstrap logic.
package app

import (
	"fmt"
	"os"
	"path/filepath"

	"myproxy/internal/core"
)

// MyAppDir returns the myproxy config directory (~/.config/myproxy/).
func MyAppDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户主目录: %w", err)
	}
	return filepath.Join(home, ".config", "myproxy"), nil
}

// MyProxyConfigPath returns the path to myproxy's own config file.
func MyProxyConfigPath() (string, error) {
	dir, err := MyAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// EnsureMyAppDir creates the myproxy config directory if it doesn't exist.
func EnsureMyAppDir() error {
	dir, err := MyAppDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

// Bootstrap ensures the basic myproxy environment is ready.
func Bootstrap() error {
	if err := EnsureMyAppDir(); err != nil {
		return fmt.Errorf("初始化 myproxy 目录失败: %w", err)
	}
	return nil
}

// LoadOrCreateAppConfig loads the existing AppConfig or returns a default one.
func LoadOrCreateAppConfig() (*core.AppConfig, error) {
	path, err := MyProxyConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return core.DefaultAppConfig(), nil
	}

	// TODO: implement actual loading with viper
	// For now just return defaults
	return core.DefaultAppConfig(), nil
}
