package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"clashctl/internal/app"
	"clashctl/internal/config"
	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	"clashctl/internal/subscription"
	"clashctl/internal/system"
)

var (
	importFile      string
	importOutput    string
	importMode      string
	importMixedPort int
	importApply     bool
	importStart     bool
	importJSON      bool
)

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

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "从本地订阅文件生成 Mihomo 配置",
	Long: `从本地文件或标准输入导入原始订阅内容并生成可直接运行的 Mihomo 配置。

支持两类输入：
  - base64 编码的原始订阅
  - 解码后的 vless:// / trojan:// / hysteria2:// 链接列表

示例：
  clashctl advanced import -f sub.txt -o config.yaml
  clashctl advanced import -f links.txt --apply --start
  cat sub.txt | clashctl advanced import -f - --apply --start`,
	Hidden: true,
	RunE:   legacyRunner("clashctl import", "clashctl advanced import", runImport),
}

func init() {
	bindImportFlags(importCmd)
	rootCmd.AddCommand(importCmd)
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

	// Validate output path for security (only when not using --apply)
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
		outputPath = filepath.Join(cfg.ConfigDir, "config.yaml")
		report.OutputPath = outputPath
		if err := system.EnsureDir(cfg.ConfigDir); err != nil {
			return finishImportReport(report, fmt.Errorf("创建配置目录失败: %w", err))
		}
		backupPath, err := plan.Save(outputPath)
		if err != nil {
			return finishImportReport(report, fmt.Errorf("写入配置文件失败: %w", err))
		}
		report.BackupPath = backupPath
		if err := app.SaveAppConfig(cfg); err != nil {
			return finishImportReport(report, fmt.Errorf("保存 clashctl 配置失败: %w", err))
		}
		if !importJSON {
			fmt.Printf("✅ 静态配置已写入: %s\n", outputPath)
			if backupPath != "" {
				fmt.Printf("   已备份旧配置: %s\n", backupPath)
			}
		}
	} else {
		report.OutputPath = outputPath
		yamlData, err := plan.RenderYAML()
		if err != nil {
			return finishImportReport(report, fmt.Errorf("YAML 渲染失败: %w", err))
		}
		if err := config.WriteConfig(outputPath, yamlData); err != nil {
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
