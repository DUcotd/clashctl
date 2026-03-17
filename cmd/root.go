package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "clashctl",
	Short: "Mihomo TUN 交互式配置工具",
	Long: `clashctl 是一个终端交互式工具，帮助你通过输入机场订阅 URL
快速完成 Mihomo 代理的配置生成、启动与管理。

只需输入订阅链接，clashctl 会自动：
  - 生成 Mihomo 配置文件
  - 启动服务
  - 进行自检

使用 clashctl init 开始交互式配置向导。`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
