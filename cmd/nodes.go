package cmd

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"clashctl/internal/mihomo"
	nodeops "clashctl/internal/nodes"
	"clashctl/internal/ui"
)

var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "管理代理节点（默认进入 TUI）",
	Long: `管理代理节点。

直接执行 'clashctl nodes' 会进入交互式节点管理界面；
需要脚本化操作时，可继续使用 list / use / groups / test 子命令。`,
	RunE: runTUINodes,
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

var (
	nodesListJSON       bool
	nodesUseJSON        bool
	nodesGroupsJSON     bool
	nodesTestAllGroups  bool
	nodesTestConcurrent int
	nodesTestJSON       bool
)

type nodesListEntry struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type nodesListReport struct {
	Group   string           `json:"group"`
	Type    string           `json:"type"`
	Current string           `json:"current,omitempty"`
	Count   int              `json:"count"`
	Nodes   []nodesListEntry `json:"nodes"`
}

type nodesUseReport struct {
	Group   string `json:"group"`
	Node    string `json:"node"`
	Success bool   `json:"success"`
}

type nodesGroupSummary struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Current   string `json:"current,omitempty"`
	NodeCount int    `json:"node_count"`
}

type nodesGroupsReport struct {
	Groups []nodesGroupSummary `json:"groups"`
}

type nodesTestReport struct {
	Concurrency int                        `json:"concurrency"`
	Groups      []*mihomo.ProxyGroupDetail `json:"groups"`
}

var nodesTestCmd = &cobra.Command{
	Use:   "test [代理组名]",
	Short: "测试代理组节点延迟",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runNodesTest,
}

func init() {
	nodesListCmd.Flags().BoolVar(&nodesListJSON, "json", false, "以 JSON 输出节点列表")
	nodesUseCmd.Flags().BoolVar(&nodesUseJSON, "json", false, "以 JSON 输出切换结果")
	nodesGroupsCmd.Flags().BoolVar(&nodesGroupsJSON, "json", false, "以 JSON 输出代理组列表")
	nodesCmd.AddCommand(nodesListCmd)
	nodesCmd.AddCommand(nodesUseCmd)
	nodesCmd.AddCommand(nodesGroupsCmd)
	nodesTestCmd.Flags().BoolVar(&nodesTestAllGroups, "all-groups", false, "遍历所有代理组并测速")
	nodesTestCmd.Flags().IntVar(&nodesTestConcurrent, "concurrency", 10, "并发测速数")
	nodesTestCmd.Flags().BoolVar(&nodesTestJSON, "json", false, "以 JSON 输出测速结果")
	nodesCmd.AddCommand(nodesTestCmd)
	rootCmd.AddCommand(nodesCmd)
}

func runTUINodes(cmd *cobra.Command, args []string) error {
	appCfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	service := nodeops.NewServiceWithSecret(appCfg.ControllerSecret)
	if err := service.CheckConnection(appCfg.ControllerAddr); err != nil {
		return fmt.Errorf("Controller API 不可达: %w\n请先运行 'clashctl service start' 或完成 'clashctl init'", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	manager := ui.NewNodeManager(appCfg)
	p := tea.NewProgram(manager, tea.WithAltScreen())

	go func() {
		<-sigCh
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("节点管理界面运行出错: %w", err)
	}
	return nil
}

func runNodesList(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	groupName := "PROXY"
	if len(args) > 0 {
		groupName = args[0]
	}

	service := nodeops.NewServiceWithSecret(cfg.ControllerSecret)
	detail, err := service.GetGroup(cfg.ControllerAddr, groupName)
	if err != nil {
		return fmt.Errorf("获取节点列表失败: %w", err)
	}
	if nodesListJSON {
		return writeJSON(buildNodesListReport(detail))
	}

	fmt.Printf("📡 代理组: %s (%s)\n\n", detail.Name, mihomo.NormalizeProxyType(detail.Type))

	if detail.Current != "" {
		fmt.Printf("当前选中: %s\n\n", detail.Current)
	}

	fmt.Printf("共 %d 个节点:\n\n", len(detail.Nodes))

	for i, node := range detail.Nodes {
		marker := "  "
		if node.Selected {
			marker = "▸ "
		}
		fmt.Printf("  %s%3d. %s\n", marker, i+1, node.Name)
	}

	return nil
}

func runNodesUse(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	nodeName := args[0]
	groupName := "PROXY"
	if len(args) > 1 {
		groupName = args[1]
	}

	service := nodeops.NewServiceWithSecret(cfg.ControllerSecret)
	if err := service.SwitchNode(cfg.ControllerAddr, groupName, nodeName); err != nil {
		return fmt.Errorf("切换节点失败: %w", err)
	}
	if nodesUseJSON {
		return writeJSON(&nodesUseReport{Group: groupName, Node: nodeName, Success: true})
	}

	fmt.Printf("✅ 代理组 %s 已切换到节点: %s\n", groupName, nodeName)
	return nil
}

func runNodesGroups(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	service := nodeops.NewServiceWithSecret(cfg.ControllerSecret)
	groups, err := service.ListGroups(cfg.ControllerAddr)
	if err != nil {
		return fmt.Errorf("获取代理组列表失败: %w", err)
	}

	if len(groups) == 0 {
		if nodesGroupsJSON {
			return writeJSON(&nodesGroupsReport{Groups: []nodesGroupSummary{}})
		}
		fmt.Println("未找到任何代理组")
		return nil
	}
	if nodesGroupsJSON {
		return writeJSON(buildNodesGroupsReport(groups))
	}

	fmt.Println("📁 代理组列表")
	fmt.Println()

	for _, group := range groups {
		typ := mihomo.NormalizeProxyType(group.Type)
		typeIcon := mihomo.GroupTypeIcon(typ)
		fmt.Printf("  %s %s [%s]", typeIcon, group.Name, typ)
		if group.Current != "" {
			fmt.Printf(" → %s", group.Current)
		}
		fmt.Printf(" (%d 节点)\n", group.NodeCount)
	}

	fmt.Println()
	fmt.Println("使用 'clashctl nodes list <组名>' 查看详细节点列表")

	return nil
}

func runNodesTest(cmd *cobra.Command, args []string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}
	if nodesTestConcurrent <= 0 {
		return fmt.Errorf("--concurrency 必须大于 0")
	}

	service := nodeops.NewServiceWithSecret(cfg.ControllerSecret)

	groupNames := []string{"PROXY"}
	if len(args) > 0 {
		groupNames = []string{args[0]}
	}
	if nodesTestAllGroups {
		groups, err := service.ListGroups(cfg.ControllerAddr)
		if err != nil {
			return fmt.Errorf("获取代理组列表失败: %w", err)
		}
		groupNames = make([]string, 0, len(groups))
		for _, group := range groups {
			groupNames = append(groupNames, group.Name)
		}
	}

	details := make([]*mihomo.ProxyGroupDetail, 0, len(groupNames))
	for _, groupName := range groupNames {
		detail, err := service.TestGroup(cfg.ControllerAddr, groupName, nodesTestConcurrent)
		if err != nil {
			return fmt.Errorf("测速代理组 %s 失败: %w", groupName, err)
		}
		details = append(details, detail)
	}

	if nodesTestJSON {
		return writeJSON(buildNodesTestReport(nodesTestConcurrent, details))
	}

	for i, detail := range details {
		if i > 0 {
			fmt.Println()
		}
		printProxyGroupLatency(os.Stdout, detail)
	}

	return nil
}

func buildNodesTestReport(concurrency int, details []*mihomo.ProxyGroupDetail) *nodesTestReport {
	groups := make([]*mihomo.ProxyGroupDetail, 0, len(details))
	groups = append(groups, details...)
	return &nodesTestReport{Concurrency: concurrency, Groups: groups}
}

func buildNodesListReport(detail *nodeops.GroupDetail) *nodesListReport {
	report := &nodesListReport{
		Group:   detail.Name,
		Type:    mihomo.NormalizeProxyType(detail.Type),
		Current: detail.Current,
		Count:   len(detail.Nodes),
		Nodes:   make([]nodesListEntry, 0, len(detail.Nodes)),
	}
	for i, node := range detail.Nodes {
		report.Nodes = append(report.Nodes, nodesListEntry{
			Index:    i + 1,
			Name:     node.Name,
			Selected: node.Selected,
		})
	}
	return report
}

func buildNodesGroupsReport(groups []nodeops.GroupSummary) *nodesGroupsReport {
	report := &nodesGroupsReport{Groups: make([]nodesGroupSummary, 0, len(groups))}
	for _, group := range groups {
		report.Groups = append(report.Groups, nodesGroupSummary{
			Name:      group.Name,
			Type:      mihomo.NormalizeProxyType(group.Type),
			Current:   group.Current,
			NodeCount: group.NodeCount,
		})
	}
	return report
}

func printProxyGroupLatency(w io.Writer, detail *mihomo.ProxyGroupDetail) {
	fmt.Fprintf(w, "📡 代理组: %s (%s)\n\n", detail.Name, mihomo.NormalizeProxyType(detail.Type))
	if detail.Now != "" {
		fmt.Fprintf(w, "当前选中: %s\n", detail.Now)
	}
	fmt.Fprintf(w, "测速完成: %d 个节点\n\n", len(detail.Nodes))

	for i, node := range detail.Nodes {
		marker := "  "
		if node.Selected {
			marker = "▸ "
		}
		fmt.Fprintf(w, "  %s%3d. %-40s %s\n", marker, i+1, node.Name, mihomo.FormatDelay(node.Delay))
	}
}
