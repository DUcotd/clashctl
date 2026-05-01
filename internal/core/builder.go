// Package core provides configuration building logic.
package core

import (
	"net/url"
)

// extractSubscriptionDomain extracts the hostname from a subscription URL
// so we can add a DIRECT rule to prevent the chicken-and-egg problem
// where Mihomo tries to fetch the subscription through itself.
func extractSubscriptionDomain(subURL string) string {
	u, err := url.Parse(subURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// BuildMihomoConfig generates a MihomoConfig from the given AppConfig.
func BuildMihomoConfig(cfg *AppConfig) *MihomoConfig {
	m := &MihomoConfig{
		MixedPort:          cfg.MixedPort,
		AllowLan:           false,
		Mode:               "rule",
		LogLevel:           "info",
		ExternalController: cfg.ControllerAddr,
		Secret:             cfg.ControllerSecret,
		ProxyProviders: map[string]*ProxyProvider{
			"airport": {
				Type:     "http",
				URL:      cfg.SubscriptionURL,
				Path:     cfg.ProviderPath,
				Interval: 3600,
				HealthCheck: &HealthCheck{
					Enable:   cfg.EnableHealthCheck,
					URL:      "https://cp.cloudflare.com/",
					Interval: 300,
				},
			},
		},
		ProxyGroups: []*ProxyGroup{
			{
				Name: "PROXY",
				Type: "select",
				Use:  []string{"airport"},
			},
			{
				Name:     "auto",
				Type:     "url-test",
				Use:      []string{"airport"},
				URL:      "https://cp.cloudflare.com/",
				Interval: 300,
			},
			{
				Name:     "fallback",
				Type:     "fallback",
				Use:      []string{"airport"},
				URL:      "https://cp.cloudflare.com/",
				Interval: 300,
			},
		},
		DNS: defaultDNSConfig(),
	}

	// Build rules list
	rules := []string{
		// Local/lan traffic
		"DOMAIN-SUFFIX,local,DIRECT",
		"IP-CIDR,127.0.0.0/8,DIRECT",
		"IP-CIDR,172.16.0.0/12,DIRECT",
		"IP-CIDR,192.168.0.0/16,DIRECT",
		"IP-CIDR,10.0.0.0/8,DIRECT",
		"IP-CIDR,100.64.0.0/10,DIRECT",
	}

	// Fix: Add subscription domain as DIRECT to avoid chicken-and-egg problem
	if subDomain := extractSubscriptionDomain(cfg.SubscriptionURL); subDomain != "" {
		rules = append(rules, "DOMAIN,"+subDomain+",DIRECT")
	}

	// Common Chinese/global domains that GeoSite may not include
	// but should always go direct to avoid proxy issues
	rules = append(rules,
		"DOMAIN-SUFFIX,ubuntu.com,DIRECT",
		"DOMAIN-SUFFIX,github.com,DIRECT",
		"DOMAIN-SUFFIX,githubusercontent.com,DIRECT",
		"DOMAIN-SUFFIX,docker.io,DIRECT",
		"DOMAIN-SUFFIX,registry-1.docker.io,DIRECT",
		"DOMAIN-SUFFIX,pypi.org,DIRECT",
		"DOMAIN-SUFFIX,npmjs.org,DIRECT",
		"DOMAIN-SUFFIX,gcr.io,DIRECT",
		"DOMAIN-SUFFIX,k8s.io,DIRECT",
		"DOMAIN-SUFFIX,cloudflare.com,DIRECT",
		"DOMAIN-SUFFIX,amazonaws.com,DIRECT",
	)

	// China mainland - direct
	rules = append(rules,
		"GEOSITE,cn,DIRECT",
		"GEOIP,CN,DIRECT",
	)

	// Fallback
	rules = append(rules, "MATCH,PROXY")
	m.Rules = rules

	// Add TUN config only in TUN mode
	if cfg.Mode == "tun" {
		m.TUN = DefaultTUNConfig()
	}

	return m
}
