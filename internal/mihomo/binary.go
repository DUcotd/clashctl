package mihomo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	binaryCandidateNames = []string{"mihomo", "clash-meta", "clash"}
	installedBinaryPath  = InstallPath
)

// FindBinary locates a healthy Mihomo binary in PATH or at the default install location.
func FindBinary() (string, error) {
	var invalid []string

	for _, name := range binaryCandidateNames {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}

		if _, err := validateBinary(path); err == nil {
			return path, nil
		} else {
			invalid = append(invalid, fmt.Sprintf("%s (%v)", path, err))
		}
	}

	if info, err := os.Stat(installedBinaryPath); err == nil && !info.IsDir() {
		if _, err := validateBinary(installedBinaryPath); err == nil {
			return installedBinaryPath, nil
		} else {
			invalid = append(invalid, fmt.Sprintf("%s (%v)", installedBinaryPath, err))
		}
	}

	if len(invalid) > 0 {
		return "", fmt.Errorf("未找到可用的 mihomo 可执行文件；已跳过异常候选: %s", strings.Join(invalid, "; "))
	}

	return "", fmt.Errorf("未找到 mihomo 可执行文件。请先安装 Mihomo 并确保其在 PATH 中")
}

// GetBinaryVersion returns the version string of the mihomo binary.
func GetBinaryVersion() (string, error) {
	binary, err := FindBinary()
	if err != nil {
		return "", err
	}

	version, err := validateBinary(binary)
	if err != nil {
		return "", fmt.Errorf("获取版本号失败: %w", err)
	}

	return version, nil
}

// ValidateConfig validates a Mihomo configuration file using `mihomo -t`.
// Returns nil if the config is valid, or an error with details if invalid.
func ValidateConfig(configPath string) error {
	binary, err := FindBinary()
	if err != nil {
		return fmt.Errorf("找不到 mihomo 可执行文件: %w", err)
	}

	cmd := exec.Command(binary, "-t", "-d", filepath.Dir(configPath), "-f", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("配置校验失败: %s", msg)
	}

	return nil
}

// ValidateConfigContent validates Mihomo configuration content by writing it to a temp file.
func ValidateConfigContent(content []byte, configDir string) error {
	// Create a temp file for validation
	tmpFile, err := os.CreateTemp("", "clashctl-validate-*.yaml")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}

	// Validate using mihomo -t
	binary, err := FindBinary()
	if err != nil {
		// Skip validation if mihomo is not installed
		return nil
	}

	cmd := exec.Command(binary, "-t", "-d", configDir, "-f", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("配置校验失败: %s", msg)
	}

	return nil
}

func validateBinary(path string) (string, error) {
	version, err := runBinaryCheck(path, "-v")
	if err == nil && version != "" {
		return version, nil
	}

	help, helpErr := runBinaryCheck(path, "-h")
	if helpErr == nil {
		if version != "" {
			return version, nil
		}
		return firstLine(help), nil
	}

	if err != nil {
		return "", err
	}
	return "", helpErr
}

func runBinaryCheck(path string, arg string) (string, error) {
	cmd := exec.Command(path, arg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		return "", fmt.Errorf("%s %s 失败: %s", path, arg, text)
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		return "", fmt.Errorf("%s %s 没有输出", path, arg)
	}

	return firstLine(text), nil
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = text[:idx]
	}
	if len(text) > 200 {
		text = text[:200]
	}
	return text
}
