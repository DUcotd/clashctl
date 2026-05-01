package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"clashctl/internal/app"
	"clashctl/internal/config"
	"clashctl/internal/core"
)

func setupCmdTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func writeTestAppConfig(t *testing.T, home string) *core.AppConfig {
	t.Helper()

	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"
	cfg.ConfigDir = filepath.Join(home, "mihomo")
	if err := os.MkdirAll(cfg.ConfigDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := app.SaveAppConfig(cfg); err != nil {
		t.Fatalf("SaveAppConfig() error = %v", err)
	}
	return cfg
}

func TestBackupDirUsesConfigHome(t *testing.T) {
	home := setupCmdTestHome(t)

	got, err := BackupDir()
	if err != nil {
		t.Fatalf("BackupDir() error = %v", err)
	}

	want := backupDirForHome(home)
	if got != want {
		t.Fatalf("BackupDir() = %q, want %q", got, want)
	}
}

func TestRunBackupCreatesMihomoAndClashctlBackups(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := writeTestAppConfig(t, home)

	if err := os.WriteFile(mihomoConfigPath(cfg), []byte("mixed-port: 7890\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := runBackup(nil, nil, cfg); err != nil {
		t.Fatalf("runBackup() error = %v", err)
	}

	entries, err := os.ReadDir(backupDirForHome(home))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	var sawMihomoBackup bool
	var sawClashctlBackup bool
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "config-") && strings.HasSuffix(name, ".yaml") {
			sawMihomoBackup = true
		}
		if strings.HasPrefix(name, "clashctl-") && strings.HasSuffix(name, ".yaml") {
			sawClashctlBackup = true
		}
	}

	if !sawMihomoBackup || !sawClashctlBackup {
		t.Fatalf("backup entries missing expected files: mihomo=%v clashctl=%v", sawMihomoBackup, sawClashctlBackup)
	}
}

func TestRunRestoreRestoresMihomoConfig(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := writeTestAppConfig(t, home)
	targetPath := mihomoConfigPath(cfg)

	if err := os.WriteFile(targetPath, []byte("mixed-port: 7890\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backupDir := backupDirForHome(home)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	backupName := "config-restore.yaml"
	want := []byte("mixed-port: 9090\n")
	if err := os.WriteFile(filepath.Join(backupDir, backupName), want, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := runRestore(nil, []string{backupName}, cfg); err != nil {
		t.Fatalf("runRestore() error = %v", err)
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("restored content = %q, want %q", string(got), string(want))
	}

	matches, err := filepath.Glob(targetPath + ".bak.*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected backup of original config before restore")
	}
}

func TestRunRestoreRestoresClashctlConfig(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := writeTestAppConfig(t, home)

	targetPath, err := app.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	backupDir := backupDirForHome(home)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	backupName := "clashctl-restore.yaml"
	want := []byte("mode: tun\nconfig_dir: /tmp/mihomo\n")
	if err := os.WriteFile(filepath.Join(backupDir, backupName), want, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := runRestore(nil, []string{backupName}, cfg); err != nil {
		t.Fatalf("runRestore() error = %v", err)
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("restored clashctl config = %q, want %q", string(got), string(want))
	}
}

func TestListBackupEntriesSortedNewestFirst(t *testing.T) {
	backupDir := t.TempDir()
	oldPath := filepath.Join(backupDir, "old.yaml")
	newPath := filepath.Join(backupDir, "new.yaml")
	if err := os.WriteFile(oldPath, []byte("a"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(newPath, []byte("b"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes(old) error = %v", err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatalf("Chtimes(new) error = %v", err)
	}

	entries, err := listBackupEntries(backupDir)
	if err != nil {
		t.Fatalf("listBackupEntries() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Name != "new.yaml" || entries[1].Name != "old.yaml" {
		t.Fatalf("entries order = %#v", entries)
	}
}

func TestCreateBackupReportIncludesWarningsWhenConfigsMissing(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := core.DefaultAppConfig()
	cfg.ConfigDir = filepath.Join(home, "missing-mihomo")

	report, err := createBackupReport(cfg, backupDirForHome(home), time.Now())
	if err != nil {
		t.Fatalf("createBackupReport() error = %v", err)
	}
	if len(report.Items) != 2 {
		t.Fatalf("len(report.Items) = %d, want 2", len(report.Items))
	}
	for _, item := range report.Items {
		if item.Created {
			t.Fatalf("item should not be created when source is missing: %#v", item)
		}
		if item.Warning == "" {
			t.Fatalf("item should include warning when source is missing: %#v", item)
		}
	}
}

func TestRunRestoreCreatesMissingTargetDir(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"
	cfg.ConfigDir = filepath.Join(home, "nested", "mihomo")
	if err := app.SaveAppConfig(cfg); err != nil {
		t.Fatalf("SaveAppConfig() error = %v", err)
	}

	backupDir := backupDirForHome(home)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	backupName := "config-createdir.yaml"
	want := []byte("mixed-port: 9090\n")
	if err := os.WriteFile(filepath.Join(backupDir, backupName), want, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := runRestore(nil, []string{backupName}, cfg); err != nil {
		t.Fatalf("runRestore() error = %v", err)
	}

	got, err := os.ReadFile(mihomoConfigPath(cfg))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("restored content = %q, want %q", string(got), string(want))
	}
}

func TestRunRestoreRejectsPathTraversalBackupName(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := writeTestAppConfig(t, home)

	backupDir := backupDirForHome(home)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	outsidePath := filepath.Join(home, "outside.yaml")
	if err := os.WriteFile(outsidePath, []byte("mixed-port: 10000\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := runRestore(nil, []string{"../outside.yaml"}, cfg)
	if err == nil {
		t.Fatal("runRestore() should reject path traversal backup names")
	}
	if !strings.Contains(err.Error(), "备份文件名不合法") {
		t.Fatalf("runRestore() error = %v, want illegal backup name", err)
	}
}

func TestRunRestoreRejectsInvalidMihomoBackup(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := writeTestAppConfig(t, home)
	targetPath := mihomoConfigPath(cfg)
	original := []byte("mixed-port: 7890\n")
	if err := os.WriteFile(targetPath, original, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backupDir := backupDirForHome(home)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	backupName := "config-invalid.yaml"
	if err := os.WriteFile(filepath.Join(backupDir, backupName), []byte("mixed-port: [broken"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := runRestore(nil, []string{backupName}, cfg)
	if err == nil {
		t.Fatal("runRestore() should reject invalid backup content")
	}
	if !strings.Contains(err.Error(), "备份文件校验失败") {
		t.Fatalf("runRestore() error = %v", err)
	}

	got, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("target content = %q, want %q", string(got), string(original))
	}
}

func TestRunRestoreRejectsOversizedBackup(t *testing.T) {
	home := setupCmdTestHome(t)
	cfg := writeTestAppConfig(t, home)

	backupDir := backupDirForHome(home)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	backupName := "config-oversized.yaml"
	oversized := strings.Repeat("a", config.MaxConfigFileSize+1)
	if err := os.WriteFile(filepath.Join(backupDir, backupName), []byte(oversized), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := runRestore(nil, []string{backupName}, cfg)
	if err == nil {
		t.Fatal("runRestore() should reject oversized backups")
	}
	if !strings.Contains(err.Error(), "配置文件过大") {
		t.Fatalf("runRestore() error = %v", err)
	}
}
