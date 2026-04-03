package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	setupflow "clashctl/internal/setup"
	"clashctl/internal/subscription"
	"clashctl/internal/system"
)

// SetupService runs the setup pipeline behind the TUI.
type SetupService interface {
	StartRemote(appCfg *core.AppConfig) <-chan setupProgressMsg
	StartImport(appCfg *core.AppConfig, filePath string) <-chan setupProgressMsg
	StartInline(appCfg *core.AppConfig, content string) <-chan setupProgressMsg
}

type defaultSetupService struct {
	newRuntimeManager   func() *mihomo.RuntimeManager
	newResolver         func() *subscription.Resolver
	readFile            func(string) ([]byte, error)
	persistShellProxy   func(int) (*system.ShellProxyResult, error)
	removeShellProxyEnv func() (*system.ShellProxyResult, error)
}

type setupRunContext struct {
	appCfg              *core.AppConfig
	steps               []ExecStep
	controllerAvailable bool
	progress            *setupProgressWriter
}

type setupProgressWriter struct {
	stream chan<- setupProgressMsg
}

func newDefaultSetupService() SetupService {
	return &defaultSetupService{
		newRuntimeManager:   mihomo.NewRuntimeManager,
		newResolver:         subscription.NewResolver,
		readFile:            os.ReadFile,
		persistShellProxy:   system.PersistShellProxyEnv,
		removeShellProxyEnv: system.RemoveShellProxyEnv,
	}
}

func cloneAppConfig(cfg *core.AppConfig) *core.AppConfig {
	if cfg == nil {
		return core.DefaultAppConfig()
	}
	cloned := *cfg
	return &cloned
}

func (w *setupProgressWriter) current(step string) {
	if w == nil || strings.TrimSpace(step) == "" {
		return
	}
	w.stream <- setupProgressMsg{currentStep: step}
}

func (w *setupProgressWriter) complete(step ExecStep) {
	if w == nil {
		return
	}
	stepCopy := step
	w.stream <- setupProgressMsg{step: &stepCopy}
}

func (w *setupProgressWriter) finish(run *setupRunContext) {
	if w == nil || run == nil {
		return
	}
	canImport, importHint := detectImportFallback(run.steps)
	w.stream <- setupProgressMsg{
		done:            true,
		controllerReady: run.controllerAvailable,
		canImport:       canImport,
		importHint:      importHint,
	}
}

func (r *setupRunContext) current(step string) {
	if r.progress != nil {
		r.progress.current(step)
	}
}

func (r *setupRunContext) addStep(step ExecStep) {
	r.steps = append(r.steps, step)
	if r.progress != nil {
		r.progress.complete(step)
	}
}

func (s *defaultSetupService) stream(appCfg *core.AppConfig, runFn func(*setupRunContext)) <-chan setupProgressMsg {
	stream := make(chan setupProgressMsg)
	go func() {
		defer close(stream)
		run := &setupRunContext{
			appCfg:   cloneAppConfig(appCfg),
			progress: &setupProgressWriter{stream: stream},
		}
		runFn(run)
		run.progress.finish(run)
	}()
	return stream
}

func (s *defaultSetupService) StartRemote(appCfg *core.AppConfig) <-chan setupProgressMsg {
	return s.stream(appCfg, func(run *setupRunContext) {
		runtime := s.newRuntimeManager()
		resolver := s.newResolver()

		run.current("检查运行模式")
		run.stepResolveRuntimeConfig(runtime)

		run.current("检查 Mihomo 可执行文件")
		binary, ok := run.stepEnsureBinary(runtime)
		if !ok {
			return
		}

		run.current("获取订阅内容")
		plan, ok := run.stepResolveRemotePlan(resolver)
		if !ok {
			return
		}

		run.current("写入配置文件")
		if !s.stepApplyPlan(run, plan, "") {
			return
		}

		run.current("启动 Mihomo")
		startResult, err := runtime.StartWithBinary(run.appCfg, binary, mihomo.StartOptions{
			VerifyInventory: true,
			WaitRetries:     30,
			WaitInterval:    2 * time.Second,
		})
		if err != nil {
			run.appendStartResult(startResult)
			run.addStep(ExecStep{
				Label:   "启动 Mihomo",
				Success: false,
				Detail:  err.Error(),
			})
			run.controllerAvailable = false
			return
		}

		run.appendStartResult(startResult)
		run.current("同步 shell 代理环境")
		s.stepManageShellProxyEnv(run)
	})
}

func (s *defaultSetupService) StartImport(appCfg *core.AppConfig, filePath string) <-chan setupProgressMsg {
	return s.stream(appCfg, func(run *setupRunContext) {
		run.current("读取本地订阅文件")
		data, err := s.readFile(filePath)
		if err != nil {
			run.addStep(ExecStep{
				Label:   "读取本地订阅文件",
				Success: false,
				Detail:  err.Error(),
			})
			return
		}

		run.addStep(ExecStep{
			Label:   "读取本地订阅文件",
			Success: true,
			Detail:  fmt.Sprintf("已读取 %s (%d bytes)", filePath, len(data)),
		})

		s.executeResolvedContent(
			run,
			data,
			"解析本地订阅文件",
			"提示: 当前使用静态导入配置，后续不会依赖服务器直连订阅 URL",
		)
	})
}

func (s *defaultSetupService) StartInline(appCfg *core.AppConfig, content string) <-chan setupProgressMsg {
	return s.stream(appCfg, func(run *setupRunContext) {
		data := []byte(content)
		run.current("读取粘贴订阅内容")
		run.addStep(ExecStep{
			Label:   "读取粘贴订阅内容",
			Success: true,
			Detail:  fmt.Sprintf("已接收 %d bytes 的订阅内容", len(data)),
		})

		s.executeResolvedContent(
			run,
			data,
			"解析粘贴订阅内容",
			"提示: 当前使用直接粘贴的静态内容，原始订阅内容不会写入 clashctl 配置",
		)
	})
}

func (r *setupRunContext) stepResolveRuntimeConfig(runtime *mihomo.RuntimeManager) {
	resolved, warnings := runtime.ResolveConfig(r.appCfg)
	r.appCfg = resolved
	if len(warnings) == 0 {
		return
	}
	r.addStep(ExecStep{
		Label:   "调整运行模式",
		Success: true,
		Detail:  strings.Join(warnings, "\n"),
	})
}

func (r *setupRunContext) stepEnsureBinary(runtime *mihomo.RuntimeManager) (*mihomo.InstallResult, bool) {
	result, err := runtime.EnsureBinary()
	if err != nil {
		r.addStep(ExecStep{
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

	r.addStep(ExecStep{
		Label:   label,
		Success: true,
		Detail:  detail,
	})
	return result, true
}

func (r *setupRunContext) stepResolveRemotePlan(resolver *subscription.Resolver) (*subscription.ResolvedConfigPlan, bool) {
	if errs := r.appCfg.Validate(); len(errs) > 0 {
		r.addStep(ExecStep{
			Label:   "验证配置参数",
			Success: false,
			Detail:  fmt.Sprintf("%v", errs),
		})
		return nil, false
	}

	plan, err := resolver.ResolveRemoteURL(r.appCfg, r.appCfg.SubscriptionURL, 15*time.Second)
	if err != nil {
		detail := err.Error() + "\n提示: 服务器若无法直连订阅，可先在本地下载订阅，再用 'clashctl config import --file sub.txt --apply --start'"
		if mihomo.IsMihomoRunningAt(r.appCfg.ControllerAddr, r.appCfg.ControllerSecret) {
			detail += "\n检测到当前已有 Mihomo 在运行；旧代理链路或系统代理可能干扰了本次检查"
		}
		r.addStep(ExecStep{Label: "获取订阅内容", Success: false, Detail: detail})
		return nil, false
	}

	detail := fmt.Sprintf("已通过订阅脚本下载内容 (%s)", plan.ContentKind)
	if plan.UsedProxyEnv {
		detail += "\n已忽略当前 shell 中的代理环境变量，直接下载订阅地址"
	}
	if plan.FetchDetail != "" {
		detail += "\n" + plan.FetchDetail
	}
	r.addStep(ExecStep{
		Label:   "获取订阅内容",
		Success: true,
		Detail:  detail,
	})

	switch plan.Kind {
	case subscription.PlanKindStatic:
		r.addStep(ExecStep{
			Label:   "解析订阅内容",
			Success: true,
			Detail:  fmt.Sprintf("检测到 %s，解析出 %d 个节点", plan.DetectedFormat, plan.ProxyCount),
		})
	case subscription.PlanKindYAML:
		detail := plan.Summary
		if plan.Sanitized {
			detail += "\n该订阅已按安全策略裁剪"
		}
		if len(plan.RemovedFields) > 0 {
			detail += "\n移除字段: " + strings.Join(plan.RemovedFields, ", ")
		}
		if len(plan.Warnings) > 0 {
			detail += "\n" + strings.Join(plan.Warnings, "\n")
		}
		r.addStep(ExecStep{
			Label:   "处理订阅 YAML",
			Success: true,
			Detail:  detail,
		})
	}

	return plan, true
}

func (s *defaultSetupService) stepApplyPlan(run *setupRunContext, plan *subscription.ResolvedConfigPlan, saveDetail string) bool {
	if plan == nil {
		run.addStep(ExecStep{Label: "写入配置文件", Success: false, Detail: "配置计划为空"})
		return false
	}
	result, err := setupflow.ApplyResolvedPlan(run.appCfg, plan, setupflow.ApplyPlanOptions{SaveAppConfig: true})
	if result != nil && result.ConfigDir != "" {
		run.addStep(ExecStep{Label: "创建配置目录", Success: true, Detail: result.ConfigDir})
	}
	if err != nil {
		if stageErr := new(setupflow.ApplyPlanError); strings.TrimSpace(err.Error()) != "" && errors.As(err, &stageErr) {
			switch stageErr.Stage {
			case setupflow.StageCreateConfigDir:
				run.addStep(ExecStep{Label: "创建配置目录", Success: false, Detail: stageErr.Error()})
				return false
			case setupflow.StageRenderYAML:
				run.addStep(ExecStep{Label: "渲染 YAML", Success: false, Detail: stageErr.Error()})
				return false
			case setupflow.StageWriteConfig:
				if result != nil && result.YAMLSize > 0 {
					run.addStep(ExecStep{Label: "渲染 YAML", Success: true, Detail: fmt.Sprintf("%d bytes", result.YAMLSize)})
					appendValidationStep(run, result.ValidationError)
				}
				run.addStep(ExecStep{Label: "写入配置文件", Success: false, Detail: stageErr.Error()})
				return false
			case setupflow.StageSaveAppConfig:
				if result != nil && result.YAMLSize > 0 {
					run.addStep(ExecStep{Label: "渲染 YAML", Success: true, Detail: fmt.Sprintf("%d bytes", result.YAMLSize)})
					appendValidationStep(run, result.ValidationError)
					appendWritePlanStep(run, plan, result)
				}
				run.addStep(ExecStep{Label: "保存 clashctl 配置", Success: false, Detail: stageErr.Error()})
				return false
			}
		}
		run.addStep(ExecStep{Label: "写入配置文件", Success: false, Detail: err.Error()})
		return false
	}
	if result == nil {
		run.addStep(ExecStep{Label: "写入配置文件", Success: false, Detail: "配置写入结果为空"})
		return false
	}
	run.addStep(ExecStep{Label: "渲染 YAML", Success: true, Detail: fmt.Sprintf("%d bytes", result.YAMLSize)})
	appendValidationStep(run, result.ValidationError)
	appendWritePlanStep(run, plan, result)
	detail := "已写入 ~/.config/clashctl/config.yaml"
	if saveDetail != "" {
		detail += "\n" + saveDetail
	}
	run.addStep(ExecStep{Label: "保存 clashctl 配置", Success: true, Detail: detail})
	return true
}

func appendValidationStep(run *setupRunContext, validationErr error) {
	if validationErr != nil {
		run.addStep(ExecStep{
			Label:   "校验配置",
			Success: false,
			Detail:  validationErr.Error() + "\n配置可能有语法错误，但仍会尝试写入",
		})
		return
	}
	run.addStep(ExecStep{Label: "校验配置", Success: true, Detail: "配置语法检查通过"})
}

func appendWritePlanStep(run *setupRunContext, plan *subscription.ResolvedConfigPlan, result *setupflow.ApplyPlanResult) {
	detail := "已写入 " + result.OutputPath
	detail += "\n静态配置优先，后续无需服务器再次拉取订阅"
	if result.BackupPath != "" {
		detail += "\n旧配置已备份至: " + result.BackupPath
	}
	if len(plan.Warnings) > 0 {
		detail += "\n" + strings.Join(plan.Warnings, "\n")
	}
	run.addStep(ExecStep{Label: "写入配置文件", Success: true, Detail: detail})
}

func (s *defaultSetupService) stepManageShellProxyEnv(run *setupRunContext) {
	if run.appCfg.Mode == "mixed" {
		result, err := s.persistShellProxy(run.appCfg.MixedPort)
		if err != nil {
			run.addStep(ExecStep{
				Label:   "持久化 shell 代理环境",
				Success: false,
				Detail:  err.Error(),
			})
			return
		}
		run.addStep(ExecStep{
			Label:   "持久化 shell 代理环境",
			Success: true,
			Detail: fmt.Sprintf(
				"已写入 %s\n代理脚本: %s\n新开终端会自动生效；当前终端请执行: source %s",
				result.ProfilePath,
				result.ScriptPath,
				result.ProfilePath,
			),
		})
		return
	}

	result, err := s.removeShellProxyEnv()
	if err != nil {
		run.addStep(ExecStep{
			Label:   "清理 shell 代理环境",
			Success: false,
			Detail:  err.Error(),
		})
		return
	}
	run.addStep(ExecStep{
		Label:   "清理 shell 代理环境",
		Success: true,
		Detail:  fmt.Sprintf("已移除 clashctl 管理的 shell 代理设置: %s", result.ProfilePath),
	})
}

func (r *setupRunContext) appendStartResult(result *mihomo.StartResult) {
	if result == nil {
		return
	}

	if result.GeoData != nil || result.GeoDataError != "" {
		if result.GeoDataError != "" {
			r.addStep(ExecStep{
				Label:   "GeoSite/GeoIP 数据",
				Success: false,
				Detail:  result.GeoDataError + "\nMihomo 启动时会自动重试下载",
			})
		} else if result.GeoData.Downloaded > 0 {
			r.addStep(ExecStep{
				Label:   "GeoSite/GeoIP 数据",
				Success: true,
				Detail:  fmt.Sprintf("已下载 %d 个数据文件到 %s", result.GeoData.Downloaded, r.appCfg.ConfigDir),
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
	r.addStep(ExecStep{
		Label:   "启动 Mihomo",
		Success: true,
		Detail:  startDetail,
	})

	if result.ControllerReady {
		r.controllerAvailable = true
		detail := "Controller API 可达"
		if result.ControllerVersion != "" {
			detail += " (Mihomo " + result.ControllerVersion + ")"
		}
		r.addStep(ExecStep{
			Label:   "检查 Controller API",
			Success: true,
			Detail:  detail,
		})
	} else {
		r.controllerAvailable = false
	}

	if result.InventoryError != "" {
		r.controllerAvailable = false
		r.addStep(ExecStep{
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
		r.controllerAvailable = false
		detail := "Controller API 已就绪，但订阅节点未成功加载"
		if result.Inventory.Loaded > 0 {
			detail += fmt.Sprintf("\n当前 PROXY 候选: %v", result.Inventory.Candidates)
		}
		detail += "\n常见原因: 服务器无法直连订阅 URL、provider 拉取失败、或订阅返回的是原始节点链接"
		detail += "\n建议: 先在本地下载订阅，再执行 'clashctl config import --file sub.txt -o config.yaml' 生成静态配置"
		r.addStep(ExecStep{
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
	r.addStep(ExecStep{
		Label:   "验证代理节点加载",
		Success: true,
		Detail:  detail,
	})
}

func (s *defaultSetupService) executeResolvedContent(run *setupRunContext, data []byte, parseLabel, saveDetail string) {
	runtime := s.newRuntimeManager()
	resolver := s.newResolver()

	run.current("检查运行模式")
	run.stepResolveRuntimeConfig(runtime)

	run.current(parseLabel)
	plan, err := resolver.ResolveContent(run.appCfg, data)
	if err != nil {
		run.addStep(ExecStep{
			Label:   parseLabel,
			Success: false,
			Detail:  err.Error(),
		})
		return
	}
	run.addStep(ExecStep{
		Label:   parseLabel,
		Success: true,
		Detail:  plan.Summary,
	})

	run.current("写入配置文件")
	if !s.stepApplyPlan(run, plan, saveDetail) {
		return
	}

	run.current("检查 Mihomo 可执行文件")
	binary, ok := run.stepEnsureBinary(runtime)
	if !ok {
		return
	}

	run.current("启动 Mihomo")
	startResult, err := runtime.StartWithBinary(run.appCfg, binary, mihomo.StartOptions{
		VerifyInventory: true,
		WaitRetries:     15,
		WaitInterval:    2 * time.Second,
	})
	if err != nil {
		run.appendStartResult(startResult)
		run.addStep(ExecStep{
			Label:   "启动 Mihomo",
			Success: false,
			Detail:  err.Error(),
		})
		run.controllerAvailable = false
		return
	}

	run.appendStartResult(startResult)
	run.current("同步 shell 代理环境")
	s.stepManageShellProxyEnv(run)
}

func detectImportFallback(steps []ExecStep) (bool, string) {
	for _, step := range steps {
		if step.Label == "验证代理节点加载" && !step.Success {
			return true, step.Detail
		}
	}
	return false, ""
}
