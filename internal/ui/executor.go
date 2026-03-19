// Package ui provides system integration execution steps.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clashctl/internal/app"
	"clashctl/internal/mihomo"
	"clashctl/internal/subscription"
	"clashctl/internal/system"
)

// executeFull performs the full configuration and startup pipeline.
func (m WizardModel) executeFull() []ExecStep {
	var steps []ExecStep

	runtime := mihomo.NewRuntimeManager()
	resolver := subscription.NewResolver()

	m.stepResolveRuntimeConfig(runtime, &steps)
	binary, ok := m.stepEnsureBinary(runtime, &steps)
	if !ok {
		return steps
	}

	plan, ok := m.stepResolveRemotePlan(resolver, &steps)
	if !ok {
		return steps
	}

	configPath := filepath.Join(m.appCfg.ConfigDir, "config.yaml")
	if !m.stepWritePlan(plan, configPath, &steps) {
		return steps
	}

	if !m.stepSaveAppConfig(&steps, "") {
		return steps
	}

	startResult, err := runtime.StartWithBinary(m.appCfg, binary, mihomo.StartOptions{
		VerifyInventory: true,
		WaitRetries:     30,
		WaitInterval:    2 * time.Second,
	})
	if err != nil {
		m.appendStartResult(startResult, &steps)
		steps = append(steps, ExecStep{
			Label:   "启动 Mihomo",
			Success: false,
			Detail:  err.Error(),
		})
		m.controllerAvailable = false
		return steps
	}
	m.appendStartResult(startResult, &steps)
	return steps
}

func (m *WizardModel) stepResolveRuntimeConfig(runtime *mihomo.RuntimeManager, steps *[]ExecStep) {
	resolved, warnings := runtime.ResolveConfig(m.appCfg)
	m.appCfg = resolved
	if len(warnings) == 0 {
		return
	}
	*steps = append(*steps, ExecStep{
		Label:   "调整运行模式",
		Success: true,
		Detail:  strings.Join(warnings, "\n"),
	})
}

func (m WizardModel) stepEnsureBinary(runtime *mihomo.RuntimeManager, steps *[]ExecStep) (*mihomo.InstallResult, bool) {
	result, err := runtime.EnsureBinary()
	if err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "检测 Mihomo 可执行文件",
			Success: false,
			Detail:  err.Error(),
		})
		return nil, false
	}

	label := "检测 Mihomo 可执行文件"
	detail := "已找到: " + result.Path
	if result.Installed {
		label = "自动下载 Mihomo"
		detail = "已安装到 " + result.Path
	}
	if result.Version != "" {
		detail += " (" + result.Version + ")"
	}
	if result.ReleaseTag != "" && result.Installed {
		detail += "\n发布版本: " + result.ReleaseTag
	}

	*steps = append(*steps, ExecStep{
		Label:   label,
		Success: true,
		Detail:  detail,
	})
	return result, true
}

func (m WizardModel) stepResolveRemotePlan(resolver *subscription.Resolver, steps *[]ExecStep) (*subscription.ResolvedConfigPlan, bool) {
	if errs := m.appCfg.Validate(); len(errs) > 0 {
		*steps = append(*steps, ExecStep{
			Label:   "验证配置参数",
			Success: false,
			Detail:  fmt.Sprintf("%v", errs),
		})
		return nil, false
	}

	plan, err := resolver.ResolveRemoteURL(m.appCfg, m.appCfg.SubscriptionURL, 15*time.Second)
	if err != nil {
		detail := err.Error() + "\n提示: 服务器若无法直连订阅，可先在本地下载订阅，再用 'clashctl import --file sub.txt --apply --start'"
		if mihomo.IsMihomoRunningAt(m.appCfg.ControllerAddr) {
			detail += "\n检测到当前已有 Mihomo 在运行；旧代理链路或系统代理可能干扰了本次检查"
		}
		*steps = append(*steps, ExecStep{Label: "获取订阅内容", Success: false, Detail: detail})
		return nil, false
	}

	detail := fmt.Sprintf("已通过订阅脚本下载内容 (%s)", plan.ContentKind)
	if plan.UsedProxyEnv {
		detail += "\n已忽略当前 shell 中的代理环境变量，直接下载订阅地址"
	}
	if plan.FetchDetail != "" {
		detail += "\n" + plan.FetchDetail
	}
	*steps = append(*steps, ExecStep{
		Label:   "获取订阅内容",
		Success: true,
		Detail:  detail,
	})

	switch plan.Kind {
	case subscription.PlanKindStatic:
		*steps = append(*steps, ExecStep{
			Label:   "解析订阅内容",
			Success: true,
			Detail:  fmt.Sprintf("检测到 %s，解析出 %d 个节点", plan.DetectedFormat, plan.ProxyCount),
		})
	case subscription.PlanKindYAML:
		*steps = append(*steps, ExecStep{
			Label:   "处理订阅 YAML",
			Success: true,
			Detail:  plan.Summary,
		})
	case subscription.PlanKindProvider:
		*steps = append(*steps, ExecStep{
			Label:   "生成 Mihomo 配置",
			Success: true,
			Detail:  plan.Summary,
		})
	}

	return plan, true
}

func (m WizardModel) stepWritePlan(plan *subscription.ResolvedConfigPlan, path string, steps *[]ExecStep) bool {
	if plan == nil {
		*steps = append(*steps, ExecStep{Label: "写入配置文件", Success: false, Detail: "配置计划为空"})
		return false
	}
	if err := system.EnsureDir(m.appCfg.ConfigDir); err != nil {
		*steps = append(*steps, ExecStep{Label: "创建配置目录", Success: false, Detail: err.Error()})
		return false
	}
	*steps = append(*steps, ExecStep{Label: "创建配置目录", Success: true, Detail: m.appCfg.ConfigDir})

	yamlData, err := plan.RenderYAML()
	if err != nil {
		*steps = append(*steps, ExecStep{Label: "渲染 YAML", Success: false, Detail: err.Error()})
		return false
	}
	*steps = append(*steps, ExecStep{
		Label:   "渲染 YAML",
		Success: true,
		Detail:  fmt.Sprintf("%d bytes", len(yamlData)),
	})

	backupPath, err := plan.Save(path)
	if err != nil {
		*steps = append(*steps, ExecStep{Label: "写入配置文件", Success: false, Detail: err.Error()})
		return false
	}

	detail := "已写入 " + path
	if plan.Kind != subscription.PlanKindProvider {
		detail += "\n静态配置优先，后续无需服务器再次拉取订阅"
	}
	if backupPath != "" {
		detail += "\n旧配置已备份至: " + backupPath
	}
	*steps = append(*steps, ExecStep{Label: "写入配置文件", Success: true, Detail: detail})
	return true
}

func (m WizardModel) stepSaveAppConfig(steps *[]ExecStep, extraDetail string) bool {
	if err := app.SaveAppConfig(m.appCfg); err != nil {
		*steps = append(*steps, ExecStep{
			Label:   "保存 clashctl 配置",
			Success: false,
			Detail:  err.Error(),
		})
		return false
	}
	detail := "已写入 ~/.config/clashctl/config.yaml"
	if extraDetail != "" {
		detail += "\n" + extraDetail
	}
	*steps = append(*steps, ExecStep{
		Label:   "保存 clashctl 配置",
		Success: true,
		Detail:  detail,
	})
	return true
}

func (m *WizardModel) appendStartResult(result *mihomo.StartResult, steps *[]ExecStep) {
	if result == nil {
		return
	}

	if result.GeoData != nil || result.GeoDataError != "" {
		if result.GeoDataError != "" {
			*steps = append(*steps, ExecStep{
				Label:   "GeoSite/GeoIP 数据",
				Success: false,
				Detail:  result.GeoDataError + "\nMihomo 启动时会自动重试下载",
			})
		} else if result.GeoData.Downloaded > 0 {
			*steps = append(*steps, ExecStep{
				Label:   "GeoSite/GeoIP 数据",
				Success: true,
				Detail:  fmt.Sprintf("已下载 %d 个数据文件到 %s", result.GeoData.Downloaded, m.appCfg.ConfigDir),
			})
		}
	}

	startDetail := "Mihomo 已启动"
	switch result.StartedBy {
	case "systemd":
		startDetail = "已通过 systemd 启动"
	case "process":
		startDetail = "Mihomo 已以子进程方式启动"
	}
	if result.ServiceStopped {
		startDetail += "\n已停止旧的 systemd 服务"
	}
	if result.ProcessStopped {
		startDetail += "\n已清理旧进程"
	}
	if len(result.Warnings) > 0 {
		startDetail += "\n" + strings.Join(result.Warnings, "\n")
	}
	*steps = append(*steps, ExecStep{
		Label:   "启动 Mihomo",
		Success: true,
		Detail:  startDetail,
	})

	if result.ControllerReady {
		m.controllerAvailable = true
		detail := "Controller API 可达"
		if result.ControllerVersion != "" {
			detail += " (Mihomo " + result.ControllerVersion + ")"
		}
		*steps = append(*steps, ExecStep{
			Label:   "检查 Controller API",
			Success: true,
			Detail:  detail,
		})
	} else {
		m.controllerAvailable = false
	}

	if result.InventoryError != "" {
		m.controllerAvailable = false
		*steps = append(*steps, ExecStep{
			Label:   "验证代理节点加载",
			Success: false,
			Detail:  "Controller API 已启动，但无法读取 PROXY 组\n提示: 请检查配置文件中的 proxy-groups 是否包含 PROXY",
		})
		return
	}

	if result.Inventory == nil {
		return
	}

	if result.Inventory.Loaded == 0 || result.Inventory.OnlyCompatible {
		m.controllerAvailable = false
		detail := "Controller API 已就绪，但订阅节点未成功加载"
		if result.Inventory.Loaded > 0 {
			detail += fmt.Sprintf("\n当前 PROXY 候选: %v", result.Inventory.Candidates)
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

	detail := fmt.Sprintf("PROXY 已加载 %d 个节点", result.Inventory.Loaded)
	if result.Inventory.Current != "" {
		detail += "\n当前节点: " + result.Inventory.Current
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

	runtime := mihomo.NewRuntimeManager()
	resolver := subscription.NewResolver()

	m.stepResolveRuntimeConfig(runtime, &steps)

	plan, err := resolver.ResolveContent(m.appCfg, data)
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
		Detail:  plan.Summary,
	})

	configPath := filepath.Join(m.appCfg.ConfigDir, "config.yaml")
	if !m.stepWritePlan(plan, configPath, &steps) {
		m.controllerAvailable = false
		return steps
	}

	if !m.stepSaveAppConfig(&steps, "提示: 当前使用静态导入配置，后续不会依赖服务器直连订阅 URL") {
		m.controllerAvailable = false
		return steps
	}

	binary, ok := m.stepEnsureBinary(runtime, &steps)
	if !ok {
		m.controllerAvailable = false
		return steps
	}

	startResult, err := runtime.StartWithBinary(m.appCfg, binary, mihomo.StartOptions{
		VerifyInventory: true,
		WaitRetries:     15,
		WaitInterval:    2 * time.Second,
	})
	if err != nil {
		m.appendStartResult(startResult, &steps)
		steps = append(steps, ExecStep{
			Label:   "启动 Mihomo",
			Success: false,
			Detail:  err.Error(),
		})
		m.controllerAvailable = false
		return steps
	}

	m.appendStartResult(startResult, &steps)
	return steps
}
