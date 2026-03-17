// Package config provides clashctl-specific config save/load operations.
package config

import (
	"fmt"

	"clashctl/internal/core"
)

// SaveMihomoConfig renders a MihomoConfig to YAML and writes it to the given path.
// It backs up any existing file first, then writes and validates.
func SaveMihomoConfig(cfg *core.MihomoConfig, path string) (backupPath string, err error) {
	// Render to YAML
	data, err := core.RenderYAML(cfg)
	if err != nil {
		return "", fmt.Errorf("YAML render failed: %w", err)
	}

	// Backup existing
	backupPath, err = BackupFile(path)
	if err != nil {
		return "", fmt.Errorf("backup failed: %w", err)
	}

	// Write new config
	if err := WriteConfig(path, data); err != nil {
		return backupPath, err
	}

	// Validate the written file
	if err := ValidateYAML(path); err != nil {
		return backupPath, fmt.Errorf("written config failed validation: %w", err)
	}

	return backupPath, nil
}
