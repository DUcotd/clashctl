package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	configfile "clashctl/internal/config"
	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	setupflow "clashctl/internal/setup"
	"clashctl/internal/subscription"
	"clashctl/internal/system"
)

var (
	configShowJSON  bool
	configPathJSON  bool
	exportSubURL    string
	exportMode      string
	exportMixedPort int
	exportOutput    string
	exportJSON      bool
	importFile      string
	importOutput    string
	importMode      string
	importMixedPort int
	importApply     bool
	importStart     bool
	importJSON      bool
)

type configShowReport struct {
	ConfigPath string `json:"config_path"`
	Content    string `json:"content"`
	Error      string `json:"error,omitempty"`
}

type configPathReport struct {
	ConfigDir    string `json:"config_dir"`
	ConfigPath   string `json:"config_path"`
	ProviderPath string `json:"provider_path"`
	Error        string `json:"error,omitempty"`
}

type exportRunReport struct {
	SubscriptionURL string   `json:"subscription_url"`
	Mode            string   `json:"mode"`
	MixedPort       int      `json:"mixed_port,omitempty"`
	OutputPath      string   `json:"output_path"`
	Written         bool     `json:"written"`
	Errors          []string `json:"errors,omitempty"`
	Error           string   `json:"error,omitempty"`
}

type importRunReport struct {
	Source          string                  `json:"source"`
	OutputPath      string                  `json:"output_path,omitempty"`
	Apply           bool                    `json:"apply"`
	Start           bool                    `json:"start"`
	Mode            string                  `json:"mode"`
	MixedPort       int                     `json:"mixed_port,omitempty"`
	PlanKind        string                  `json:"plan_kind,omitempty"`
	ContentKind     string                  `json:"content_kind,omitempty"`
	DetectedFormat  string                  `json:"detected_format,omitempty"`
	Summary         string                  `json:"summary,omitempty"`
	ProxyCount      int                     `json:"proxy_count,omitempty"`
	VerifyInventory bool                    `json:"verify_inventory"`
	UsedProxyEnv    bool                    `json:"used_proxy_env,omitempty"`
	BackupPath      string                  `json:"backup_path,omitempty"`
	Warnings        []string                `json:"warnings,omitempty"`
	Runtime         *runtimeStartJSONReport `json:"runtime,omitempty"`
	Error           string                  `json:"error,omitempty"`
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "导入导出与查看配置",
	Long:  `统一管理 Mihomo 配置的导入、导出、查看和路径查询。`,
}

var configImportCmd = &cobra.Command{
	Use:   "import",
	Short: "从本地订阅文件生成 Mihomo 配置",
	RunE:  runImport,
}

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出 Mihomo 配置文件",
	RunE:  runExport,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前 Mihomo 配置",
	RunE:  runConfigShow,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "显示配置文件路径",
	RunE:  runConfigPath,
}

func init() {
	bindExportFlags(configExportCmd)
	bindImportFlags(configImportCmd)
	bindConfigFlags(configShowCmd, configPathCmd)
	configCmd.AddCommand(configImportCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	rootCmd.AddCommand(configCmd)
}

func bindConfigFlags(showCmd, pathCmd *cobra.Command) {
	showCmd.Flags().BoolVar(&configShowJSON, "json", false, "以 JSON 输出当前配置")
	pathCmd.Flags().BoolVar(&configPathJSON, "json", false, "以 JSON 输出配置路径")
}

func bindExportFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&exportSubURL, "url", "u", "", "订阅 URL（必填）")
	cmd.Flags().StringVarP(&exportMode, "mode", "m", "mixed", "运行模式: tun 或 mixed")
	cmd.Flags().IntVarP(&exportMixedPort, "port", "p", core.DefaultMixedPort, "mixed-port 值")
	cmd.Flags().StringVarP(&exportOutput, "output", "o", "config.yaml", "输出文件路径")
	cmd.Flags().BoolVar(&exportJSON, "json", false, "以 JSON 输出导出结果")
	if err := cmd.MarkFlagRequired("url"); err != nil {
		panic(err)
	}
}

func bindImportFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&importFile, "file", "f", "", "本地订阅文件路径（必填）")
	cmd.Flags().StringVarP(&importOutput, "output", "o", "config.yaml", "输出文件路径")
	cmd.Flags().StringVarP(&importMode, "mode", "m", "mixed", "运行模式: tun 或 mixed")
	cmd.Flags().IntVarP(&importMixedPort, "port", "p", core.DefaultMixedPort, "mixed-port 值")
	cmd.Flags().BoolVar(&importApply, "apply", false, "直接写入当前 clashctl 配置目录")
	cmd.Flags().BoolVar(&importStart, "start", false, "写入后立即启动 Mihomo（隐含 --apply）")
	cmd.Flags().BoolVar(&importJSON, "json", false, "以 JSON 输出导入结果")
	if err := cmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}
}

func runExport(cmd *cobra.Command, args []string) error {
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = exportSubURL
	cfg.Mode = exportMode
	cfg.MixedPort = exportMixedPort
	report := buildExportReport(cfg, exportOutput)

	if errs := cfg.Validate(); len(errs) > 0 {
		report.Errors = append(report.Errors, errs...)
		if !exportJSON {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "❌ %s\n", e)
			}
		}
		return finishExportReport(report, fmt.Errorf("配置校验失败"))
	}

	if err := system.ValidateOutputPath(exportOutput); err != nil {
		return finishExportReport(report, fmt.Errorf("输出路径不安全: %w", err))
	}

	mihomoCfg := core.BuildMihomoConfig(cfg)
	yamlData, err := core.RenderYAML(mihomoCfg)
	if err != nil {
		return finishExportReport(report, fmt.Errorf("YAML 渲染失败: %w", err))
	}

	if err := configfile.WriteConfig(exportOutput, yamlData); err != nil {
		return finishExportReport(report, fmt.Errorf("写入文件失败: %w", err))
	}
	report.Written = true
	if exportJSON {
		return finishExportReport(report, nil)
	}

	fmt.Printf("✅ 配置已导出到: %s\n", exportOutput)
	fmt.Printf("   模式: %s\n", cfg.Mode)
	fmt.Printf("   订阅: %s\n", cfg.SubscriptionURL)

	return finishExportReport(report, nil)
}

func buildExportReport(cfg *core.AppConfig, outputPath string) *exportRunReport {
	report := &exportRunReport{
		SubscriptionURL: cfg.SubscriptionURL,
		Mode:            cfg.Mode,
		OutputPath:      outputPath,
	}
	if cfg.Mode == "mixed" {
		report.MixedPort = cfg.MixedPort
	}
	return report
}

func finishExportReport(report *exportRunReport, err error) error {
	if err != nil && report != nil {
		report.Error = err.Error()
	}
	if exportJSON && report != nil {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func runImport(cmd *cobra.Command, args []string) error {
	report := &importRunReport{
		Apply:      importApply || importStart,
		Start:      importStart,
		Mode:       importMode,
		OutputPath: importOutput,
	}
	if importMode == "mixed" {
		report.MixedPort = importMixedPort
	}
	data, sourceDesc, err := readImportSource(importFile)
	if err != nil {
		return finishImportReport(report, fmt.Errorf("读取订阅文件失败: %w", err))
	}
	report.Source = sourceDesc

	if !importApply && !importStart {
		if err := system.ValidateOutputPath(importOutput); err != nil {
			return finishImportReport(report, fmt.Errorf("输出路径不安全: %w", err))
		}
	}

	cfg := core.DefaultAppConfig()
	if importApply || importStart {
		loaded, err := loadAppConfig()
		if err != nil {
			return finishImportReport(report, err)
		}
		cfg = loaded
	}
	cfg.SubscriptionURL = ""
	cfg.Mode = importMode
	cfg.MixedPort = importMixedPort

	runtime := mihomo.NewRuntimeManager()
	if importApply || importStart {
		resolved, warnings := runtime.ResolveConfig(cfg)
		cfg = resolved
		report.Mode = cfg.Mode
		report.MixedPort = 0
		if cfg.Mode == "mixed" {
			report.MixedPort = cfg.MixedPort
		}
		report.Warnings = append(report.Warnings, warnings...)
		if !importJSON {
			for _, warning := range warnings {
				fmt.Printf("⚠️  %s\n", warning)
			}
		}
	}

	resolver := subscription.NewResolver()
	plan, err := resolver.ResolveContent(cfg, data)
	if err != nil {
		return finishImportReport(report, fmt.Errorf("解析订阅文件失败: %w", err))
	}
	populateImportReport(report, cfg, plan)

	outputPath := importOutput
	if importApply || importStart {
		applyResult, err := setupflow.ApplyResolvedPlan(cfg, plan, setupflow.ApplyPlanOptions{SaveAppConfig: true})
		if applyResult != nil {
			outputPath = applyResult.OutputPath
			report.OutputPath = applyResult.OutputPath
			report.BackupPath = applyResult.BackupPath
			if applyResult.ValidationError != nil {
				report.Warnings = append(report.Warnings, "配置校验提示: "+applyResult.ValidationError.Error())
				if !importJSON {
					fmt.Printf("⚠️  %s\n", "配置校验提示: "+applyResult.ValidationError.Error())
				}
			}
		}
		if err != nil {
			if wrapped := setupflow.WrapStageError(err, setupflow.StageCreateConfigDir, "创建配置目录失败: %w"); wrapped != err {
				return finishImportReport(report, wrapped)
			}
			if wrapped := setupflow.WrapStageError(err, setupflow.StageWriteConfig, "写入配置文件失败: %w"); wrapped != err {
				return finishImportReport(report, wrapped)
			}
			if wrapped := setupflow.WrapStageError(err, setupflow.StageSaveAppConfig, "保存 clashctl 配置失败: %w"); wrapped != err {
				return finishImportReport(report, wrapped)
			}
			return finishImportReport(report, err)
		}
		if !importJSON {
			fmt.Printf("✅ 静态配置已写入: %s\n", outputPath)
			if report.BackupPath != "" {
				fmt.Printf("   已备份旧配置: %s\n", report.BackupPath)
			}
		}
	} else {
		report.OutputPath = outputPath
		yamlData, err := plan.RenderYAML()
		if err != nil {
			return finishImportReport(report, fmt.Errorf("YAML 渲染失败: %w", err))
		}
		if err := configfile.WriteConfig(outputPath, yamlData); err != nil {
			return finishImportReport(report, fmt.Errorf("写入文件失败: %w", err))
		}
		if !importJSON {
			fmt.Printf("✅ 配置已导出到: %s\n", outputPath)
		}
	}

	if !importJSON {
		fmt.Printf("   来源格式: %s\n", plan.DetectedFormat)
		fmt.Printf("   读取来源: %s\n", sourceDesc)
		if plan.ProxyCount > 0 {
			fmt.Printf("   节点数量: %d\n", plan.ProxyCount)
		}
		fmt.Printf("   模式: %s\n", cfg.Mode)
		if plan.Kind != subscription.PlanKindProvider {
			fmt.Println("   说明: 这是静态配置，不依赖服务器再次拉取订阅 URL")
		}
	}

	if importStart {
		if !importJSON {
			fmt.Println("🚀 正在启动 Mihomo...")
		}
		result, err := runtime.Start(cfg, mihomo.StartOptions{
			VerifyInventory: true,
			WaitRetries:     15,
			WaitInterval:    2 * time.Second,
		})
		report.Runtime = buildRuntimeStartJSONReport(result)
		if !importJSON {
			printRuntimeStartResult(os.Stdout, result)
		}
		if err != nil {
			return finishImportReport(report, err)
		}
	}

	return finishImportReport(report, nil)
}

func populateImportReport(report *importRunReport, cfg *core.AppConfig, plan *subscription.ResolvedConfigPlan) {
	if report == nil || plan == nil || cfg == nil {
		return
	}
	report.Mode = cfg.Mode
	report.MixedPort = 0
	if cfg.Mode == "mixed" {
		report.MixedPort = cfg.MixedPort
	}
	report.PlanKind = string(plan.Kind)
	report.ContentKind = plan.ContentKind
	report.DetectedFormat = plan.DetectedFormat
	report.Summary = plan.Summary
	report.ProxyCount = plan.ProxyCount
	report.VerifyInventory = plan.VerifyInventory
	report.UsedProxyEnv = plan.UsedProxyEnv
	report.Warnings = append(report.Warnings, plan.Warnings...)
	if plan.Sanitized {
		report.Warnings = append(report.Warnings, "订阅 YAML 已按安全策略裁剪")
	}
	if len(plan.RemovedFields) > 0 {
		report.Warnings = append(report.Warnings, "已移除字段: "+strings.Join(plan.RemovedFields, ", "))
	}
}

func finishImportReport(report *importRunReport, err error) error {
	if err != nil && report != nil {
		report.Error = err.Error()
	}
	if importJSON && report != nil {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func readImportSource(path string) ([]byte, string, error) {
	if path == "-" {
		data, err := io.ReadAll(os.Stdin)
		return data, "stdin", err
	}
	data, err := os.ReadFile(path)
	return data, path, err
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	configPath := mihomoConfigPath(cfg)
	report := &configShowReport{ConfigPath: configPath}
	data, err := configfile.ReadConfigWithLimit(configPath)
	if err != nil {
		return finishConfigShowReport(report, fmt.Errorf("无法读取配置文件 %s: %w", configPath, err))
	}
	report.Content = string(data)
	if configShowJSON {
		return finishConfigShowReport(report, nil)
	}
	fmt.Println(string(data))
	return finishConfigShowReport(report, nil)
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}
	report := buildConfigPathReport(cfg)
	if configPathJSON {
		return finishConfigPathReport(report, nil)
	}

	fmt.Printf("配置目录: %s\n", cfg.ConfigDir)
	fmt.Printf("配置文件: %s\n", mihomoConfigPath(cfg))
	fmt.Printf("Provider: %s\n", mihomoProviderPath(cfg))
	return finishConfigPathReport(report, nil)
}

func buildConfigPathReport(cfg *core.AppConfig) *configPathReport {
	return &configPathReport{
		ConfigDir:    cfg.ConfigDir,
		ConfigPath:   mihomoConfigPath(cfg),
		ProviderPath: mihomoProviderPath(cfg),
	}
}

func finishConfigShowReport(report *configShowReport, err error) error {
	if err != nil && report != nil {
		report.Error = err.Error()
	}
	if configShowJSON && report != nil {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func finishConfigPathReport(report *configPathReport, err error) error {
	if err != nil && report != nil {
		report.Error = err.Error()
	}
	if configPathJSON && report != nil {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	}
	return err
}
