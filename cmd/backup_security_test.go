package cmd

import (
	"path/filepath"
	"testing"
)

func TestResolveBackupPath_RejectsNonYAMLExtension(t *testing.T) {
	backupDir := "/tmp/test-backups"

	tests := []string{
		"config.txt",
		"config.json",
		"config.sh",
		"config.exe",
		"config",
	}

	for _, name := range tests {
		_, err := resolveBackupPath(backupDir, name)
		if err == nil {
			t.Errorf("resolveBackupPath should reject: %s", name)
		}
	}
}

func TestResolveBackupPath_AcceptsYAMLExtension(t *testing.T) {
	backupDir := "/tmp/test-backups"

	tests := []string{
		"config-20260101-120000.yaml",
		"clashctl-20260101-120000.yml",
	}

	for _, name := range tests {
		_, err := resolveBackupPath(backupDir, name)
		if err != nil {
			t.Errorf("resolveBackupPath should accept: %s, got: %v", name, err)
		}
	}
}

func TestResolveBackupPath_RejectsPathTraversal(t *testing.T) {
	backupDir := "/tmp/test-backups"

	tests := []string{
		"../../../etc/passwd.yaml",
		"config/../../etc/shadow.yaml",
	}

	for _, name := range tests {
		_, err := resolveBackupPath(backupDir, name)
		if err == nil {
			t.Errorf("resolveBackupPath should reject traversal: %s", name)
		}
	}
}

func TestResolveBackupPath_RejectsEmptyName(t *testing.T) {
	_, err := resolveBackupPath("/tmp", "")
	if err == nil {
		t.Error("resolveBackupPath should reject empty name")
	}
}

func TestResolveBackupPath_RejectsAbsoluteName(t *testing.T) {
	_, err := resolveBackupPath("/tmp", "/etc/config.yaml")
	if err == nil {
		t.Error("resolveBackupPath should reject absolute path in name")
	}
}

func TestResolveBackupPath_NormalizesPath(t *testing.T) {
	backupDir := "/tmp/test-backups"
	name := "config.yaml"

	path, err := resolveBackupPath(backupDir, name)
	if err != nil {
		t.Fatalf("resolveBackupPath failed: %v", err)
	}

	expected := filepath.Join(backupDir, name)
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}
