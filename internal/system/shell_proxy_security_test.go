package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertManagedBlock_RejectsSymlink(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home dir")
	}
	configDir := filepath.Join(home, ".config", "clashctl")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Skip("Cannot create config dir")
	}
	tmpDir, err := os.MkdirTemp(configDir, "test-*")
	if err != nil {
		t.Skip("Cannot create temp in config dir")
	}
	defer os.RemoveAll(tmpDir)

	target := filepath.Join(tmpDir, "target.txt")
	symlink := filepath.Join(tmpDir, "shell_profile")

	os.WriteFile(target, []byte("target content"), 0644)
	os.Symlink(target, symlink)

	err = upsertManagedBlock(symlink, "test block")
	if err == nil {
		t.Error("upsertManagedBlock should reject symlink")
	}
	if !strings.Contains(err.Error(), "符号链接") {
		t.Errorf("Error should mention symlink: %v", err)
	}
}

func TestUpsertManagedBlock_RejectsWorldWritable(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home dir")
	}
	configDir := filepath.Join(home, ".config", "clashctl")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Skip("Cannot create config dir")
	}
	tmpDir, err := os.MkdirTemp(configDir, "test-*")
	if err != nil {
		t.Skip("Cannot create temp in config dir")
	}
	defer os.RemoveAll(tmpDir)

	profile := filepath.Join(tmpDir, ".bashrc")
	os.WriteFile(profile, []byte("existing content"), 0644)
	os.Chmod(profile, 0666)

	err = upsertManagedBlock(profile, "test block")
	if err == nil {
		t.Error("upsertManagedBlock should reject world-writable files")
	}
	if !strings.Contains(err.Error(), "全局可写") {
		t.Errorf("Error should mention world-writable: %v", err)
	}
}

func TestUpsertManagedBlock_WritesNormalFile(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home dir")
	}
	configDir := filepath.Join(home, ".config", "clashctl")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Skip("Cannot create config dir")
	}
	tmpDir, err := os.MkdirTemp(configDir, "test-*")
	if err != nil {
		t.Skip("Cannot create temp in config dir")
	}
	defer os.RemoveAll(tmpDir)

	profile := filepath.Join(tmpDir, ".bashrc")

	err = upsertManagedBlock(profile, "test block")
	if err != nil {
		t.Errorf("upsertManagedBlock should accept normal file: %v", err)
	}

	content, _ := os.ReadFile(profile)
	if !strings.Contains(string(content), "test block") {
		t.Error("Block should be written to file")
	}
}
