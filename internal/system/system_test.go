package system

import (
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestProxyEnvForDisplay(t *testing.T) {
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	t.Setenv("all_proxy", "")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:7890")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:7890")
	t.Setenv("ALL_PROXY", "")

	got := ProxyEnvForDisplay()
	want := []string{
		"HTTP_PROXY=http://127.0.0.1:7890",
		"HTTPS_PROXY=http://127.0.0.1:7890",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ProxyEnvForDisplay() = %#v, want %#v", got, want)
	}
}

func TestPersistShellProxyEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")

	result, err := PersistShellProxyEnv(7890)
	if err != nil {
		t.Fatalf("PersistShellProxyEnv() error: %v", err)
	}

	if result.ProfilePath != filepath.Join(home, ".bashrc") {
		t.Fatalf("ProfilePath = %q", result.ProfilePath)
	}
	script, err := os.ReadFile(result.ScriptPath)
	if err != nil {
		t.Fatalf("ReadFile(script) error: %v", err)
	}
	if !strings.Contains(string(script), `export HTTP_PROXY="http://127.0.0.1:7890"`) {
		t.Fatalf("script content = %s", string(script))
	}

	profile, err := os.ReadFile(result.ProfilePath)
	if err != nil {
		t.Fatalf("ReadFile(profile) error: %v", err)
	}
	if !strings.Contains(string(profile), clashctlProxyBlockStart) {
		t.Fatalf("profile missing managed block: %s", string(profile))
	}

	if _, err := PersistShellProxyEnv(7891); err != nil {
		t.Fatalf("second PersistShellProxyEnv() error: %v", err)
	}
	profile, err = os.ReadFile(result.ProfilePath)
	if err != nil {
		t.Fatalf("ReadFile(profile) error: %v", err)
	}
	if strings.Count(string(profile), clashctlProxyBlockStart) != 1 {
		t.Fatalf("expected one managed block, got profile: %s", string(profile))
	}
}

func TestRemoveShellProxyEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")

	result, err := PersistShellProxyEnv(7890)
	if err != nil {
		t.Fatalf("PersistShellProxyEnv() error: %v", err)
	}
	if _, err := RemoveShellProxyEnv(); err != nil {
		t.Fatalf("RemoveShellProxyEnv() error: %v", err)
	}

	profile, err := os.ReadFile(result.ProfilePath)
	if err != nil {
		t.Fatalf("ReadFile(profile) error: %v", err)
	}
	if strings.Contains(string(profile), clashctlProxyBlockStart) {
		t.Fatalf("profile should not contain managed block: %s", string(profile))
	}
	if _, err := os.Stat(result.ScriptPath); !os.IsNotExist(err) {
		t.Fatalf("script should be removed, stat err = %v", err)
	}
}
