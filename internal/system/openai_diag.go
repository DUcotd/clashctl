// Package system provides network diagnostics for OpenAI/Codex login flows.
package system

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"clashctl/internal/core"
)

// HTTPProbeResult captures the HTTP-level outcome of a reachability probe.
type HTTPProbeResult struct {
	URL         string
	FinalURL    string
	StatusCode  int
	BodyPreview string
}

// EgressInfo describes the externally observed IP and country/region.
type EgressInfo struct {
	Source      string
	IP          string
	Country     string
	CountryCode string
}

// NewProxyHTTPClient creates an HTTP client that always uses the given proxy URL.
func NewProxyHTTPClient(timeout time.Duration, proxyURL string) (*http.Client, error) {
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("解析代理地址失败: %w", err)
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyURL(parsed),
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

// DetectEgressInfo queries a public endpoint to determine the current egress country/region.
func DetectEgressInfo(client HTTPDoer) (*EgressInfo, error) {
	var errs []string
	for _, rawURL := range []string{"https://ifconfig.co/json", "https://ifconfig.io/json"} {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("无法构建请求: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "clashctl/"+core.AppVersion)

		resp, err := client.Do(req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", rawURL, err))
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8192))
		resp.Body.Close()
		if readErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", rawURL, readErr))
			continue
		}
		if resp.StatusCode >= 400 {
			errs = append(errs, fmt.Sprintf("%s: HTTP %d", rawURL, resp.StatusCode))
			continue
		}

		info, err := parseEgressInfo(body)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", rawURL, err))
			continue
		}
		info.Source = rawURL
		return info, nil
	}

	return nil, fmt.Errorf("无法获取出口信息: %s", strings.Join(errs, "; "))
}

// ProbeEndpoint performs a lightweight GET and captures the resulting HTTP status/body.
func ProbeEndpoint(client HTTPDoer, rawURL string) (*HTTPProbeResult, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("无法构建请求: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/plain;q=0.9, */*;q=0.8")
	req.Header.Set("User-Agent", "clashctl/"+core.AppVersion)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	return &HTTPProbeResult{
		URL:         rawURL,
		FinalURL:    finalURL,
		StatusCode:  resp.StatusCode,
		BodyPreview: strings.TrimSpace(string(body)),
	}, nil
}

func parseEgressInfo(body []byte) (*EgressInfo, error) {
	var payload struct {
		IP          string `json:"ip"`
		Country     string `json:"country"`
		CountryISO  string `json:"country_iso"`
		CountryCode string `json:"country_code"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析出口信息失败: %w", err)
	}

	code := strings.TrimSpace(payload.CountryISO)
	if code == "" {
		code = strings.TrimSpace(payload.CountryCode)
	}
	info := &EgressInfo{
		IP:          strings.TrimSpace(payload.IP),
		Country:     strings.TrimSpace(payload.Country),
		CountryCode: strings.ToUpper(code),
	}
	if info.IP == "" && info.Country == "" && info.CountryCode == "" {
		return nil, fmt.Errorf("返回内容缺少 IP 和国家/地区字段")
	}
	return info, nil
}
