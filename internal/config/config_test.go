package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"clashctl/internal/core"
)

func TestBackupFile(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config.yaml")

	// Write test content
	testContent := []byte("test: content")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Backup
	backupPath, err := BackupFile(testFile)
	if err != nil {
		t.Fatalf("BackupFile failed: %v", err)
	}
	if backupPath == "" {
		t.Fatal("expected backup path, got empty")
	}

	// Verify backup exists and has correct content
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("can't read backup: %v", err)
	}
	if string(backupData) != string(testContent) {
		t.Errorf("backup content = %q, want %q", string(backupData), string(testContent))
	}

	// Verify original still exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("original file was removed")
	}
}

func TestBackupFileNoExist(t *testing.T) {
	backupPath, err := BackupFile("/nonexistent/file.yaml")
	if err != nil {
		t.Fatalf("BackupFile should not error for nonexistent file: %v", err)
	}
	if backupPath != "" {
		t.Errorf("expected empty backup path for nonexistent file, got %q", backupPath)
	}
}

func TestWriteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "config.yaml")

	data := []byte("test: data")
	if err := WriteConfig(path, data); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Verify written
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("can't read written file: %v", err)
	}
	if string(read) != string(data) {
		t.Errorf("content = %q, want %q", string(read), string(data))
	}
}

func TestValidateYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid YAML
	validPath := filepath.Join(tmpDir, "valid.yaml")
	os.WriteFile(validPath, []byte("key: value\nlist:\n  - a\n  - b\n"), 0644)
	if err := ValidateYAML(validPath); err != nil {
		t.Errorf("ValidateYAML(valid) failed: %v", err)
	}

	// Invalid YAML
	invalidPath := filepath.Join(tmpDir, "invalid.yaml")
	os.WriteFile(invalidPath, []byte("key: [unclosed"), 0644)
	if err := ValidateYAML(invalidPath); err == nil {
		t.Error("ValidateYAML(invalid) should have failed")
	}
}

func TestSaveMihomoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"
	m := core.BuildMihomoConfig(cfg)

	backup, err := SaveMihomoConfig(m, path)
	if err != nil {
		t.Fatalf("SaveMihomoConfig failed: %v", err)
	}
	if backup != "" {
		t.Error("first save should not create backup")
	}

	// Save again to test backup
	backup, err = SaveMihomoConfig(m, path)
	if err != nil {
		t.Fatalf("second SaveMihomoConfig failed: %v", err)
	}
	if backup == "" {
		t.Error("second save should create backup")
	}

	// Verify backup exists
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		t.Errorf("backup file %s doesn't exist", backup)
	}
}

func TestSaveRawYAMLRejectsInvalidInputWithoutOverwriting(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")
	original := []byte("mixed-port: 7890\n")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backup, err := SaveRawYAML([]byte("key: [broken"), path)
	if err == nil {
		t.Fatal("SaveRawYAML() should reject invalid YAML")
	}
	if backup != "" {
		t.Fatalf("backup = %q, want empty on pre-write validation failure", backup)
	}

	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("config content = %q, want %q", string(got), string(original))
	}
}

func TestValidateProxyCountRejectsOversizedInlineProxyList(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("proxies:\n")
	for i := 0; i < MaxProxyCount+1; i++ {
		builder.WriteString("  - name: node\n")
	}

	err := ValidateProxyCount([]byte(builder.String()))
	if err == nil {
		t.Fatal("ValidateProxyCount() should reject oversized inline proxy lists")
	}
}

func TestValidateProxyCountIgnoresProxyProviderURLs(t *testing.T) {
	data := []byte(`
proxy-providers:
  airport:
    type: http
    url: https://example.com/provider.yaml
`)

	if err := ValidateProxyCount(data); err != nil {
		t.Fatalf("ValidateProxyCount() error = %v, want nil for provider-only config", err)
	}
}

func TestBackupFileRejectsOversizedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	oversized := strings.Repeat("a", MaxConfigFileSize+1)
	if err := os.WriteFile(path, []byte(oversized), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := BackupFile(path)
	if err == nil {
		t.Fatal("BackupFile() should reject oversized configs")
	}
	if !strings.Contains(err.Error(), "配置文件过大") {
		t.Fatalf("BackupFile() error = %v", err)
	}
}
