package mihomo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFormatDelay(t *testing.T) {
	tests := []struct {
		name  string
		delay int
		want  string
	}{
		{"untested", 0, "未测试"},
		{"timeout", -1, "超时"},
		{"fast", 50, "50ms ✨"},
		{"ok", 150, "150ms"},
		{"slow", 500, "500ms ⚠️"},
		{"very slow", 1500, "1.5s 🔴"},
		{"border fast", 99, "99ms ✨"},
		{"border ok", 100, "100ms"},
		{"border slow", 300, "300ms ⚠️"},
		{"border very slow", 1000, "1.0s 🔴"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDelay(tt.delay)
			if got != tt.want {
				t.Errorf("FormatDelay(%d) = %q, want %q", tt.delay, got, tt.want)
			}
		})
	}
}

func TestSortNodesByDelay(t *testing.T) {
	detail := &ProxyGroupDetail{
		Nodes: []ProxyNode{
			{Name: "slow", Delay: 500},
			{Name: "fast", Delay: 50},
			{Name: "timeout", Delay: -1},
			{Name: "untested", Delay: 0},
			{Name: "ok", Delay: 200},
		},
	}

	detail.SortNodesByDelay()

	// Expected order: fast(50), ok(200), slow(500), untested(0), timeout(-1)
	// Both <= 0 go to end; when both <= 0: ai > aj → 0 > -1 is true, so untested before timeout
	expected := []string{"fast", "ok", "slow", "untested", "timeout"}
	for i, node := range detail.Nodes {
		if node.Name != expected[i] {
			t.Errorf("SortNodesByDelay: position %d = %q, want %q", i, node.Name, expected[i])
		}
	}
}

func TestHasSystemd(t *testing.T) {
	// Just ensure it doesn't panic
	_ = HasSystemd()
}

func TestFindBinary(t *testing.T) {
	// On a system without mihomo, this should return an error
	// On a system with mihomo, it should return a path
	// Either way, it should not panic
	_, err := FindBinary()
	// We don't assert on the result since it depends on the system
	_ = err
}

func TestNewClient(t *testing.T) {
	client := NewClient("http://127.0.0.1:9090")
	if client.BaseURL != "http://127.0.0.1:9090" {
		t.Errorf("BaseURL = %q, want http://127.0.0.1:9090", client.BaseURL)
	}
	if client.HTTP == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestNewProcess(t *testing.T) {
	proc := NewProcess("/etc/mihomo")
	if proc.ConfigDir != "/etc/mihomo" {
		t.Errorf("ConfigDir = %q, want /etc/mihomo", proc.ConfigDir)
	}
	if proc.IsRunning() {
		t.Error("new process should not be running")
	}
}

func TestGetProxyGroupEscapesPath(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"测试 组/一","type":"select","now":"节点 A","all":["节点 A"]}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if _, err := client.GetProxyGroup("测试 组/一"); err != nil {
		t.Fatalf("GetProxyGroup() error: %v", err)
	}

	if requestedPath != "/proxies/%E6%B5%8B%E8%AF%95%20%E7%BB%84%2F%E4%B8%80" {
		t.Errorf("escaped path = %q", requestedPath)
	}
}

func TestSwitchProxyEscapesPath(t *testing.T) {
	var requestedPath string
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.EscapedPath()
		payload, _ := io.ReadAll(r.Body)
		body = string(payload)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if err := client.SwitchProxy("测试/组", "节点 A"); err != nil {
		t.Fatalf("SwitchProxy() error: %v", err)
	}

	if requestedPath != "/proxies/%E6%B5%8B%E8%AF%95%2F%E7%BB%84" {
		t.Errorf("escaped path = %q", requestedPath)
	}
	if !strings.Contains(body, `"name":"节点 A"`) {
		t.Errorf("body = %q", body)
	}
}

func TestTestNodeEscapesPath(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"history":[{"time":"2026-03-19T00:00:00Z","delay":123}]}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if got := client.TestNode("组 /A", "节点/%25"); got != 123 {
		t.Fatalf("TestNode() = %d, want 123", got)
	}

	if requestedPath != "/proxies/%E7%BB%84%20%2FA/%E8%8A%82%E7%82%B9%2F%2525" {
		t.Errorf("escaped path = %q", requestedPath)
	}
}

func TestProcessUsesConfigDir(t *testing.T) {
	if !processUsesConfigDir([]string{"/usr/local/bin/mihomo", "-d", "/etc/mihomo"}, "/etc/mihomo") {
		t.Fatal("expected processUsesConfigDir to match managed process")
	}
	if processUsesConfigDir([]string{"/usr/bin/bash", "-d", "/etc/mihomo"}, "/etc/mihomo") {
		t.Fatal("unexpected match for unrelated process")
	}
}

func TestCheckTUNPermission(t *testing.T) {
	// Just ensure it doesn't panic - result depends on system
	_ = CheckTUNPermission()
}

func TestCanUseTUN(t *testing.T) {
	// Just ensure it doesn't panic - result depends on system
	_ = CanUseTUN()
}
