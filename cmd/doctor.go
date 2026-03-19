package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
)

var doctorTunMode bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "环境自检",
	Long:  `检查当前环境是否满足 Mihomo 的运行条件；传入 --tun 时会额外检查 TUN 相关条件。`,
	RunE:  runDoctor,
}

var doctorOpenAICmd = &cobra.Command{
	Use:   "openai",
	Short: "诊断 OpenAI/Codex 登录链路",
	Long:  `检查当前 shell 代理环境、直连/代理出口地区，以及 auth.openai.com / api.openai.com 的可达性。`,
	RunE:  runDoctorOpenAI,
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorTunMode, "tun", false, "是否检查 TUN 模式相关条件")
	doctorCmd.AddCommand(doctorOpenAICmd)
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("🩺 环境自检")
	fmt.Println()

	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	results := mihomo.RunDoctor(cfg.ConfigDir, cfg.ControllerAddr, doctorTunMode)

	return printDoctorResults(results)
}

func runDoctorOpenAI(cmd *cobra.Command, args []string) error {
	fmt.Println("🩺 OpenAI / Codex 登录诊断")
	fmt.Println()

	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	report := mihomo.RunOpenAIDoctor(cfg.MixedPort)
	if len(report.Hints) > 0 {
		defer func() {
			fmt.Println()
			fmt.Println("结论:")
			for _, hint := range report.Hints {
				fmt.Printf("  - %s\n", hint)
			}
		}()
	}

	return printDoctorResults(report.Results)
}

func printDoctorResults(results []mihomo.CheckResult) error {

	passed := 0
	failed := 0

	for _, r := range results {
		status := "✅"
		if !r.Passed {
			status = "❌"
			failed++
		} else {
			passed++
		}

		fmt.Printf("  %s %s\n", status, r.Name)
		if r.Problem != "" {
			if r.Passed {
				fmt.Printf("     %s\n", r.Problem)
			} else {
				fmt.Printf("     问题: %s\n", r.Problem)
			}
		}
		if r.Suggest != "" {
			fmt.Printf("     建议: %s\n", r.Suggest)
		}
		fmt.Println()
	}

	fmt.Printf("检查完成: %d 通过, %d 失败\n", passed, failed)

	if failed > 0 {
		return fmt.Errorf("存在 %d 项检查未通过", failed)
	}
	return nil
}
