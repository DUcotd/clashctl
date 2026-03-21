package ui

import (
	"fmt"
	"strings"

	"clashctl/internal/mihomo"
)

func (m NodeManagerModel) viewGroupSelect() string {
	if m.loading {
		content := m.spinner.View() + " " + m.loadingMsg
		return BoxStyle.Render(content)
	}

	errorBlock := ""
	if m.loadError != "" {
		errorBlock = ErrorStyle.Render(m.loadError) + "\n\n"
	}

	if len(m.groups) == 0 {
		content := errorBlock + WarningStyle.Render("未找到任何代理组") + "\n\n"
		content += InfoStyle.Render("请确认 Mihomo 已启动并且有可用的代理组") + "\n"
		content += HelpStyle.Render("r 重试 │ Esc 退出")
		return BoxStyle.Render(content)
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

		header := HeaderStyle.Render(fmt.Sprintf("选择代理组 (%d)", len(m.groups)))
		legend := BadgeStyle.Render("🔀 select") + " 选择  " +
			BadgeStyle.Render("⚡ url-test") + " 测试  " +
			BadgeStyle.Render("🔄 fallback") + " 故障转移"
		footer := HelpStyle.Render("↑/↓ 选择 │ Enter 查看节点 │ r 刷新 │ Esc 退出")

		return m.renderSelectablePage(header, errorBlock+list.String(), legend+"\n"+footer, m.groupIndex)
	}

	return BoxStyle.Render(HeaderStyle.Render("选择代理组") + "\n\n" + errorBlock + list.String())
}

func (m NodeManagerModel) viewNodeSelect() string {
	if m.loading {
		content := m.spinner.View() + " " + m.loadingMsg
		return BoxStyle.Render(content)
	}

	if m.testing {
		content := m.spinner.View() + fmt.Sprintf(" 正在测试延迟... (%d/%d)", m.testDone, m.testTotal)
		return BoxStyle.Render(content)
	}

	if len(m.nodes) == 0 {
		content := HeaderStyle.Render(fmt.Sprintf("代理组: %s", m.selectedGroup)) + "\n\n"
		if m.loadError != "" {
			content += ErrorStyle.Render(m.loadError) + "\n\n"
		}
		content += WarningStyle.Render("该组没有可用节点") + "\n"
		content += HelpStyle.Render("r 重试 │ Esc 返回")
		return BoxStyle.Render(content)
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

		header := HeaderStyle.Render(fmt.Sprintf("代理组: %s (%d 节点)", m.selectedGroup, len(m.nodes)))
		resultLine := ""
		if m.switchResult != "" {
			resultLine = "\n" + m.switchResult
		}
		if m.loadError != "" {
			resultLine += "\n" + ErrorStyle.Render(m.loadError)
		}
		footer := HelpStyle.Render("↑/↓ 选择 │ Enter 切换 │ t 测试延迟 │ r 刷新 │ Esc 返回")

		return m.renderSelectablePage(header, list.String()+resultLine, footer, m.nodeIndex)
	}

	content := HeaderStyle.Render(fmt.Sprintf("代理组: %s", m.selectedGroup)) + "\n\n" + list.String()
	if m.switchResult != "" {
		content += "\n" + m.switchResult + "\n"
	}
	if m.loadError != "" {
		content += "\n" + ErrorStyle.Render(m.loadError) + "\n"
	}
	content += "\n" + HelpStyle.Render("↑/↓ 选择 │ Enter 切换 │ t 测试延迟 │ r 刷新 │ Esc 返回")
	return BoxStyle.Render(content)
}

func (m NodeManagerModel) renderStaticCard(header, body, footer string) string {
	content := HeaderStyle.Render(header) + "\n\n" + body
	if footer != "" {
		content += "\n\n" + HelpStyle.Render(footer)
	}
	return BoxStyle.Render(content)
}

func (m NodeManagerModel) renderScrollablePage(header, body, footer string) string {
	if !m.vpReady {
		return m.renderStaticCard(header, body, footer)
	}
	innerWidth, innerHeight := m.baseViewportSize()
	vp := m.vp
	vp.Width = innerWidth
	headerBlock := HeaderStyle.Render(header)
	footerBlock := HelpStyle.Render(footer)
	contentHeight := max(5, innerHeight-lineCount(headerBlock)-lineCount(footerBlock)-2)
	vp.Height = contentHeight
	vp.SetContent(body)
	if off, ok := m.screenOffsets[m.screen]; ok {
		vp.SetYOffset(off)
	}
	scrollHint := ""
	if vp.TotalLineCount() > vp.Height {
		scrollHint = InfoStyle.Render(fmt.Sprintf("位置 %d/%d", min(vp.YOffset+vp.Height, vp.TotalLineCount()), vp.TotalLineCount())) + "\n"
	}
	content := headerBlock + "\n\n" + vp.View()
	if footer != "" {
		content += "\n" + scrollHint + footerBlock
	}
	return BoxStyle.Render(content)
}

func (m NodeManagerModel) renderSelectablePage(header, body, footer string, selectedIndex int) string {
	if !m.vpReady {
		return m.renderStaticCard(header, body, footer)
	}
	innerWidth, innerHeight := m.baseViewportSize()
	vp := m.vp
	vp.Width = innerWidth
	headerBlock := HeaderStyle.Render(header)
	footerBlock := HelpStyle.Render(footer)
	contentHeight := max(5, innerHeight-lineCount(headerBlock)-lineCount(footerBlock)-2)
	vp.Height = contentHeight
	vp.SetContent(body)
	if selectedIndex < vp.YOffset {
		vp.SetYOffset(selectedIndex)
	} else if selectedIndex >= vp.YOffset+vp.Height {
		vp.SetYOffset(selectedIndex - vp.Height + 1)
	} else if off, ok := m.screenOffsets[m.screen]; ok {
		vp.SetYOffset(off)
	}
	scrollHint := ""
	if vp.TotalLineCount() > vp.Height {
		scrollHint = InfoStyle.Render(fmt.Sprintf("位置 %d/%d", min(vp.YOffset+vp.Height, vp.TotalLineCount()), vp.TotalLineCount())) + "\n"
	}
	content := headerBlock + "\n\n" + vp.View()
	if footer != "" {
		content += "\n" + scrollHint + footerBlock
	}
	return BoxStyle.Render(content)
}
