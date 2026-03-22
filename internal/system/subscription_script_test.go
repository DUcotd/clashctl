package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPreparedSubscriptionBodyRejectsOversizeFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subscription.txt")
	data := make([]byte, MaxPreparedSubscriptionBytes+1)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := readPreparedSubscriptionBody(path)
	if err == nil {
		t.Fatal("readPreparedSubscriptionBody() should reject oversized files")
	}
	if !strings.Contains(err.Error(), "过大") {
		t.Fatalf("readPreparedSubscriptionBody() error = %v, want size hint", err)
	}
}

func TestPreparedSubscriptionCleanupRemovesTempDir(t *testing.T) {
	dir := t.TempDir()
	prepared := &PreparedSubscription{TempDir: dir}

	if err := prepared.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("TempDir should be removed, stat err = %v", err)
	}
}

func TestEnsurePathWithinBase(t *testing.T) {
	base := t.TempDir()
	inside := filepath.Join(base, "output", "subscription.txt")
	if err := os.MkdirAll(filepath.Dir(inside), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	got, err := ensurePathWithinBase(base, inside)
	if err != nil {
		t.Fatalf("ensurePathWithinBase(inside) error = %v", err)
	}
	if got != inside {
		t.Fatalf("ensurePathWithinBase(inside) = %q, want %q", got, inside)
	}

	outside := filepath.Join(base, "..", "outside.txt")
	if _, err := ensurePathWithinBase(base, outside); err == nil {
		t.Fatal("ensurePathWithinBase(outside) should fail")
	}
}
