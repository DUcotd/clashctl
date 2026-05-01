// Package system provides network utility functions.
package system

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// UserAgentVersion is set at startup from core.AppVersion.
// Default "dev" is overwritten by cmd.Execute() before any command runs.
var UserAgentVersion = "dev"

type URLProbeResult struct {
	StatusCode  int
	ContentKind string
	BodyPreview string
	UsedProxy   bool
}

func directHTTPClient(timeout time.Duration) *http.Client {
	return NewHTTPClient(timeout, true)
}

// CheckURLReachable performs an HTTP GET request to verify a URL is accessible.
// Uses GET instead of HEAD because many subscription servers don't support HEAD.
func CheckURLReachable(rawURL string, timeout time.Duration) error {
	_, err := ProbeURL(rawURL, timeout)
	return err
}

// ProbeURL fetches a URL with a lightweight GET and classifies the response shape.
func ProbeURL(rawURL string, timeout time.Duration) (*URLProbeResult, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("无法构建请求: %w", err)
	}
	// Some providers require a User-Agent to return proper content
	req.Header.Set("User-Agent", "clashctl/"+UserAgentVersion)

	client := directHTTPClient(timeout)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法访问 %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s 返回 HTTP %d", rawURL, resp.StatusCode)
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return &URLProbeResult{
		StatusCode:  resp.StatusCode,
		ContentKind: classifyBody(body),
		BodyPreview: string(body),
		UsedProxy:   hasProxyEnv(),
	}, nil
}

// FetchURLContent fetches the full body of a URL using a direct connection.
func FetchURLContent(rawURL string, timeout time.Duration, maxSize int64) ([]byte, *URLProbeResult, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("无法构建请求: %w", err)
	}
	req.Header.Set("User-Agent", "clashctl/"+UserAgentVersion)

	resp, err := directHTTPClient(timeout).Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("无法访问 %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("%s 返回 HTTP %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return nil, nil, fmt.Errorf("读取 %s 响应失败: %w", rawURL, err)
	}
	probe := &URLProbeResult{
		StatusCode:  resp.StatusCode,
		ContentKind: classifyBody(body),
		BodyPreview: string(body[:minInt(len(body), 4096)]),
		UsedProxy:   hasProxyEnv(),
	}
	return body, probe, nil
}

var (
	proxyEnvKeys = []string{
		"http_proxy", "https_proxy", "all_proxy",
		"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY",
	}
	proxyEnvDisplayKeys = func() []string {
		out := append([]string(nil), proxyEnvKeys...)
		return append(out, "NODE_USE_ENV_PROXY")
	}()
	proxyEnvStripKeys = func() []string {
		out := append([]string(nil), proxyEnvDisplayKeys...)
		return append(out, "no_proxy", "NO_PROXY")
	}()
)

func hasProxyEnv() bool {
	for _, key := range proxyEnvKeys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

// HasProxyEnvForDisplay reports whether the current shell exports proxy variables.
func HasProxyEnvForDisplay() bool {
	return hasProxyEnv()
}

// ProxyEnvForDisplay returns the currently exported proxy variables.
func ProxyEnvForDisplay() []string {
	seen := make(map[string]struct{}, len(proxyEnvDisplayKeys))
	var out []string
	for _, key := range proxyEnvDisplayKeys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		entry := key + "=" + value
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		out = append(out, entry)
	}
	return out
}

// StripProxyEnv removes proxy-related variables from an environment list.
func StripProxyEnv(env []string) []string {
	blocked := make(map[string]struct{}, len(proxyEnvStripKeys))
	for _, k := range proxyEnvStripKeys {
		blocked[k] = struct{}{}
	}
	out := make([]string, 0, len(env))
	for _, entry := range env {
		key := entry
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			key = entry[:idx]
		}
		if _, skip := blocked[key]; skip {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func classifyBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "empty"
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<html") || strings.Contains(lower, "<body") {
		return "html"
	}
	if strings.Contains(trimmed, "proxies:") || strings.Contains(trimmed, "proxy-groups:") || strings.Contains(trimmed, "mixed-port:") {
		return "mihomo-yaml"
	}
	if looksLikeRawLinks(trimmed) {
		return "raw-links"
	}
	compact := strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t', ' ':
			return -1
		default:
			return r
		}
	}, trimmed)
	if decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(compact))); err == nil && looksLikeRawLinks(string(decoded)) {
		return "base64-links"
	}
	return "unknown"
}

// ProbeContentKind classifies fetched subscription content.
func ProbeContentKind(body []byte) string {
	return classifyBody(body)
}

func looksLikeRawLinks(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "vless://") || strings.HasPrefix(line, "trojan://") || strings.HasPrefix(line, "hysteria2://") {
			return true
		}
	}
	return false
}

// CheckPortInUse checks if a TCP port is already in use.
func CheckPortInUse(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	_ = ln.Close()
	return false
}

// LookupHost resolves a hostname and returns the first IP address.
func LookupHost(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("未找到 %s 的解析地址", host)
	}
	return addrs[0], nil
}
