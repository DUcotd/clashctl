package app

import (
	"strings"
	"testing"
)

func TestSanitizeForLog_JWTToken(t *testing.T) {
	input := "auth header: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	result := sanitizeForLog(input)
	if strings.Contains(result, "eyJhbG") {
		t.Errorf("JWT token should be redacted: %s", result)
	}
	if !strings.Contains(result, "[JWT_REDACTED]") {
		t.Errorf("Expected [JWT_REDACTED] in result: %s", result)
	}
}

func TestSanitizeForLog_SlackToken(t *testing.T) {
	input := "token: xoxb-REDACTED-REDACTED"
	result := sanitizeForLog(input)
	if strings.Contains(result, "xoxb-REDACTED") {
		t.Errorf("Slack token should be redacted: %s", result)
	}
}

func TestSanitizeForLog_URLPathRedacted(t *testing.T) {
	input := "request to https://api.example.com/v1/users?id=123&token=secret"
	result := sanitizeForLog(input)
	if strings.Contains(result, "/v1/users") {
		t.Errorf("URL path should be redacted: %s", result)
	}
	if !strings.Contains(result, "api.example.com") {
		t.Errorf("Host should be preserved: %s", result)
	}
}

func TestSanitizeForLog_UUID(t *testing.T) {
	input := "user id: 550e8400-e29b-41d4-a716-446655440000"
	result := sanitizeForLog(input)
	if strings.Contains(result, "550e8400") {
		t.Errorf("UUID should be redacted: %s", result)
	}
	if !strings.Contains(result, "[UUID_REDACTED]") {
		t.Errorf("Expected [UUID_REDACTED] in result: %s", result)
	}
}

func TestSanitizeForLog_BearerToken(t *testing.T) {
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.test"
	result := sanitizeForLog(input)
	if strings.Contains(result, "eyJhbG") {
		t.Errorf("Bearer token should be redacted: %s", result)
	}
}

func TestSanitizeForLog_GitHubToken(t *testing.T) {
	input := "token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef12"
	result := sanitizeForLog(input)
	if strings.Contains(result, "ghp_") {
		t.Errorf("GitHub token should be redacted: %s", result)
	}
}

func TestSanitizeForLog_OpenAIKey(t *testing.T) {
	input := "api key: sk-proj-ABCDEFghijklmnop1234567890"
	result := sanitizeForLog(input)
	if strings.Contains(result, "sk-proj") {
		t.Errorf("OpenAI key should be redacted: %s", result)
	}
}
