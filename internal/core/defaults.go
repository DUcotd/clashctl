// Package core holds default values and constants.
package core

const (
	// AppVersion is the canonical version string for clashctl.
	// Update this in one place; all consumers reference it.
	AppVersion            = "v2.6.11"
	DefaultConfigDir      = "/etc/mihomo"
	DefaultControllerAddr = "127.0.0.1:9090"
	DefaultMixedPort      = 7890
	DefaultProviderPath   = "./providers/airport.yaml"
	DefaultServiceName    = "clashctl-mihomo"
)
