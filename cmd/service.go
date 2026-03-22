package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
	"clashctl/internal/system"
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

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装 Mihomo 内核",
	RunE:  runInstall,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Mihomo 服务",
	RunE:  runStart,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Mihomo 服务",
	RunE:  runStop,
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "重启 Mihomo 服务",
	RunE:  runRestart,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看 Mihomo 运行状态",
	RunE:  runStatus,
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

func bindInstallFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&installJSON, "json", false, "以 JSON 输出安装结果")
}

func runInstall(cmd *cobra.Command, args []string) error {
	report := &installRunReport{}
	if err := system.RequireRoot(); err != nil {
		report.RequiresRoot = true
		return finishInstallReport(report, err)
	}

	if binary, err := mihomo.FindBinary(); err == nil {
		version, _ := mihomo.GetBinaryVersion()
		report.AlreadyExists = true
		report.Binary = &installJSONReport{Path: binary, Version: version, Installed: false}
		if !installJSON {
			printInstallStatus(os.Stdout, binary, version)
		}
		return finishInstallReport(report, nil)
	}

	result, err := mihomo.InstallMihomo()
	if err != nil {
		return finishInstallReport(report, fmt.Errorf("安装失败: %w", err))
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

	return finishInstallReport(report, nil)
}

func finishInstallReport(report *installRunReport, err error) error {
	if err != nil && report != nil {
		report.Error = err.Error()
	}
	if installJSON && report != nil {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func runStart(cmd *cobra.Command, args []string) error {
	if mihomo.HasSystemd() {
		if err := system.RequireRootForOperation("systemd 服务启动"); err != nil {
			return err
		}
	}

	fmt.Println("🚀 正在启动 Mihomo...")

	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	runtime := mihomo.NewRuntimeManager()
	result, err := runtime.Start(cfg, mihomo.StartOptions{
		VerifyInventory: true,
		WaitRetries:     15,
		WaitInterval:    2 * time.Second,
	})
	printRuntimeStartResult(os.Stdout, result)
	if err != nil {
		return err
	}
	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	if mihomo.HasSystemd() {
		if err := system.RequireRootForOperation("systemd 服务停止"); err != nil {
			return err
		}
	}

	fmt.Println("🛑 正在停止 Mihomo...")
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	if mihomo.HasSystemd() {
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

func runRestart(cmd *cobra.Command, args []string) error {
	if mihomo.HasSystemd() {
		if err := system.RequireRootForOperation("systemd 服务重启"); err != nil {
			return err
		}
	}

	fmt.Println("🔄 正在重启 Mihomo...")
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	if mihomo.HasSystemd() {
		if err := mihomo.RestartService(mihomo.DefaultServiceName); err == nil {
			fmt.Println("✅ Mihomo 已重启")
			return nil
		} else {
			fmt.Printf("⚠️  systemd 重启失败: %v\n正在回退到进程模式...\n", err)
		}
	}

	if _, err := mihomo.StopManagedProcess(cfg.ConfigDir); err != nil {
		return fmt.Errorf("重启失败: %w", err)
	}

	proc := mihomo.NewProcess(cfg.ConfigDir)
	if err := proc.Start(); err != nil {
		return fmt.Errorf("重启失败: %w", err)
	}

	fmt.Println("✅ Mihomo 已重启")
	return nil
}
