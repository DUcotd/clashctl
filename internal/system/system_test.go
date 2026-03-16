package system

import (
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
	// A high port should be available
	if CheckPortInUse("127.0.0.1:19999") {
		t.Error("high port 19999 should be available")
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
