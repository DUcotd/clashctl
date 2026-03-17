// Package ui provides system integration execution steps.
package ui

import (
	"fmt"
	"time"

	"clashctl/internal/config"
	"clashctl/internal/core"
	"clashctl/internal/mihomo"
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
	_ = binary

	// Step 3: Build config
	mihomoCfg, ok := m.stepBuildConfig(&steps)
	if !ok {
		return steps
	}

	// Step 4: Render YAML
	yamlData, ok := m.stepRenderYAML(mihomoCfg, &steps)
	if !ok {
		return steps
	}

	// Step 5: Write config file
	configPath := m.appCfg.ConfigDir + "/config.yaml"
	if !m.stepWriteConfig(mihomoCfg, configPath, yamlData, &steps) {
		return steps
	}

	// Step 6: Check /dev/net/tun (TUN mode only)
	if m.appCfg.Mode == "tun" {
		m.stepCheckTUN(&steps)
	}

	// Step 7: Setup systemd or start process
	if m.appCfg.EnableSystemd {
		m.stepSystemd(binary, &steps)
	} else {
		m.stepStartProcess(&steps)
	}

	// Step 8: Verify controller API (non-blocking check)
	m.stepCheckController(&steps)

	return steps
}

func (m WizardModel) stepCheckURL() ExecStep {
	if err := system.CheckURLReachable(m.appCfg.SubscriptionURL, 10*time.Second); err != nil {
		return ExecStep{
			Label:   "检查订阅 URL 可达性",
			Success: false,
			Detail:  err.Error() + "\n(仅警告，配置仍会生成)",
		}
	}
	return ExecStep{
		Label:   "检查订阅 URL 可达性",
		Success: true,
		Detail:  "URL 可正常访问",
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

func (m WizardModel) stepWriteConfig(cfg *core.MihomoConfig, path string, data []byte, steps *[]ExecStep) bool {
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

func (m WizardModel) stepCheckTUN(steps *[]ExecStep) {
	if _, err := system.StatFile("/dev/net/tun"); err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "检查 TUN 设备",
			Success: false,
			Detail:  "/dev/net/tun 不存在，请运行: sudo modprobe tun",
		})
		return
	}
	if err := mihomo.CheckTUNPermission(); err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "检查 TUN 权限",
			Success: false,
			Detail:  err.Error(),
		})
		return
	}
	*steps = append(*steps, ExecStep{
		Label:   "TUN 设备 & 权限",
		Success: true,
		Detail:  "/dev/net/tun 可用，权限正常",
	})
}

func (m WizardModel) stepSystemd(binary string, steps *[]ExecStep) {
	svcCfg := mihomo.ServiceConfig{
		Binary:      binary,
		ConfigDir:   m.appCfg.ConfigDir,
		ServiceName: core.DefaultServiceName,
	}

	if err := mihomo.SetupSystemd(svcCfg, m.appCfg.AutoStart); err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "配置 systemd 服务",
			Success: false,
			Detail:  err.Error() + "\n回退到子进程启动...",
		})
		// Fallback to direct process
		m.stepStartProcess(steps)
		return
	}

	detail := "服务已创建并启用"
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

func (m *WizardModel) stepCheckController(steps *[]ExecStep) {
	client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)
	if err := client.CheckConnection(); err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "检查 Controller API",
			Success: false,
			Detail:  err.Error() + "\nMihomo 可能还在启动中，请稍后用 'clashctl status' 检查",
		})
		m.controllerAvailable = false
		return
	}

	version, _ := client.Version()
	detail := "Controller API 可达"
	if version != "" {
		detail += " (Mihomo " + version + ")"
	}

	if group, err := client.GetProxyGroup("PROXY"); err == nil {
		detail += fmt.Sprintf("\n代理组 PROXY: %d 个节点可用", len(group.All))
	}

	m.controllerAvailable = true
	*steps = append(*steps, ExecStep{
		Label:   "检查 Controller API",
		Success: true,
		Detail:  detail,
	})
}
