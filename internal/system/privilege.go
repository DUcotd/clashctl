// Package system provides privilege and permission utilities.
package system

import (
	"fmt"
	"os"
)

// IsRoot checks if the current process is running as root.
func IsRoot() bool {
	return os.Geteuid() == 0
}

// RequireRoot returns an error if not running as root.
func RequireRoot() error {
	if !IsRoot() {
		return fmt.Errorf("此操作需要 root 权限，请使用 sudo 运行")
	}
	return nil
}

// RequireRootForOperation returns an error with a specific operation name.
func RequireRootForOperation(operation string) error {
	if !IsRoot() {
		return fmt.Errorf("操作 %q 需要 root 权限，请使用 sudo 运行", operation)
	}
	return nil
}

// SuggestSudo returns a message suggesting to use sudo.
func SuggestSudo(command string) string {
	return fmt.Sprintf("请使用 sudo 运行: sudo %s", command)
}

// CanWritePath checks if the current user can write to a path.
func CanWritePath(path string) error {
	// Try to open the file for writing (or create it)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("没有写入 %s 的权限", path)
		}
		return err
	}
	f.Close()
	// Clean up if we created a new file
	os.Remove(path)
	return nil
}
