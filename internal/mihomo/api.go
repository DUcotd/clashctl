// Package mihomo provides interaction with the Mihomo controller API.
package mihomo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// APITimeout is the timeout for controller API requests.
	APITimeout = 5 * time.Second
	// APIResponseMaxSize is the maximum size for API error responses (1MB).
	APIResponseMaxSize = 1024 * 1024
)

// Client communicates with the Mihomo controller REST API.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient creates a new Mihomo API client.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: APITimeout},
	}
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
	resp, err := c.HTTP.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("无法连接 controller API (%s): %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, APIResponseMaxSize))
		return nil, fmt.Errorf("controller API 返回 %d: %s", resp.StatusCode, string(body))
	}

	var group ProxyGroup
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
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

	resp, err := c.HTTP.Do(req)
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
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return "", fmt.Errorf("无法连接 controller API: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析版本信息失败: %w", err)
	}
	return result.Version, nil
}

// CheckConnection tests if the controller API is reachable.
func (c *Client) CheckConnection() error {
	_, err := c.Version()
	return err
}

// TestNode tests a node's latency via the controller API (GET /proxies/{group}/{name}).
// Returns the delay in ms (0 = no data, negative = timeout/error).
func (c *Client) TestNode(groupName, nodeName string) int {
	type proxyInfo struct {
		History []History `json:"history"`
	}

	apiURL := fmt.Sprintf("%s/proxies/%s/%s", c.BaseURL, url.PathEscape(groupName), url.PathEscape(nodeName))
	resp, err := c.HTTP.Get(apiURL)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1
	}

	var info proxyInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return -1
	}

	if len(info.History) == 0 {
		return 0
	}
	return info.History[len(info.History)-1].Delay
}
