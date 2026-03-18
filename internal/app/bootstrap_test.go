package app

import (
	"os"
	"testing"

	"clashctl/internal/core"
)

func TestMyAppDir(t *testing.T) {
	dir, err := MyAppDir()
	if err != nil {
		t.Fatalf("MyAppDir() error: %v", err)
	}
	if dir == "" {
		t.Error("MyAppDir() returned empty string")
	}
	// Should end with .config/clashctl
	if len(dir) < len(".config/clashctl") {
		t.Errorf("MyAppDir() = %q, too short", dir)
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}
	if path == "" {
		t.Error("ConfigPath() returned empty string")
	}
}

func TestBootstrap(t *testing.T) {
	if err := Bootstrap(); err != nil {
		t.Fatalf("Bootstrap() error: %v", err)
	}

	dir, _ := MyAppDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("Bootstrap() did not create dir %s", dir)
	}
}

func TestLoadOrCreateAppConfig(t *testing.T) {
	cfg, err := LoadOrCreateAppConfig()
	if err != nil {
		t.Fatalf("LoadOrCreateAppConfig() error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadOrCreateAppConfig() returned nil")
	}
	if cfg.Mode != "tun" && cfg.Mode != "mixed" {
		t.Errorf("default mode = %q, want tun or mixed", cfg.Mode)
	}
}

func TestSaveAndLoadAppConfig(t *testing.T) {
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://test.example.com/sub"
	cfg.Mode = "mixed"

	if err := SaveAppConfig(cfg); err != nil {
		t.Fatalf("SaveAppConfig() error: %v", err)
	}

	loaded, err := LoadOrCreateAppConfig()
	if err != nil {
		t.Fatalf("LoadOrCreateAppConfig() error: %v", err)
	}
	if loaded.SubscriptionURL != cfg.SubscriptionURL {
		t.Errorf("loaded SubscriptionURL = %q, want %q", loaded.SubscriptionURL, cfg.SubscriptionURL)
	}
	if loaded.Mode != cfg.Mode {
		t.Errorf("loaded Mode = %q, want %q", loaded.Mode, cfg.Mode)
	}
}
