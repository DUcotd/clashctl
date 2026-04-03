package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceFile_RejectsSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "real_target.txt")
	symlink := filepath.Join(tmpDir, "symlink.txt")
	src := filepath.Join(tmpDir, "src.txt")

	os.WriteFile(target, []byte("original"), 0644)
	os.Symlink(target, symlink)
	os.WriteFile(src, []byte("new content"), 0644)

	err := ReplaceFile(src, symlink, ReplaceFileOptions{})
	if err == nil {
		t.Error("ReplaceFile should reject symlink replacement")
	}
	if !strings.Contains(err.Error(), "符号链接") {
		t.Errorf("Error should mention symlink: %v", err)
	}
}

func TestValidateOutputPath_RelativePathEscapes(t *testing.T) {
	err := ValidateOutputPath("../../etc/passwd")
	if err == nil {
		t.Error("ValidateOutputPath should reject escaping relative paths")
	}
}

func TestValidateOutputPath_RelativePathInAllowedRoot(t *testing.T) {
	err := ValidateOutputPath("config.yaml")
	if err != nil {
		t.Logf("ValidateOutputPath for relative path: %v (may fail depending on cwd)", err)
	}
}

func TestValidateOutputPath_DangerousPaths(t *testing.T) {
	dangerous := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/boot/grub.cfg",
		"/bin/bash",
		"/usr/bin/sudo",
	}
	for _, path := range dangerous {
		err := ValidateOutputPath(path)
		if err == nil {
			t.Errorf("ValidateOutputPath should reject: %s", path)
		}
	}
}

func TestValidateOutputPath_AllowedPaths(t *testing.T) {
	allowed := []string{
		"/etc/mihomo/config.yaml",
		"/tmp/test.yaml",
		"/var/tmp/test.yaml",
	}
	for _, path := range allowed {
		err := ValidateOutputPath(path)
		if err != nil {
			t.Errorf("ValidateOutputPath should allow: %s, got: %v", path, err)
		}
	}
}

func TestCreateSiblingTempFile_Permissions(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")

	path, err := CreateSiblingTempFile(target, ".tmp-*")
	if err != nil {
		t.Fatalf("CreateSiblingTempFile failed: %v", err)
	}
	defer os.Remove(path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected 0600 permissions, got %o", info.Mode().Perm())
	}
}
