// Package system provides network utility functions.
package system

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// CheckURLReachable performs an HTTP HEAD request to verify a URL is accessible.
func CheckURLReachable(rawURL string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Head(rawURL)
	if err != nil {
		return fmt.Errorf("无法访问 %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s 返回 HTTP %d", rawURL, resp.StatusCode)
	}
	return nil
}

// CheckPortInUse checks if a TCP port is already in use.
func CheckPortInUse(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	ln.Close()
	return false
}

// LookupHost resolves a hostname and returns the first IP address.
func LookupHost(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("no addresses found for %s", host)
	}
	return addrs[0], nil
}
