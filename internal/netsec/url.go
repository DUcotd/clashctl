package netsec

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

const localSubscriptionOverrideEnv = "CLASHCTL_ALLOW_LOCAL_SUBSCRIPTION"

var lookupIPAddr = net.DefaultResolver.LookupIPAddr

type URLValidationOptions struct {
	ResolveHost bool
	AllowLocal  bool
	Timeout     time.Duration
}

type ResolvedHost struct {
	Host  string
	Addrs []net.IPAddr
}

// AllowLocalSubscriptionTargets returns whether local/private subscription targets are allowed.
func AllowLocalSubscriptionTargets() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(localSubscriptionOverrideEnv)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ValidateRemoteHTTPURL validates an HTTP(S) URL and optionally rejects local/private targets.
func ValidateRemoteHTTPURL(rawURL string, opts URLValidationOptions) (*url.URL, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, fmt.Errorf("URL 不能为空")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("仅支持 http/https 协议，当前: %s", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("URL 缺少主机名")
	}
	if strings.Contains(u.Host, "://") {
		return nil, fmt.Errorf("URL 包含非法字符")
	}

	if _, err := ResolveRemoteHost(u.Hostname(), opts); err != nil {
		return nil, err
	}
	return u, nil
}

// ResolveRemoteHost validates a target host and optionally resolves it to safe IPs.
func ResolveRemoteHost(host string, opts URLValidationOptions) (*ResolvedHost, error) {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return nil, fmt.Errorf("URL 缺少主机名")
	}

	if opts.AllowLocal || AllowLocalSubscriptionTargets() {
		if ip := net.ParseIP(host); ip != nil {
			return &ResolvedHost{Host: host, Addrs: []net.IPAddr{{IP: ip}}}, nil
		}
		if !opts.ResolveHost {
			return &ResolvedHost{Host: host}, nil
		}
		addrs, err := lookupResolvedHost(host, opts.Timeout)
		if err != nil {
			return nil, fmt.Errorf("无法解析主机 %s: %w", host, err)
		}
		return &ResolvedHost{Host: host, Addrs: addrs}, nil
	}

	if isLocalHostname(host) {
		return nil, fmt.Errorf("不允许使用本地或内网地址: %s", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return nil, fmt.Errorf("不允许使用本地或内网地址: %s", host)
		}
		return &ResolvedHost{Host: host, Addrs: []net.IPAddr{{IP: ip}}}, nil
	}
	if !opts.ResolveHost {
		return &ResolvedHost{Host: host}, nil
	}

	addrs, err := lookupResolvedHost(host, opts.Timeout)
	if err != nil {
		return nil, fmt.Errorf("无法安全解析主机 %s: %w", host, err)
	}
	for _, addr := range addrs {
		if isPrivateIP(addr.IP) {
			return nil, fmt.Errorf("不允许使用本地或内网地址: %s", host)
		}
	}
	return &ResolvedHost{Host: host, Addrs: addrs}, nil
}

func lookupResolvedHost(host string, timeout time.Duration) ([]net.IPAddr, error) {
	if host == "" {
		return nil, fmt.Errorf("host 不能为空")
	}
	timeout = resolveTimeout(timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return lookupIPAddr(ctx, host)
}

func resolveTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 3 * time.Second
	}
	return timeout
}

// IsPrivateIP checks if an IP address is private, loopback, or otherwise non-routable.
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		privateCIDRs := []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"100.64.0.0/10",
			"169.254.0.0/16",
			"127.0.0.0/8",
		}
		for _, cidr := range privateCIDRs {
			_, network, _ := net.ParseCIDR(cidr)
			if network.Contains(ip4) {
				return true
			}
		}
		return false
	}
	privateCIDRs := []string{
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range privateCIDRs {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func isLocalHostname(host string) bool {
	return host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local")
}

func isPrivateIP(ip net.IP) bool {
	return IsPrivateIP(ip)
}
