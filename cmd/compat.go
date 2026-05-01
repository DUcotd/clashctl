package cmd

import "github.com/spf13/cobra"

var restoreCmd = &cobra.Command{
	Use:    "restore [version]",
	Short:  "恢复历史配置",
	Long:   `从备份中恢复配置文件。不指定版本则列出可用备份。`,
	Args:   cobra.MaximumNArgs(1),
	Hidden: true,
	RunE:   legacyRunner("clashctl restore", "clashctl backup restore", withAppConfig(runRestore)),
}

var advancedCmd = &cobra.Command{
	Use:    "advanced",
	Short:  "旧版高级命令入口（兼容保留）",
	Long:   `旧版高级命令入口，仅为兼容保留；请改用 service/config/backup 等按任务分组的命令。`,
	Hidden: true,
}

var advancedInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装 Mihomo 内核",
	RunE:  legacyRunner("clashctl advanced install", "clashctl service install", runInstall),
}

var advancedExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出 Mihomo 配置文件",
	RunE:  legacyRunner("clashctl advanced export", "clashctl config export", runExport),
}

var advancedImportCmd = &cobra.Command{
	Use:   "import",
	Short: "从本地订阅文件生成 Mihomo 配置",
	RunE:  legacyRunner("clashctl advanced import", "clashctl config import", runImport),
}

var advancedConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "查看当前配置与路径",
}

var advancedConfigShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前 Mihomo 配置",
	RunE:  legacyRunner("clashctl advanced config show", "clashctl config show", withAppConfig(runConfigShow)),
}

var advancedConfigPathCmd = &cobra.Command{
	Use:   "path",
	Short: "显示配置文件路径",
	RunE:  legacyRunner("clashctl advanced config path", "clashctl config path", withAppConfig(runConfigPath)),
}

var startCmd = &cobra.Command{
	Use:    "start",
	Short:  "启动 Mihomo 服务",
	Long:   `根据已有配置文件启动 Mihomo。优先使用 systemd，否则以子进程方式启动。`,
	Hidden: true,
	RunE:   legacyRunner("clashctl start", "clashctl service start", withAppConfig(runStart)),
}

var stopCmd = &cobra.Command{
	Use:    "stop",
	Short:  "停止 Mihomo 服务",
	Hidden: true,
	RunE:   legacyRunner("clashctl stop", "clashctl service stop", withAppConfig(runStop)),
}

var restartCmd = &cobra.Command{
	Use:    "restart",
	Short:  "重启 Mihomo 服务",
	Hidden: true,
	RunE:   legacyRunner("clashctl restart", "clashctl service restart", withAppConfig(runRestart)),
}

var statusCmd = &cobra.Command{
	Use:    "status",
	Short:  "查看 Mihomo 运行状态",
	Long:   `显示 Mihomo 服务状态、配置路径、controller 连接情况和当前代理组信息。`,
	Hidden: true,
	RunE:   legacyRunner("clashctl status", "clashctl service status", withAppConfig(runStatus)),
}

var installCmd = &cobra.Command{
	Use:    "install",
	Short:  "安装 Mihomo 内核",
	Long:   `自动下载并安装最新版本的 Mihomo 内核到 /usr/local/bin/mihomo。`,
	Hidden: true,
	RunE:   legacyRunner("clashctl install", "clashctl service install", runInstall),
}

var exportCmd = &cobra.Command{
	Use:    "export",
	Short:  "导出 Mihomo 配置文件",
	Long:   "根据指定的参数生成 Mihomo 配置文件并导出。",
	Hidden: true,
	RunE:   legacyRunner("clashctl export", "clashctl config export", runExport),
}

var importCmd = &cobra.Command{
	Use:    "import",
	Short:  "从本地订阅文件生成 Mihomo 配置",
	Long:   "从本地文件或标准输入导入原始订阅内容并生成可直接运行的 Mihomo 配置。",
	Hidden: true,
	RunE:   legacyRunner("clashctl import", "clashctl config import", runImport),
}

var tuiCmd = &cobra.Command{
	Use:    "tui",
	Short:  "启动交互式管理界面",
	Hidden: true,
}

var tuiNodesCmd = &cobra.Command{
	Use:    "nodes",
	Short:  "直接进入节点测速与切换界面",
	Long:   `跳过 init 向导，直接进入代理组/节点管理 TUI，可测速并切换节点。`,
	Hidden: true,
	RunE:   legacyRunner("clashctl tui nodes", "clashctl nodes", withAppConfig(runTUINodes)),
}

func init() {
	restoreCmd.Flags().BoolVar(&restoreJSON, "json", false, "以 JSON 输出恢复结果")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "以 JSON 输出状态信息")
	bindExportFlags(advancedExportCmd)
	bindImportFlags(advancedImportCmd)
	bindConfigFlags(advancedConfigShowCmd, advancedConfigPathCmd)
	bindInstallFlags(serviceInstallCmd)
	bindInstallFlags(advancedInstallCmd)
	bindInstallFlags(installCmd)
	bindExportFlags(exportCmd)
	bindImportFlags(importCmd)
	advancedConfigCmd.AddCommand(advancedConfigShowCmd)
	advancedConfigCmd.AddCommand(advancedConfigPathCmd)
	advancedCmd.AddCommand(advancedInstallCmd)
	advancedCmd.AddCommand(advancedExportCmd)
	advancedCmd.AddCommand(advancedImportCmd)
	advancedCmd.AddCommand(advancedConfigCmd)
	tuiCmd.AddCommand(tuiNodesCmd)
	rootCmd.AddCommand(advancedCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(tuiCmd)
}
