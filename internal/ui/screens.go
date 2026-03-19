// Package ui contains screen view rendering.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"clashctl/internal/mihomo"
)

func (m WizardModel) viewWelcome() string {
	body := strings.Join([]string{
		"欢迎使用 clashctl — Mihomo 交互式配置工具",
		"",
		"这个向导将帮助你：",
		"",
		"• 输入机场订阅 URL 或后续改走本地订阅导入",
		"• 选择 TUN / mixed-port 运行模式",
		"• 调整高级设置（可选）",
		"• 自动生成并写入 Mihomo 配置",
		"• 启动 Mihomo 并验证节点是否真正加载",
		"• 进入代理组和节点管理",
		"",
		"如果服务器拉不到订阅，向导会自动提示本地导入静态配置。",
	}, "\n")
	return m.renderScrollablePage("开始使用", body, "Enter 开始 │ Esc / q 退出")
}

func (m WizardModel) viewSubscription() string {
	content := HeaderStyle.Render("输入订阅链接或本地订阅文件") + "\n\n" +
		InfoStyle.Render("推荐直接粘贴订阅 URL；如果服务器拉取失败，也可以填本地文件路径") + "\n" +
		InfoStyle.Render("默认会自动转换成更适合服务器的静态配置，尽量减少后续问题") + "\n\n" +
		m.urlInput.View() + "\n\n" +
		HelpStyle.Render("Enter 下一步 │ Esc 返回")

	return BoxStyle.Render(content)
}

func (m WizardModel) viewMode() string {
	options := []string{"TUN 模式（全局代理，需 root）", "mixed-port 模式（推荐，兼容性好）"}

	var lines []string
	lines = append(lines, "选择代理运行模式", "")

	for i, opt := range options {
		if i == m.modeIndex {
			lines = append(lines, SelectedStyle.Render("▸ "+opt))
		} else {
			lines = append(lines, UnselectedStyle.Render("  "+opt))
		}
	}

	lines = append(lines, "", InfoStyle.Render("mixed-port 模式：提供本地代理端口，兼容性最好；服务器流量需显式走代理"))
	lines = append(lines, InfoStyle.Render("TUN 模式：接管整机流量，但依赖 /dev/net/tun、iptables 和更严格的环境条件"))

	if !mihomo.CanUseTUN() {
		lines = append(lines, "", WarningStyle.Render("⚠ 检测到 /dev/net/tun 或 iptables 不可用，TUN 模式不可用"))
	}

	return m.renderScrollablePage("运行模式", strings.Join(lines, "\n"), "↑/↓ 选择 │ PgUp/PgDn 滚动 │ Enter 确认 │ Esc 返回")
}

func (m WizardModel) viewAdvanced() string {
	content := HeaderStyle.Render("高级设置（可直接按 Enter 使用默认值）") + "\n\n"

	for i, field := range m.advancedFields {
		val := m.advancedInputs[i].Value()
		if i == m.advancedIndex {
			content += SelectedStyle.Render("▸ "+field+": ") + m.advancedInputs[i].View() + "\n"
		} else {
			content += UnselectedStyle.Render("  "+field+": ") + InfoStyle.Render(val) + "\n"
		}
	}

	content += "\n" + HelpStyle.Render("↑/↓ 切换字段 │ 输入修改值 │ Enter 确认 │ Esc 返回")

	return BoxStyle.Render(content)
}

func (m WizardModel) viewPreview() string {
	cfg := m.appCfg
	sourceLabel := "订阅 URL"
	sourceValue := cfg.SubscriptionURL
	if strings.TrimSpace(m.localImportPath) != "" {
		sourceLabel = "本地订阅文件"
		sourceValue = m.localImportPath
	}

	width, _ := m.baseViewportSize()
	rows := []string{
		formatKV(sourceLabel, sourceValue, width),
		formatKV("运行模式", cfg.Mode, width),
		formatKV("配置目录", cfg.ConfigDir, width),
		formatKV("控制器地址", cfg.ControllerAddr, width),
		formatKV("mixed-port", fmt.Sprintf("%d", cfg.MixedPort), width),
		formatKV("Provider", cfg.ProviderPath, width),
		formatKV("健康检查", boolToYesNo(cfg.EnableHealthCheck), width),
		formatKV("systemd", boolToYesNo(cfg.EnableSystemd), width),
		formatKV("自动启动", boolToYesNo(cfg.AutoStart), width),
		"",
		"首次启动可能需要下载 GeoSite/GeoIP 数据（~33MB），以及验证节点是否真正加载。",
	}
	return m.renderScrollablePage("配置预览", strings.Join(rows, "\n"), "↑/↓ 滚动 │ Enter 确认并开始配置 │ Esc 返回修改")
}

func (m WizardModel) viewExecution() string {
	content := m.spinner.View() + " " + HeaderStyle.Render("正在配置 Mihomo...") + "\n\n"

	content += InfoStyle.Render("这可能需要一些时间，特别是首次运行：") + "\n"
	content += InfoStyle.Render("  • 检查并下载 Mihomo（如未安装）") + "\n"
	content += InfoStyle.Render("  • 生成配置文件") + "\n"
	content += InfoStyle.Render("  • 下载 GeoSite/GeoIP 数据（~33MB）") + "\n"
	content += InfoStyle.Render("  • 启动 Mihomo 服务") + "\n"
	content += InfoStyle.Render("  • 等待 Controller API 就绪") + "\n\n"
	content += HelpStyle.Render("请稍候...")

	return BoxStyle.Render(content)
}

func (m WizardModel) viewResult() string {
	allSuccess := true
	width, _ := m.baseViewportSize()
	var body strings.Builder
	for _, step := range m.execSteps {
		if step.Success {
			body.WriteString(SuccessStyle.Render("✅ " + step.Label))
			body.WriteString("\n")
		} else {
			body.WriteString(ErrorStyle.Render("❌ " + step.Label))
			body.WriteString("\n")
			allSuccess = false
		}
		if step.Detail != "" {
			for _, line := range wrapText(step.Detail, max(20, width-4)) {
				body.WriteString(InfoStyle.Render("   " + line))
				body.WriteString("\n")
			}
		}
	}

	body.WriteString("\n")
	if allSuccess {
		body.WriteString(SuccessStyle.Render("🎉 配置完成！Mihomo 已配置就绪。"))
		body.WriteString("\n")
	} else {
		body.WriteString(WarningStyle.Render("⚠️ 部分步骤失败，请检查上述错误信息。"))
		body.WriteString("\n")
	}

	footer := "↑/↓ 滚动 │ Enter 继续 │ Esc 退出"
	if m.controllerAvailable {
		body.WriteString("\n")
		body.WriteString(InfoStyle.Render("Controller API 可用，可以管理节点。"))
		body.WriteString("\n")
		footer = "↑/↓ 滚动 │ Enter/n 进入节点管理 │ Esc 退出"
	} else {
		if m.canImportFallback {
			body.WriteString("\n")
			body.WriteString(WarningStyle.Render("检测到订阅未成功加载，可切换为本地导入。"))
			body.WriteString("\n")
			body.WriteString(InfoStyle.Render("先在本地下载/解码订阅文件，再输入文件路径。"))
			body.WriteString("\n")
			footer = "↑/↓ 滚动 │ i 导入本地订阅文件 │ Enter/Esc 退出"
			return m.renderScrollablePage("执行结果", body.String(), footer)
		}
		body.WriteString("\n")
		body.WriteString(InfoStyle.Render("使用 'clashctl start' 启动服务"))
		body.WriteString("\n")
		body.WriteString(InfoStyle.Render("使用 'clashctl doctor' 检查环境"))
		body.WriteString("\n")
	}

	return m.renderScrollablePage("执行结果", body.String(), footer)
}

func (m WizardModel) viewImportLocal() string {
	content := strings.Join([]string{
		InfoStyle.Render("检测到服务器无法成功拉取订阅，建议改用本地导入。"),
		InfoStyle.Render("支持两类文件：base64 原始订阅，或解码后的 vless:// / trojan:// / hysteria2:// 链接列表。"),
		"",
		TextStyle.Render("文件路径: ") + m.importInput.View(),
	}, "\n")
	return m.renderStaticCard("导入本地订阅文件", content, "Enter 开始导入 │ Esc 返回结果页")
}

func (m WizardModel) viewGroupSelect() string {
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

	// Build full list content
	var list strings.Builder
	for i, g := range m.groups {
		typeIcon := groupIcon(g.Type)
		line := fmt.Sprintf("%s %s", typeIcon, g.Name)
		if g.Now != "" {
			line += InfoStyle.Render(fmt.Sprintf(" → %s", g.Now))
		}
		line += InfoStyle.Render(fmt.Sprintf(" (%d)", g.NodeCount))

		if i == m.groupIndex {
			list.WriteString(SelectedStyle.Render("▸ "+line) + "\n")
		} else {
			list.WriteString(UnselectedStyle.Render("  "+line) + "\n")
		}
	}

	if m.vpReady {
		m.vp.SetContent(list.String())
		// Auto-scroll to keep selected item visible
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

	// Fallback without viewport
	return BoxStyle.Render(HeaderStyle.Render("选择代理组") + "\n\n" + errorBlock + list.String())
}

func (m WizardModel) viewNodeSelect() string {
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

	// Build full node list
	var list strings.Builder
	for i, node := range m.nodes {
		marker := "  "
		if node.Selected {
			marker = ActiveMarkerStyle.Render("✓ ")
		}

		line := marker + node.Name

		// Show protocol type
		if node.Protocol != "" {
			line += " " + protocolBadge(node.Protocol)
		}

		if node.Delay != 0 {
			delayStr := mihomo.FormatDelay(node.Delay)
			line += " " + delayStyle(node.Delay).Render(delayStr)
		}

		if i == m.nodeIndex {
			list.WriteString(SelectedStyle.Render("▸ "+strings.TrimSpace(line)) + "\n")
		} else {
			list.WriteString(UnselectedStyle.Render("  "+strings.TrimSpace(line)) + "\n")
		}
	}

	if m.vpReady {
		m.vp.SetContent(list.String())
		// Auto-scroll to keep selected item visible
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

	// Fallback without viewport
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

func (m WizardModel) renderStaticCard(header, body, footer string) string {
	content := HeaderStyle.Render(header) + "\n\n" + body
	if footer != "" {
		content += "\n\n" + HelpStyle.Render(footer)
	}
	return BoxStyle.Render(content)
}

func (m WizardModel) renderScrollablePage(header, body, footer string) string {
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

func (m WizardModel) renderSelectablePage(header, body, footer string, selectedIndex int) string {
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

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func formatKV(label, value string, width int) string {
	wrapped := wrapText(value, max(20, width-len([]rune(label))-4))
	if len(wrapped) == 0 {
		return TextStyle.Render(label + ":")
	}
	var out strings.Builder
	for i, line := range wrapped {
		if i == 0 {
			out.WriteString(TextStyle.Render(label + ": "))
			out.WriteString(InputStyle.Render(line))
		} else {
			out.WriteString("\n")
			out.WriteString(strings.Repeat(" ", len([]rune(label))+2))
			out.WriteString(InputStyle.Render(line))
		}
	}
	return out.String()
}

func wrapText(text string, width int) []string {
	if width < 8 {
		return []string{text}
	}
	var out []string
	for _, rawLine := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimRight(rawLine, " ")
		if line == "" {
			out = append(out, "")
			continue
		}
		for len([]rune(line)) > width {
			cut := width
			runes := []rune(line)
			for i := width; i > width/2; i-- {
				if runes[i] == ' ' {
					cut = i
					break
				}
			}
			out = append(out, strings.TrimSpace(string(runes[:cut])))
			line = strings.TrimSpace(string(runes[cut:]))
		}
		out = append(out, line)
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func delayStyle(delay int) lipgloss.Style {
	switch {
	case delay < 0:
		return DelayBadStyle
	case delay < 100:
		return DelayGoodStyle
	case delay < 300:
		return DelayOkStyle
	case delay < 1000:
		return DelayOkStyle
	default:
		return DelayBadStyle
	}
}

func groupIcon(t string) string {
	switch mihomo.NormalizeProxyType(t) {
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

func protocolBadge(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "vless":
		return ProtocolVlessStyle.Render("Vless")
	case "hysteria2", "hy2":
		return ProtocolHy2Style.Render("Hy2")
	case "trojan":
		return ProtocolTrojanStyle.Render("Trojan")
	case "vmess":
		return ProtocolVMessStyle.Render("VMess")
	case "shadowsocks", "ss":
		return ProtocolSSStyle.Render("SS")
	default:
		return ProtocolUnknownStyle.Render(p)
	}
}

func boolToYesNo(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
