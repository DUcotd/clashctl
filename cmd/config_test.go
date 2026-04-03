package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"clashctl/internal/app"
	"clashctl/internal/config"
	"clashctl/internal/core"
	"clashctl/internal/system"
)

func TestRunConfigShowRejectsOversizedConfig(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := core.DefaultAppConfig()
	cfg.ConfigDir = home
	cfg.SubscriptionURL = "https://example.com/sub"
	if err := app.SaveAppConfig(cfg); err != nil {
		t.Fatalf("SaveAppConfig() error = %v", err)
	}

	oversized := strings.Repeat("a", config.MaxConfigFileSize+1)
	if err := os.WriteFile(mihomoConfigPath(cfg), []byte(oversized), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := runConfigShow(nil, nil)
	if err == nil {
		t.Fatal("runConfigShow() should reject oversized config files")
	}
	if !strings.Contains(err.Error(), "配置文件过大") {
		t.Fatalf("runConfigShow() error = %v", err)
	}
}

func TestReadImportSourceRejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub.txt")
	oversized := strings.Repeat("a", system.MaxPreparedSubscriptionBytes+1)
	if err := os.WriteFile(path, []byte(oversized), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, _, err := readImportSource(path)
	if err == nil {
		t.Fatal("readImportSource() should reject oversized files")
	}
	if !strings.Contains(err.Error(), "订阅文件过大") {
		t.Fatalf("readImportSource() error = %v", err)
	}
}

func TestReadImportSourceRejectsOversizedStdin(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer tmpFile.Close()

	oversized := strings.Repeat("a", system.MaxPreparedSubscriptionBytes+1)
	if _, err := io.WriteString(tmpFile, oversized); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if _, err := tmpFile.Seek(0, 0); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}

	oldStdin := os.Stdin
	os.Stdin = tmpFile
	defer func() { os.Stdin = oldStdin }()

	_, _, err = readImportSource("-")
	if err == nil {
		t.Fatal("readImportSource() should reject oversized stdin")
	}
	if !strings.Contains(err.Error(), "订阅文件过大") {
		t.Fatalf("readImportSource() error = %v", err)
	}
}
