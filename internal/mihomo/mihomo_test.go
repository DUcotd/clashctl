package mihomo

import (
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

func TestCheckTUNPermission(t *testing.T) {
	// Just ensure it doesn't panic - result depends on system
	_ = CheckTUNPermission()
}

func TestCanUseTUN(t *testing.T) {
	// Just ensure it doesn't panic - result depends on system
	_ = CanUseTUN()
}
