// Package system provides shared HTTP helpers.
package system

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	// MaxRedirects limits the number of HTTP redirects to prevent redirect loops
	MaxRedirects = 5
	// DefaultTimeout is the default HTTP request timeout
	DefaultTimeout = 30 * time.Second
	// ConnectTimeout is the TCP connection timeout
	ConnectTimeout = 10 * time.Second
)

// HTTPDoer is the minimal interface implemented by http.Client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewHTTPClient creates an HTTP client with consistent timeout handling.
// When direct is true, proxy environment variables are ignored.
func NewHTTPClient(timeout time.Duration, direct bool) *http.Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}

	if direct {
		transport.Proxy = nil
		transport.DialContext = (&net.Dialer{
			Timeout:   ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return fmt.Errorf("重定向次数过多 (超过 %d 次)，可能存在重定向循环", MaxRedirects)
			}
			return nil
		},
	}
}

// NewHTTPClientWithRedirectLimit creates an HTTP client with custom redirect limit.
func NewHTTPClientWithRedirectLimit(timeout time.Duration, direct bool, maxRedirects int) *http.Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if maxRedirects <= 0 {
		maxRedirects = MaxRedirects
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       90 * time.Second,
	}

	if direct {
		transport.Proxy = nil
		transport.DialContext = (&net.Dialer{
			Timeout:   ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("重定向次数过多 (超过 %d 次)", maxRedirects)
			}
			return nil
		},
	}
}
