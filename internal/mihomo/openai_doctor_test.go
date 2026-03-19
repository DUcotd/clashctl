package mihomo

import (
	"strings"
	"testing"
)

func TestExtractCountry(t *testing.T) {
	country, code := extractCountry("United States (US) - 203.0.113.8 via https://ifconfig.co/json")
	if country != "United States" || code != "US" {
		t.Fatalf("extractCountry() = %q, %q", country, code)
	}
}

func TestBuildOpenAIHints(t *testing.T) {
	hints := buildOpenAIHints(false, true, true, false, true, false, "China", "CN", "United States", "US")
	joined := strings.Join(hints, "\n")

	for _, want := range []string{
		"当前 shell 没有代理环境",
		"直连 auth.openai.com 正常、代理路径失败",
		"直连 api.openai.com 正常、代理路径失败",
		"直连出口是 China (CN)，代理出口是 United States (US)",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("hints missing %q in:\n%s", want, joined)
		}
	}
}

func TestFormatCountry(t *testing.T) {
	if got := formatCountry("United States", "US"); got != "United States (US)" {
		t.Fatalf("formatCountry() = %q", got)
	}
	if got := formatCountry("", "US"); got != "US" {
		t.Fatalf("formatCountry() = %q", got)
	}
}
