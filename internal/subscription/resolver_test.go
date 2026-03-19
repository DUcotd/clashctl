package subscription

import (
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

func TestResolveRemoteURLFallsBackToProvider(t *testing.T) {
	resolver := &Resolver{
		prepareURL: func(string, time.Duration) (*system.PreparedSubscription, error) {
			return &system.PreparedSubscription{
				Body:        []byte("<html>login</html>"),
				FetchDetail: "bytes=18",
			}, nil
		},
	}
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"

	plan, err := resolver.ResolveRemoteURL(cfg, cfg.SubscriptionURL, time.Second)
	if err != nil {
		t.Fatalf("ResolveRemoteURL() error = %v", err)
	}
	if plan.Kind != PlanKindProvider {
		t.Fatalf("Kind = %q", plan.Kind)
	}
	if plan.MihomoConfig == nil || plan.MihomoConfig.ProxyProviders == nil {
		t.Fatalf("unexpected provider plan: %#v", plan)
	}
}
