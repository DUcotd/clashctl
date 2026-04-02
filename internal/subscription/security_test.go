package subscription

import (
	"path/filepath"
	"strings"
	"testing"

	"clashctl/internal/core"
)

func TestValidateYAMLSecurity_DangerousFields(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		allowUnsafe bool
		wantErr     bool
		wantWarn    bool
	}{
		{
			name: "safe config",
			yaml: `
mixed-port: 7890
proxies:
  - name: test
    type: vless
    server: example.com
    port: 443
`,
			allowUnsafe: false,
			wantErr:     false,
			wantWarn:    false,
		},
		{
			name: "dangerous script field",
			yaml: `
mixed-port: 7890
script: |
  os.execute("rm -rf /")
`,
			allowUnsafe: false,
			wantErr:     true,
			wantWarn:    false,
		},
		{
			name: "dangerous script field with unsafe allowed",
			yaml: `
mixed-port: 7890
script: |
  print("hello")
`,
			allowUnsafe: true,
			wantErr:     false,
			wantWarn:    true,
		},
		{
			name: "dangerous tun field",
			yaml: `
mixed-port: 7890
tun:
  enable: true
  stack: system
`,
			allowUnsafe: false,
			wantErr:     true,
			wantWarn:    false,
		},
		{
			name: "dangerous dns hijacking",
			yaml: `
mixed-port: 7890
dns:
  enable: true
  nameserver:
    - 8.8.8.8
  fake-ip-filter:
    - "*.lan"
`,
			allowUnsafe: false,
			wantErr:     true,
			wantWarn:    false,
		},
		{
			name: "dangerous external-ui",
			yaml: `
mixed-port: 7890
external-ui: /tmp/malicious
`,
			allowUnsafe: false,
			wantErr:     true,
			wantWarn:    false,
		},
		{
			name: "warn disallowed auth fields",
			yaml: `
mixed-port: 7890
authentication:
  - user:pass
`,
			allowUnsafe: false,
			wantErr:     false,
			wantWarn:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := ValidateYAMLSecurity([]byte(tt.yaml), tt.allowUnsafe)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateYAMLSecurity() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (len(warnings) > 0) != tt.wantWarn {
				t.Errorf("ValidateYAMLSecurity() warnings = %v, wantWarn %v", warnings, tt.wantWarn)
			}
		})
	}
}

func TestSanitizeYAML(t *testing.T) {
	yaml := `
mixed-port: 7890
script: |
  os.execute("bad")
dns:
  enable: true
proxies:
  - name: test
    type: vless
    server: example.com
    port: 443
`
	sanitized, removed, err := SanitizeYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("SanitizeYAML() error = %v", err)
	}

	if len(removed) == 0 {
		t.Error("SanitizeYAML() should have removed dangerous fields")
	}

	// Verify dangerous fields are removed
	warnings, err := ValidateYAMLSecurity(sanitized, false)
	if err != nil {
		t.Errorf("Sanitized YAML should be safe, got error: %v", err)
	}
	_ = warnings
}

func TestPatchRemoteYAMLRemovesUnsafeTopLevelFields(t *testing.T) {
	cfg := core.DefaultAppConfig()
	result, err := PatchRemoteYAML([]byte(`
mixed-port: 7890
bind-address: 0.0.0.0
hosts:
  example.com: 127.0.0.1
authentication:
  - user:pass
skip-auth-prefixes:
  - /public
proxies:
  - name: test
    type: socks5
    server: example.com
    port: 1080
`), cfg)
	if err != nil {
		t.Fatalf("PatchRemoteYAML() error = %v", err)
	}
	text := string(result.YAML)
	for _, fragment := range []string{"bind-address:", "hosts:", "authentication:", "skip-auth-prefixes:"} {
		if strings.Contains(text, fragment) {
			t.Fatalf("patched YAML still contains %q: %s", fragment, text)
		}
	}
	if !result.Sanitized {
		t.Fatal("PatchRemoteYAML() should mark result as sanitized")
	}
	if len(result.RemovedFields) == 0 {
		t.Fatal("PatchRemoteYAML() should report removed fields")
	}
}

func TestPatchRemoteYAMLSanitizesProxyProviders(t *testing.T) {
	cfg := core.DefaultAppConfig()
	result, err := PatchRemoteYAML([]byte(`
proxy-providers:
  airport 1:
    type: http
    url: https://example.com/provider.yaml
    path: /etc/passwd
    script: os.execute("bad")
    health-check:
      enable: true
      url: https://cp.cloudflare.com/
      interval: 300
      lazy: true
unknown-top-level: true
`), cfg)
	if err != nil {
		t.Fatalf("PatchRemoteYAML() error = %v", err)
	}
	text := string(result.YAML)
	if strings.Contains(text, "unknown-top-level") || strings.Contains(text, "script:") {
		t.Fatalf("patched YAML still contains stripped fields: %s", text)
	}
	if !strings.Contains(text, filepath.ToSlash("providers/airport-1.yaml")) {
		t.Fatalf("patched YAML should contain sanitized provider path, got: %s", text)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("PatchRemoteYAML() should surface warnings")
	}
}

func TestPatchRemoteYAMLInjectsTUNConfigForTUNMode(t *testing.T) {
	cfg := core.DefaultAppConfig()
	cfg.Mode = "tun"
	result, err := PatchRemoteYAML([]byte(`
mixed-port: 7890
proxies:
  - name: test
    type: socks5
    server: example.com
    port: 1080
`), cfg)
	if err != nil {
		t.Fatalf("PatchRemoteYAML() error = %v", err)
	}
	text := string(result.YAML)
	if strings.Contains(text, "mixed-port:") {
		t.Fatalf("patched YAML should drop mixed-port in tun mode: %s", text)
	}
	if !strings.Contains(text, "tun:") {
		t.Fatalf("patched YAML should contain tun config: %s", text)
	}
}

func TestPatchRemoteYAMLPreservesSafeRuleProviders(t *testing.T) {
	cfg := core.DefaultAppConfig()
	result, err := PatchRemoteYAML([]byte(`
rule-providers:
  apple:
    type: http
    behavior: classical
    url: https://example.com/apple.yaml
    interval: 600
rules:
  - RULE-SET,apple,DIRECT
  - MATCH,PROXY
`), cfg)
	if err != nil {
		t.Fatalf("PatchRemoteYAML() error = %v", err)
	}

	text := string(result.YAML)
	if !strings.Contains(text, "rule-providers:") {
		t.Fatalf("patched YAML should preserve rule-providers: %s", text)
	}
	if !strings.Contains(text, "RULE-SET,apple,DIRECT") {
		t.Fatalf("patched YAML should preserve RULE-SET rules: %s", text)
	}
	if !strings.Contains(text, filepath.ToSlash("rules/apple.yaml")) {
		t.Fatalf("patched YAML should rewrite rule provider path, got: %s", text)
	}
}

func TestPatchRemoteYAMLDropsRulesForRemovedRuleProviders(t *testing.T) {
	cfg := core.DefaultAppConfig()
	result, err := PatchRemoteYAML([]byte(`
rule-providers:
  local-only:
    type: file
    behavior: classical
    path: ./rules.yaml
rules:
  - RULE-SET,local-only,DIRECT
  - MATCH,PROXY
`), cfg)
	if err != nil {
		t.Fatalf("PatchRemoteYAML() error = %v", err)
	}

	text := string(result.YAML)
	if strings.Contains(text, "RULE-SET,local-only,DIRECT") {
		t.Fatalf("patched YAML should drop RULE-SET for removed provider: %s", text)
	}
	if !strings.Contains(text, "MATCH,PROXY") {
		t.Fatalf("patched YAML should keep unrelated rules: %s", text)
	}
}

func TestContainsDangerousScript(t *testing.T) {
	tests := []struct {
		script string
		want   bool
	}{
		{"print('hello')", false},
		{"os.execute('rm -rf /')", true},
		{"os.system('bad')", true},
		{"io.popen('ls')", true},
		{"require('malicious')", true},
		{"eval('code')", true},
		{"import os", true},
		{"import subprocess", true},
		{"Runtime.getRuntime().exec('bad')", true},
		{"ProcessBuilder pb = new ProcessBuilder()", true},
	}

	for _, tt := range tests {
		t.Run(tt.script, func(t *testing.T) {
			if got := containsDangerousScript(tt.script); got != tt.want {
				t.Errorf("containsDangerousScript(%q) = %v, want %v", tt.script, got, tt.want)
			}
		})
	}
}
