package system

import (
	"fmt"
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

func TestDirWritableLeavesNoProbeFile(t *testing.T) {
	dir := t.TempDir()
	if err := DirWritable(dir); err != nil {
		t.Fatalf("DirWritable() error = %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, ".clashctl_write_test-*"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		fatalf := "DirWritable() should clean probe files, got %v"
		t.Fatalf(fatalf, matches)
	}
}

func TestSiblingTempPathHelpers(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "binary")

	tmpPath, err := CreateSiblingTempFile(target, ".tmp-*")
	if err != nil {
		t.Fatalf("CreateSiblingTempFile() error = %v", err)
	}
	if filepath.Dir(tmpPath) != dir {
		t.Fatalf("CreateSiblingTempFile() dir = %q, want %q", filepath.Dir(tmpPath), dir)
	}
	if _, err := os.Stat(tmpPath); err != nil {
		t.Fatalf("temp file should exist, stat err = %v", err)
	}

	reservedPath, err := ReserveSiblingPath(target, ".bak-*")
	if err != nil {
		t.Fatalf("ReserveSiblingPath() error = %v", err)
	}
	if filepath.Dir(reservedPath) != dir {
		t.Fatalf("ReserveSiblingPath() dir = %q, want %q", filepath.Dir(reservedPath), dir)
	}
	if _, err := os.Stat(reservedPath); !os.IsNotExist(err) {
		t.Fatalf("reserved path should not exist, stat err = %v", err)
	}

	_ = os.Remove(tmpPath)
}

func TestWriteFileAtomicCreatesFinalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	want := []byte("mixed-port: 7890\n")

	if err := WriteFileAtomic(path, want, 0600); err != nil {
		t.Fatalf("WriteFileAtomic() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("file content = %q, want %q", string(got), string(want))
	}

	matches, err := filepath.Glob(path + ".tmp-*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files should be cleaned up, found %v", matches)
	}
}

func TestReplaceFileRestoresOriginalOnValidationFailure(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "binary")
	src := filepath.Join(dir, "binary.new")

	if err := os.WriteFile(dest, []byte("original"), 0755); err != nil {
		t.Fatalf("WriteFile(dest) error = %v", err)
	}
	if err := os.WriteFile(src, []byte("candidate"), 0755); err != nil {
		t.Fatalf("WriteFile(src) error = %v", err)
	}

	err := ReplaceFile(src, dest, ReplaceFileOptions{
		Validate: func(path string) error {
			return fmt.Errorf("boom")
		},
	})
	if err == nil {
		t.Fatal("ReplaceFile() should fail validation")
	}

	got, readErr := os.ReadFile(dest)
	if readErr != nil {
		t.Fatalf("ReadFile(dest) error = %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("dest content = %q, want original", string(got))
	}

	if _, statErr := os.Stat(src); !os.IsNotExist(statErr) {
		t.Fatalf("src should be moved away after attempted replace, stat err = %v", statErr)
	}

	matches, globErr := filepath.Glob(filepath.Join(dir, "binary.bak-*"))
	if globErr != nil {
		t.Fatalf("Glob() error = %v", globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("backup files should be cleaned up, found %v", matches)
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
		"NODE_USE_ENV_PROXY=1",
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
	t.Setenv("NODE_USE_ENV_PROXY", "1")

	got := ProxyEnvForDisplay()
	want := []string{
		"HTTP_PROXY=http://127.0.0.1:7890",
		"HTTPS_PROXY=http://127.0.0.1:7890",
		"NODE_USE_ENV_PROXY=1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ProxyEnvForDisplay() = %#v, want %#v", got, want)
	}
}

func TestPersistShellProxyEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")

	profilePath := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(profilePath, []byte("# existing\n"), 0600); err != nil {
		t.Fatalf("WriteFile(profile) error: %v", err)
	}

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
	if !strings.Contains(string(script), "export NODE_USE_ENV_PROXY=1") {
		t.Fatalf("script content = %s", string(script))
	}

	profile, err := os.ReadFile(result.ProfilePath)
	if err != nil {
		t.Fatalf("ReadFile(profile) error: %v", err)
	}
	if !strings.Contains(string(profile), clashctlProxyBlockStart) {
		t.Fatalf("profile missing managed block: %s", string(profile))
	}
	info, err := os.Stat(result.ProfilePath)
	if err != nil {
		t.Fatalf("Stat(profile) error: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("profile mode = %o, want 0600", info.Mode().Perm())
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

func TestPersistShellProxyEnvRejectsOversizedProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")

	profilePath := filepath.Join(home, ".bashrc")
	oversized := strings.Repeat("a", maxShellProfileBytes+1)
	if err := os.WriteFile(profilePath, []byte(oversized), 0600); err != nil {
		t.Fatalf("WriteFile(profile) error: %v", err)
	}

	_, err := PersistShellProxyEnv(7890)
	if err == nil {
		t.Fatal("PersistShellProxyEnv() should reject oversized shell profile")
	}
	if !strings.Contains(err.Error(), "shell 配置文件过大") {
		t.Fatalf("PersistShellProxyEnv() error = %v", err)
	}
}

func TestCanWritePathDoesNotDeleteExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	const want = "keep me"
	if err := os.WriteFile(path, []byte(want), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := CanWritePath(path); err != nil {
		t.Fatalf("CanWritePath() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != want {
		t.Fatalf("existing file content changed: got %q want %q", string(data), want)
	}
}

func TestCanWritePathDoesNotCreateTargetFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "future-config.yaml")

	if err := CanWritePath(path); err != nil {
		t.Fatalf("CanWritePath() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("target file should not be created, stat err = %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "future-config.yaml.perm-*"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("permission probe files should be removed, got %v", matches)
	}
}
