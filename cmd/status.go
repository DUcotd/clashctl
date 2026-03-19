package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

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
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	// Check systemd service
	serviceActive := false
	if mihomo.HasSystemd() {
		serviceActive, _ = mihomo.ServiceStatus(mihomo.DefaultServiceName)
	}

	client := mihomo.NewClient("http://" + cfg.ControllerAddr)
	controllerOK := client.CheckConnection() == nil

	if serviceActive {
		fmt.Println("  服务状态: ✅ 运行中 (systemd)")
	} else if controllerOK {
		fmt.Println("  服务状态: ✅ 运行中 (子进程/API 可达)")
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
			fmt.Printf("  版本: %s\n", version)
		}
	}

	// Check config path
	fmt.Printf("  配置目录: %s\n", cfg.ConfigDir)

	// Check controller API
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
	if controllerOK {
		groups, err := client.GetAllProxyGroups()
		if err == nil && len(groups) > 0 {
			fmt.Println("\n  ── 代理组 ──")
			for name, group := range groups {
				typ := mihomo.NormalizeProxyType(group.Type)
				marker := "  "
				if group.Now != "" {
					marker = "▸ "
				}
				fmt.Printf("\n  %s%s [%s]\n", marker, name, typ)
				if group.Now != "" {
					fmt.Printf("     当前: %s\n", group.Now)
				}
				fmt.Printf("     节点数: %d\n", len(group.All))
			}

			if inv, err := client.InspectProxyInventory("PROXY"); err == nil {
				if inv.OnlyCompatible {
					fmt.Println("\n  ⚠ 订阅节点未成功加载；当前仅剩 COMPATIBLE。")
					fmt.Println("    常见原因: 服务器无法直连订阅 URL，或 provider 拉取失败。")
					fmt.Println("    可改用 'clashctl import --file sub.txt -o /etc/mihomo/config.yaml' 生成静态配置。")
				}
			}
		}
	}

	return nil
}
