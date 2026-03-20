// Package app provides audit logging for clashctl operations.
package app

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// sanitizeForLog removes sensitive information from log strings.
func sanitizeForLog(input string) string {
	result := input

	// Sanitize URLs - keep only scheme and host
	urlRegex := regexp.MustCompile(`(https?://)([^/]*@)?([^/\s]+)([^\s]*)`)
	result = urlRegex.ReplaceAllStringFunc(result, func(match string) string {
		u, err := url.Parse(match)
		if err != nil {
			return "[URL_REDACTED]"
		}
		// Keep scheme and host, redact path and query
		sanitized := u.Scheme + "://" + u.Host
		if u.Path != "" || u.RawQuery != "" {
			sanitized += "/[PATH_REDACTED]"
		}
		return sanitized
	})

	// Sanitize UUIDs
	uuidRegex := regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	result = uuidRegex.ReplaceAllString(result, "[UUID_REDACTED]")

	// Sanitize passwords/tokens (common patterns)
	passwordPatterns := []struct {
		regex *regexp.Regexp
	}{
		{regexp.MustCompile(`(?i)(password|passwd|pwd|token|secret|key|auth)[=:\s]+[^\s&;]+`)},
		{regexp.MustCompile(`(?i)(ghp_|gho_|ghu_|ghs_|ghr_)[A-Za-z0-9_]+`)},  // GitHub tokens
		{regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]+`)},              // Bearer tokens
	}

	for _, p := range passwordPatterns {
		result = p.regex.ReplaceAllStringFunc(result, func(match string) string {
			// Find the key part and redact the value
			for _, prefix := range []string{"password", "passwd", "pwd", "token", "secret", "key", "auth", "Bearer", "ghp_", "gho_", "ghu_", "ghs_", "ghr_"} {
				idx := strings.Index(strings.ToLower(match), strings.ToLower(prefix))
				if idx >= 0 {
					return match[:idx+len(prefix)] + "=[REDACTED]"
				}
			}
			return "[REDACTED]"
		})
	}

	return result
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

	// Sanitize sensitive data before logging
	sanitizedOp := sanitizeForLog(operation)
	sanitizedDetail := sanitizeForLog(detail)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s - %s", timestamp, level, sanitizedOp)
	if sanitizedDetail != "" {
		logEntry += " | " + sanitizedDetail
	}
	logEntry += "\n"

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
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
