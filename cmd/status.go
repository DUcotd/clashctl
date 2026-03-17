package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看 Mihomo 运行状态",
	Long:  `显示 Mihomo 服务状态、配置路径、controller 连接情况和当前代理组信息。`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("📊 Mihomo 状态")
	fmt.Println()

	// Check systemd service
	active, _ := mihomo.ServiceStatus("clashctl-mihomo")
	if active {
		fmt.Println("  服务状态: ✅ 运行中 (systemd)")
	} else {
		fmt.Println("  服务状态: ❌ 未运行")
	}

	// Check binary
	binary, err := mihomo.FindBinary()
	if err != nil {
		fmt.Printf("  可执行文件: ❌ %s\n", err.Error())
	} else {
		version, _ := mihomo.GetBinaryVersion()
		fmt.Printf("  可执行文件: ✅ %s\n", binary)
		if version != "" {
			fmt.Printf("  版本: %s", version)
		}
	}

	// Check config path
	fmt.Printf("  配置目录: %s\n", core.DefaultConfigDir)

	// Check controller API
	client := mihomo.NewClient("http://" + core.DefaultControllerAddr)
	if err := client.CheckConnection(); err != nil {
		fmt.Printf("  Controller API: ❌ %s\n", err.Error())
	} else {
		mihomoVer, _ := client.Version()
		fmt.Printf("  Controller API: ✅ 可达")
		if mihomoVer != "" {
			fmt.Printf(" (Mihomo %s)", mihomoVer)
		}
		fmt.Println()
	}

	// Show all proxy groups if API is reachable
	if active {
		groups, err := client.GetAllProxyGroups()
		if err == nil && len(groups) > 0 {
			fmt.Println("\n  ── 代理组 ──")
			for name, group := range groups {
				// Only show select-type groups (the ones users care about)
				if group.Type != "select" && group.Type != "url-test" {
					continue
				}
				marker := "  "
				if group.Now != "" {
					marker = "▸ "
				}
				fmt.Printf("\n  %s%s [%s]\n", marker, name, group.Type)
				if group.Now != "" {
					fmt.Printf("     当前: %s\n", group.Now)
				}
				fmt.Printf("     节点数: %d\n", len(group.All))
			}
		}
	}

	return nil
}
