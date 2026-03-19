// Package system provides shared HTTP helpers.
package system

import (
	"net"
	"net/http"
	"time"
)

// HTTPDoer is the minimal interface implemented by http.Client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewHTTPClient creates an HTTP client with consistent timeout handling.
// When direct is true, proxy environment variables are ignored.
func NewHTTPClient(timeout time.Duration, direct bool) *http.Client {
	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
	}

	if direct {
		transport.Proxy = nil
		transport.DialContext = (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
