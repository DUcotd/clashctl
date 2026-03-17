package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "管理代理节点",
}

var nodesListCmd = &cobra.Command{
	Use:   "list [代理组名]",
	Short: "列出代理组中的节点",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runNodesList,
}

var nodesUseCmd = &cobra.Command{
	Use:   "use <节点名称> [代理组名]",
	Short: "切换到指定节点",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runNodesUse,
}

var nodesGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "列出所有代理组",
	RunE:  runNodesGroups,
}

func init() {
	nodesCmd.AddCommand(nodesListCmd)
	nodesCmd.AddCommand(nodesUseCmd)
	nodesCmd.AddCommand(nodesGroupsCmd)
	rootCmd.AddCommand(nodesCmd)
}

func runNodesList(cmd *cobra.Command, args []string) error {
	groupName := "PROXY"
	if len(args) > 0 {
		groupName = args[0]
	}

	client := mihomo.NewClient("http://" + core.DefaultControllerAddr)

	detail, err := client.GetProxyGroupDetail(groupName)
	if err != nil {
		return fmt.Errorf("获取节点列表失败: %w", err)
	}

	fmt.Printf("📡 代理组: %s (%s)\n\n", detail.Name, detail.Type)

	if detail.Now != "" {
		fmt.Printf("当前选中: %s\n\n", detail.Now)
	}

	fmt.Printf("共 %d 个节点:\n\n", len(detail.All))

	for i, nodeName := range detail.All {
		marker := "  "
		if nodeName == detail.Now {
			marker = "▸ "
		}
		fmt.Printf("  %s%3d. %s\n", marker, i+1, nodeName)
	}

	return nil
}

func runNodesUse(cmd *cobra.Command, args []string) error {
	nodeName := args[0]
	groupName := "PROXY"
	if len(args) > 1 {
		groupName = args[1]
	}

	client := mihomo.NewClient("http://" + core.DefaultControllerAddr)

	if err := client.SwitchProxy(groupName, nodeName); err != nil {
		return fmt.Errorf("切换节点失败: %w", err)
	}

	fmt.Printf("✅ 代理组 %s 已切换到节点: %s\n", groupName, nodeName)
	return nil
}

func runNodesGroups(cmd *cobra.Command, args []string) error {
	client := mihomo.NewClient("http://" + core.DefaultControllerAddr)

	groups, err := client.GetAllProxyGroups()
	if err != nil {
		return fmt.Errorf("获取代理组列表失败: %w", err)
	}

	if len(groups) == 0 {
		fmt.Println("未找到任何代理组")
		return nil
	}

	fmt.Println("📁 代理组列表")
	fmt.Println()

	for name, group := range groups {
		typeIcon := groupTypeIcon(group.Type)
		fmt.Printf("  %s %s [%s]", typeIcon, name, group.Type)
		if group.Now != "" {
			fmt.Printf(" → %s", group.Now)
		}
		fmt.Printf(" (%d 节点)\n", len(group.All))
	}

	fmt.Println()
	fmt.Println("使用 'clashctl nodes list <组名>' 查看详细节点列表")

	return nil
}

func groupTypeIcon(t string) string {
	switch t {
	case "select":
		return "🔀"
	case "url-test":
		return "⚡"
	case "fallback":
		return "🔄"
	case "load-balance":
		return "⚖️"
	default:
		return "📦"
	}
}
