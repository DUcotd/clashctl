package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"clashctl/internal/app"
	"clashctl/internal/core"
)

// withAppConfig wraps a RunE function that needs an AppConfig.
// It loads the config before calling the inner function.
func withAppConfig(runE func(cmd *cobra.Command, args []string, cfg *core.AppConfig) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := loadAppConfigFn()
		if err != nil {
			return err
		}
		return runE(cmd, args, cfg)
	}
}

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
