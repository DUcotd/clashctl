// Package ui provides system integration execution steps.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clashctl/internal/app"
	"clashctl/internal/config"
	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	"clashctl/internal/subscription"
	"clashctl/internal/system"
)

// executeFull performs the full configuration and startup pipeline.
func (m WizardModel) executeFull() []ExecStep {
	var steps []ExecStep

	// Step 1: Validate URL (network check)
	steps = append(steps, m.stepCheckURL())

	// Step 2: Check mihomo binary
	binary, binaryOK := m.stepCheckBinary(&steps)
	if !binaryOK {
		return steps
	}

	// Step 3: Build config
	mihomoCfg, ok := m.stepBuildConfig(&steps)
	if !ok {
		return steps
	}

	// Step 4: Render YAML
	_, ok = m.stepRenderYAML(mihomoCfg, &steps)
	if !ok {
		return steps
	}

	// Step 5: Write config file
	configPath := m.appCfg.ConfigDir + "/config.yaml"
	if !m.stepWriteConfig(mihomoCfg, configPath, &steps) {
		return steps
	}

	// Step 6: Check /dev/net/tun (TUN mode only) - auto fallback to mixed-port
	if m.appCfg.Mode == "tun" {
		if !m.stepCheckTUNWithFallback(mihomoCfg, &steps) {
			// TUN not available, re-generate config in mixed-port mode
			m.appCfg.Mode = "mixed"
			mihomoCfg = core.BuildMihomoConfig(m.appCfg)
			_, ok = m.stepRenderYAML(mihomoCfg, &steps)
			if !ok {
				return steps
			}
			if !m.stepWriteConfig(mihomoCfg, configPath, &steps) {
				return steps
			}
		}
	}

	if err := app.SaveAppConfig(m.appCfg); err != nil {
		steps = append(steps, ExecStep{
			Label:   "保存 clashctl 配置",
			Success: false,
			Detail:  err.Error(),
		})
	} else {
		steps = append(steps, ExecStep{
			Label:   "保存 clashctl 配置",
			Success: true,
			Detail:  "已写入 ~/.config/clashctl/config.yaml",
		})
	}

	// Step 6.5: Pre-download geodata to avoid blocking mihomo startup
	m.stepEnsureGeoData(&steps)

	// Step 7: Setup systemd or start process
	if m.appCfg.EnableSystemd {
		m.stepSystemd(binary, &steps)
	} else {
		m.stepStartProcess(&steps)
	}

	// Step 8: Verify controller API (with retry, since mihomo may need time)
	m.stepCheckControllerWithRetry(&steps)
	if m.controllerAvailable {
		m.stepVerifyProxyInventory(&steps)
	}

	return steps
}

func (m WizardModel) stepCheckURL() ExecStep {
	probe, err := system.ProbeURL(m.appCfg.SubscriptionURL, 10*time.Second)
	if err != nil {
		return ExecStep{
			Label:   "检查订阅 URL 可达性",
			Success: false,
			Detail:  err.Error() + "\n提示: 服务器若无法直连订阅，可先在本地下载订阅，再用 'clashctl import --file sub.txt' 生成静态配置\n(仅警告，配置仍会生成)",
		}
	}
	detail := "URL 可正常访问"
	switch probe.ContentKind {
	case "base64-links", "raw-links":
		detail += "\n检测到原始节点订阅（非 YAML），必要时可用 'clashctl import --file ...' 转为静态配置"
	case "mihomo-yaml":
		detail += "\n检测到 Mihomo/Clash YAML 内容"
	case "html":
		detail += "\n警告: 返回内容像 HTML 页面，请确认这是真正的订阅链接"
	}
	return ExecStep{
		Label:   "检查订阅 URL 可达性",
		Success: true,
		Detail:  detail,
	}
}

func (m WizardModel) stepCheckBinary(steps *[]ExecStep) (string, bool) {
	binary, err := mihomo.FindBinary()
	if err != nil {
		// Not found, try auto-download
		*steps = append(*steps, ExecStep{
			Label:   "检测 Mihomo 可执行文件",
			Success: false,
			Detail:  "未找到，尝试自动下载...",
		})

		binary, err = mihomo.InstallMihomo()
		if err != nil {
			*steps = append(*steps, ExecStep{
				Label:   "自动下载 Mihomo",
				Success: false,
				Detail:  err.Error(),
			})
			return "", false
		}
		*steps = append(*steps, ExecStep{
			Label:   "自动下载 Mihomo",
			Success: true,
			Detail:  "已安装到 " + binary,
		})
		return binary, true
	}
	version, _ := mihomo.GetBinaryVersion()
	detail := "已找到: " + binary
	if version != "" {
		detail += " (" + version + ")"
	}
	*steps = append(*steps, ExecStep{
		Label:   "检测 Mihomo 可执行文件",
		Success: true,
		Detail:  detail,
	})
	return binary, true
}

func (m WizardModel) stepBuildConfig(steps *[]ExecStep) (*core.MihomoConfig, bool) {
	if errs := m.appCfg.Validate(); len(errs) > 0 {
		*steps = append(*steps, ExecStep{
			Label:   "验证配置参数",
			Success: false,
			Detail:  fmt.Sprintf("%v", errs),
		})
		return nil, false
	}

	cfg := core.BuildMihomoConfig(m.appCfg)
	*steps = append(*steps, ExecStep{
		Label:   "生成 Mihomo 配置",
		Success: true,
		Detail:  fmt.Sprintf("模式: %s, 订阅: %s", m.appCfg.Mode, m.appCfg.SubscriptionURL),
	})
	return cfg, true
}

func (m WizardModel) stepRenderYAML(cfg *core.MihomoConfig, steps *[]ExecStep) ([]byte, bool) {
	data, err := core.RenderYAML(cfg)
	if err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "渲染 YAML",
			Success: false,
			Detail:  err.Error(),
		})
		return nil, false
	}
	*steps = append(*steps, ExecStep{
		Label:   "渲染 YAML",
		Success: true,
		Detail:  fmt.Sprintf("%d bytes", len(data)),
	})
	return data, true
}

func (m WizardModel) stepWriteConfig(cfg *core.MihomoConfig, path string, steps *[]ExecStep) bool {
	if err := system.EnsureDir(m.appCfg.ConfigDir); err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "创建配置目录",
			Success: false,
			Detail:  err.Error(),
		})
		return false
	}
	*steps = append(*steps, ExecStep{
		Label:   "创建配置目录",
		Success: true,
		Detail:  m.appCfg.ConfigDir,
	})

	backupPath, err := config.SaveMihomoConfig(cfg, path)
	if err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "写入配置文件",
			Success: false,
			Detail:  err.Error(),
		})
		return false
	}
	detail := "已写入 " + path
	if backupPath != "" {
		detail += "\n旧配置已备份至: " + backupPath
	}
	*steps = append(*steps, ExecStep{
		Label:   "写入配置文件",
		Success: true,
		Detail:  detail,
	})
	return true
}

// stepCheckTUNWithFallback checks TUN availability.
// Returns true if TUN is usable, false if we need to fall back to mixed-port.
func (m WizardModel) stepCheckTUNWithFallback(mihomoCfg *core.MihomoConfig, steps *[]ExecStep) bool {
	if !mihomo.CanUseTUN() {
		reasons := []string{}
		if _, err := system.StatFile("/dev/net/tun"); err != nil {
			reasons = append(reasons, "/dev/net/tun 不存在")
		}
		if !system.CommandExists("iptables") {
			reasons = append(reasons, "iptables 未安装")
		}
		*steps = append(*steps, ExecStep{
			Label:   "检查 TUN 设备",
			Success: false,
			Detail:  fmt.Sprintf("TUN 不可用: %s\n自动降级到 mixed-port 模式...", fmt.Sprintf("%v", reasons)),
		})
		return false
	}
	*steps = append(*steps, ExecStep{
		Label:   "TUN 设备 & 权限",
		Success: true,
		Detail:  "/dev/net/tun 可用，iptables 已安装",
	})
	return true
}

// stepEnsureGeoData pre-downloads geodata files to avoid mihomo blocking on first startup.
func (m WizardModel) stepEnsureGeoData(steps *[]ExecStep) {
	configDir := m.appCfg.ConfigDir

	if mihomo.GeoDataReady(configDir) {
		*steps = append(*steps, ExecStep{
			Label:   "GeoSite/GeoIP 数据",
			Success: true,
			Detail:  "已存在，跳过下载",
		})
		return
	}

	downloaded, err := mihomo.EnsureGeoData(configDir)
	if err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "GeoSite/GeoIP 数据",
			Success: false,
			Detail:  err.Error() + "\nMihomo 启动时会自动重试下载",
		})
		return
	}

	if downloaded > 0 {
		*steps = append(*steps, ExecStep{
			Label:   "GeoSite/GeoIP 数据",
			Success: true,
			Detail:  fmt.Sprintf("已下载 %d 个数据文件到 %s", downloaded, configDir),
		})
	}
}

func (m WizardModel) stepSystemd(binary string, steps *[]ExecStep) {
	svcCfg := mihomo.ServiceConfig{
		Binary:      binary,
		ConfigDir:   m.appCfg.ConfigDir,
		ServiceName: core.DefaultServiceName,
	}

	if err := mihomo.SetupSystemd(svcCfg, m.appCfg.AutoStart, m.appCfg.AutoStart); err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "配置 systemd 服务",
			Success: false,
			Detail:  err.Error() + "\n回退到子进程启动...",
		})
		// Fallback to direct process
		m.stepStartProcess(steps)
		return
	}

	detail := "服务文件已写入"
	if m.appCfg.AutoStart {
		detail += "，已启用开机自启"
	} else {
		detail += "，未启用开机自启"
	}
	if m.appCfg.AutoStart {
		detail += "，已启动"
	}
	*steps = append(*steps, ExecStep{
		Label:   "配置 systemd 服务",
		Success: true,
		Detail:  detail,
	})
}

func (m WizardModel) stepStartProcess(steps *[]ExecStep) {
	if mihomo.HasSystemd() {
		if active, _ := mihomo.ServiceStatus(mihomo.DefaultServiceName); active {
			if err := mihomo.StopService(mihomo.DefaultServiceName); err != nil {
				*steps = append(*steps, ExecStep{
					Label:   "停止 systemd 服务",
					Success: false,
					Detail:  err.Error(),
				})
			} else {
				*steps = append(*steps, ExecStep{
					Label:   "停止 systemd 服务",
					Success: true,
					Detail:  "已停止已有的 clashctl-mihomo 服务",
				})
			}
		}
	}

	// Stop any managed Mihomo processes using the same config directory.
	stopped, err := mihomo.StopManagedProcess(m.appCfg.ConfigDir)
	if err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "清理旧进程",
			Success: false,
			Detail:  err.Error(),
		})
	} else if stopped {
		*steps = append(*steps, ExecStep{
			Label:   "清理旧进程",
			Success: true,
			Detail:  "已停止当前配置目录对应的 Mihomo 进程",
		})
	}

	proc := mihomo.NewProcess(m.appCfg.ConfigDir)
	if err := proc.Start(); err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "启动 Mihomo 进程",
			Success: false,
			Detail:  err.Error(),
		})
		return
	}

	*steps = append(*steps, ExecStep{
		Label:   "启动 Mihomo 进程",
		Success: true,
		Detail:  "Mihomo 已以子进程方式启动",
	})
}

// stepCheckControllerWithRetry waits for the controller API to become ready.
// Mihomo may need time to download subscription, geodata, etc.
func (m *WizardModel) stepCheckControllerWithRetry(steps *[]ExecStep) {
	client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)

	// First check: maybe already ready
	if err := client.CheckConnection(); err == nil {
		m.reportControllerReady(client, steps)
		return
	}

	// Not ready yet — retry up to 30 times (60 seconds total, 2s interval)
	// This covers: geodata loading, subscription fetch, health check
	*steps = append(*steps, ExecStep{
		Label:   "等待 Mihomo 就绪",
		Success: true,
		Detail:  "首次启动可能需要下载 GeoSite/GeoIP 数据和订阅，正在等待...",
	})

	err := mihomo.WaitForController(m.appCfg.ControllerAddr, 30, 2*time.Second)
	if err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "检查 Controller API",
			Success: false,
			Detail:  err.Error() + "\nMihomo 可能仍在加载，请用 'clashctl status' 检查",
		})
		m.controllerAvailable = false
		return
	}

	m.reportControllerReady(client, steps)
}

func (m *WizardModel) reportControllerReady(client *mihomo.Client, steps *[]ExecStep) {
	version, _ := client.Version()
	detail := "Controller API 可达"
	if version != "" {
		detail += " (Mihomo " + version + ")"
	}

	if group, err := client.GetProxyGroup("PROXY"); err == nil {
		detail += fmt.Sprintf("\n代理组 PROXY: %d 个节点已加载", len(group.All))
	}

	m.controllerAvailable = true
	*steps = append(*steps, ExecStep{
		Label:   "检查 Controller API",
		Success: true,
		Detail:  detail,
	})
}

func (m *WizardModel) stepVerifyProxyInventory(steps *[]ExecStep) {
	client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)
	inv, err := client.InspectProxyInventory("PROXY")
	if err != nil {
		m.controllerAvailable = false
		*steps = append(*steps, ExecStep{
			Label:   "验证代理节点加载",
			Success: false,
			Detail:  "Controller API 已启动，但无法读取 PROXY 组\n提示: 请检查配置文件中的 proxy-groups 是否包含 PROXY",
		})
		return
	}

	providerPath := filepath.Join(m.appCfg.ConfigDir, m.appCfg.ProviderPath)
	providerMissing := false
	if strings.TrimSpace(m.appCfg.SubscriptionURL) != "" {
		if info, err := system.StatFile(providerPath); err != nil || info.Size() == 0 {
			providerMissing = true
		}
	}

	loaded := inv.Loaded
	if loaded == 0 || inv.OnlyCompatible || providerMissing {
		m.controllerAvailable = false
		detail := "Controller API 已就绪，但订阅节点未成功加载"
		if providerMissing {
			detail += fmt.Sprintf("\nprovider 文件不存在或为空: %s", providerPath)
		}
		if loaded > 0 {
			detail += fmt.Sprintf("\n当前 PROXY 候选: %v", inv.Candidates)
		}
		detail += "\n常见原因: 服务器无法直连订阅 URL、provider 拉取失败、或订阅返回的是原始节点链接"
		detail += "\n建议: 先在本地下载订阅，再执行 'clashctl import --file sub.txt -o config.yaml' 生成静态配置"
		*steps = append(*steps, ExecStep{
			Label:   "验证代理节点加载",
			Success: false,
			Detail:  detail,
		})
		return
	}

	detail := fmt.Sprintf("PROXY 已加载 %d 个节点", loaded)
	if inv.Current != "" {
		detail += "\n当前节点: " + inv.Current
	}
	*steps = append(*steps, ExecStep{
		Label:   "验证代理节点加载",
		Success: true,
		Detail:  detail,
	})
}

func (m WizardModel) executeImport(filePath string) []ExecStep {
	var steps []ExecStep

	data, err := os.ReadFile(filePath)
	if err != nil {
		steps = append(steps, ExecStep{
			Label:   "读取本地订阅文件",
			Success: false,
			Detail:  err.Error(),
		})
		m.controllerAvailable = false
		return steps
	}
	steps = append(steps, ExecStep{
		Label:   "读取本地订阅文件",
		Success: true,
		Detail:  fmt.Sprintf("已读取 %s (%d bytes)", filePath, len(data)),
	})

	parsed, err := subscription.Parse(data)
	if err != nil {
		steps = append(steps, ExecStep{
			Label:   "解析本地订阅文件",
			Success: false,
			Detail:  err.Error(),
		})
		m.controllerAvailable = false
		return steps
	}
	steps = append(steps, ExecStep{
		Label:   "解析本地订阅文件",
		Success: true,
		Detail:  fmt.Sprintf("检测到 %s，解析出 %d 个节点", parsed.DetectedFormat, len(parsed.Names)),
	})

	mihomoCfg := core.BuildStaticMihomoConfig(m.appCfg, parsed.Proxies, parsed.Names)
	steps = append(steps, ExecStep{
		Label:   "生成静态配置",
		Success: true,
		Detail:  "已生成不依赖订阅 URL 的 Mihomo 配置",
	})

	configPath := filepath.Join(m.appCfg.ConfigDir, "config.yaml")
	if _, ok := m.stepRenderYAML(mihomoCfg, &steps); !ok {
		m.controllerAvailable = false
		return steps
	}
	if !m.stepWriteConfig(mihomoCfg, configPath, &steps) {
		m.controllerAvailable = false
		return steps
	}

	if err := app.SaveAppConfig(m.appCfg); err != nil {
		steps = append(steps, ExecStep{
			Label:   "保存 clashctl 配置",
			Success: false,
			Detail:  err.Error(),
		})
	} else {
		steps = append(steps, ExecStep{
			Label:   "保存 clashctl 配置",
			Success: true,
			Detail:  "已写入 ~/.config/clashctl/config.yaml\n提示: 当前使用静态导入配置，后续不会依赖服务器直连订阅 URL",
		})
	}

	m.stepEnsureGeoData(&steps)
	if m.appCfg.EnableSystemd {
		binary, binaryOK := m.stepCheckBinary(&steps)
		if !binaryOK {
			m.controllerAvailable = false
			return steps
		}
		m.stepSystemd(binary, &steps)
	} else {
		m.stepStartProcess(&steps)
	}
	m.stepCheckControllerWithRetry(&steps)
	if m.controllerAvailable {
		m.stepVerifyProxyInventory(&steps)
	}

	return steps
}
