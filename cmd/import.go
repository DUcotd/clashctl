package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"clashctl/internal/app"
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
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "从本地订阅文件生成 Mihomo 配置",
	Long: `从本地文件或标准输入导入原始订阅内容并生成可直接运行的 Mihomo 配置。

支持两类输入：
  - base64 编码的原始订阅
  - 解码后的 vless:// / trojan:// / hysteria2:// 链接列表

示例：
  clashctl import -f sub.txt -o config.yaml
  clashctl import -f links.txt --apply --start
  cat sub.txt | clashctl import -f - --apply --start`,
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVarP(&importFile, "file", "f", "", "本地订阅文件路径（必填）")
	importCmd.Flags().StringVarP(&importOutput, "output", "o", "config.yaml", "输出文件路径")
	importCmd.Flags().StringVarP(&importMode, "mode", "m", "mixed", "运行模式: tun 或 mixed")
	importCmd.Flags().IntVarP(&importMixedPort, "port", "p", core.DefaultMixedPort, "mixed-port 值")
	importCmd.Flags().BoolVar(&importApply, "apply", false, "直接写入当前 clashctl 配置目录")
	importCmd.Flags().BoolVar(&importStart, "start", false, "写入后立即启动 Mihomo（隐含 --apply）")
	importCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	data, sourceDesc, err := readImportSource(importFile)
	if err != nil {
		return fmt.Errorf("读取订阅文件失败: %w", err)
	}

	cfg := core.DefaultAppConfig()
	if importApply || importStart {
		loaded, err := loadAppConfig()
		if err != nil {
			return err
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
		for _, warning := range warnings {
			fmt.Printf("⚠️  %s\n", warning)
		}
	}

	resolver := subscription.NewResolver()
	plan, err := resolver.ResolveContent(cfg, data)
	if err != nil {
		return fmt.Errorf("解析订阅文件失败: %w", err)
	}

	outputPath := importOutput
	if importApply || importStart {
		outputPath = filepath.Join(cfg.ConfigDir, "config.yaml")
		if err := system.EnsureDir(cfg.ConfigDir); err != nil {
			return fmt.Errorf("创建配置目录失败: %w", err)
		}
		backupPath, err := plan.Save(outputPath)
		if err != nil {
			return fmt.Errorf("写入配置文件失败: %w", err)
		}
		if err := app.SaveAppConfig(cfg); err != nil {
			return fmt.Errorf("保存 clashctl 配置失败: %w", err)
		}
		fmt.Printf("✅ 静态配置已写入: %s\n", outputPath)
		if backupPath != "" {
			fmt.Printf("   已备份旧配置: %s\n", backupPath)
		}
	} else {
		yamlData, err := plan.RenderYAML()
		if err != nil {
			return fmt.Errorf("YAML 渲染失败: %w", err)
		}
		if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
		fmt.Printf("✅ 配置已导出到: %s\n", outputPath)
	}

	fmt.Printf("   来源格式: %s\n", plan.DetectedFormat)
	fmt.Printf("   读取来源: %s\n", sourceDesc)
	if plan.ProxyCount > 0 {
		fmt.Printf("   节点数量: %d\n", plan.ProxyCount)
	}
	fmt.Printf("   模式: %s\n", cfg.Mode)
	if plan.Kind != subscription.PlanKindProvider {
		fmt.Println("   说明: 这是静态配置，不依赖服务器再次拉取订阅 URL")
	}

	if importStart {
		fmt.Println("🚀 正在启动 Mihomo...")
		result, err := runtime.Start(cfg, mihomo.StartOptions{
			VerifyInventory: true,
			WaitRetries:     15,
			WaitInterval:    2 * time.Second,
		})
		printRuntimeStartResult(os.Stdout, result)
		if err != nil {
			return err
		}
	}

	return nil
}

func readImportSource(path string) ([]byte, string, error) {
	if path == "-" {
		data, err := io.ReadAll(os.Stdin)
		return data, "stdin", err
	}
	data, err := os.ReadFile(path)
	return data, path, err
}
