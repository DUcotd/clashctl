package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"clashctl/internal/core"
	"clashctl/internal/system"
)

// ValidateManagedPaths ensures clashctl-managed paths stay within approved write roots.
func ValidateManagedPaths(cfg *core.AppConfig) error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}
	if err := validateManagedConfigDir(cfg.ConfigDir); err != nil {
		return fmt.Errorf("配置目录不安全: %w", err)
	}
	if err := validateManagedProviderPath(cfg.ConfigDir, cfg.ProviderPath); err != nil {
		return fmt.Errorf("Provider 路径不安全: %w", err)
	}
	return nil
}

func validateManagedConfigDir(configDir string) error {
	dir := strings.TrimSpace(configDir)
	if dir == "" {
		return fmt.Errorf("配置目录不能为空")
	}
	return system.ValidateOutputPath(filepath.Join(dir, "config.yaml"))
}

func validateManagedProviderPath(configDir, providerPath string) error {
	trimmed := strings.TrimSpace(providerPath)
	if trimmed == "" {
		return fmt.Errorf("Provider 路径不能为空")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("必须使用相对路径: %s", providerPath)
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("不允许路径遍历: %s", providerPath)
	}

	fullPath := filepath.Join(configDir, cleaned)
	rel, err := filepath.Rel(configDir, fullPath)
	if err != nil {
		return fmt.Errorf("无法解析 Provider 路径: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("不允许路径遍历: %s", providerPath)
	}

	return system.ValidateOutputPath(fullPath)
}
