package system

import (
	"net"
	"reflect"
	"testing"
)

func TestCommandExists(t *testing.T) {
	// sh should exist on all Linux systems
	if !CommandExists("sh") {
		t.Error("CommandExists(sh) = false, want true")
	}

	// nonexistent command
	if CommandExists("nonexistent_command_xyz_123") {
		t.Error("CommandExists(nonexistent) = true, want false")
	}
}

func TestDirExists(t *testing.T) {
	// /tmp should exist
	if !DirExists("/tmp") {
		t.Error("DirExists(/tmp) = false, want true")
	}

	// Nonexistent dir
	if DirExists("/nonexistent/dir/xyz") {
		t.Error("DirExists(nonexistent) = true, want false")
	}

	// File is not dir
	if DirExists("/etc/hostname") {
		t.Error("DirExists(file) = true, want false")
	}
}

func TestIsRoot(t *testing.T) {
	// Just ensure it doesn't panic
	_ = IsRoot()
}

func TestCheckPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("sandbox does not allow local listeners: %v", err)
	}
	defer ln.Close()

	if !CheckPortInUse(ln.Addr().String()) {
		t.Error("listening port should be reported as in use")
	}
}

func TestLookupHost(t *testing.T) {
	// localhost should always resolve
	addr, err := LookupHost("localhost")
	if err != nil {
		t.Fatalf("LookupHost(localhost) failed: %v", err)
	}
	if addr == "" {
		t.Error("LookupHost returned empty address")
	}
	// localhost should resolve to 127.0.0.1 or ::1
	if addr != "127.0.0.1" && addr != "::1" {
		t.Errorf("LookupHost(localhost) = %q, want 127.0.0.1 or ::1", addr)
	}
}

func TestStripProxyEnv(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"http_proxy=http://127.0.0.1:7890",
		"HTTPS_PROXY=http://127.0.0.1:7890",
		"NO_PROXY=localhost",
		"HOME=/root",
	}
	want := []string{
		"PATH=/usr/bin",
		"HOME=/root",
	}
	got := StripProxyEnv(env)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StripProxyEnv() = %#v, want %#v", got, want)
	}
}
