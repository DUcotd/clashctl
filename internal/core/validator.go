// Package core provides validation logic for clashctl configurations.
package core

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"clashctl/internal/netsec"
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
	} else if err := ValidateControllerAddr(cfg.ControllerAddr); err != nil {
		errs = append(errs, fmt.Sprintf("控制器地址不安全: %v", err))
	}

	return errs
}

// ValidateControllerAddr ensures the controller only binds to a local loopback address.
func ValidateControllerAddr(addr string) error {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return fmt.Errorf("控制器地址不能为空")
	}

	host, portText, err := net.SplitHostPort(trimmed)
	if err != nil {
		return fmt.Errorf("必须使用 host:port 格式: %w", err)
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("控制器主机不能为空")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("控制器端口超出范围: %s", portText)
	}

	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("仅允许监听本地回环地址，当前: %s", host)
	}
	return nil
}

// validateURL checks that a string is a valid HTTP/HTTPS URL.
func validateURL(rawURL string) error {
	_, err := netsec.ValidateRemoteHTTPURL(rawURL, netsec.URLValidationOptions{})
	if err != nil {
		return err
	}
	return nil
}
