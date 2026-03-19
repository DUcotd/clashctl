// Package mihomo provides richer API interaction with enhanced node display.
package mihomo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// ProxyGroupDetail is the enhanced proxy group info with full node details.
type ProxyGroupDetail struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Now     string      `json:"now"`
	History []History   `json:"history"`
	All     []string    `json:"all"`
	Nodes   []ProxyNode `json:"-"` // populated by fetching each proxy
}

// History represents a connection history entry.
type History struct {
	Time  string `json:"time"`
	Delay int    `json:"delay"`
}

// ProxyNode represents a single proxy node with its details.
type ProxyNode struct {
	Name     string
	Type     string
	Protocol string // Vless, Hysteria2, Trojan, etc.
	Delay    int    // latest delay in ms, 0 = unknown, -1 = timeout
	Selected bool
}

// ProxyInventory summarizes whether a config has loaded usable proxy entries.
type ProxyInventory struct {
	Loaded         int
	Current        string
	Candidates     []string
	OnlyCompatible bool
}

// GetAllProxyGroups returns all proxy groups from the controller API.
func (c *Client) GetAllProxyGroups() (map[string]ProxyGroup, error) {
	url := fmt.Sprintf("%s/proxies", c.BaseURL)
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return nil, fmt.Errorf("无法连接 controller API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("controller API 返回 %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Proxies map[string]ProxyGroup `json:"proxies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
	}

	groups := make(map[string]ProxyGroup)
	for name, group := range result.Proxies {
		if !IsProxyGroupType(group.Type) {
			continue
		}
		groups[name] = group
	}

	return groups, nil
}

// NormalizeProxyType maps Mihomo API type names to stable lowercase values.
func NormalizeProxyType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "selector", "select":
		return "select"
	case "urltest", "url-test":
		return "url-test"
	case "fallback":
		return "fallback"
	case "loadbalance", "load-balance":
		return "load-balance"
	case "relay":
		return "relay"
	case "direct":
		return "direct"
	case "reject":
		return "reject"
	case "rejectdrop", "reject-drop":
		return "reject-drop"
	case "pass":
		return "pass"
	case "compatible":
		return "compatible"
	default:
		return strings.ToLower(strings.TrimSpace(t))
	}
}

// IsProxyGroupType reports whether a Mihomo proxy entry is a group rather than a single node.
func IsProxyGroupType(t string) bool {
	switch NormalizeProxyType(t) {
	case "select", "url-test", "fallback", "load-balance", "relay":
		return true
	default:
		return false
	}
}

// ProxyInfo represents a single proxy with its type.
type ProxyInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	ProviderName string `json:"provider-name"`
}

// GetAllProxies returns all individual proxies (not just groups) with their types.
func (c *Client) GetAllProxies() (map[string]ProxyInfo, error) {
	url := fmt.Sprintf("%s/proxies", c.BaseURL)
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return nil, fmt.Errorf("无法连接 controller API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("controller API 返回 %d", resp.StatusCode)
	}

	var result struct {
		Proxies map[string]ProxyInfo `json:"proxies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 API 响应失败: %w", err)
	}

	return result.Proxies, nil
}

// GetProxyGroupDetail fetches a proxy group with enriched node information.
func (c *Client) GetProxyGroupDetail(name string) (*ProxyGroupDetail, error) {
	group, err := c.GetProxyGroup(name)
	if err != nil {
		return nil, err
	}

	detail := &ProxyGroupDetail{
		Name:  group.Name,
		Type:  group.Type,
		Now:   group.Now,
		All:   group.All,
		Nodes: make([]ProxyNode, 0, len(group.All)),
	}

	// Copy history from the API response
	for _, h := range group.History {
		detail.History = append(detail.History, History{Time: h.Time, Delay: h.Delay})
	}

	for _, nodeName := range group.All {
		node := ProxyNode{
			Name:     nodeName,
			Selected: nodeName == group.Now,
		}

		// Get delay from group history
		if len(group.History) > 0 && nodeName == group.Now {
			node.Delay = group.History[len(group.History)-1].Delay
		}

		detail.Nodes = append(detail.Nodes, node)
	}

	return detail, nil
}

// SortNodesByDelay sorts nodes by delay (fastest first, unknown/timeout last).
func (d *ProxyGroupDetail) SortNodesByDelay() {
	sort.Slice(d.Nodes, func(i, j int) bool {
		ai, aj := d.Nodes[i].Delay, d.Nodes[j].Delay
		// 0 = unknown, -1 = timeout; both sorted last, unknown before timeout
		if ai <= 0 && aj <= 0 {
			return ai > aj // unknown (0) before timeout (-1)
		}
		if ai <= 0 {
			return false
		}
		if aj <= 0 {
			return true
		}
		return ai < aj
	})
}

// TestProxyGroupNodes refreshes delay for all nodes in a group with bounded concurrency.
func (c *Client) TestProxyGroupNodes(name string, maxConcurrent int) (*ProxyGroupDetail, error) {
	detail, err := c.GetProxyGroupDetail(name)
	if err != nil {
		return nil, err
	}

	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for i := range detail.Nodes {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			detail.Nodes[idx].Delay = c.TestNode(name, detail.Nodes[idx].Name)
		}(i)
	}

	wg.Wait()
	detail.SortNodesByDelay()
	return detail, nil
}

// FormatDelay returns a human-readable delay string.
func FormatDelay(delay int) string {
	switch {
	case delay == 0:
		return "未测试"
	case delay < 0:
		return "超时"
	case delay < 100:
		return fmt.Sprintf("%dms ✨", delay)
	case delay < 300:
		return fmt.Sprintf("%dms", delay)
	case delay < 1000:
		return fmt.Sprintf("%dms ⚠️", delay)
	default:
		return fmt.Sprintf("%.1fs 🔴", float64(delay)/1000)
	}
}

// InspectProxyInventory inspects the PROXY group and reports whether real nodes were loaded.
func (c *Client) InspectProxyInventory(groupName string) (*ProxyInventory, error) {
	group, err := c.GetProxyGroup(groupName)
	if err != nil {
		return nil, err
	}
	inv := &ProxyInventory{
		Loaded:     len(group.All),
		Current:    group.Now,
		Candidates: append([]string{}, group.All...),
	}
	inv.OnlyCompatible = inv.Loaded == 1 && (group.All[0] == "COMPATIBLE" || group.All[0] == "DIRECT")
	return inv, nil
}
