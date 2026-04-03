// Package system provides filesystem utilities.
package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReplaceFileOptions controls replacement behavior.
type ReplaceFileOptions struct {
	Validate func(string) error
}

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
	f, err := os.CreateTemp(path, ".clashctl_write_test-*")
	if err != nil {
		return fmt.Errorf("目录 %s 不可写: %w", path, err)
	}
	testFile := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(testFile)
		return fmt.Errorf("关闭目录写入测试文件失败: %w", err)
	}
	_ = os.Remove(testFile)
	return nil
}

// EnsureDir creates a directory with 0755 permissions if it doesn't exist.
func EnsureDir(path string) error {
	if DirExists(path) {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

// CreateSiblingTempFile creates a temp file next to the target path and returns its path.
func CreateSiblingTempFile(targetPath, patternSuffix string) (string, error) {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	tmpFile, err := os.CreateTemp(dir, filepath.Base(targetPath)+patternSuffix)
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

// ReserveSiblingPath returns a unique path next to the target path that does not yet exist.
func ReserveSiblingPath(targetPath, patternSuffix string) (string, error) {
	reservedPath, err := CreateSiblingTempFile(targetPath, patternSuffix)
	if err != nil {
		return "", err
	}
	if err := os.Remove(reservedPath); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return reservedPath, nil
}

// WriteFileAtomic writes data to a sibling temp file and renames it into place.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	tmpPath, err := CreateSiblingTempFile(path, ".tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("打开临时文件失败: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("同步临时文件失败: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("设置临时文件权限失败: %w", err)
	}

	if err := ReplaceFile(tmpPath, path, ReplaceFileOptions{}); err != nil {
		return err
	}
	return nil
}

// ReplaceFile replaces destPath with srcPath and rolls back if validation fails.
func ReplaceFile(srcPath, destPath string, opts ReplaceFileOptions) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	backupPath, err := ReserveSiblingPath(destPath, ".bak-*")
	if err != nil {
		return fmt.Errorf("准备备份路径失败: %w", err)
	}

	hadExisting := false
	if _, err := os.Stat(destPath); err == nil {
		hadExisting = true
		if err := os.Rename(destPath, backupPath); err != nil {
			return fmt.Errorf("备份旧文件失败: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查目标文件失败: %w", err)
	}

	if err := os.Rename(srcPath, destPath); err != nil {
		if hadExisting {
			_ = os.Rename(backupPath, destPath)
		}
		return fmt.Errorf("替换文件失败: %w", err)
	}

	if opts.Validate != nil {
		if err := opts.Validate(destPath); err != nil {
			_ = os.Remove(destPath)
			if hadExisting {
				_ = os.Rename(backupPath, destPath)
			}
			return fmt.Errorf("替换后校验失败: %w", err)
		}
	}

	if hadExisting {
		_ = os.Remove(backupPath)
	}
	if err := syncDir(filepath.Dir(destPath)); err != nil {
		return fmt.Errorf("同步目标目录失败: %w", err)
	}
	return nil
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

	// Reject paths with obvious traversal attempts before cleaning
	if strings.Contains(path, "../") || strings.Contains(path, "..\\") {
		return fmt.Errorf("不允许使用路径遍历: %s", path)
	}

	// Clean the path to normalize it (resolve . and ..)
	cleanPath := filepath.Clean(path)
	isRelative := !filepath.IsAbs(cleanPath)

	// After cleaning, check if it's trying to escape to parent directories
	// This catches cases like "/etc/mihomo/../../../etc/passwd"
	if strings.HasPrefix(cleanPath, "../") {
		return fmt.Errorf("不允许使用路径遍历: %s", path)
	}

	// Get absolute path for comparison
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("无法解析绝对路径: %w", err)
	}
	resolvedPath, err := resolvePathForWrite(absPath)
	if err != nil {
		return fmt.Errorf("无法解析输出路径: %w", err)
	}

	// Check against dangerous system paths
	for _, dangerous := range dangerousPaths {
		if resolvedPath == dangerous || strings.HasPrefix(resolvedPath, dangerous+"/") {
			return fmt.Errorf("不允许写入系统路径: %s", resolvedPath)
		}
	}

	if isRelative {
		return nil
	}

	for _, root := range allowedOutputRoots() {
		if pathWithinRoot(resolvedPath, root) {
			return nil
		}
	}

	return fmt.Errorf("输出路径必须位于允许的目录中: %s", resolvedPath)
}

func resolvePathForWrite(path string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved, nil
	}

	current := path
	var relParts []string
	for {
		if _, err := os.Stat(current); err == nil {
			resolvedCurrent, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			parts := append([]string{resolvedCurrent}, relParts...)
			return filepath.Join(parts...), nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		relParts = append([]string{filepath.Base(current)}, relParts...)
		current = parent
	}
	return path, nil
}

func syncDir(path string) error {
	dirHandle, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dirHandle.Close()
	return dirHandle.Sync()
}

func allowedOutputRoots() []string {
	roots := []string{"/etc/mihomo", "/tmp", "/var/tmp"}

	if home, err := os.UserHomeDir(); err == nil {
		configRoot := filepath.Join(home, ".config", "clashctl")
		if resolved, err := resolvePathForWrite(configRoot); err == nil && strings.TrimSpace(resolved) != "/" {
			roots = append(roots, resolved)
		}
	}

	return roots
}

func pathWithinRoot(path, root string) bool {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "/" {
		return false
	}
	path = filepath.Clean(path)
	return path == root || strings.HasPrefix(path, root+string(os.PathSeparator))
}
