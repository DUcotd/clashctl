package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"clashctl/internal/config"
	"clashctl/internal/core"
	"clashctl/internal/system"
)

var (
	exportSubURL    string
	exportMode      string
	exportMixedPort int
	exportOutput    string
	exportJSON      bool
)

type exportRunReport struct {
	SubscriptionURL string   `json:"subscription_url"`
	Mode            string   `json:"mode"`
	MixedPort       int      `json:"mixed_port,omitempty"`
	OutputPath      string   `json:"output_path"`
	Written         bool     `json:"written"`
	Errors          []string `json:"errors,omitempty"`
	Error           string   `json:"error,omitempty"`
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出 Mihomo 配置文件",
	Long: `根据指定的参数生成 Mihomo 配置文件并导出。

示例：
  clashctl advanced export -u https://example.com/sub -o config.yaml
  clashctl advanced export --url https://example.com/sub --mode mixed --output config.yaml`,
	Hidden: true,
	RunE:   legacyRunner("clashctl export", "clashctl advanced export", runExport),
}

func init() {
	bindExportFlags(exportCmd)
	rootCmd.AddCommand(exportCmd)
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

	if err := config.WriteConfig(exportOutput, yamlData); err != nil {
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
