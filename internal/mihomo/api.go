// Package mihomo provides interaction with the Mihomo controller API.
package mihomo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// APITimeout is the timeout for controller API requests.
	APITimeout = 5 * time.Second
	// APIResponseMaxSize is the maximum size for API error responses (1MB).
	APIResponseMaxSize = 1024 * 1024
	// DelayTestURL is the default URL used for active proxy delay checks.
	DelayTestURL = "https://cp.cloudflare.com/"
	// DelayTestTimeoutMS is the default timeout used for active proxy delay checks.
	DelayTestTimeoutMS = 5000
)

// Client communicates with the Mihomo controller REST API.
type Client struct {
	BaseURL string
	Secret  string
	HTTP    *http.Client
}

// NewClient creates a new Mihomo API client.
func NewClient(baseURL string) *Client {
	return NewClientWithSecret(baseURL, "")
}

// NewClientWithSecret creates a new Mihomo API client with optional bearer authentication.
func NewClientWithSecret(baseURL, secret string) *Client {
	return &Client{
		BaseURL: baseURL,
		Secret:  strings.TrimSpace(secret),
		HTTP:    &http.Client{Timeout: APITimeout},
	}
}

func (c *Client) setAuthHeader(req *http.Request) {
	if req == nil || c == nil || c.Secret == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.Secret)
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.setAuthHeader(req)
	return c.HTTP.Do(req)
}

func decodeAPIJSON(body io.Reader, dest any) error {
	data, err := io.ReadAll(io.LimitReader(body, APIResponseMaxSize+1))
	if err != nil {
		return fmt.Errorf("读取 API 响应失败: %w", err)
	}
	if len(data) > APIResponseMaxSize {
		return fmt.Errorf("API 响应体过大 (超过 %d bytes)", APIResponseMaxSize)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("解析 API 响应失败: %w", err)
	}
	return nil
}

// ProxyGroup represents a proxy group from the controller API.
type ProxyGroup struct {
	Name    string    `json:"name"`
	Type    string    `json:"type"`
	Now     string    `json:"now"`
	All     []string  `json:"all"`
	History []History `json:"history"`
}

// GetProxyGroup fetches details of a specific proxy group.
func (c *Client) GetProxyGroup(name string) (*ProxyGroup, error) {
	apiURL := fmt.Sprintf("%s/proxies/%s", c.BaseURL, url.PathEscape(name))
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接 controller API (%s): %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, APIResponseMaxSize))
		return nil, fmt.Errorf("controller API 返回 %d: %s", resp.StatusCode, string(body))
	}

	var group ProxyGroup
	if err := decodeAPIJSON(resp.Body, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// SwitchProxy switches the selected node in a proxy group.
func (c *Client) SwitchProxy(groupName, nodeName string) error {
	apiURL := fmt.Sprintf("%s/proxies/%s", c.BaseURL, url.PathEscape(groupName))
	body := map[string]string{"name": nodeName}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("构建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("切换节点请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, APIResponseMaxSize))
		return fmt.Errorf("切换节点失败 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// Version fetches the Mihomo version from the controller API.
func (c *Client) Version() (string, error) {
	url := fmt.Sprintf("%s/version", c.BaseURL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("构建请求失败: %w", err)
	}
	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("无法连接 controller API: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Version string `json:"version"`
	}
	if err := decodeAPIJSON(resp.Body, &result); err != nil {
		return "", fmt.Errorf("解析版本信息失败: %w", err)
	}
	return result.Version, nil
}

// CheckConnection tests if the controller API is reachable.
func (c *Client) CheckConnection() error {
	_, err := c.Version()
	return err
}

// TestNode actively tests a node's latency via the controller API
// (GET /proxies/{name}/delay?url=...&timeout=...).
// Returns the delay in ms (0 = no data, negative = timeout/error).
func (c *Client) TestNode(groupName, nodeName string) int {
	type delayInfo struct {
		Delay   int       `json:"delay"`
		History []History `json:"history"`
	}

	name := strings.TrimSpace(nodeName)
	if name == "" {
		name = strings.TrimSpace(groupName)
	}
	params := url.Values{
		"url":     []string{DelayTestURL},
		"timeout": []string{strconv.Itoa(DelayTestTimeoutMS)},
	}
	apiURL := fmt.Sprintf("%s/proxies/%s/delay?%s", c.BaseURL, url.PathEscape(name), params.Encode())
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return -1
	}
	resp, err := c.do(req)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1
	}

	var info delayInfo
	if err := decodeAPIJSON(resp.Body, &info); err != nil {
		return -1
	}

	if info.Delay > 0 {
		return info.Delay
	}
	if len(info.History) == 0 {
		return 0
	}
	return info.History[len(info.History)-1].Delay
}
