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
