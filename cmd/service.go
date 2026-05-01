package cmd

import (
	"fmt"
	"os"
	"time"

	"clashctl/internal/core"
	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
	"clashctl/internal/system"
)

var (
	loadAppConfigFn   = loadAppConfig
	hasSystemdFn      = mihomo.HasSystemd
	newRuntimeManager = mihomo.NewRuntimeManager
	startRuntimeFn    = func(runtime *mihomo.RuntimeManager, cfg *core.AppConfig, opts mihomo.StartOptions) (*mihomo.StartResult, error) {
		return runtime.Start(cfg, opts)
	}
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "管理 Mihomo 服务",
	Long:  `统一管理 Mihomo 的安装、启动、停止、重启和状态查看。`,
}

var installJSON bool

type installRunReport struct {
	Installed     bool               `json:"installed"`
	AlreadyExists bool               `json:"already_exists"`
	RequiresRoot  bool               `json:"requires_root,omitempty"`
	Binary        *installJSONReport `json:"binary,omitempty"`
	Error         string             `json:"error,omitempty"`
}

func (r *installRunReport) SetError(msg string) { r.Error = msg }

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装 Mihomo 内核",
	RunE:  runInstall,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Mihomo 服务",
	RunE:  withAppConfig(runStart),
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Mihomo 服务",
	RunE:  withAppConfig(runStop),
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "重启 Mihomo 服务",
	RunE:  withAppConfig(runRestart),
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看 Mihomo 运行状态",
	RunE:  withAppConfig(runStatus),
}

func init() {
	serviceStatusCmd.Flags().BoolVar(&statusJSON, "json", false, "以 JSON 输出状态信息")
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	rootCmd.AddCommand(serviceCmd)
}

func defaultStartOptions() mihomo.StartOptions {
	return mihomo.StartOptions{
		VerifyInventory: true,
		WaitRetries:     15,
		WaitInterval:    2 * time.Second,
	}
}

func bindInstallFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&installJSON, "json", false, "以 JSON 输出安装结果")
}

func runInstall(cmd *cobra.Command, args []string) error {
	report := &installRunReport{}
	if err := system.RequireRoot(); err != nil {
		report.RequiresRoot = true
		return finishReport(report, err, installJSON)
	}

	if binary, err := mihomo.FindBinary(); err == nil {
		version, _ := mihomo.GetBinaryVersion()
		report.AlreadyExists = true
		report.Binary = &installJSONReport{Path: binary, Version: version, Installed: false}
		if !installJSON {
			printInstallStatus(os.Stdout, binary, version)
		}
		return finishReport(report, nil, installJSON)
	}

	result, err := mihomo.InstallMihomo()
	if err != nil {
		return finishReport(report, fmt.Errorf("安装失败: %w", err), installJSON)
	}
	report.Installed = true
	report.Binary = &installJSONReport{
		Path:       result.Path,
		Version:    result.Version,
		ReleaseTag: result.ReleaseTag,
		Installed:  result.Installed,
	}
	if !installJSON {
		printInstallResult(os.Stdout, result)
	}

	return finishReport(report, nil, installJSON)
}

func runStart(cmd *cobra.Command, args []string, cfg *core.AppConfig) error {
	if hasSystemdFn() {
		if err := system.RequireRootForOperation("systemd 服务启动"); err != nil {
			return err
		}
	}

	fmt.Println("🚀 正在启动 Mihomo...")

	runtime := newRuntimeManager()
	result, err := startRuntimeFn(runtime, cfg, defaultStartOptions())
	printRuntimeStartResult(os.Stdout, result)
	if err != nil {
		return err
	}
	return nil
}

func runStop(cmd *cobra.Command, args []string, cfg *core.AppConfig) error {
	if hasSystemdFn() {
		if err := system.RequireRootForOperation("systemd 服务停止"); err != nil {
			return err
		}
	}

	fmt.Println("🛑 正在停止 Mihomo...")

	if hasSystemdFn() {
		if err := mihomo.StopService(mihomo.DefaultServiceName); err == nil {
			fmt.Println("✅ 已通过 systemd 停止")
			return nil
		}
	}

	stopped, err := mihomo.StopManagedProcess(cfg.ConfigDir)
	if err != nil {
		return err
	}
	if stopped {
		fmt.Println("✅ 已停止 Mihomo 进程")
		return nil
	}

	fmt.Println("⚠️  Mihomo 未在运行")
	return nil
}

func runRestart(cmd *cobra.Command, args []string, cfg *core.AppConfig) error {
	if hasSystemdFn() {
		if err := system.RequireRootForOperation("systemd 服务重启"); err != nil {
			return err
		}
	}

	fmt.Println("🔄 正在重启 Mihomo...")

	runtime := newRuntimeManager()
	result, err := startRuntimeFn(runtime, cfg, defaultStartOptions())
	printRuntimeStartResult(os.Stdout, result)
	if err != nil {
		return fmt.Errorf("重启失败: %w", err)
	}
	return nil
}
