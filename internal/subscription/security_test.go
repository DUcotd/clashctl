package subscription

import (
	"testing"
)

func TestValidateYAMLSecurity_DangerousFields(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		allowUnsafe bool
		wantErr   bool
		wantWarn  bool
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
