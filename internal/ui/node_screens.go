package ui

import (
	"fmt"
	"strings"

	"clashctl/internal/mihomo"
)

// minTerminalWidthForProgressBar is the minimum terminal width (columns) at
// which the animated progress bar is rendered. Below this threshold only the
// spinner and percentage text are shown to avoid layout overflow.
const minTerminalWidthForProgressBar = 50

func (m NodeManagerModel) viewGroupSelect() string {
	if m.loading {
		content := m.spinner.View() + " " + m.loadingMsg
		return renderCard("选择代理组", m.feedback, content, "请稍候... │ q 退出")
	}

	if len(m.groups) == 0 {
		content := WarningStyle.Render("未找到任何代理组") + "\n\n"
		content += InfoStyle.Render("请确认 Mihomo 已启动并且有可用的代理组") + "\n"
		return renderCard("选择代理组", m.feedback, content, "r 重试 │ Esc 退出 │ q 退出")
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
			return renderCard("选择代理组", m.feedback, groupSearchBar+WarningStyle.Render("未找到匹配的代理组"), "Esc 清除搜索 │ r 刷新 │ q 退出")
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
			list.WriteString(SelectedStyle.Render("▸ "+line) + "\n")
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
		footer := "↑/↓ 选择 │ Enter 查看节点 │ / 搜索 │ r 刷新 │ ? 帮助 │ Esc 退出"

		return m.renderSelectablePage(header, groupSearchBar+list.String(), legend+"\n"+footer, m.groupIndex)
	}

	return renderCard("选择代理组", m.feedback, groupSearchBar+list.String(), "↑/↓ 选择 │ Enter 查看节点 │ / 搜索 │ r 刷新 │ Esc 退出")
}

func (m NodeManagerModel) viewNodeSelect() string {
	if m.loading {
		content := m.spinner.View() + " " + m.loadingMsg
		return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, content, "请稍候... │ q 退出")
	}

	if m.testing {
		progress := ""
		if m.testTotal > 0 && m.width >= minTerminalWidthForProgressBar {
			pct := float64(m.testDone) / float64(m.testTotal)
			barWidth := max(10, min(40, m.width/3))
			filled := int(pct * float64(barWidth))
			if filled > barWidth {
				filled = barWidth
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			progress = fmt.Sprintf(" [%s] %d/%d (%.0f%%)", bar, m.testDone, m.testTotal, pct*100)
		}
		content := m.spinner.View() + fmt.Sprintf(" 正在测试延迟...%s", progress)
		return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, content, "请稍候... │ q 退出")
	}

	displayNodes := m.getDisplayNodes()

	if len(m.nodes) == 0 {
		return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, WarningStyle.Render("该组没有可用节点"), "r 重试 │ Esc 返回 │ q 退出")
	}

	searchBar := ""
	if m.searchInput.Focused() || m.searchQuery != "" {
		if m.searchInput.Focused() {
			searchBar = m.searchInput.View() + "\n"
		} else {
			searchBar = InfoStyle.Render("搜索: "+m.searchQuery+" (按 / 重新搜索)") + "\n"
		}
		if len(displayNodes) == 0 {
			return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, searchBar+WarningStyle.Render("未找到匹配的节点"), "Esc 清除搜索 │ r 刷新 │ q 退出")
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
			list.WriteString(SelectedStyle.Render("▸ "+strings.TrimSpace(line)) + "\n")
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
		footer := "↑/↓ 选择 │ Enter 切换 │ t 测速 │ s 排序 │ / 搜索 │ i 详情 │ g 跳到选中 │ c 复制 │ ? 帮助 │ Esc 返回"

		return m.renderSelectablePage(header+sortInfo, searchBar+list.String(), footer, m.nodeIndex)
	}

	return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup)+sortInfo, m.feedback, searchBar+list.String(), "↑/↓ 选择 │ Enter 切换 │ t 测速 │ s 排序 │ / 搜索 │ i 详情 │ g 跳到选中 │ c 复制 │ r 刷新 │ Esc 返回")
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
	help.WriteString(HeaderStyle.Render("快捷键帮助"))
	help.WriteString("\n\n")

	help.WriteString(TextStyle.Render("全局操作:") + "\n")
	help.WriteString(InfoStyle.Render("  q / Ctrl+C    退出") + "\n")
	help.WriteString(InfoStyle.Render("  ?             显示/隐藏此帮助") + "\n")
	help.WriteString("\n")

	help.WriteString(TextStyle.Render("代理组列表:") + "\n")
	help.WriteString(InfoStyle.Render("  ↑/↓ 或 j/k    上下选择") + "\n")
	help.WriteString(InfoStyle.Render("  PgUp/PgDown   快速翻页") + "\n")
	help.WriteString(InfoStyle.Render("  Home/End      跳到首/尾") + "\n")
	help.WriteString(InfoStyle.Render("  Enter         查看节点") + "\n")
	help.WriteString(InfoStyle.Render("  r             刷新列表") + "\n")
	help.WriteString(InfoStyle.Render("  /             搜索代理组") + "\n")
	help.WriteString("\n")

	help.WriteString(TextStyle.Render("节点列表:") + "\n")
	help.WriteString(InfoStyle.Render("  ↑/↓ 或 j/k    上下选择") + "\n")
	help.WriteString(InfoStyle.Render("  PgUp/PgDown   快速翻页") + "\n")
	help.WriteString(InfoStyle.Render("  Home/End      跳到首/尾") + "\n")
	help.WriteString(InfoStyle.Render("  Enter         切换节点 (需确认)") + "\n")
	help.WriteString(InfoStyle.Render("  t             测试所有节点延迟") + "\n")
	help.WriteString(InfoStyle.Render("  r             刷新列表") + "\n")
	help.WriteString(InfoStyle.Render("  /             搜索/过滤节点") + "\n")
	help.WriteString(InfoStyle.Render("  s             切换排序 (默认/延迟/名称/协议)") + "\n")
	help.WriteString(InfoStyle.Render("  i             查看节点详情") + "\n")
	help.WriteString(InfoStyle.Render("  g / *         跳到当前选中节点") + "\n")
	help.WriteString(InfoStyle.Render("  c             复制节点名到剪贴板") + "\n")
	help.WriteString(InfoStyle.Render("  Esc           返回代理组列表") + "\n")
	help.WriteString("\n")

	help.WriteString(TextStyle.Render("搜索模式:") + "\n")
	help.WriteString(InfoStyle.Render("  Enter         确认搜索") + "\n")
	help.WriteString(InfoStyle.Render("  Esc           退出搜索") + "\n")
	help.WriteString("\n")

	help.WriteString(TextStyle.Render("确认对话框:") + "\n")
	help.WriteString(InfoStyle.Render("  y / Enter     确认操作") + "\n")
	help.WriteString(InfoStyle.Render("  n / Esc       取消操作") + "\n")
	help.WriteString("\n")

	help.WriteString(TextStyle.Render("节点详情:") + "\n")
	help.WriteString(InfoStyle.Render("  c             复制节点名") + "\n")
	help.WriteString(InfoStyle.Render("  i / Enter / Esc  关闭详情") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpStyle.Render("按任意键返回"))
	return BoxStyle.Render(help.String())
}

func (m NodeManagerModel) viewQuitConfirm() string {
	var d strings.Builder
	d.WriteString(HeaderStyle.Render("确认退出"))
	d.WriteString("\n\n")
	d.WriteString(WarningStyle.Render("确定要退出节点管理吗？"))
	d.WriteString("\n\n")
	d.WriteString(HelpStyle.Render("y / Enter 确认退出  │  n / Esc 取消"))
	return BoxStyle.Render(d.String())
}

func (m NodeManagerModel) viewConfirmDialog() string {
	var dialog strings.Builder
	dialog.WriteString(HeaderStyle.Render("确认操作"))
	dialog.WriteString("\n\n")

	switch m.confirmAction {
	case ConfirmSwitchNode:
		dialog.WriteString(WarningStyle.Render("确定要切换到以下节点吗？"))
		dialog.WriteString("\n\n")
		dialog.WriteString(TextStyle.Render("  " + m.confirmTarget))
		dialog.WriteString("\n")
	}

	dialog.WriteString("\n")
	dialog.WriteString(HelpStyle.Render("y / Enter 确认  │  n / Esc 取消"))
	return BoxStyle.Render(dialog.String())
}

func (m NodeManagerModel) viewNodeDetail() string {
	displayNodes := m.getDisplayNodes()
	if m.detailNodeIndex < 0 || m.detailNodeIndex >= len(displayNodes) {
		return renderCard("节点详情", m.feedback, WarningStyle.Render("无效的节点索引"), "Esc 返回")
	}

	node := displayNodes[m.detailNodeIndex]

	var detail strings.Builder
	detail.WriteString(HeaderStyle.Render("节点详情"))
	detail.WriteString("\n\n")

	detail.WriteString(TextStyle.Render("名称:") + "\n")
	detail.WriteString(InputStyle.Render("  "+node.Name) + "\n")
	detail.WriteString("\n")

	if node.Protocol != "" {
		detail.WriteString(TextStyle.Render("协议:") + "\n")
		detail.WriteString("  " + protocolBadge(node.Protocol) + "\n")
		detail.WriteString("\n")
	}

	if node.Delay != 0 {
		detail.WriteString(TextStyle.Render("延迟:") + "\n")
		detail.WriteString("  " + delayStyle(node.Delay).Render(mihomo.FormatDelay(node.Delay)) + "\n")
		detail.WriteString("\n")
	}

	if node.Selected {
		detail.WriteString(ActiveMarkerStyle.Render("  ✓ 当前已选中") + "\n")
		detail.WriteString("\n")
	}

	detail.WriteString(InfoStyle.Render("代理组: "+m.selectedGroup) + "\n")

	detail.WriteString("\n")
	detail.WriteString(HelpStyle.Render("c 复制节点名 │ i / Enter / Esc 关闭"))
	return BoxStyle.Render(detail.String())
}

func (m NodeManagerModel) renderStatusBar() string {
	if m.width < 4 {
		return ""
	}

	var bar strings.Builder

	barWidth := m.width - 2
	bar.WriteString("┌")
	bar.WriteString(strings.Repeat("─", barWidth))
	bar.WriteString("┐\n")

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
	bar.WriteString(statusLine)
	bar.WriteString("\n")

	helpLine := " "
	if m.screen == ScreenNodeSelect {
		helpLine += "↑/↓ 选择 │ Enter 切换 │ t 测速 │ s 排序 │ / 搜索 │ i 详情 │ g 跳到选中 │ c 复制 │ ? 帮助 │ Esc 返回"
	} else {
		helpLine += "↑/↓ 选择 │ Enter 查看 │ / 搜索 │ r 刷新 │ ? 帮助"
	}
	if len(helpLine) < barWidth {
		helpLine += strings.Repeat(" ", barWidth-len(helpLine))
	}
	bar.WriteString(helpLine)

	bar.WriteString("\n└")
	bar.WriteString(strings.Repeat("─", barWidth))
	bar.WriteString("┘")

	return bar.String()
}
