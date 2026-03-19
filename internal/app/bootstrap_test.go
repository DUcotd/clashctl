package app

import (
	"os"
	"path/filepath"
	"testing"

	"clashctl/internal/core"
)

func withTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestMyAppDir(t *testing.T) {
	home := withTempHome(t)

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
	if dir != filepath.Join(home, ".config", "clashctl") {
		t.Errorf("MyAppDir() = %q", dir)
	}
}

func TestConfigPath(t *testing.T) {
	withTempHome(t)

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}
	if path == "" {
		t.Error("ConfigPath() returned empty string")
	}
}

func TestBootstrap(t *testing.T) {
	withTempHome(t)

	if err := Bootstrap(); err != nil {
		t.Fatalf("Bootstrap() error: %v", err)
	}

	dir, _ := MyAppDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("Bootstrap() did not create dir %s", dir)
	}
}

func TestLoadOrCreateAppConfig(t *testing.T) {
	withTempHome(t)

	cfg, err := LoadOrCreateAppConfig()
	if err != nil {
		t.Fatalf("LoadOrCreateAppConfig() error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadOrCreateAppConfig() returned nil")
	}
	if cfg.Mode != "mixed" && cfg.Mode != "tun" {
		t.Errorf("default mode = %q, want mixed or tun", cfg.Mode)
	}
}

func TestSaveAndLoadAppConfig(t *testing.T) {
	withTempHome(t)

	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://test.example.com/sub"
	cfg.Mode = "mixed"
	cfg.ConfigDir = "/tmp/test-mihomo"
	cfg.ControllerAddr = "127.0.0.1:9091"

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
	if loaded.ConfigDir != cfg.ConfigDir {
		t.Errorf("loaded ConfigDir = %q, want %q", loaded.ConfigDir, cfg.ConfigDir)
	}
	if loaded.ControllerAddr != cfg.ControllerAddr {
		t.Errorf("loaded ControllerAddr = %q, want %q", loaded.ControllerAddr, cfg.ControllerAddr)
	}
}
