package core

// BuildStaticMihomoConfig builds a standalone Mihomo config from inline proxies.
func BuildStaticMihomoConfig(cfg *AppConfig, proxies []map[string]any, names []string) *MihomoConfig {
	m := &MihomoConfig{
		MixedPort:          cfg.MixedPort,
		AllowLan:           false,
		Mode:               "rule",
		LogLevel:           "info",
		ExternalController: cfg.ControllerAddr,
		Secret:             cfg.ControllerSecret,
		Proxies:            proxies,
		ProxyGroups: []*ProxyGroup{
			{
				Name:    "PROXY",
				Type:    "select",
				Proxies: append(append([]string{}, names...), "DIRECT"),
			},
		},
		DNS: &DNSConfig{
			Enable:       true,
			IPv6:         false,
			EnhancedMode: "redir-host",
			NameServer: []string{
				"223.5.5.5",
				"119.29.29.29",
			},
			Fallback: []string{
				"https://1.1.1.1/dns-query",
				"https://dns.google/dns-query",
				"tls://8.8.4.4:853",
			},
			DefaultNameserver: []string{
				"223.5.5.5",
				"119.29.29.29",
			},
			DirectNameserver: []string{
				"223.5.5.5",
				"119.29.29.29",
			},
		},
		Rules: []string{
			"GEOSITE,cn,DIRECT",
			"GEOIP,CN,DIRECT",
			"MATCH,PROXY",
		},
	}

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
