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
	if m.DNS.EnhancedMode != "fake-ip" {
		t.Errorf("DNS enhanced mode = %q, want fake-ip", m.DNS.EnhancedMode)
	}

	// Check rules
	if len(m.Rules) == 0 {
		t.Error("no rules generated")
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
		"tun:",
		"enable: true",
		"dns:",
		"enhanced-mode: fake-ip",
		"rules:",
		"MATCH,PROXY",
	}

	for _, check := range checks {
		if !contains(yaml, check) {
			t.Errorf("YAML missing expected content: %q", check)
		}
	}
}

func TestDefaultAppConfig(t *testing.T) {
	cfg := DefaultAppConfig()

	if cfg.Mode != "tun" {
		t.Errorf("default mode = %q, want tun", cfg.Mode)
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
