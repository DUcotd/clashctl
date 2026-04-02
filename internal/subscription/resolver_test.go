package subscription

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"clashctl/internal/core"
	"clashctl/internal/system"
)

func TestResolveContentStaticPlan(t *testing.T) {
	resolver := NewResolver()
	cfg := core.DefaultAppConfig()
	input := []byte("vless://uuid@example.com:443?type=tcp&security=reality&sni=music.apple.com&pbk=pub&sid=short#node-a\n" +
		"trojan://pass@host.example:443?type=ws&host=cdn.example&path=%2Fws#node-b\n")

	plan, err := resolver.ResolveContent(cfg, input)
	if err != nil {
		t.Fatalf("ResolveContent() error = %v", err)
	}
	if plan.Kind != PlanKindStatic {
		t.Fatalf("Kind = %q", plan.Kind)
	}
	if plan.ProxyCount != 2 {
		t.Fatalf("ProxyCount = %d", plan.ProxyCount)
	}
	if plan.MihomoConfig == nil || len(plan.MihomoConfig.Proxies) != 2 {
		t.Fatalf("unexpected mihomo config: %#v", plan.MihomoConfig)
	}
}

func TestResolveContentYAMLPlan(t *testing.T) {
	resolver := NewResolver()
	cfg := core.DefaultAppConfig()
	input := []byte("mixed-port: 7890\nproxies:\n  - name: a\n")

	plan, err := resolver.ResolveContent(cfg, input)
	if err != nil {
		t.Fatalf("ResolveContent() error = %v", err)
	}
	if plan.Kind != PlanKindYAML {
		t.Fatalf("Kind = %q", plan.Kind)
	}
	if len(plan.RawYAML) == 0 {
		t.Fatal("RawYAML should not be empty")
	}
	if !strings.Contains(string(plan.RawYAML), "external-controller: 127.0.0.1:9090") {
		t.Fatalf("patched yaml = %s", string(plan.RawYAML))
	}
}

func TestResolveContentYAMLPlanHonorsTUNMode(t *testing.T) {
	resolver := NewResolver()
	cfg := core.DefaultAppConfig()
	cfg.Mode = "tun"
	input := []byte("mixed-port: 7890\nproxies:\n  - name: a\n")

	plan, err := resolver.ResolveContent(cfg, input)
	if err != nil {
		t.Fatalf("ResolveContent() error = %v", err)
	}
	text := string(plan.RawYAML)
	if strings.Contains(text, "mixed-port:") {
		t.Fatalf("patched yaml should not keep mixed-port in tun mode: %s", text)
	}
	if !strings.Contains(text, "tun:") {
		t.Fatalf("patched yaml should contain tun config: %s", text)
	}
}

func TestResolveRemoteURLRejectsProviderOnlyConfig(t *testing.T) {
	tempDir := t.TempDir()
	preparedDir := filepath.Join(tempDir, "prepared-provider")
	if err := os.Mkdir(preparedDir, 0755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	resolver := &Resolver{
		prepareURL: func(string, time.Duration) (*system.PreparedSubscription, error) {
			return &system.PreparedSubscription{
				Body: []byte(`
proxy-providers:
  airport:
    type: http
    url: https://example.com/provider.yaml
    filter: hk
    health-check:
      enable: true
      url: https://cp.cloudflare.com/
      interval: 300
`),
				FetchDetail: "bytes=21",
				TempDir:     preparedDir,
			}, nil
		},
	}
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"

	_, err := resolver.ResolveRemoteURL(cfg, cfg.SubscriptionURL, time.Second)
	if err == nil {
		t.Fatal("ResolveRemoteURL() should reject provider-only config")
	}
	if !strings.Contains(err.Error(), "provider-only") {
		t.Fatalf("ResolveRemoteURL() error = %v, want provider-only hint", err)
	}
	if _, err := os.Stat(preparedDir); !os.IsNotExist(err) {
		t.Fatalf("prepared temp dir should be removed, stat error = %v", err)
	}
}

func TestResolveRemoteURLRejectsUnknownContent(t *testing.T) {
	resolver := &Resolver{
		prepareURL: func(string, time.Duration) (*system.PreparedSubscription, error) {
			return &system.PreparedSubscription{
				Body:        []byte("provider payload v2025"),
				FetchDetail: "bytes=21",
				TempDir:     t.TempDir(),
			}, nil
		},
	}
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"

	_, err := resolver.ResolveRemoteURL(cfg, cfg.SubscriptionURL, time.Second)
	if err == nil {
		t.Fatal("ResolveRemoteURL() should reject unknown content")
	}
	if !strings.Contains(err.Error(), "无法识别") {
		t.Fatalf("ResolveRemoteURL() error = %v, want unknown content hint", err)
	}
}

func TestResolveRemoteURLRejectsHTMLContent(t *testing.T) {
	tempDir := t.TempDir()
	preparedDir := filepath.Join(tempDir, "prepared-html")
	if err := os.Mkdir(preparedDir, 0755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	resolver := &Resolver{
		prepareURL: func(string, time.Duration) (*system.PreparedSubscription, error) {
			return &system.PreparedSubscription{
				Body:        []byte("<html>login</html>"),
				FetchDetail: "bytes=18",
				TempDir:     preparedDir,
			}, nil
		},
	}
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"

	_, err := resolver.ResolveRemoteURL(cfg, cfg.SubscriptionURL, time.Second)
	if err == nil {
		t.Fatal("ResolveRemoteURL() should reject HTML responses")
	}
	if !strings.Contains(err.Error(), "html") {
		t.Fatalf("ResolveRemoteURL() error = %v, want html hint", err)
	}
	if _, statErr := os.Stat(preparedDir); !os.IsNotExist(statErr) {
		t.Fatalf("prepared temp dir should be removed, stat error = %v", statErr)
	}
}
