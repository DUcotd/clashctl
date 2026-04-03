package system

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeHTTPDoer func(*http.Request) (*http.Response, error)

func (f fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestExtractSHA256(t *testing.T) {
	tests := []struct {
		name    string
		content string
		target  string
		want    string
	}{
		{
			name:    "sha256sum format",
			content: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  clashctl-linux-amd64\n",
			target:  "clashctl-linux-amd64",
			want:    strings.Repeat("a", 64),
		},
		{
			name:    "bsd format",
			content: "SHA256 (clashctl-linux-amd64) = BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB\n",
			target:  "clashctl-linux-amd64",
			want:    strings.Repeat("b", 64),
		},
		{
			name:    "bare hash",
			content: "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC\n",
			target:  "ignored",
			want:    strings.Repeat("c", 64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractSHA256([]byte(tt.content), tt.target)
			if err != nil {
				t.Fatalf("ExtractSHA256() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ExtractSHA256() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDownloadFileWithOptionsAtomicPreservesOriginalOnChecksumMismatch(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "binary")
	if err := os.WriteFile(dest, []byte("original"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "https://example.com/file", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	err = DownloadFileWithOptions(fakeHTTPDoer(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("new-binary")),
		}, nil
	}), req, dest, DownloadOptions{ExpectedSHA256: strings.Repeat("0", 64), Atomic: true})
	if err == nil {
		t.Fatal("DownloadFileWithOptions() should fail checksum validation")
	}

	got, readErr := os.ReadFile(dest)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("dest content = %q, want original", string(got))
	}
}

func TestDownloadFileWithOptionsAtomicWritesVerifiedFile(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "binary")
	body := []byte("verified-binary")
	hash := sha256.Sum256(body)

	req, err := http.NewRequest(http.MethodGet, "https://example.com/file", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	err = DownloadFileWithOptions(fakeHTTPDoer(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(body))),
		}, nil
	}), req, dest, DownloadOptions{ExpectedSHA256: hex.EncodeToString(hash[:]), Atomic: true})
	if err != nil {
		t.Fatalf("DownloadFileWithOptions() error = %v", err)
	}

	got, readErr := os.ReadFile(dest)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(got) != string(body) {
		t.Fatalf("dest content = %q, want %q", string(got), string(body))
	}
}

func TestDownloadBytesWithDoerLimitRejectsOversizedBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/file", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	_, err = DownloadBytesWithDoerLimit(fakeHTTPDoer(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("12345")),
		}, nil
	}), req, 4)
	if err == nil {
		t.Fatal("DownloadBytesWithDoerLimit() should reject oversized bodies")
	}
	if !strings.Contains(err.Error(), "响应体过大") {
		t.Fatalf("DownloadBytesWithDoerLimit() error = %v", err)
	}
}

func TestFetchJSONWithDoerRejectsOversizedBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/meta.json", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	oversizedErr := FetchJSONWithDoer(fakeHTTPDoer(func(*http.Request) (*http.Response, error) {
		body := strings.Repeat("a", MaxJSONResponseBytes+1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}), req, &map[string]any{})
	if oversizedErr == nil {
		t.Fatal("FetchJSONWithDoer() should reject oversized bodies")
	}
	if !strings.Contains(oversizedErr.Error(), "JSON 响应体过大") {
		t.Fatalf("FetchJSONWithDoer() error = %v", oversizedErr)
	}
}
