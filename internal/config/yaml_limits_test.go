package config

import (
	"strings"
	"testing"
)

func TestValidateYAMLBytes_DepthLimit(t *testing.T) {
	var buf strings.Builder
	buf.WriteString("a:\n")
	for i := 0; i < 60; i++ {
		buf.WriteString(strings.Repeat("  ", i+1) + "b:\n")
	}
	buf.WriteString(strings.Repeat("  ", 61) + "c: 1\n")

	err := ValidateYAMLBytes([]byte(buf.String()), "test")
	if err == nil {
		t.Error("ValidateYAMLBytes should reject deeply nested YAML")
	}
	if !strings.Contains(err.Error(), "嵌套过深") {
		t.Errorf("Expected depth error, got: %v", err)
	}
}

func TestValidateYAMLBytes_NodeLimit(t *testing.T) {
	var buf strings.Builder
	buf.WriteString("items:\n")
	for i := 0; i < 100001; i++ {
		buf.WriteString("  - item_" + string(rune('0'+i%10)) + "\n")
	}

	err := ValidateYAMLBytes([]byte(buf.String()), "test")
	if err == nil {
		t.Error("ValidateYAMLBytes should reject YAML with too many nodes")
	}
	if !strings.Contains(err.Error(), "节点过多") {
		t.Errorf("Expected node count error, got: %v", err)
	}
}

func TestValidateYAMLBytes_NormalYAML(t *testing.T) {
	yaml := `
mixed-port: 7890
allow-lan: false
log-level: info
proxies:
  - name: test
    type: trojan
    server: example.com
    port: 443
    password: test
`
	err := ValidateYAMLBytes([]byte(yaml), "test")
	if err != nil {
		t.Errorf("ValidateYAMLBytes should accept normal YAML: %v", err)
	}
}

func TestCheckYAMLDepth_Boundary(t *testing.T) {
	var buf strings.Builder
	buf.WriteString("root:\n")
	for i := 0; i < 20; i++ {
		buf.WriteString(strings.Repeat("  ", i+1) + "child:\n")
	}
	buf.WriteString(strings.Repeat("  ", 21) + "leaf: 1\n")

	err := ValidateYAMLBytes([]byte(buf.String()), "test")
	if err != nil {
		t.Errorf("ValidateYAMLBytes should accept YAML at depth boundary: %v", err)
	}
}
