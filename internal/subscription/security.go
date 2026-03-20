// Package subscription provides security validation for subscription content.
package subscription

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// dangerousFields are configuration keys that should be blocked from remote subscriptions.
// These fields could be used for malicious purposes (DNS hijacking, script execution, etc.)
var dangerousFields = map[string]bool{
	// Script execution risks
	"script":           true,
	"script-context":   true,
	"rule-providers":   false, // Allow but warn

	// DNS hijacking risks
	"dns":              true,
	"fake-ip-filter":   false, // Allow with validation
	"nameserver":       false, // Allow with validation

	// TUN mode risks (requires explicit user consent)
	"tun":              true,

	// External process risks
	"external-ui":      true,
	"external-ui-url":  true,

	// File system risks
	"mmdb":             true,
	"geo-auto-update":  true,
	"geodata-mode":     false,
}

// allowedTopLevelFields defines the safe configuration structure
var allowedTopLevelFields = map[string]bool{
	"mixed-port":        true,
	"allow-lan":         true,
	"log-level":         true,
	"mode":              true,
	"external-controller": true,
	"proxies":           true,
	"proxy-providers":   true,
	"proxy-groups":      true,
	"rules":             true,
	"hosts":             false, // Careful with hosts
	"bind-address":      true,
	"authentication":    false,
	"skip-auth-prefixes": false,
}

// ValidateYAMLSecurity checks a YAML document for dangerous fields.
// Returns a list of warnings and an error if critical threats are found.
func ValidateYAMLSecurity(data []byte, allowDangerous bool) ([]string, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("YAML 解析失败: %w", err)
	}

	var warnings []string
	var criticalErrors []string

	for key := range doc {
		lowerKey := strings.ToLower(key)

		// Check dangerous fields
		if blocked, exists := dangerousFields[lowerKey]; exists && blocked {
			if allowDangerous {
				warnings = append(warnings, fmt.Sprintf("⚠️ 允许高风险字段: %s (使用了 --unsafe 选项)", key))
			} else {
				criticalErrors = append(criticalErrors, fmt.Sprintf("❌ 检测到高风险字段: %s (使用 --unsafe 可强制启用)", key))
			}
		}

		// Warn about unknown fields
		if !allowedTopLevelFields[lowerKey] && !exists {
			warnings = append(warnings, fmt.Sprintf("⚠️ 未知配置字段: %s", key))
		}
	}

	// Check for script content in nested fields
	if err := checkNestedScripts(doc, &warnings, &criticalErrors, allowDangerous); err != nil {
		return warnings, err
	}

	if len(criticalErrors) > 0 {
		return warnings, fmt.Errorf("订阅内容包含危险配置:\n%s", strings.Join(criticalErrors, "\n"))
	}

	return warnings, nil
}

// checkNestedScripts recursively checks for script content
func checkNestedScripts(doc map[string]any, warnings *[]string, errors *[]string, allowDangerous bool) error {
	for key, value := range doc {
		lowerKey := strings.ToLower(key)

		// Check proxy-providers for script content
		if lowerKey == "proxy-providers" {
			if providers, ok := value.(map[string]any); ok {
				for name, provider := range providers {
					if p, ok := provider.(map[string]any); ok {
						if script, exists := p["script"]; exists {
							scriptStr := fmt.Sprintf("%v", script)
							if containsDangerousScript(scriptStr) {
								if allowDangerous {
									*warnings = append(*warnings, fmt.Sprintf("⚠️ Provider %s 包含脚本内容", name))
								} else {
									*errors = append(*errors, fmt.Sprintf("❌ Provider %s 包含危险脚本", name))
								}
							}
						}
					}
				}
			}
		}

		// Check rules for script-based rules
		if lowerKey == "rules" {
			if rules, ok := value.([]any); ok {
				for i, rule := range rules {
					ruleStr := fmt.Sprintf("%v", rule)
					if strings.Contains(strings.ToLower(ruleStr), "script") {
						*warnings = append(*warnings, fmt.Sprintf("⚠️ 规则 #%d 可能包含脚本: %s", i+1, truncate(ruleStr, 50)))
					}
				}
			}
		}

		// Recursively check nested maps
		if nested, ok := value.(map[string]any); ok {
			if err := checkNestedScripts(nested, warnings, errors, allowDangerous); err != nil {
				return err
			}
		}
	}
	return nil
}

// containsDangerousScript checks if a script string contains dangerous patterns
func containsDangerousScript(script string) bool {
	lower := strings.ToLower(script)
	dangerousPatterns := []string{
		"os.execute",
		"os.system",
		"io.popen",
		"require(",
		"dofile(",
		"loadfile(",
		"eval(",
		"exec(",
		"system(",
		"popen(",
		"subprocess",
		"import os",
		"import subprocess",
		"__import__",
		"Runtime.getRuntime",
		"ProcessBuilder",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// SanitizeYAML removes dangerous fields from a YAML document.
// Returns the sanitized YAML and a list of removed fields.
func SanitizeYAML(data []byte) ([]byte, []string, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("YAML 解析失败: %w", err)
	}

	var removed []string

	// Remove dangerous top-level fields
	for key := range doc {
		lowerKey := strings.ToLower(key)
		if blocked, exists := dangerousFields[lowerKey]; exists && blocked {
			delete(doc, key)
			removed = append(removed, key)
		}
	}

	// Sanitize proxy-providers
	if providers, ok := doc["proxy-providers"].(map[string]any); ok {
		for name, provider := range providers {
			if p, ok := provider.(map[string]any); ok {
				if _, hasScript := p["script"]; hasScript {
					delete(p, "script")
					removed = append(removed, fmt.Sprintf("proxy-providers.%s.script", name))
				}
			}
		}
	}

	sanitized, err := yaml.Marshal(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("YAML 序列化失败: %w", err)
	}

	return sanitized, removed, nil
}

// truncate shortens a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
