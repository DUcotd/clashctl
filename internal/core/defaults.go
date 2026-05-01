// Package core holds default values and constants.
package core

const (
	// AppVersion is the canonical version string for clashctl.
	// Update this in one place; all consumers reference it.
	AppVersion            = "v2.7.2"
	DefaultConfigDir      = "/etc/mihomo"
	DefaultControllerAddr = "127.0.0.1:9090"
	DefaultMixedPort      = 7890
	DefaultProviderPath   = "./providers/airport.yaml"
	DefaultServiceName    = "clashctl-mihomo"
)

// defaultDNSConfig returns the default DNS configuration for Mihomo.
func defaultDNSConfig() *DNSConfig {
	return &DNSConfig{
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
	}
}

// DefaultTUNConfig returns the default TUN configuration for Mihomo.
func DefaultTUNConfig() *TUNConfig {
	return &TUNConfig{
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
