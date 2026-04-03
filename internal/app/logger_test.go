package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    []string // Should contain these after sanitization
		notContains []string // Should NOT contain these
	}{
		{
			name:        "URL sanitization",
			input:       "Fetching https://example.com/sub/path?token=secret",
			contains:    []string{"https://example.com"},
			notContains: []string{"secret", "token"},
		},
		{
			name:        "UUID sanitization",
			input:       "UUID: 550e8400-e29b-41d4-a716-446655440000",
			contains:    []string{"[UUID_REDACTED]"},
			notContains: []string{"550e8400"},
		},
		{
			name:        "password sanitization",
			input:       "password=mysecretpassword",
			contains:    []string{"[REDACTED]"},
			notContains: []string{"mysecretpassword"},
		},
		{
			name:        "GitHub token sanitization",
			input:       "token=ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			contains:    []string{"[REDACTED]"},
			notContains: []string{"ghp_1234567890"},
		},
		{
			name:        "Bearer token sanitization",
			input:       "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			contains:    []string{"[REDACTED]"},
			notContains: []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		},
		{
			name:        "api key sanitization",
			input:       "api_key=super-secret-value",
			contains:    []string{"api_key=[REDACTED]"},
			notContains: []string{"super-secret-value"},
		},
		{
			name:        "session token sanitization",
			input:       "__Secure-next-auth.session-token=very-secret-cookie",
			contains:    []string{"session-token=[REDACTED]"},
			notContains: []string{"very-secret-cookie"},
		},
		{
			name:        "openai key sanitization",
			input:       "using sk-proj-abc123secret456 for request",
			contains:    []string{"[REDACTED]"},
			notContains: []string{"sk-proj-abc123secret456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeForLog(tt.input)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("sanitizeForLog(%q) = %q, want to contain %q", tt.input, result, s)
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(result, s) {
					t.Errorf("sanitizeForLog(%q) = %q, should NOT contain %q", tt.input, result, s)
				}
			}
		})
	}
}

func TestGetRecentLogsReadsTailOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logDir, err := LogDir()
	if err != nil {
		t.Fatalf("LogDir() error = %v", err)
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")

	var builder strings.Builder
	for builder.Len() < maxRecentLogReadBytes+4096 {
		builder.WriteString("old-line-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n")
	}
	builder.WriteString("tail-one\n")
	builder.WriteString("tail-two\n")
	if err := os.WriteFile(logFile, []byte(builder.String()), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	logs, err := GetRecentLogs(2)
	if err != nil {
		t.Fatalf("GetRecentLogs() error = %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("len(logs) = %d, want 2 (%v)", len(logs), logs)
	}
	if logs[0] != "tail-one" || logs[1] != "tail-two" {
		t.Fatalf("GetRecentLogs() = %v, want tail lines", logs)
	}
}
