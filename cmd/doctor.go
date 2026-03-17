package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

var doctorTunMode bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "环境自检",
	Long:  `检查当前环境是否满足 Mihomo TUN 模式的运行条件。`,
	RunE:  runDoctor,
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorTunMode, "tun", true, "是否检查 TUN 模式相关条件")
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("🩺 环境自检")
	fmt.Println()

	results := mihomo.RunDoctor(core.DefaultConfigDir, core.DefaultControllerAddr, doctorTunMode)

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
