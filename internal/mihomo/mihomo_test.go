package mihomo

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

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

func TestFindBinarySkipsBrokenCandidate(t *testing.T) {
	tmp := t.TempDir()
	badPath := filepath.Join(tmp, "mihomo")
	goodPath := filepath.Join(tmp, "clash-meta")

	writeExecutable(t, badPath, "#!/bin/sh\nexit 139\n")
	writeExecutable(t, goodPath, "#!/bin/sh\nif [ \"$1\" = \"-v\" ]; then\n  echo 'Mihomo Meta v1.2.3'\n  exit 0\nfi\necho 'usage'\nexit 0\n")

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmp)
	oldInstall := installedBinaryPath
	installedBinaryPath = filepath.Join(tmp, "missing-install")
	t.Cleanup(func() {
		installedBinaryPath = oldInstall
		t.Setenv("PATH", oldPath)
	})

	got, err := FindBinary()
	if err != nil {
		t.Fatalf("FindBinary() error: %v", err)
	}
	if got != goodPath {
		t.Fatalf("FindBinary() = %q, want %q", got, goodPath)
	}
}

func TestValidateBinaryRejectsSilentOrBrokenBinary(t *testing.T) {
	tmp := t.TempDir()
	broken := filepath.Join(tmp, "mihomo-broken")
	silent := filepath.Join(tmp, "mihomo-silent")
	good := filepath.Join(tmp, "mihomo-good")

	writeExecutable(t, broken, "#!/bin/sh\nexit 139\n")
	writeExecutable(t, silent, "#!/bin/sh\nexit 0\n")
	writeExecutable(t, good, "#!/bin/sh\nif [ \"$1\" = \"-v\" ]; then\n  echo 'Mihomo Meta v9.9.9'\n  exit 0\nfi\necho 'usage'\nexit 0\n")

	if _, err := validateBinary(broken); err == nil {
		t.Fatal("validateBinary() should reject broken binary")
	}
	if _, err := validateBinary(silent); err == nil {
		t.Fatal("validateBinary() should reject silent binary")
	}
	version, err := validateBinary(good)
	if err != nil {
		t.Fatalf("validateBinary() error: %v", err)
	}
	if version != "Mihomo Meta v9.9.9" {
		t.Fatalf("validateBinary() = %q", version)
	}
}

func TestActivateBinaryRollsBackOnInvalidInstall(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "mihomo")
	bad := filepath.Join(tmp, "mihomo.bad")

	writeExecutable(t, dest, "#!/bin/sh\necho 'Mihomo Meta v1.0.0'\n")
	writeExecutable(t, bad, "#!/bin/sh\nexit 139\n")

	if err := activateBinary(bad, dest); err == nil {
		t.Fatal("activateBinary() should fail for invalid replacement")
	}

	version, err := validateBinary(dest)
	if err != nil {
		t.Fatalf("validateBinary(dest) error after rollback: %v", err)
	}
	if version != "Mihomo Meta v1.0.0" {
		t.Fatalf("restored binary version = %q", version)
	}
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
	client := NewClient("http://example.invalid")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestedPath = r.URL.EscapedPath()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"name":"测试 组/一","type":"select","now":"节点 A","all":["节点 A"]}`)),
		}, nil
	})}
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
	client := NewClient("http://example.invalid")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestedPath = r.URL.EscapedPath()
		payload, _ := io.ReadAll(r.Body)
		body = string(payload)
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})}
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
	client := NewClient("http://example.invalid")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestedPath = r.URL.EscapedPath()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"history":[{"time":"2026-03-19T00:00:00Z","delay":123}]}`)),
		}, nil
	})}
	if got := client.TestNode("组 /A", "节点/%25"); got != 123 {
		t.Fatalf("TestNode() = %d, want 123", got)
	}

	if requestedPath != "/proxies/%E7%BB%84%20%2FA/%E8%8A%82%E7%82%B9%2F%2525" {
		t.Errorf("escaped path = %q", requestedPath)
	}
}

func TestTestProxyGroupNodes(t *testing.T) {
	var requestedPaths []string
	client := NewClient("http://example.invalid")
	client.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestedPaths = append(requestedPaths, r.URL.EscapedPath())
		switch r.URL.EscapedPath() {
		case "/proxies/PROXY":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"name":"PROXY","type":"select","now":"Node B","all":["Node A","Node B","Node C"]}`)),
			}, nil
		case "/proxies/PROXY/Node%20A":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"history":[{"time":"2026-03-19T00:00:00Z","delay":180}]}`)),
			}, nil
		case "/proxies/PROXY/Node%20B":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"history":[{"time":"2026-03-19T00:00:00Z","delay":80}]}`)),
			}, nil
		case "/proxies/PROXY/Node%20C":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"history":[]}`)),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`not found`)),
			}, nil
		}
	})}

	detail, err := client.TestProxyGroupNodes("PROXY", 2)
	if err != nil {
		t.Fatalf("TestProxyGroupNodes() error: %v", err)
	}

	if detail.Name != "PROXY" {
		t.Fatalf("detail.Name = %q, want PROXY", detail.Name)
	}

	if got := []string{detail.Nodes[0].Name, detail.Nodes[1].Name, detail.Nodes[2].Name}; !slices.Equal(got, []string{"Node B", "Node A", "Node C"}) {
		t.Fatalf("sorted nodes = %#v", got)
	}
	if detail.Nodes[0].Delay != 80 || !detail.Nodes[0].Selected {
		t.Fatalf("Node B = %#v, want selected with 80ms", detail.Nodes[0])
	}
	if detail.Nodes[2].Delay != 0 {
		t.Fatalf("Node C delay = %d, want 0", detail.Nodes[2].Delay)
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

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("WriteFile(%q) error: %v", path, err)
	}
}
