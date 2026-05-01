package cmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

func TestModeLabel(t *testing.T) {
	cases := map[string]string{
		"tun":   "TUN (透明接管)",
		"mixed": "mixed-port",
		"rule":  "rule",
	}

	for input, want := range cases {
		if got := modeLabel(input); got != want {
			t.Fatalf("modeLabel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestProxyStatusLinesMixedWithoutEnv(t *testing.T) {
	cfg := &core.AppConfig{Mode: "mixed", MixedPort: 7890}
	got := proxyStatusLines(cfg, nil)
	want := []string{
		"Shell 代理: ⚠ 未设置",
		"  当前为 mixed-port 模式；像 codex/opencode/Node.js 这类 CLI 需要显式导出 HTTP_PROXY/HTTPS_PROXY/ALL_PROXY，并为 Node.js 额外设置 NODE_USE_ENV_PROXY=1（代理地址 127.0.0.1:7890）",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("proxyStatusLines() = %#v, want %#v", got, want)
	}
}

func TestProxyStatusLinesTunWithoutEnv(t *testing.T) {
	cfg := &core.AppConfig{Mode: "tun"}
	got := proxyStatusLines(cfg, nil)
	want := []string{"Shell 代理: 未设置 (TUN 模式通常不需要)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("proxyStatusLines() = %#v, want %#v", got, want)
	}
}

func TestProxyStatusLinesWithEnv(t *testing.T) {
	cfg := &core.AppConfig{Mode: "mixed", MixedPort: 7890}
	got := proxyStatusLines(cfg, []string{
		"HTTP_PROXY=http://127.0.0.1:7890",
		"ALL_PROXY=socks5h://127.0.0.1:7890",
	})
	want := []string{
		"Shell 代理: ✅ 已设置",
		"  HTTP_PROXY=http://127.0.0.1:7890",
		"  ALL_PROXY=socks5h://127.0.0.1:7890",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("proxyStatusLines() = %#v, want %#v", got, want)
	}
}

func TestBuildStatusReportIncludesSortedGroupsAndWarnings(t *testing.T) {
	cfg := &core.AppConfig{
		Mode:           "mixed",
		MixedPort:      7890,
		ConfigDir:      "/etc/mihomo",
		ControllerAddr: "127.0.0.1:9090",
	}
	groups := map[string]mihomo.ProxyGroup{
		"PROXY": {Name: "PROXY", Type: "Selector", Now: "Node A", All: []string{"Node A", "Node B"}},
		"Auto":  {Name: "Auto", Type: "url-test", All: []string{"Node C"}},
	}
	inventory := &mihomo.ProxyInventory{
		Loaded:         1,
		Current:        "COMPATIBLE",
		Candidates:     []string{"COMPATIBLE"},
		OnlyCompatible: true,
	}

	report := buildStatusReport(&statusReportInput{
		Cfg: cfg, ProxyEnv: []string{"HTTP_PROXY=http://127.0.0.1:7890"},
		Binary: "/usr/local/bin/mihomo", BinaryVersion: "1.19.10",
		ControllerVersion: "1.19.10",
		Groups: groups, Inventory: inventory,
	})

	if !report.Service.Active || report.Service.Mode != "process" {
		t.Fatalf("service = %#v", report.Service)
	}
	if !report.Controller.Reachable || report.Controller.Version != "1.19.10" {
		t.Fatalf("controller = %#v", report.Controller)
	}
	if len(report.Groups) != 2 || report.Groups[0].Name != "Auto" || report.Groups[1].Name != "PROXY" {
		t.Fatalf("groups = %#v", report.Groups)
	}
	if report.Inventory == nil || !report.Inventory.OnlyCompatible {
		t.Fatalf("inventory = %#v", report.Inventory)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected compatibility warning")
	}
}

func TestPrintStatusReportIncludesFallbackHint(t *testing.T) {
	report := &statusReport{
		Service:    statusServiceReport{Active: true, Mode: "systemd"},
		Binary:     statusBinaryReport{Found: true, Path: "/usr/local/bin/mihomo", Version: "1.19.10"},
		Config:     statusConfigReport{Dir: "/etc/mihomo", Mode: "mixed", MixedPort: 7890},
		ProxyEnv:   statusProxyEnvReport{Messages: []string{"Shell 代理: ✅ 已设置"}},
		Controller: statusControllerReport{Reachable: true, Version: "1.19.10"},
		Inventory:  &statusInventoryReport{OnlyCompatible: true},
	}

	var buf bytes.Buffer
	if err := printStatusReport(&buf, report); err != nil {
		t.Fatalf("printStatusReport() error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{"服务状态: ✅ 运行中 (systemd)", "Controller API: ✅ 可达 (Mihomo 1.19.10)", "当前仅剩 COMPATIBLE", "config import --file sub.txt"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q in:\n%s", want, out)
		}
	}
}
