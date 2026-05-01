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
		DNS: defaultDNSConfig(),
		Rules: []string{
			"GEOSITE,cn,DIRECT",
			"GEOIP,CN,DIRECT",
			"MATCH,PROXY",
		},
	}

	if cfg.Mode == "tun" {
		m.TUN = DefaultTUNConfig()
	}

	return m
}
