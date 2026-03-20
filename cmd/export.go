package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/system"
)

var (
	exportSubURL    string
	exportMode      string
	exportMixedPort int
	exportOutput    string
)

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
	cmd.MarkFlagRequired("url")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Build app config
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = exportSubURL
	cfg.Mode = exportMode
	cfg.MixedPort = exportMixedPort

	// Validate
	if errs := cfg.Validate(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "❌ %s\n", e)
		}
		return fmt.Errorf("配置校验失败")
	}

	// Validate output path for security
	if err := system.ValidateOutputPath(exportOutput); err != nil {
		return fmt.Errorf("输出路径不安全: %w", err)
	}

	// Build mihomo config
	mihomoCfg := core.BuildMihomoConfig(cfg)

	// Render to YAML
	yamlData, err := core.RenderYAML(mihomoCfg)
	if err != nil {
		return fmt.Errorf("YAML 渲染失败: %w", err)
	}

	// Write to file
	if err := os.WriteFile(exportOutput, yamlData, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	fmt.Printf("✅ 配置已导出到: %s\n", exportOutput)
	fmt.Printf("   模式: %s\n", cfg.Mode)
	fmt.Printf("   订阅: %s\n", cfg.SubscriptionURL)

	return nil
}
