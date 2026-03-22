package setup

import (
	"errors"
	"fmt"
	"path/filepath"

	"clashctl/internal/app"
	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	"clashctl/internal/subscription"
	"clashctl/internal/system"
)

const (
	StageCreateConfigDir = "create_config_dir"
	StageRenderYAML      = "render_yaml"
	StageWriteConfig     = "write_config"
	StageSaveAppConfig   = "save_app_config"
)

type ApplyPlanOptions struct {
	SaveAppConfig bool
}

type ApplyPlanResult struct {
	ConfigDir       string
	OutputPath      string
	YAMLSize        int
	ValidationError error
	BackupPath      string
	AppConfigSaved  bool
}

type ApplyPlanError struct {
	Stage string
	Err   error
}

func (e *ApplyPlanError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ApplyPlanError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func ApplyResolvedPlan(cfg *core.AppConfig, plan *subscription.ResolvedConfigPlan, opts ApplyPlanOptions) (*ApplyPlanResult, error) {
	if cfg == nil {
		return nil, &ApplyPlanError{Stage: StageWriteConfig, Err: errors.New("配置为空")}
	}
	if plan == nil {
		return nil, &ApplyPlanError{Stage: StageWriteConfig, Err: errors.New("配置计划为空")}
	}

	result := &ApplyPlanResult{
		ConfigDir:  cfg.ConfigDir,
		OutputPath: filepath.Join(cfg.ConfigDir, "config.yaml"),
	}

	if err := system.EnsureDir(cfg.ConfigDir); err != nil {
		return result, &ApplyPlanError{Stage: StageCreateConfigDir, Err: err}
	}

	yamlData, err := plan.RenderYAML()
	if err != nil {
		return result, &ApplyPlanError{Stage: StageRenderYAML, Err: err}
	}
	result.YAMLSize = len(yamlData)
	result.ValidationError = mihomo.ValidateConfigContent(yamlData, cfg.ConfigDir)

	backupPath, err := plan.Save(result.OutputPath)
	if err != nil {
		return result, &ApplyPlanError{Stage: StageWriteConfig, Err: err}
	}
	result.BackupPath = backupPath

	if !opts.SaveAppConfig {
		return result, nil
	}
	if err := app.SaveAppConfig(cfg); err != nil {
		return result, &ApplyPlanError{Stage: StageSaveAppConfig, Err: err}
	}
	result.AppConfigSaved = true
	return result, nil
}

func WrapStageError(err error, stage string, format string) error {
	if err == nil {
		return nil
	}
	var stageErr *ApplyPlanError
	if errors.As(err, &stageErr) && stageErr.Stage == stage {
		return fmt.Errorf(format, stageErr.Err)
	}
	return err
}
