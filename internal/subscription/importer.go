package subscription

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type ParseResult struct {
	DetectedFormat string
	Proxies        []map[string]any
	Names          []string
}

func Parse(content []byte) (*ParseResult, error) {
	text := strings.TrimSpace(string(content))
	if text == "" {
		return nil, fmt.Errorf("订阅内容为空")
	}

	decoded, format := normalize(text)
	lines := splitLines(decoded)
	if len(lines) == 0 {
		return nil, fmt.Errorf("未找到可解析的节点链接")
	}

	proxies := make([]map[string]any, 0, len(lines))
	names := make([]string, 0, len(lines))
	seen := map[string]int{}

	for _, line := range lines {
		proxy, name, err := parseLine(line)
		if err != nil {
			continue
		}
		seen[name]++
		if seen[name] > 1 {
			name = fmt.Sprintf("%s-%d", name, seen[name])
			proxy["name"] = name
		}
		proxies = append(proxies, proxy)
		names = append(names, name)
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("未能从订阅内容中解析出支持的节点；当前支持 vless/trojan/hysteria2")
	}

	return &ParseResult{
		DetectedFormat: format,
		Proxies:        proxies,
		Names:          names,
	}, nil
}

func normalize(text string) (string, string) {
	if looksLikeLinks(text) {
		return text, "raw-links"
	}

	compact := strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t', ' ':
			return -1
		default:
			return r
		}
	}, text)

	if decoded, err := base64.StdEncoding.DecodeString(compact); err == nil {
		decodedText := strings.TrimSpace(string(decoded))
		if looksLikeLinks(decodedText) {
			return decodedText, "base64-links"
		}
	}

	return text, "unknown"
}

func looksLikeLinks(text string) bool {
	for _, line := range splitLines(text) {
		if strings.HasPrefix(line, "vless://") || strings.HasPrefix(line, "trojan://") || strings.HasPrefix(line, "hysteria2://") {
			return true
		}
	}
	return false
}

func splitLines(text string) []string {
	s := bufio.NewScanner(strings.NewReader(text))
	var out []string
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func parseLine(line string) (map[string]any, string, error) {
	u, err := url.Parse(line)
	if err != nil {
		return nil, "", err
	}

	name := decodeName(u.Fragment)
	if name == "" {
		name = u.Hostname()
	}

	switch u.Scheme {
	case "vless":
		return parseVLESS(u, name)
	case "trojan":
		return parseTrojan(u, name)
	case "hysteria2":
		return parseHysteria2(u, name)
	default:
		return nil, "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}

func parseVLESS(u *url.URL, name string) (map[string]any, string, error) {
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return nil, "", err
	}
	q := u.Query()
	proxy := map[string]any{
		"name":    name,
		"type":    "vless",
		"server":  u.Hostname(),
		"port":    port,
		"uuid":    user(u),
		"network": defaultString(q.Get("type"), "tcp"),
		"udp":     true,
	}
	if security := strings.ToLower(q.Get("security")); security == "reality" || security == "tls" {
		proxy["tls"] = true
	}
	if v := q.Get("sni"); v != "" {
		proxy["servername"] = v
	}
	if v := q.Get("flow"); v != "" {
		proxy["flow"] = v
	}
	if v := q.Get("fp"); v != "" {
		proxy["client-fingerprint"] = v
	}
	if q.Get("security") == "reality" {
		reality := map[string]any{}
		if v := q.Get("pbk"); v != "" {
			reality["public-key"] = v
		}
		if v := q.Get("sid"); v != "" {
			reality["short-id"] = v
		}
		if len(reality) > 0 {
			proxy["reality-opts"] = reality
		}
	}
	return proxy, name, nil
}

func parseTrojan(u *url.URL, name string) (map[string]any, string, error) {
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return nil, "", err
	}
	q := u.Query()
	proxy := map[string]any{
		"name":     name,
		"type":     "trojan",
		"server":   u.Hostname(),
		"port":     port,
		"password": user(u),
		"udp":      true,
	}
	if v := q.Get("sni"); v != "" {
		proxy["sni"] = v
	}
	if insecure := q.Get("allowInsecure"); insecure == "1" || strings.EqualFold(insecure, "true") {
		proxy["skip-cert-verify"] = true
	}
	if strings.EqualFold(q.Get("type"), "ws") {
		proxy["network"] = "ws"
		ws := map[string]any{}
		if v := q.Get("path"); v != "" {
			ws["path"] = v
		}
		host := q.Get("host")
		if host == "" {
			host = q.Get("peer")
		}
		if host != "" {
			ws["headers"] = map[string]any{"Host": host}
		}
		if len(ws) > 0 {
			proxy["ws-opts"] = ws
		}
	}
	return proxy, name, nil
}

func parseHysteria2(u *url.URL, name string) (map[string]any, string, error) {
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return nil, "", err
	}
	q := u.Query()
	proxy := map[string]any{
		"name":     name,
		"type":     "hysteria2",
		"server":   u.Hostname(),
		"port":     port,
		"password": user(u),
	}
	if v := q.Get("sni"); v != "" {
		proxy["sni"] = v
	}
	if insecure := q.Get("insecure"); insecure == "1" || strings.EqualFold(insecure, "true") {
		proxy["skip-cert-verify"] = true
	}
	return proxy, name, nil
}

func user(u *url.URL) string {
	if u.User == nil {
		return ""
	}
	return u.User.Username()
}

func decodeName(raw string) string {
	if raw == "" {
		return ""
	}
	name, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return sanitizeNodeName(name)
}

// sanitizeNodeName cleans up node names for safe use in configs.
// It removes or replaces characters that could cause issues.
func sanitizeNodeName(name string) string {
	if name == "" {
		return "unnamed"
	}

	var result []rune
	for _, r := range name {
		switch {
		case r >= 0x20 && r < 0x7F:
			// ASCII printable characters
			if r == '"' || r == '\\' || r == '\n' || r == '\r' || r == '\t' {
				// Skip or replace problematic characters
				continue
			}
			result = append(result, r)
		case r >= 0x00A0:
			// Unicode characters (including CJK) - keep them
			result = append(result, r)
		default:
			// Skip control characters
			continue
		}
	}

	sanitized := string(result)
	if sanitized == "" {
		return "unnamed"
	}

	// Trim spaces
	sanitized = strings.TrimSpace(sanitized)
	if sanitized == "" {
		return "unnamed"
	}

	return sanitized
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func SortedNames(names []string) []string {
	out := append([]string{}, names...)
	sort.Strings(out)
	return out
}
