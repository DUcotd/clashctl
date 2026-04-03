package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
)

var (
	doctorTunMode bool
	doctorJSON    bool
)

type doctorSummary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

type doctorReport struct {
	Command string               `json:"command"`
	TunMode bool                 `json:"tun_mode,omitempty"`
	Summary doctorSummary        `json:"summary"`
	Results []mihomo.CheckResult `json:"results"`
	Hints   []string             `json:"hints,omitempty"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "环境自检",
	Long:  `检查当前环境是否满足 Mihomo 的运行条件；传入 --tun 时会额外检查 TUN 相关条件。`,
	RunE:  runDoctor,
}

var doctorOpenAICmd = &cobra.Command{
	Use:   "openai",
	Short: "诊断 OpenAI/Codex 登录链路",
	Long:  `检查当前 shell 代理环境、直连/代理出口地区，以及 auth.openai.com / api.openai.com / chatgpt.com/backend-api 的可达性。`,
	RunE:  runDoctorOpenAI,
}

func init() {
	doctorCmd.PersistentFlags().BoolVar(&doctorJSON, "json", false, "以 JSON 输出检查结果")
	doctorCmd.Flags().BoolVar(&doctorTunMode, "tun", false, "是否检查 TUN 模式相关条件")
	doctorCmd.AddCommand(doctorOpenAICmd)
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	results := mihomo.RunDoctor(cfg.ConfigDir, cfg.ControllerAddr, cfg.ControllerSecret, doctorTunMode)
	report := buildDoctorReport("doctor", doctorTunMode, results, nil)
	if doctorJSON {
		if err := writeJSON(report); err != nil {
			return err
		}
		return doctorSummaryError(report.Summary)
	}

	fmt.Println("🩺 环境自检")
	fmt.Println()
	return printDoctorResults(os.Stdout, report)
}

func runDoctorOpenAI(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	report := mihomo.RunOpenAIDoctor(cfg.MixedPort)
	result := buildDoctorReport("doctor openai", false, report.Results, report.Hints)
	if doctorJSON {
		if err := writeJSON(result); err != nil {
			return err
		}
		return doctorSummaryError(result.Summary)
	}

	fmt.Println("🩺 OpenAI / Codex 登录诊断")
	fmt.Println()
	return printDoctorResults(os.Stdout, result)
}

func buildDoctorReport(command string, tunMode bool, results []mihomo.CheckResult, hints []string) *doctorReport {
	passed, failed := summarizeDoctorResults(results)
	return &doctorReport{
		Command: command,
		TunMode: tunMode,
		Summary: doctorSummary{Passed: passed, Failed: failed},
		Results: results,
		Hints:   append([]string(nil), hints...),
	}
}

func summarizeDoctorResults(results []mihomo.CheckResult) (passed int, failed int) {
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}
	return passed, failed
}

func printDoctorResults(w io.Writer, report *doctorReport) error {
	for _, r := range report.Results {
		status := "✅"
		if !r.Passed {
			status = "❌"
		}

		fmt.Fprintf(w, "  %s %s\n", status, r.Name)
		if r.Problem != "" {
			if r.Passed {
				fmt.Fprintf(w, "     %s\n", r.Problem)
			} else {
				fmt.Fprintf(w, "     问题: %s\n", r.Problem)
			}
		}
		if r.Suggest != "" {
			fmt.Fprintf(w, "     建议: %s\n", r.Suggest)
		}
		fmt.Fprintln(w)
	}

	if len(report.Hints) > 0 {
		fmt.Fprintln(w, "结论:")
		for _, hint := range report.Hints {
			fmt.Fprintf(w, "  - %s\n", hint)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "检查完成: %d 通过, %d 失败\n", report.Summary.Passed, report.Summary.Failed)

	return doctorSummaryError(report.Summary)
}

func doctorSummaryError(summary doctorSummary) error {
	if summary.Failed > 0 {
		return fmt.Errorf("存在 %d 项检查未通过", summary.Failed)
	}
	return nil
}
