package core

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://example.com/sub", false},
		{"valid http", "http://example.com/sub", false},
		{"empty", "", true},
		{"missing scheme", "example.com/sub", true},
		{"ftp not allowed", "ftp://example.com/sub", true},
		{"missing host", "https://", true},
		{"just domain", "https://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestAppConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *AppConfig
		wantErrs  int
	}{
		{"valid tun", &AppConfig{SubscriptionURL: "https://example.com/sub", Mode: "tun", MixedPort: 7890, ConfigDir: "/etc/mihomo", ControllerAddr: "127.0.0.1:9090"}, 0},
		{"empty url", &AppConfig{SubscriptionURL: "", Mode: "tun", MixedPort: 7890, ConfigDir: "/etc/mihomo", ControllerAddr: "127.0.0.1:9090"}, 1},
		{"invalid mode", &AppConfig{SubscriptionURL: "https://example.com/sub", Mode: "bad", MixedPort: 7890, ConfigDir: "/etc/mihomo", ControllerAddr: "127.0.0.1:9090"}, 1},
		{"bad port", &AppConfig{SubscriptionURL: "https://example.com/sub", Mode: "tun", MixedPort: 0, ConfigDir: "/etc/mihomo", ControllerAddr: "127.0.0.1:9090"}, 1},
		{"empty config dir", &AppConfig{SubscriptionURL: "https://example.com/sub", Mode: "tun", MixedPort: 7890, ConfigDir: "", ControllerAddr: "127.0.0.1:9090"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.cfg.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("Validate() got %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestBuildMihomoConfig(t *testing.T) {
	cfg := &AppConfig{
		SubscriptionURL:   "https://example.com/sub",
		Mode:              "tun",
		MixedPort:         7890,
		ConfigDir:         "/etc/mihomo",
		ControllerAddr:    "127.0.0.1:9090",
		ProviderPath:      "./providers/airport.yaml",
		EnableHealthCheck: true,
	}

	m := BuildMihomoConfig(cfg)

	// Check basic fields
	if m.MixedPort != 7890 {
		t.Errorf("MixedPort = %d, want 7890", m.MixedPort)
	}
	if m.ExternalController != "127.0.0.1:9090" {
		t.Errorf("ExternalController = %q, want 127.0.0.1:9090", m.ExternalController)
	}

	// Check proxy provider
	p, ok := m.ProxyProviders["airport"]
	if !ok {
		t.Fatal("missing 'airport' proxy provider")
	}
	if p.URL != "https://example.com/sub" {
		t.Errorf("provider URL = %q", p.URL)
	}

	// Check TUN config exists for TUN mode
	if m.TUN == nil {
		t.Fatal("TUN config is nil for tun mode")
	}
	if !m.TUN.Enable {
		t.Error("TUN not enabled")
	}

	// Check mixed mode has no TUN
	mixed := BuildMihomoConfig(&AppConfig{
		SubscriptionURL: "https://example.com/sub",
		Mode:            "mixed",
		MixedPort:       7890,
		ConfigDir:       "/etc/mihomo",
		ControllerAddr:  "127.0.0.1:9090",
		ProviderPath:    "./providers/airport.yaml",
	})
	if mixed.TUN != nil {
		t.Error("TUN config should be nil for mixed mode")
	}

	// Check DNS config
	if m.DNS == nil {
		t.Fatal("DNS config is nil")
	}
	if m.DNS.EnhancedMode != "redir-host" {
		t.Errorf("DNS enhanced mode = %q, want redir-host", m.DNS.EnhancedMode)
	}

	// Check proxy groups - should have PROXY (select), auto (url-test), fallback
	if len(m.ProxyGroups) != 3 {
		t.Errorf("expected 3 proxy groups, got %d", len(m.ProxyGroups))
	}

	groupTypes := make(map[string]string)
	for _, g := range m.ProxyGroups {
		groupTypes[g.Name] = g.Type
	}
	if groupTypes["PROXY"] != "select" {
		t.Errorf("PROXY group type = %q, want select", groupTypes["PROXY"])
	}
	if groupTypes["auto"] != "url-test" {
		t.Errorf("auto group type = %q, want url-test", groupTypes["auto"])
	}
	if groupTypes["fallback"] != "fallback" {
		t.Errorf("fallback group type = %q, want fallback", groupTypes["fallback"])
	}

	// Check DNS enhancements
	if len(m.DNS.Fallback) == 0 {
		t.Error("DNS fallback should not be empty")
	}
	if len(m.DNS.DefaultNameserver) == 0 {
		t.Error("DNS default-nameserver should not be empty")
	}
	if len(m.DNS.DirectNameserver) == 0 {
		t.Error("DNS direct-nameserver should not be empty")
	}
	// Check domestic DNS is used as primary
	if m.DNS.NameServer[0] != "223.5.5.5" {
		t.Errorf("DNS nameserver[0] = %q, want 223.5.5.5", m.DNS.NameServer[0])
	}

	// Check routing rules - should have DIRECT rules for local/China traffic
	hasLocalRule := false
	hasGeoIPRule := false
	hasMatchRule := false
	for _, r := range m.Rules {
		if r == "IP-CIDR,127.0.0.0/8,DIRECT" {
			hasLocalRule = true
		}
		if r == "GEOIP,CN,DIRECT" {
			hasGeoIPRule = true
		}
		if r == "MATCH,PROXY" {
			hasMatchRule = true
		}
	}
	if !hasLocalRule {
		t.Error("missing local IP direct rule")
	}
	if !hasGeoIPRule {
		t.Error("missing GEOIP,CN,DIRECT rule")
	}
	if !hasMatchRule {
		t.Error("missing MATCH,PROXY fallback rule")
	}
}

func TestRenderYAML(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"
	m := BuildMihomoConfig(cfg)

	data, err := RenderYAML(m)
	if err != nil {
		t.Fatalf("RenderYAML failed: %v", err)
	}

	yaml := string(data)

	// Check key fields are present
	checks := []string{
		"mixed-port: 7890",
		"proxy-providers:",
		"airport:",
		"https://example.com/sub",
		"proxy-groups:",
		"name: PROXY",
		"dns:",
		"enhanced-mode: redir-host",
		"rules:",
		"MATCH,PROXY",
		"DOMAIN-SUFFIX,ubuntu.com,DIRECT",
		"DOMAIN-SUFFIX,github.com,DIRECT",
	}

	// TUN mode should include tun config
	tunCfg := BuildMihomoConfig(&AppConfig{
		SubscriptionURL: "https://example.com/sub",
		Mode:            "tun",
		MixedPort:       7890,
		ConfigDir:       "/etc/mihomo",
		ControllerAddr:  "127.0.0.1:9090",
		ProviderPath:    "./providers/airport.yaml",
	})
	tunData, err := RenderYAML(tunCfg)
	if err != nil {
		t.Fatalf("RenderYAML for TUN config failed: %v", err)
	}
	tunYAML := string(tunData)
	if !contains(tunYAML, "tun:") {
		t.Error("TUN config YAML missing 'tun:' section")
	}
	if !contains(tunYAML, "enable: true") {
		t.Error("TUN config YAML missing 'enable: true'")
	}

	for _, check := range checks {
		if !contains(yaml, check) {
			t.Errorf("YAML missing expected content: %q", check)
		}
	}
}

func TestDefaultAppConfig(t *testing.T) {
	cfg := DefaultAppConfig()

	if cfg.Mode != "mixed" {
		t.Errorf("default mode = %q, want mixed", cfg.Mode)
	}
	if cfg.MixedPort != 7890 {
		t.Errorf("default mixed port = %d, want 7890", cfg.MixedPort)
	}
	if cfg.ConfigDir != "/etc/mihomo" {
		t.Errorf("default config dir = %q", cfg.ConfigDir)
	}
	if !cfg.EnableHealthCheck {
		t.Error("health check should be enabled by default")
	}
	if !cfg.EnableSystemd {
		t.Error("systemd should be enabled by default")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
