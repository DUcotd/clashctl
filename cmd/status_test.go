package cmd

import (
	"reflect"
	"testing"

	"clashctl/internal/core"
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
		"  当前为 mixed-port 模式；像 codex/opencode 这类 CLI 需要显式导出 HTTP_PROXY/HTTPS_PROXY/ALL_PROXY 指向 127.0.0.1:7890",
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
