package subscription

import (
	"testing"
)

func TestSanitizeProxyGroups_RemovesInvalidURLs(t *testing.T) {
	groups := []any{
		map[string]any{
			"name": "PROXY",
			"type": "select",
			"url":  "http://192.168.1.1/health",
		},
		map[string]any{
			"name": "AUTO",
			"type": "url-test",
			"url":  "http://10.0.0.1/health",
		},
		map[string]any{
			"name":     "FALLBACK",
			"type":     "fallback",
			"url":      "https://cp.cloudflare.com/",
			"interval": 300,
		},
	}

	result := sanitizeProxyGroups(groups)
	if len(result) != 3 {
		t.Fatalf("Expected 3 groups (URLs removed but groups kept), got %d", len(result))
	}

	for _, g := range result {
		group := g.(map[string]any)
		if _, hasURL := group["url"]; hasURL {
			if group["name"] == "PROXY" || group["name"] == "AUTO" {
				t.Errorf("Group %s should have had its private URL removed", group["name"])
			}
		}
	}

	fallback := result[2].(map[string]any)
	if fallback["url"] != "https://cp.cloudflare.com/" {
		t.Errorf("FALLBACK should keep valid URL, got: %v", fallback["url"])
	}
}

func TestSanitizeProxyGroups_KeepsGroupsWithoutURL(t *testing.T) {
	groups := []any{
		map[string]any{
			"name":    "PROXY",
			"type":    "select",
			"proxies": []any{"node1", "node2"},
		},
	}

	result := sanitizeProxyGroups(groups)
	if len(result) != 1 {
		t.Errorf("Expected 1 group, got %d", len(result))
	}
}
