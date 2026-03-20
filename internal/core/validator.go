// Package core provides validation logic for clashctl configurations.
package core

import (
	"fmt"
	"net/url"
	"strings"
)

// Validate checks the AppConfig for errors and returns a list of problems.
func (cfg *AppConfig) Validate() []string {
	var errs []string

	if strings.TrimSpace(cfg.SubscriptionURL) == "" {
		errs = append(errs, "订阅 URL 不能为空")
	} else if err := validateURL(cfg.SubscriptionURL); err != nil {
		errs = append(errs, fmt.Sprintf("订阅 URL 格式非法: %v", err))
	}

	if cfg.Mode != "tun" && cfg.Mode != "mixed" {
		errs = append(errs, fmt.Sprintf("不支持的运行模式: %q (可选: tun, mixed)", cfg.Mode))
	}

	if cfg.MixedPort < 1 || cfg.MixedPort > 65535 {
		errs = append(errs, fmt.Sprintf("mixed-port 超出范围: %d (1-65535)", cfg.MixedPort))
	}

	if strings.TrimSpace(cfg.ConfigDir) == "" {
		errs = append(errs, "配置目录不能为空")
	}

	if strings.TrimSpace(cfg.ControllerAddr) == "" {
		errs = append(errs, "控制器地址不能为空")
	}

	return errs
}

// validateURL checks that a string is a valid HTTP/HTTPS URL.
func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https 协议，当前: %s", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("URL 缺少主机名")
	}
	// Block file:// protocol masquerading as host
	if strings.Contains(u.Host, "://") {
		return fmt.Errorf("URL 包含非法字符")
	}
	// Block local/private addresses to prevent SSRF (optional, commented out for now)
	// This can be enabled for stricter security
	// if isPrivateHost(u.Hostname()) {
	//     return fmt.Errorf("不允许使用私有地址")
	// }
	return nil
}

// isPrivateHost checks if a hostname resolves to a private IP range.
// This is disabled by default but can be enabled for SSRF protection.
// func isPrivateHost(host string) bool {
//     ip := net.ParseIP(host)
//     if ip == nil {
//         return false // DNS name, not IP
//     }
//     privateRanges := []string{
//         "10.0.0.0/8",
//         "172.16.0.0/12",
//         "192.168.0.0/16",
//         "127.0.0.0/8",
//         "169.254.0.0/16",
//         "::1/128",
//         "fc00::/7",
//     }
//     for _, cidr := range privateRanges {
//         _, network, _ := net.ParseCIDR(cidr)
//         if network != nil && network.Contains(ip) {
//             return true
//         }
//     }
//     return false
// }
