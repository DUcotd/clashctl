package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "clashctl",
	Short: "Mihomo 交互式配置工具",
	Long: `clashctl 是一个终端交互式工具，帮助你通过输入订阅 URL
快速完成 Mihomo 代理的配置生成、启动与管理。

推荐主入口：
  - clashctl init            交互式完成安装、配置、启动
  - clashctl nodes           进入节点测速与切换 TUI
  - clashctl service status  查看运行状态
  - clashctl doctor          环境自检

高级/脚本化能力统一放在：
  - clashctl advanced ...`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
