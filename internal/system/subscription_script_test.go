package system

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestPrepareSubscriptionURLDownloadsContent(t *testing.T) {
	t.Setenv("CLASHCTL_ALLOW_LOCAL_SUBSCRIPTION", "1")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "trojan://secret@example.com:443#node")
	}))
	defer server.Close()

	prepared, err := PrepareSubscriptionURL(server.URL, 5*time.Second)
	if err != nil {
		t.Fatalf("PrepareSubscriptionURL() error = %v", err)
	}
	defer prepared.Cleanup()

	if got := string(prepared.Body); got != "trojan://secret@example.com:443#node" {
		t.Fatalf("prepared.Body = %q", got)
	}
	if !strings.Contains(prepared.FetchDetail, "fetcher=go-http") {
		t.Fatalf("FetchDetail = %q, want go-http marker", prepared.FetchDetail)
	}
	if !strings.Contains(prepared.FetchDetail, "status=200") {
		t.Fatalf("FetchDetail = %q, want status marker", prepared.FetchDetail)
	}
}

func TestDialPreparedSubscriptionRejectsPrivateTarget(t *testing.T) {
	dialer := &net.Dialer{Timeout: time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := dialPreparedSubscription(ctx, dialer, "tcp", "127.0.0.1:80", time.Second)
	if err == nil {
		t.Fatal("dialPreparedSubscription() should reject local targets")
	}
}
