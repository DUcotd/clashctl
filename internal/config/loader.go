// Package config handles reading and writing clashctl configuration files.
package config

import (
	"fmt"
	"io"
	"os"
	"time"

	"clashctl/internal/system"

	"gopkg.in/yaml.v3"
)

const (
	// MaxConfigFileSize is the maximum allowed config file size (10MB)
	MaxConfigFileSize = 10 * 1024 * 1024
	// MaxProxyCount is the maximum number of proxies to prevent memory exhaustion
	MaxProxyCount = 50000
)

// BackupFile creates a timestamped backup of an existing file.
// Returns the backup path or an error.
func BackupFile(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil // nothing to back up
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.bak.%s", path, timestamp)

	data, err := ReadConfigWithLimit(path)
	if err != nil {
		return "", fmt.Errorf("读取 %s 备份失败: %w", path, err)
	}

	if err := system.WriteFileAtomic(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("写入备份到 %s 失败: %w", backupPath, err)
	}

	return backupPath, nil
}

// WriteConfig writes data to a path, creating parent directories if needed.
func WriteConfig(path string, data []byte) error {
	return WriteConfigAtomic(path, data)
}

// WriteConfigAtomic writes data to a temp file and renames it into place.
func WriteConfigAtomic(path string, data []byte) error {
	if err := system.WriteFileAtomic(path, data, 0600); err != nil {
		return fmt.Errorf("写入配置到 %s 失败: %w", path, err)
	}
	return nil
}

// ValidateYAML reads back a YAML file and checks it can be parsed.
func ValidateYAML(path string) error {
	data, err := ReadConfigWithLimit(path)
	if err != nil {
		return err
	}
	return ValidateYAMLBytes(data, path)
}

// ValidateYAMLBytes checks whether a YAML document is parseable.
func ValidateYAMLBytes(data []byte, source string) error {
	var dummy any
	if err := yaml.Unmarshal(data, &dummy); err != nil {
		return fmt.Errorf("YAML 解析错误 %s: %w", source, err)
	}

	return nil
}

// ReadConfigWithLimit reads a config file with size limit.
func ReadConfigWithLimit(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", path, err)
	}
	defer f.Close()

	// Check file size
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	if info.Size() > MaxConfigFileSize {
		return nil, fmt.Errorf("配置文件过大: %d bytes (最大允许 %d bytes)，建议拆分配置", info.Size(), MaxConfigFileSize)
	}

	// Read with limit
	data, err := io.ReadAll(io.LimitReader(f, MaxConfigFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	if len(data) > MaxConfigFileSize {
		return nil, fmt.Errorf("配置文件过大 (超过 %d bytes)，建议拆分配置", MaxConfigFileSize)
	}

	return data, nil
}

// ValidateProxyCount checks if a config has too many proxies.
func ValidateProxyCount(data []byte) error {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil // Will be caught by other validation
	}

	// Check proxies count
	if proxies, ok := doc["proxies"].([]any); ok {
		if len(proxies) > MaxProxyCount {
			return fmt.Errorf("代理节点数量过多: %d (最大允许 %d)，建议拆分配置", len(proxies), MaxProxyCount)
		}
	}

	return nil
}

// Loader provides configuration loading from files.
type Loader struct {
	Path string
}

// NewLoader creates a Loader for the given config file path.
func NewLoader(path string) *Loader {
	return &Loader{Path: path}
}

// Load reads and unmarshals the YAML config file into dest.
func (l *Loader) Load(dest any) error {
	data, err := ReadConfigWithLimit(l.Path)
	if err != nil {
		return err
	}

	// Validate proxy count before parsing
	if err := ValidateProxyCount(data); err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("解析 YAML 文件 %s 失败: %w", l.Path, err)
	}
	return nil
}
