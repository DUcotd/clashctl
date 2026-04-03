package ui

import (
	"fmt"
	"strings"

	"clashctl/internal/mihomo"
)

const minTerminalWidthForProgressBar = 50

func (m NodeManagerModel) viewGroupSelect() string {
	if m.loading {
		content := m.spinner.View() + " " + m.loadingMsg
		return renderCard("选择代理组", m.feedback, content, renderKeyHelp([][2]string{
			{"请稍候", ""},
			{"q", "退出"},
		}))
	}

	if len(m.groups) == 0 {
		content := EmptyStateIconStyle.Render("📦") + "\n\n"
		content += EmptyStateTextStyle.Render("未找到任何代理组") + "\n"
		content += InfoStyle.Render("请确认 Mihomo 已启动并且有可用的代理组") + "\n"
		return renderCardWithStyle("选择代理组", m.feedback, content, renderKeyHelp([][2]string{
			{"r", "重试"},
			{"Esc", "退出"},
			{"q", "退出"},
		}), BoxWarningStyle)
	}

	displayGroups := m.getDisplayGroups()

	groupSearchBar := ""
	if m.groupSearchInput.Focused() || m.groupSearchQuery != "" {
		if m.groupSearchInput.Focused() {
			groupSearchBar = m.groupSearchInput.View() + "\n"
		} else {
			groupSearchBar = InfoStyle.Render("搜索: "+m.groupSearchQuery+" (按 / 重新搜索)") + "\n"
		}
		if len(displayGroups) == 0 {
			return renderCardWithStyle("选择代理组", m.feedback, groupSearchBar+WarningStyle.Render("未找到匹配的代理组"), renderKeyHelp([][2]string{
				{"Esc", "清除搜索"},
				{"r", "刷新"},
				{"q", "退出"},
			}), BoxWarningStyle)
		}
	}

	var list strings.Builder
	for i, group := range displayGroups {
		line := fmt.Sprintf("%s %s", groupIcon(group.Type), group.Name)
		if group.Now != "" {
			line += InfoStyle.Render(fmt.Sprintf(" → %s", group.Now))
		}
		line += InfoStyle.Render(fmt.Sprintf(" (%d)", group.NodeCount))

		if i == m.groupIndex {
			list.WriteString(SelectedBarStyle.Render(line) + "\n")
		} else {
			list.WriteString(UnselectedStyle.Render("  "+line) + "\n")
		}
	}

	if m.vpReady {
		m.vp.SetContent(groupSearchBar + list.String())
		m.followSelected(m.groupIndex)

		header := fmt.Sprintf("选择代理组 (%d)", len(m.groups))
		if m.groupSearchQuery != "" {
			header = fmt.Sprintf("选择代理组 (搜索: %d/%d)", len(displayGroups), len(m.groups))
		}
		legend := BadgeStyle.Render("🔀 select") + " 选择  " +
			BadgeStyle.Render("⚡ url-test") + " 测试  " +
			BadgeStyle.Render("🔄 fallback") + " 故障转移"
		footer := renderKeyHelp([][2]string{
			{"↑/↓", "选择"},
			{"Enter", "查看节点"},
			{"/", "搜索"},
			{"r", "刷新"},
			{"?", "帮助"},
			{"Esc", "退出"},
		})

		return m.renderSelectablePage(header, groupSearchBar+list.String(), legend+"\n"+footer, m.groupIndex)
	}

	return renderCard("选择代理组", m.feedback, groupSearchBar+list.String(), renderKeyHelp([][2]string{
		{"↑/↓", "选择"},
		{"Enter", "查看节点"},
		{"/", "搜索"},
		{"r", "刷新"},
		{"Esc", "退出"},
	}))
}

func (m NodeManagerModel) viewNodeSelect() string {
	if m.loading {
		content := m.spinner.View() + " " + m.loadingMsg
		return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, content, renderKeyHelp([][2]string{
			{"请稍候", ""},
			{"q", "退出"},
		}))
	}

	if m.testing {
		progress := ""
		if m.testTotal > 0 {
			if m.width >= minTerminalWidthForProgressBar {
				progress = " " + renderProgressBar(m.testDone, m.testTotal, m.width/3)
			} else {
				progress = ProgressTextStyle.Render(fmt.Sprintf(" %d/%d", m.testDone, m.testTotal))
			}
		}
		content := m.spinner.View() + fmt.Sprintf(" 正在测试延迟...%s", progress)
		return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, content, renderKeyHelp([][2]string{
			{"请稍候", ""},
			{"q", "退出"},
		}))
	}

	displayNodes := m.getDisplayNodes()

	if len(m.nodes) == 0 {
		content := EmptyStateIconStyle.Render("📦") + "\n\n"
		content += EmptyStateTextStyle.Render("该组没有可用节点") + "\n"
		return renderCardWithStyle(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, content, renderKeyHelp([][2]string{
			{"r", "重试"},
			{"Esc", "返回"},
			{"q", "退出"},
		}), BoxWarningStyle)
	}

	searchBar := ""
	if m.searchInput.Focused() || m.searchQuery != "" {
		if m.searchInput.Focused() {
			searchBar = m.searchInput.View() + "\n"
		} else {
			searchBar = InfoStyle.Render("搜索: "+m.searchQuery+" (按 / 重新搜索)") + "\n"
		}
		if len(displayNodes) == 0 {
			return renderCardWithStyle(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, searchBar+WarningStyle.Render("未找到匹配的节点"), renderKeyHelp([][2]string{
				{"Esc", "清除搜索"},
				{"r", "刷新"},
				{"q", "退出"},
			}), BoxWarningStyle)
		}
	}

	sortInfo := ""
	if m.sortMode != NodeSortDefault {
		sortInfo = " [" + m.sortMode.Label() + "排序]"
	}

	var list strings.Builder
	for i, node := range displayNodes {
		marker := "  "
		if node.Selected {
			marker = ActiveMarkerStyle.Render("✓ ")
		}

		line := marker + node.Name
		if node.Protocol != "" {
			line += " " + protocolBadge(node.Protocol)
		}
		if node.Delay != 0 {
			line += " " + delayStyle(node.Delay).Render(mihomo.FormatDelay(node.Delay))
		}

		if i == m.nodeIndex {
			list.WriteString(SelectedBarStyle.Render(strings.TrimSpace(line)) + "\n")
		} else {
			list.WriteString(UnselectedStyle.Render("  "+strings.TrimSpace(line)) + "\n")
		}
	}

	if m.vpReady {
		m.vp.SetContent(searchBar + list.String())
		m.followSelected(m.nodeIndex)

		header := fmt.Sprintf("代理组: %s (%d 节点)", m.selectedGroup, len(m.nodes))
		if m.searchQuery != "" {
			header = fmt.Sprintf("代理组: %s (搜索: %d/%d 节点)", m.selectedGroup, len(displayNodes), len(m.nodes))
		}
		footer := renderKeyHelp([][2]string{
			{"↑/↓", "选择"},
			{"Enter", "切换"},
			{"t", "测速"},
			{"s", "排序"},
			{"/", "搜索"},
			{"i", "详情"},
			{"g", "跳到选中"},
			{"c", "复制"},
			{"?", "帮助"},
			{"Esc", "返回"},
		})

		return m.renderSelectablePage(header+sortInfo, searchBar+list.String(), footer, m.nodeIndex)
	}

	return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup)+sortInfo, m.feedback, searchBar+list.String(), renderKeyHelp([][2]string{
		{"↑/↓", "选择"},
		{"Enter", "切换"},
		{"t", "测速"},
		{"s", "排序"},
		{"/", "搜索"},
		{"i", "详情"},
		{"g", "跳到选中"},
		{"c", "复制"},
		{"r", "刷新"},
		{"Esc", "返回"},
	}))
}

func (m NodeManagerModel) renderStaticCard(header, body, footer string) string {
	return renderStaticCard(m.viewportState, m.screen, m.baseViewportSize, header, m.feedback, body, footer)
}

func (m NodeManagerModel) renderScrollablePage(header, body, footer string) string {
	return renderScrollablePage(m.viewportState, m.screen, m.baseViewportSize, header, m.feedback, body, footer)
}

func (m NodeManagerModel) renderSelectablePage(header, body, footer string, selectedIndex int) string {
	return renderSelectablePage(m.viewportState, m.screen, m.baseViewportSize, header, m.feedback, body, footer, selectedIndex)
}

func (m NodeManagerModel) viewHelp() string {
	var help strings.Builder
	help.WriteString(HeaderBarStyle.Render("快捷键帮助"))
	help.WriteString("\n\n")

	help.WriteString(HelpSectionStyle.Render("全局操作:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("q", "退出") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Ctrl+C", "退出") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("?", "显示/隐藏此帮助") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("代理组列表:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("↑/↓", "上下选择") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("j/k", "上下选择") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("PgUp/PgDown", "快速翻页") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Home/End", "跳到首/尾") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Enter", "查看节点") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("r", "刷新列表") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("/", "搜索代理组") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("节点列表:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("↑/↓", "上下选择") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("j/k", "上下选择") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("PgUp/PgDown", "快速翻页") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Home/End", "跳到首/尾") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Enter", "切换节点 (需确认)") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("t", "测试所有节点延迟") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("r", "刷新列表") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("/", "搜索/过滤节点") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("s", "切换排序 (默认/延迟/名称/协议)") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("i", "查看节点详情") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("g/*", "跳到当前选中节点") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("c", "复制节点名到剪贴板") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Esc", "返回代理组列表") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("搜索模式:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Enter", "确认搜索") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Esc", "退出搜索") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("确认对话框:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("y", "确认操作") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Enter", "确认操作") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("n", "取消操作") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Esc", "取消操作") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("节点详情:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("c", "复制节点名") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("i/Enter/Esc", "关闭详情") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpStyle.Render("按任意键返回"))
	return BoxStyle.Render(help.String())
}

func (m NodeManagerModel) viewQuitConfirm() string {
	var d strings.Builder
	d.WriteString(HeaderBarStyle.Render("确认退出"))
	d.WriteString("\n\n")
	d.WriteString(ConfirmStyle.Render("确定要退出节点管理吗？"))
	d.WriteString("\n\n")
	d.WriteString(renderKeyHelp([][2]string{
		{"y", "确认退出"},
		{"Enter", "确认退出"},
		{"n", "取消"},
		{"Esc", "取消"},
	}))
	return BoxWarningStyle.Render(d.String())
}

func (m NodeManagerModel) viewConfirmDialog() string {
	var dialog strings.Builder
	dialog.WriteString(HeaderBarStyle.Render("确认操作"))
	dialog.WriteString("\n\n")

	switch m.confirmAction {
	case ConfirmSwitchNode:
		dialog.WriteString(ConfirmStyle.Render("确定要切换到以下节点吗？"))
		dialog.WriteString("\n\n")
		dialog.WriteString(SelectedBarStyle.Render(m.confirmTarget))
		dialog.WriteString("\n")
	}

	dialog.WriteString("\n")
	dialog.WriteString(renderKeyHelp([][2]string{
		{"y", "确认"},
		{"Enter", "确认"},
		{"n", "取消"},
		{"Esc", "取消"},
	}))
	return BoxWarningStyle.Render(dialog.String())
}

func (m NodeManagerModel) viewNodeDetail() string {
	displayNodes := m.getDisplayNodes()
	if m.detailNodeIndex < 0 || m.detailNodeIndex >= len(displayNodes) {
		return renderCard("节点详情", m.feedback, WarningStyle.Render("无效的节点索引"), renderKeyHelp([][2]string{
			{"Esc", "返回"},
		}))
	}

	node := displayNodes[m.detailNodeIndex]

	var detail strings.Builder
	detail.WriteString(HeaderBarStyle.Render("节点详情"))
	detail.WriteString("\n\n")

	detail.WriteString(DetailLabelStyle.Render("名称:") + "\n")
	detail.WriteString(DetailValueStyle.Render("  "+node.Name) + "\n")
	detail.WriteString("\n")

	if node.Protocol != "" {
		detail.WriteString(DetailLabelStyle.Render("协议:") + "\n")
		detail.WriteString("  " + protocolBadge(node.Protocol) + "\n")
		detail.WriteString("\n")
	}

	if node.Delay != 0 {
		detail.WriteString(DetailLabelStyle.Render("延迟:") + "\n")
		detail.WriteString("  " + delayStyle(node.Delay).Render(mihomo.FormatDelay(node.Delay)) + "\n")
		detail.WriteString("\n")
	}

	if node.Selected {
		detail.WriteString(ActiveMarkerStyle.Render("  ✓ 当前已选中") + "\n")
		detail.WriteString("\n")
	}

	detail.WriteString(InfoStyle.Render("代理组: "+m.selectedGroup) + "\n")

	detail.WriteString("\n")
	detail.WriteString(renderKeyHelp([][2]string{
		{"c", "复制节点名"},
		{"i/Enter/Esc", "关闭"},
	}))
	return BoxStyle.Render(detail.String())
}

func (m NodeManagerModel) renderStatusBar() string {
	if m.width < 4 {
		return ""
	}

	barWidth := m.width - 2

	statusLine := " "
	switch m.screen {
	case ScreenGroupSelect:
		displayGroups := m.getDisplayGroups()
		if len(displayGroups) > 0 && m.groupIndex < len(displayGroups) {
			statusLine += fmt.Sprintf("代理组: %d 个 │ 当前: %s", len(displayGroups), displayGroups[m.groupIndex].Name)
		} else {
			statusLine += fmt.Sprintf("代理组: %d 个", len(displayGroups))
		}
		if m.groupSearchQuery != "" {
			statusLine += fmt.Sprintf(" (搜索: %d/%d)", len(displayGroups), len(m.groups))
		}
	case ScreenNodeSelect:
		displayCount := len(m.getDisplayNodes())
		totalCount := len(m.nodes)
		if displayCount > 0 {
			statusLine += fmt.Sprintf("节点: %d/%d", m.nodeIndex+1, displayCount)
		} else {
			statusLine += fmt.Sprintf("节点: 0 个")
		}
		if m.searchQuery != "" {
			statusLine += fmt.Sprintf(" (搜索: %s, 共 %d 个节点)", m.searchQuery, totalCount)
		}
		if m.sortMode != NodeSortDefault {
			statusLine += fmt.Sprintf(" [%s排序]", m.sortMode.Label())
		}
	}
	if m.testing {
		statusLine += fmt.Sprintf(" │ 测速中: %d/%d", m.testDone, m.testTotal)
	}
	if len(statusLine) < barWidth {
		statusLine += strings.Repeat(" ", barWidth-len(statusLine))
	}

	helpLine := " "
	if m.screen == ScreenNodeSelect {
		helpLine += "↑/↓ 选择 │ Enter 切换 │ t 测速 │ s 排序 │ / 搜索 │ i 详情 │ g 跳到选中 │ c 复制 │ ? 帮助 │ Esc 返回"
	} else {
		helpLine += "↑/↓ 选择 │ Enter 查看 │ / 搜索 │ r 刷新 │ ? 帮助"
	}
	if len(helpLine) < barWidth {
		helpLine += strings.Repeat(" ", barWidth-len(helpLine))
	}

	topBorder := StatusBarStyle.Render("┌" + strings.Repeat("─", barWidth) + "┐")
	statusRender := StatusBarStyle.Render(statusLine)
	helpRender := StatusHelpStyle.Render(helpLine)
	bottomBorder := StatusBarStyle.Render("└" + strings.Repeat("─", barWidth) + "┘")

	return topBorder + "\n" + statusRender + "\n" + helpRender + "\n" + bottomBorder
}
