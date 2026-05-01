// Package app provides audit logging for clashctl operations.
package app

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"clashctl/internal/system"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LogLevelInfo          LogLevel = "INFO"
	LogLevelWarn          LogLevel = "WARN"
	LogLevelError         LogLevel = "ERROR"
	maxRecentLogReadBytes          = 1 * 1024 * 1024
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
		regex       *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9._~+/=-]+`), `${1}[REDACTED]`},
		{regexp.MustCompile(`(?i)([a-z0-9._-]*(?:password|passwd|pwd|token|secret|api[_-]?key|key|auth)[a-z0-9._-]*\s*[=:]\s*)[^\s&;]+`), `${1}[REDACTED]`},
		{regexp.MustCompile(`(?i)(ghp_|gho_|ghu_|ghs_|ghr_)[A-Za-z0-9_]+`), `[REDACTED]`},
		{regexp.MustCompile(`(?i)sk-(?:proj-|live-|test-)?[A-Za-z0-9_-]+`), `[REDACTED]`},
		{regexp.MustCompile(`(?i)(xoxb-|xoxp-|xoxa-|xoxr-)[A-Za-z0-9-]+`), `[REDACTED]`},
		{regexp.MustCompile(`(?i)(ey[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,})`), `[JWT_REDACTED]`},
	}

	for _, p := range passwordPatterns {
		result = p.regex.ReplaceAllString(result, p.replacement)
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
	data, err := readRecentLogTail(logFile, maxRecentLogReadBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lines := system.SplitLines(string(data))
	if len(lines) <= count {
		return lines, nil
	}
	return lines[len(lines)-count:], nil
}

func readRecentLogTail(path string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("invalid log read limit: %d", maxBytes)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	offset := int64(0)
	if info.Size() > maxBytes {
		offset = info.Size() - maxBytes
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(io.LimitReader(f, maxBytes))
	if err != nil {
		return nil, err
	}
	if offset > 0 {
		if idx := strings.IndexByte(string(data), '\n'); idx >= 0 {
			data = data[idx+1:]
		} else {
			return nil, nil
		}
	}
	return data, nil
}
