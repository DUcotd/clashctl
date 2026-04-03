package mihomo

import (
	"errors"
	"testing"
	"time"

	"clashctl/internal/core"
)

type fakeProcess struct {
	started bool
	err     error
}

func (p *fakeProcess) Start() error {
	if p.err != nil {
		return p.err
	}
	p.started = true
	return nil
}

func TestRuntimeManagerResolveConfigFallsBackFromTUN(t *testing.T) {
	manager := &RuntimeManager{
		canUseTUN: func() bool { return false },
		checkTUNPermission: func() error {
			return nil
		},
	}

	cfg := core.DefaultAppConfig()
	cfg.Mode = "tun"
	next, warnings := manager.ResolveConfig(cfg)

	if next.Mode != "mixed" {
		t.Fatalf("Mode = %q", next.Mode)
	}
	if len(warnings) == 0 {
		t.Fatal("expected warnings")
	}
	if cfg.Mode != "tun" {
		t.Fatal("ResolveConfig should not mutate input")
	}
}

func TestRuntimeManagerStartFallsBackToProcess(t *testing.T) {
	proc := &fakeProcess{}
	manager := &RuntimeManager{
		ensureBinary: func() (*InstallResult, error) {
			return &InstallResult{Path: "/usr/local/bin/mihomo", Version: "v1.0.0"}, nil
		},
		hasSystemd: func() bool { return true },
		serviceStatus: func(string) (bool, error) {
			return true, nil
		},
		stopService: func(string) error { return nil },
		setupSystemd: func(ServiceConfig, bool, bool) error {
			return errors.New("boom")
		},
		stopManagedProcess: func(string) (bool, error) { return true, nil },
		newProcess:         func(string) processStarter { return proc },
		geoDataReady:       func(string) bool { return true },
		ensureGeoData:      func(string) (*GeoDataResult, error) { return &GeoDataResult{}, nil },
		waitForController:  func(string, string, int, time.Duration) error { return nil },
		controllerVersion:  func(string, string) (string, error) { return "v1.2.3", nil },
		inspectProxyInventory: func(string, string, string) (*ProxyInventory, error) {
			return &ProxyInventory{Loaded: 3, Current: "node-a"}, nil
		},
		canUseTUN:          func() bool { return true },
		checkTUNPermission: func() error { return nil },
	}

	cfg := core.DefaultAppConfig()
	result, err := manager.Start(cfg, StartOptions{VerifyInventory: true})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if result.StartedBy != "process" {
		t.Fatalf("StartedBy = %q", result.StartedBy)
	}
	if !proc.started {
		t.Fatal("expected process fallback to start")
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected systemd fallback warning")
	}
	if !result.ControllerReady {
		t.Fatal("controller should be ready")
	}
	if result.Inventory == nil || result.Inventory.Loaded != 3 {
		t.Fatalf("unexpected inventory: %#v", result.Inventory)
	}
}
