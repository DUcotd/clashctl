// Package core defines the data models and default configurations for clashctl.
package core

// AppConfig holds the user-facing configuration collected from the wizard or CLI flags.
type AppConfig struct {
	SubscriptionURL   string `yaml:"subscription_url" mapstructure:"subscription_url"`
	Mode              string `yaml:"mode" mapstructure:"mode"` // "tun" or "mixed"
	ConfigDir         string `yaml:"config_dir" mapstructure:"config_dir"`
	ControllerAddr    string `yaml:"controller_addr" mapstructure:"controller_addr"`
	ControllerSecret  string `yaml:"controller_secret,omitempty" mapstructure:"controller_secret"`
	MixedPort         int    `yaml:"mixed_port" mapstructure:"mixed_port"`
	ProviderPath      string `yaml:"provider_path" mapstructure:"provider_path"`
	EnableHealthCheck bool   `yaml:"enable_health_check" mapstructure:"enable_health_check"`
	EnableSystemd     bool   `yaml:"enable_systemd" mapstructure:"enable_systemd"`
	AutoStart         bool   `yaml:"auto_start" mapstructure:"auto_start"`
}

// DefaultAppConfig returns an AppConfig with sensible defaults.
func DefaultAppConfig() *AppConfig {
	return &AppConfig{
		Mode:              "mixed",
		ConfigDir:         DefaultConfigDir,
		ControllerAddr:    DefaultControllerAddr,
		ControllerSecret:  GenerateControllerSecret(),
		MixedPort:         DefaultMixedPort,
		ProviderPath:      DefaultProviderPath,
		EnableHealthCheck: true,
		EnableSystemd:     true,
		AutoStart:         true,
	}
}
