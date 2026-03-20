// Package system provides filesystem utilities.
package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DirWritable checks if a directory is writable by creating and removing a temp file.
func DirWritable(path string) error {
	testFile := fmt.Sprintf("%s/.clashctl_write_test", path)
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("目录 %s 不可写: %w", path, err)
	}
	f.Close()
	os.Remove(testFile)
	return nil
}

// EnsureDir creates a directory with 0755 permissions if it doesn't exist.
func EnsureDir(path string) error {
	if DirExists(path) {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

// StatFile checks if a file exists and returns any error.
func StatFile(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// dangerousPaths are system paths that should not be overwritten by user output
var dangerousPaths = []string{
	"/etc/passwd",
	"/etc/shadow",
	"/etc/sudoers",
	"/etc/group",
	"/etc/gshadow",
	"/etc/hosts",
	"/etc/fstab",
	"/etc/crontab",
	"/etc/profile",
	"/etc/bash.bashrc",
	"/etc/environment",
	"/boot",
	"/bin",
	"/sbin",
	"/usr/bin",
	"/usr/sbin",
	"/lib",
	"/lib64",
	"/usr/lib",
	"/usr/lib64",
}

// ValidateOutputPath validates that an output path is safe to write to.
// It prevents path traversal attacks and writing to dangerous system paths.
func ValidateOutputPath(path string) error {
	if path == "" {
		return fmt.Errorf("输出路径不能为空")
	}

	// Clean the path to normalize it (resolve . and ..)
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		// After cleaning, if it still contains .., it's trying to go up
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return fmt.Errorf("无法解析路径: %w", err)
		}
		// Verify the resolved path doesn't escape expected boundaries
		if !strings.HasPrefix(absPath, "/") {
			return fmt.Errorf("路径解析异常: %s", absPath)
		}
	}

	// Get absolute path for comparison
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("无法解析绝对路径: %w", err)
	}

	// Check against dangerous system paths
	for _, dangerous := range dangerousPaths {
		if absPath == dangerous || strings.HasPrefix(absPath, dangerous+"/") {
			return fmt.Errorf("不允许写入系统路径: %s", absPath)
		}
	}

	// Allow writing to these safe directories
	safePrefixes := []string{
		"/etc/mihomo",
		"/tmp",
		"/var/tmp",
	}

	// Check if path is in a safe location or is relative
	isSafe := false
	for _, prefix := range safePrefixes {
		if strings.HasPrefix(absPath, prefix) {
			isSafe = true
			break
		}
	}

	// Allow current directory and relative paths
	if !strings.HasPrefix(absPath, "/etc/") && !strings.HasPrefix(absPath, "/usr/") {
		isSafe = true
	}

	// Special case: allow /etc/mihomo/ paths
	if strings.HasPrefix(absPath, "/etc/mihomo") {
		isSafe = true
	}

	if !isSafe {
		// For /etc paths, only allow /etc/mihomo
		if strings.HasPrefix(absPath, "/etc/") && !strings.HasPrefix(absPath, "/etc/mihomo") {
			return fmt.Errorf("不允许写入 /etc/ 下的路径（仅允许 /etc/mihomo/）: %s", absPath)
		}
	}

	return nil
}
