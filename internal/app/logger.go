// Package app provides audit logging for clashctl operations.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogDir returns the clashctl log directory.
func LogDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户主目录: %w", err)
	}
	return filepath.Join(home, ".config", "clashctl", "logs"), nil
}

// EnsureLogDir creates the log directory if it doesn't exist.
func EnsureLogDir() error {
	dir, err := LogDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

// LogOperation records an operation to the audit log.
func LogOperation(operation string, success bool, detail string) error {
	if err := EnsureLogDir(); err != nil {
		return err
	}

	logDir, err := LogDir()
	if err != nil {
		return err
	}

	// Use daily log files
	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")

	level := LogLevelInfo
	if !success {
		level = LogLevelError
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s - %s", timestamp, level, operation)
	if detail != "" {
		logEntry += " | " + detail
	}
	logEntry += "\n"

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(logEntry)
	return err
}

// LogInfo logs an informational operation.
func LogInfo(operation string, detail string) error {
	return LogOperation(operation, true, detail)
}

// LogError logs a failed operation.
func LogError(operation string, detail string) error {
	return LogOperation(operation, false, detail)
}

// GetRecentLogs returns the most recent log entries.
func GetRecentLogs(count int) ([]string, error) {
	logDir, err := LogDir()
	if err != nil {
		return nil, err
	}

	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lines := splitLines(string(data))
	if len(lines) <= count {
		return lines, nil
	}
	return lines[len(lines)-count:], nil
}

func splitLines(text string) []string {
	var lines []string
	start := 0
	for i, c := range text {
		if c == '\n' {
			line := text[start:i]
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(text) {
		line := text[start:]
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
