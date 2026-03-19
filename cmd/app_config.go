package cmd

import (
	"fmt"
	"path/filepath"

	"clashctl/internal/app"
	"clashctl/internal/core"
)

func loadAppConfig() (*core.AppConfig, error) {
	if err := app.Bootstrap(); err != nil {
		return nil, err
	}

	cfg, err := app.LoadOrCreateAppConfig()
	if err != nil {
		return nil, fmt.Errorf("加载 clashctl 配置失败: %w", err)
	}

	return cfg, nil
}

func mihomoConfigPath(cfg *core.AppConfig) string {
	return filepath.Join(cfg.ConfigDir, "config.yaml")
}

func mihomoProviderPath(cfg *core.AppConfig) string {
	return filepath.Join(cfg.ConfigDir, cfg.ProviderPath)
}
