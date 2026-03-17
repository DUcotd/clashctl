package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("clashctl %s\n", currentVer)
		fmt.Println("Mihomo TUN 交互式 CLI 配置工具")
		fmt.Printf("https://github.com/%s/%s\n", githubOwner, githubRepo)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
