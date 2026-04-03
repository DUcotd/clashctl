package mihomo

import (
	"fmt"
	"time"

	"clashctl/internal/core"
)

type processStarter interface {
	Start() error
}

// StartOptions controls runtime startup behavior.
type StartOptions struct {
	VerifyInventory bool
	WaitRetries     int
	WaitInterval    time.Duration
}

// StartResult summarizes a runtime startup attempt.
type StartResult struct {
	Binary            *InstallResult
	GeoData           *GeoDataResult
	GeoDataError      string
	StartedBy         string
	ServiceStopped    bool
	ProcessStopped    bool
	ControllerReady   bool
	ControllerVersion string
	Inventory         *ProxyInventory
	InventoryError    string
	Warnings          []string
}

// RuntimeManager centralizes Mihomo runtime operations.
type RuntimeManager struct {
	ensureBinary          func() (*InstallResult, error)
	hasSystemd            func() bool
	serviceStatus         func(string) (bool, error)
	stopService           func(string) error
	setupSystemd          func(ServiceConfig, bool, bool) error
	stopManagedProcess    func(string) (bool, error)
	newProcess            func(string) processStarter
	geoDataReady          func(string) bool
	ensureGeoData         func(string) (*GeoDataResult, error)
	waitForController     func(string, string, int, time.Duration) error
	controllerVersion     func(string, string) (string, error)
	inspectProxyInventory func(string, string, string) (*ProxyInventory, error)
	canUseTUN             func() bool
	checkTUNPermission    func() error
}

// NewRuntimeManager creates a runtime manager with default dependencies.
func NewRuntimeManager() *RuntimeManager {
	return &RuntimeManager{
		ensureBinary: EnsureMihomo,
		hasSystemd:   HasSystemd,
		serviceStatus: func(name string) (bool, error) {
			return ServiceStatus(name)
		},
		stopService:        StopService,
		setupSystemd:       SetupSystemd,
		stopManagedProcess: StopManagedProcess,
		newProcess: func(configDir string) processStarter {
			return NewProcess(configDir)
		},
		geoDataReady:  GeoDataReady,
		ensureGeoData: EnsureGeoData,
		waitForController: func(addr, secret string, maxRetries int, interval time.Duration) error {
			return WaitForController(addr, secret, maxRetries, interval)
		},
		controllerVersion: func(addr, secret string) (string, error) {
			return NewClientWithSecret("http://"+addr, secret).Version()
		},
		inspectProxyInventory: func(addr, secret, group string) (*ProxyInventory, error) {
			return NewClientWithSecret("http://"+addr, secret).InspectProxyInventory(group)
		},
		canUseTUN:          CanUseTUN,
		checkTUNPermission: CheckTUNPermission,
	}
}

// ResolveConfig normalizes the config for local runtime use.
func (m *RuntimeManager) ResolveConfig(cfg *core.AppConfig) (*core.AppConfig, []string) {
	next := *cfg
	if next.Mode != "tun" {
		return &next, nil
	}

	if !m.canUseTUN() {
		next.Mode = "mixed"
		return &next, []string{"TUN 不可用，已自动降级到 mixed-port 模式"}
	}
	if err := m.checkTUNPermission(); err != nil {
		next.Mode = "mixed"
		return &next, []string{err.Error() + "，已自动降级到 mixed-port 模式"}
	}
	return &next, nil
}

// EnsureBinary resolves a usable Mihomo binary, installing one if needed.
func (m *RuntimeManager) EnsureBinary() (*InstallResult, error) {
	return m.ensureBinary()
}

// Start launches Mihomo using the resolved binary.
func (m *RuntimeManager) Start(cfg *core.AppConfig, opts StartOptions) (*StartResult, error) {
	binary, err := m.EnsureBinary()
	if err != nil {
		return nil, err
	}
	return m.StartWithBinary(cfg, binary, opts)
}

// StartWithBinary launches Mihomo using a caller-provided binary resolution.
func (m *RuntimeManager) StartWithBinary(cfg *core.AppConfig, binary *InstallResult, opts StartOptions) (*StartResult, error) {
	if opts.WaitRetries <= 0 {
		opts.WaitRetries = 15
	}
	if opts.WaitInterval <= 0 {
		opts.WaitInterval = 2 * time.Second
	}

	if cfg.Mode == "tun" {
		if !m.canUseTUN() {
			return nil, fmt.Errorf("TUN 模式当前不可用，请重新生成 mixed-port 配置或修复 /dev/net/tun 与 iptables 环境")
		}
		if err := m.checkTUNPermission(); err != nil {
			return nil, err
		}
	}

	result := &StartResult{Binary: binary}

	if !m.geoDataReady(cfg.ConfigDir) {
		geoData, err := m.ensureGeoData(cfg.ConfigDir)
		result.GeoData = geoData
		if err != nil {
			result.GeoDataError = err.Error()
			result.Warnings = append(result.Warnings, err.Error())
		}
	}

	if m.hasSystemd() {
		if active, _ := m.serviceStatus(DefaultServiceName); active {
			if err := m.stopService(DefaultServiceName); err != nil {
				result.Warnings = append(result.Warnings, "停止已有 systemd 服务失败: "+err.Error())
			} else {
				result.ServiceStopped = true
			}
		}
	}

	if stopped, err := m.stopManagedProcess(cfg.ConfigDir); err != nil {
		result.Warnings = append(result.Warnings, "清理旧进程失败: "+err.Error())
	} else {
		result.ProcessStopped = stopped
	}

	startedBySystemd := false
	if cfg.EnableSystemd && m.hasSystemd() {
		svcCfg := ServiceConfig{
			Binary:      binary.Path,
			ConfigDir:   cfg.ConfigDir,
			ServiceName: DefaultServiceName,
		}
		if err := m.setupSystemd(svcCfg, cfg.AutoStart, true); err == nil {
			startedBySystemd = true
			result.StartedBy = "systemd"
		} else {
			result.Warnings = append(result.Warnings, "systemd 启动失败: "+err.Error())
		}
	}

	if !startedBySystemd {
		proc := m.newProcess(cfg.ConfigDir)
		if err := proc.Start(); err != nil {
			return result, fmt.Errorf("启动失败: %w", err)
		}
		result.StartedBy = "process"
	}

	if err := m.waitForController(cfg.ControllerAddr, cfg.ControllerSecret, opts.WaitRetries, opts.WaitInterval); err != nil {
		return result, fmt.Errorf("Controller API 未就绪: %w", err)
	}
	result.ControllerReady = true
	result.ControllerVersion, _ = m.controllerVersion(cfg.ControllerAddr, cfg.ControllerSecret)

	if opts.VerifyInventory {
		inv, err := m.inspectProxyInventory(cfg.ControllerAddr, cfg.ControllerSecret, "PROXY")
		if err != nil {
			result.InventoryError = err.Error()
		} else {
			result.Inventory = inv
		}
	}

	return result, nil
}
