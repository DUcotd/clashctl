package mihomo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePIDFile_RejectsSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "real_pid")
	symlink := filepath.Join(tmpDir, "clashctl.pid")

	os.WriteFile(target, []byte("0"), 0644)
	os.Symlink(target, symlink)

	err := writePIDFile(tmpDir, 12345)
	if err == nil {
		t.Error("writePIDFile should reject symlink")
	}
	if !strings.Contains(err.Error(), "符号链接") {
		t.Errorf("Error should mention symlink: %v", err)
	}
}

func TestWritePIDFile_WritesNormalFile(t *testing.T) {
	tmpDir := t.TempDir()

	err := writePIDFile(tmpDir, 12345)
	if err != nil {
		t.Errorf("writePIDFile should accept normal path: %v", err)
	}

	content, _ := os.ReadFile(pidFilePath(tmpDir))
	if string(content) != "12345" {
		t.Errorf("Expected PID 12345, got: %s", string(content))
	}
}
