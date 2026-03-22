package setup

import (
	"os"
	"path/filepath"
	"testing"

	"clashctl/internal/core"
	"clashctl/internal/subscription"
)

func TestApplyResolvedPlanSavesConfigAndAppConfig(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(t.TempDir(), "mihomo")
	t.Setenv("HOME", home)

	cfg := core.DefaultAppConfig()
	cfg.ConfigDir = configDir
	plan := &subscription.ResolvedConfigPlan{
		Kind: subscription.PlanKindStatic,
		RawYAML: []byte("mixed-port: 7890\nmode: rule\nproxies: []\nproxy-groups: []\nrules: []\n"),
	}

	result, err := ApplyResolvedPlan(cfg, plan, ApplyPlanOptions{SaveAppConfig: true})
	if err != nil {
		t.Fatalf("ApplyResolvedPlan() error = %v", err)
	}
	if result == nil {
		t.Fatal("ApplyResolvedPlan() result is nil")
	}
	if !result.AppConfigSaved {
		t.Fatal("expected AppConfigSaved to be true")
	}
	if result.OutputPath != filepath.Join(configDir, "config.yaml") {
		t.Fatalf("OutputPath = %q", result.OutputPath)
	}
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Fatalf("config file missing: %v", err)
	}
	appConfigPath := filepath.Join(home, ".config", "clashctl", "config.yaml")
	if _, err := os.Stat(appConfigPath); err != nil {
		t.Fatalf("app config missing: %v", err)
	}
}

func TestWrapStageErrorWrapsMatchingStage(t *testing.T) {
	err := &ApplyPlanError{Stage: StageWriteConfig, Err: os.ErrPermission}
	wrapped := WrapStageError(err, StageWriteConfig, "写入失败: %w")
	if wrapped == nil || wrapped.Error() != "写入失败: permission denied" {
		t.Fatalf("WrapStageError() = %v", wrapped)
	}
	unchanged := WrapStageError(err, StageSaveAppConfig, "保存失败: %w")
	if unchanged != err {
		t.Fatalf("WrapStageError() should return original error for unmatched stage")
	}
}
