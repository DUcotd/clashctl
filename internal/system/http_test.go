package system

import (
	"net/http"
	"os"
	"testing"
)

func TestNewHTTPClientHonorsProxyEnvWhenNotDirect(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:7890")
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	client := NewHTTPClient(DefaultTimeout, false)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %#v", client.Transport)
	}

	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy() error = %v", err)
	}
	if proxyURL == nil || proxyURL.String() != os.Getenv("HTTP_PROXY") {
		t.Fatalf("proxyURL = %#v, want %q", proxyURL, os.Getenv("HTTP_PROXY"))
	}
}

func TestNewHTTPClientIgnoresProxyEnvWhenDirect(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:7890")
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	client := NewHTTPClient(DefaultTimeout, true)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %#v", client.Transport)
	}

	if transport.Proxy != nil {
		proxyURL, err := transport.Proxy(req)
		if err != nil {
			t.Fatalf("transport.Proxy() error = %v", err)
		}
		if proxyURL != nil {
			t.Fatalf("proxyURL = %#v, want nil", proxyURL)
		}
	}
}

func TestNewHTTPClientRejectsUnsafeRedirectTarget(t *testing.T) {
	client := NewHTTPClient(DefaultTimeout, false)
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080/sub", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	if err := client.CheckRedirect(req, nil); err == nil {
		t.Fatal("CheckRedirect() should reject localhost redirect target")
	}
}

func TestNewHTTPClientAllowsPublicRedirectTarget(t *testing.T) {
	client := NewHTTPClient(DefaultTimeout, false)
	req, err := http.NewRequest(http.MethodGet, "https://93.184.216.34/download", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	if err := client.CheckRedirect(req, nil); err != nil {
		t.Fatalf("CheckRedirect() error = %v, want nil", err)
	}
}
