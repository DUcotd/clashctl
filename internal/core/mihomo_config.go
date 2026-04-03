// Package core defines Mihomo YAML configuration structures.
package core

// MihomoConfig represents the full Mihomo configuration file.
type MihomoConfig struct {
	MixedPort          int                       `yaml:"mixed-port"`
	AllowLan           bool                      `yaml:"allow-lan"`
	Mode               string                    `yaml:"mode"`
	LogLevel           string                    `yaml:"log-level"`
	ExternalController string                    `yaml:"external-controller"`
	Secret             string                    `yaml:"secret,omitempty"`
	Proxies            []map[string]any          `yaml:"proxies,omitempty"`
	ProxyProviders     map[string]*ProxyProvider `yaml:"proxy-providers,omitempty"`
	ProxyGroups        []*ProxyGroup             `yaml:"proxy-groups"`
	DNS                *DNSConfig                `yaml:"dns,omitempty"`
	TUN                *TUNConfig                `yaml:"tun,omitempty"`
	Rules              []string                  `yaml:"rules"`
}

// ProxyProvider defines a proxy provider (typically an HTTP subscription).
type ProxyProvider struct {
	Type        string       `yaml:"type"`
	URL         string       `yaml:"url"`
	Path        string       `yaml:"path"`
	Interval    int          `yaml:"interval"`
	Override    *Override    `yaml:"override,omitempty"`
	HealthCheck *HealthCheck `yaml:"health-check"`
}

// Override defines provider-level overrides.
type Override struct {
	SkipCertVerify bool `yaml:"skip-cert-verify,omitempty"`
}

// HealthCheck defines the health check configuration for a provider.
type HealthCheck struct {
	Enable   bool   `yaml:"enable"`
	URL      string `yaml:"url"`
	Interval int    `yaml:"interval"`
}

// ProxyGroup defines a proxy group.
type ProxyGroup struct {
	Name     string   `yaml:"name" json:"name"`
	Type     string   `yaml:"type" json:"type"`
	Now      string   `yaml:"now,omitempty" json:"now,omitempty"`
	All      []string `yaml:"all,omitempty" json:"all,omitempty"`
	Proxies  []string `yaml:"proxies,omitempty" json:"proxies,omitempty"`
	Use      []string `yaml:"use,omitempty" json:"use,omitempty"`
	URL      string   `yaml:"url,omitempty" json:"url,omitempty"`
	Interval int      `yaml:"interval,omitempty" json:"interval,omitempty"`
}

// DNSConfig defines the DNS configuration.
type DNSConfig struct {
	Enable            bool     `yaml:"enable"`
	IPv6              bool     `yaml:"ipv6"`
	EnhancedMode      string   `yaml:"enhanced-mode"`
	FakeIPRange       string   `yaml:"fake-ip-range,omitempty"`
	NameServer        []string `yaml:"nameserver"`
	Fallback          []string `yaml:"fallback,omitempty"`
	DefaultNameserver []string `yaml:"default-nameserver,omitempty"`
	DirectNameserver  []string `yaml:"direct-nameserver,omitempty"`
}

// TUNConfig defines the TUN device configuration.
type TUNConfig struct {
	Enable              bool     `yaml:"enable"`
	Stack               string   `yaml:"stack"`
	AutoRoute           bool     `yaml:"auto-route"`
	AutoRedirect        bool     `yaml:"auto-redirect"`
	AutoDetectInterface bool     `yaml:"auto-detect-interface"`
	DNSHijack           []string `yaml:"dns-hijack"`
}
