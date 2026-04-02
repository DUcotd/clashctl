package ui

import (
	"fmt"
	"strings"

	"clashctl/internal/mihomo"
)

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

	var list strings.Builder
	for i, group := range m.groups {
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
		m.vp.SetContent(list.String())
		selectedY := m.groupIndex
		if selectedY < m.vp.YOffset {
			m.vp.SetYOffset(selectedY)
		} else if selectedY >= m.vp.YOffset+m.vp.Height {
			m.vp.SetYOffset(selectedY - m.vp.Height + 1)
		}

		header := fmt.Sprintf("选择代理组 (%d)", len(m.groups))
		legend := BadgeStyle.Render("🔀 select") + " 选择  " +
			BadgeStyle.Render("⚡ url-test") + " 测试  " +
			BadgeStyle.Render("🔄 fallback") + " 故障转移"
		footer := "↑/↓ 选择 │ Enter 查看节点 │ r 刷新 │ Esc 退出 │ q 退出"

		return m.renderSelectablePage(header, list.String(), legend+"\n"+footer, m.groupIndex)
	}

	return renderCard("选择代理组", m.feedback, list.String(), "↑/↓ 选择 │ Enter 查看节点 │ r 刷新 │ Esc 退出 │ q 退出")
}

func (m NodeManagerModel) viewNodeSelect() string {
	if m.loading {
		content := m.spinner.View() + " " + m.loadingMsg
		return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, content, "请稍候... │ q 退出")
	}

	if m.testing {
		content := m.spinner.View() + fmt.Sprintf(" 正在测试延迟... (%d/%d)", m.testDone, m.testTotal)
		return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, content, "请稍候... │ q 退出")
	}

	if len(m.nodes) == 0 {
		return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, WarningStyle.Render("该组没有可用节点"), "r 重试 │ Esc 返回 │ q 退出")
	}

	var list strings.Builder
	for i, node := range m.nodes {
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
		m.vp.SetContent(list.String())
		selectedY := m.nodeIndex
		if selectedY < m.vp.YOffset {
			m.vp.SetYOffset(selectedY)
		} else if selectedY >= m.vp.YOffset+m.vp.Height {
			m.vp.SetYOffset(selectedY - m.vp.Height + 1)
		}

		header := fmt.Sprintf("代理组: %s (%d 节点)", m.selectedGroup, len(m.nodes))
		footer := "↑/↓ 选择 │ Enter 切换 │ t 测试延迟 │ r 刷新 │ Esc 返回 │ q 退出"

		return m.renderSelectablePage(header, list.String(), footer, m.nodeIndex)
	}

	return renderCard(fmt.Sprintf("代理组: %s", m.selectedGroup), m.feedback, list.String(), "↑/↓ 选择 │ Enter 切换 │ t 测试延迟 │ r 刷新 │ Esc 返回 │ q 退出")
}

func (m NodeManagerModel) renderStaticCard(header, body, footer string) string {
	return renderCard(header, m.feedback, body, footer)
}

func (m NodeManagerModel) renderScrollablePage(header, body, footer string) string {
	return renderScrollableCard(m.viewportState, m.screen, m.baseViewportSize, header, m.feedback, body, footer)
}

func (m NodeManagerModel) renderSelectablePage(header, body, footer string, selectedIndex int) string {
	return renderSelectableCard(m.viewportState, m.screen, m.baseViewportSize, header, m.feedback, body, footer, selectedIndex)
}
