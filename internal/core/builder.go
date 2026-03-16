// Package core provides configuration building logic.
package core

// BuildMihomoConfig generates a MihomoConfig from the given AppConfig.
func BuildMihomoConfig(cfg *AppConfig) *MihomoConfig {
	m := &MihomoConfig{
		MixedPort:         cfg.MixedPort,
		AllowLan:          false,
		Mode:              "rule",
		LogLevel:          "info",
		ExternalController: cfg.ControllerAddr,
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
		},
		DNS: &DNSConfig{
			Enable:       true,
			IPv6:         false,
			EnhancedMode: "fake-ip",
			NameServer:   []string{
				"https://1.1.1.1/dns-query",
				"https://dns.google/dns-query",
			},
		},
		Rules: []string{
			"MATCH,PROXY",
		},
	}

	// Add TUN config only in TUN mode
	if cfg.Mode == "tun" {
		m.TUN = &TUNConfig{
			Enable:              true,
			Stack:               "mixed",
			AutoRoute:           true,
			AutoRedirect:        true,
			AutoDetectInterface: true,
			DNSHijack: []string{
				"any:53",
				"tcp://any:53",
			},
		}
	}

	return m
}
