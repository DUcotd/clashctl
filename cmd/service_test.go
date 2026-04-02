package cmd

import (
	"errors"
	"strings"
	"testing"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

func TestRunRestartReturnsControllerReadinessError(t *testing.T) {
	prevLoad := loadAppConfigFn
	prevHasSystemd := hasSystemdFn
	prevRuntime := newRuntimeManager
	prevStart := startRuntimeFn
	t.Cleanup(func() {
		loadAppConfigFn = prevLoad
		hasSystemdFn = prevHasSystemd
		newRuntimeManager = prevRuntime
		startRuntimeFn = prevStart
	})

	loadAppConfigFn = func() (*core.AppConfig, error) {
		return core.DefaultAppConfig(), nil
	}
	hasSystemdFn = func() bool { return false }
	newRuntimeManager = func() *mihomo.RuntimeManager {
		return &mihomo.RuntimeManager{}
	}
	startRuntimeFn = func(_ *mihomo.RuntimeManager, _ *core.AppConfig, _ mihomo.StartOptions) (*mihomo.StartResult, error) {
		return &mihomo.StartResult{StartedBy: "process"}, errors.New("Controller API 未就绪: boom")
	}

	err := runRestart(nil, nil)
	if err == nil {
		t.Fatal("runRestart() should fail when controller never becomes ready")
	}
	if !strings.Contains(err.Error(), "Controller API 未就绪") {
		t.Fatalf("runRestart() error = %v", err)
	}
}
