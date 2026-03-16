package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"myproxy/internal/core"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理配置",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前 Mihomo 配置",
	RunE:  runConfigShow,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "显示配置文件路径",
	RunE:  runConfigPath,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	path := core.DefaultConfigDir + "/config.yaml"
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("无法读取配置文件 %s: %w", path, err)
	}
	fmt.Println(string(data))
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	fmt.Printf("配置目录: %s\n", core.DefaultConfigDir)
	fmt.Printf("配置文件: %s/config.yaml\n", core.DefaultConfigDir)
	fmt.Printf("Provider: %s/%s\n", core.DefaultConfigDir, core.DefaultProviderPath)
	return nil
}
