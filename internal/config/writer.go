// Package config provides clashctl-specific config save/load operations.
package config

import (
	"fmt"

	"clashctl/internal/core"
	"clashctl/internal/system"
)

// SaveMihomoConfig renders a MihomoConfig to YAML and writes it to the given path.
// It backs up any existing file first, then writes and validates.
func SaveMihomoConfig(cfg *core.MihomoConfig, path string) (backupPath string, err error) {
	if err := system.ValidateOutputPath(path); err != nil {
		return "", fmt.Errorf("输出路径不安全: %w", err)
	}
	// Render to YAML
	data, err := core.RenderYAML(cfg)
	if err != nil {
		return "", fmt.Errorf("YAML 渲染失败: %w", err)
	}
	if err := ValidateYAMLBytes(data, path); err != nil {
		return "", fmt.Errorf("写入前配置校验失败: %w", err)
	}

	// Backup existing
	backupPath, err = BackupFile(path)
	if err != nil {
		return "", fmt.Errorf("备份失败: %w", err)
	}

	// Write new config
	if err := WriteConfig(path, data); err != nil {
		return backupPath, err
	}

	return backupPath, nil
}

// SaveRawYAML writes already-prepared YAML data with backup and validation.
func SaveRawYAML(data []byte, path string) (backupPath string, err error) {
	if err := system.ValidateOutputPath(path); err != nil {
		return "", fmt.Errorf("输出路径不安全: %w", err)
	}
	if err := ValidateYAMLBytes(data, path); err != nil {
		return "", fmt.Errorf("写入前配置校验失败: %w", err)
	}
	backupPath, err = BackupFile(path)
	if err != nil {
		return "", fmt.Errorf("备份失败: %w", err)
	}
	if err := WriteConfig(path, data); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}
