// Package system provides filesystem utilities.
package system

import (
	"fmt"
	"os"
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
		return fmt.Errorf("directory %s is not writable: %w", path, err)
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
