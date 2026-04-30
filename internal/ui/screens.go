// Package ui contains screen view rendering.
package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"clashctl/internal/mihomo"
	"clashctl/internal/system"
)

func (m WizardModel) viewWelcome() string {
	body := strings.Join([]string{
		"欢迎使用 clashctl — Mihomo 交互式配置工具",
		"",
		"这个向导将帮助你：",
		"",
		"  • 输入订阅 URL、直接粘贴订阅内容，或使用本地订阅文件",
		"  • 选择 TUN / mixed-port 运行模式",
		"  • 调整高级设置（可选）",
		"  • 自动生成并写入 Mihomo 配置",
		"  • 启动 Mihomo 并验证节点是否真正加载",
		"  • 进入代理组和节点管理",
		"",
		"如果服务器拉不到订阅，向导会自动提示本地导入静态配置。",
	}, "\n")
	return m.renderScrollablePage("开始使用", body, renderKeyHelp([][2]string{
		{"Enter", "开始"},
		{"Esc/q", "退出"},
	}))
}

func (m WizardModel) viewSubscription() string {
	inputWidth := max(40, m.width-16)
	m.urlInput.Width = inputWidth
	m.fileInput.Width = inputWidth
	m.inlineInput.SetWidth(inputWidth)
	m.inlineInput.SetHeight(8)

	selector := renderSourceSelector(m.sourceMode)
	body := selector + "\n\n"
	footer := ""

	switch m.sourceMode {
	case SubscriptionSourceURL:
		body += InfoStyle.Render("推荐使用订阅 URL；向导会先尝试抓取并尽量转成服务器更稳定的静态配置") + "\n\n"
		body += WarningStyle.Render("远程 provider-only 订阅会被拒绝；这类订阅请先在本地展开后再导入") + "\n\n"
		body += m.urlInput.View()
		footer = renderKeyHelp([][2]string{
			{"←/→", "切换来源"},
			{"Enter", "下一步"},
			{"Esc/q", "退出"},
		})
	case SubscriptionSourceInline:
		body += InfoStyle.Render("可直接粘贴 Base64、原始节点链接列表或 Mihomo/Clash YAML") + "\n"
		body += InfoStyle.Render("Enter 提交；终端支持时可用 Shift+Enter 手动换行，多行粘贴也可直接使用") + "\n\n"
		body += m.inlineInput.View()
		footer = renderKeyHelp([][2]string{
			{"←/→", "切换来源"},
			{"Enter", "提交"},
			{"Shift+Enter", "换行"},
			{"Esc/q", "退出"},
		})
	case SubscriptionSourceFile:
		body += InfoStyle.Render("适合服务器无法直连订阅时使用本地文件导入") + "\n"
		body += InfoStyle.Render("支持 Base64 原始订阅、解码后的节点链接列表或 YAML 文件") + "\n\n"
		body += m.fileInput.View()
		footer = renderKeyHelp([][2]string{
			{"←/→", "切换来源"},
			{"Enter", "下一步"},
			{"Esc/q", "退出"},
		})
	}

	return m.renderStaticCard("选择订阅来源", body, footer)
}

func (m WizardModel) viewMode() string {
	options := []string{"TUN 模式（全局代理，需 CAP_NET_ADMIN）", "mixed-port 模式（推荐，兼容性好）"}

	var lines []string
	lines = append(lines, "选择代理运行模式", "")

	for i, opt := range options {
		if i == m.modeIndex {
			lines = append(lines, SelectedBarStyle.Render("▸ "+opt))
		} else {
			lines = append(lines, UnselectedStyle.Render("  "+opt))
		}
	}

	lines = append(lines, "", InfoStyle.Render("mixed-port 模式：提供本地代理端口，兼容性最好；服务器流量需显式走代理"))
	lines = append(lines, InfoStyle.Render("TUN 模式：接管整机流量，但依赖 /dev/net/tun、iptables 和更严格的环境条件"))

	if !mihomo.CanUseTUN() {
		lines = append(lines, "", WarningStyle.Render("⚠ 检测到 /dev/net/tun 或 iptables 不可用，TUN 模式不可用"))
	}

	return m.renderScrollablePage("运行模式", strings.Join(lines, "\n"), renderKeyHelp([][2]string{
		{"↑/↓", "选择"},
		{"Enter", "继续"},
		{"a", "高级设置"},
		{"Esc", "返回"},
		{"q", "退出"},
	}))
}

func (m WizardModel) viewAdvanced() string {
	content := HeaderBarStyle.Render("高级设置（可选）") + "\n"
	content += InfoStyle.Render("普通场景保持默认即可；修改后按 Enter 返回预览并开始确认") + "\n\n"

	for i, field := range m.advancedFields {
		val := m.advancedInputs[i].Value()
		if i == m.advancedIndex {
			content += SelectedBarStyle.Render("▸ "+field+": ") + InputFocusedStyle.Render(m.advancedInputs[i].View()) + "\n"
		} else {
			content += UnselectedStyle.Render("  "+field+": ") + InfoStyle.Render(val) + "\n"
		}
	}

	content += "\n" + WarningStyle.Render("控制器地址仅允许 127.0.0.1 / localhost / ::1 这类本地回环地址")

	content += "\n" + renderKeyHelp([][2]string{
		{"↑/↓", "切换字段"},
		{"输入", "修改值"},
		{"Enter", "保存"},
		{"Esc", "放弃"},
		{"q", "退出"},
	})

	return BoxStyle.Render(content)
}

func (m WizardModel) viewPreview() string {
	cfg := m.appCfg
	sourceLabel, sourceValue := m.previewSource()

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
		"安全约束：控制器地址仅允许本地回环；远程 provider-only 订阅会被拒绝。",
		"",
		"首次启动可能需要下载 GeoSite/GeoIP 数据（~33MB），以及验证节点是否真正加载。",
	}
	return m.renderScrollablePage("配置预览", strings.Join(rows, "\n"), renderKeyHelp([][2]string{
		{"↑/↓", "滚动"},
		{"Enter", "开始配置"},
		{"a", "高级设置"},
		{"Esc", "返回"},
		{"q", "退出"},
	}))
}

func renderSourceSelector(current SubscriptionSource) string {
	options := []SubscriptionSource{
		SubscriptionSourceURL,
		SubscriptionSourceInline,
		SubscriptionSourceFile,
	}
	parts := make([]string, 0, len(options))
	for _, opt := range options {
		label := opt.Title()
		if opt == current {
			parts = append(parts, SelectorActiveStyle.Render("▣ "+label))
		} else {
			parts = append(parts, SelectorInactiveStyle.Render("□ "+label))
		}
	}
	return strings.Join(parts, "   ")
}

func (m WizardModel) previewSource() (string, string) {
	switch {
	case strings.TrimSpace(m.inlineContent) != "":
		content := strings.TrimSpace(m.inlineContent)
		contentKind := system.ProbeContentKind([]byte(content))
		detail := fmt.Sprintf("%s (%d bytes)", inlineContentKindLabel(contentKind), len(content))
		return "直接粘贴内容", detail
	case strings.TrimSpace(m.localImportPath) != "":
		return "本地订阅文件", m.localImportPath
	default:
		return "订阅 URL", m.appCfg.SubscriptionURL
	}
}

func inlineContentKindLabel(kind string) string {
	switch kind {
	case "base64-links":
		return "Base64 节点订阅"
	case "raw-links":
		return "原始节点链接"
	case "mihomo-yaml":
		return "Mihomo/Clash YAML"
	default:
		return "未识别内容"
	}
}

func (m WizardModel) viewExecution() string {
	content := m.spinner.View() + " " + InfoStyle.Render("执行中") + "\n\n"
	if m.currentStep != "" {
		content += CardHeaderStyle.Render("当前步骤: "+m.currentStep) + "\n\n"
	}
	if len(m.execSteps) > 0 {
		content += HeaderStyle.Render("进度:") + "\n"
		for _, step := range m.execSteps {
			marker := SuccessStyle.Render("✓ ")
			if !step.Success {
				marker = ErrorStyle.Render("✗ ")
			}
			content += marker + step.Label + "\n"
		}
		content += "\n"
	}

	content += InfoStyle.Render("这可能需要一些时间，特别是首次运行：") + "\n"
	content += InfoStyle.Render("  • 检查并下载 Mihomo（如未安装）") + "\n"
	content += InfoStyle.Render("  • 生成配置文件") + "\n"
	content += InfoStyle.Render("  • 下载 GeoSite/GeoIP 数据（~33MB）") + "\n"
	content += InfoStyle.Render("  • 启动 Mihomo 服务") + "\n"
	content += InfoStyle.Render("  • 等待 Controller API 就绪")

	if m.canAbortExecution {
		footer := renderKeyHelp([][2]string{
			{"Esc/q", "中止并退出"},
		})
		return m.renderScrollablePage("正在配置 Mihomo...", content, footer)
	}
	return m.renderScrollablePage("正在配置 Mihomo...", content, "请稍候，此阶段不可中断...")
}

func (m WizardModel) viewResult() string {
	allSuccess := true
	width, _ := m.baseViewportSize()
	var body strings.Builder
	for _, step := range m.execSteps {
		if step.Success {
			body.WriteString(SuccessStyle.Render("✓ " + step.Label))
			body.WriteString("\n")
		} else {
			body.WriteString(ErrorStyle.Render("✗ " + step.Label))
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
		body.WriteString(WarningStyle.Render("⚠ 部分步骤失败，请检查上述错误信息。"))
		body.WriteString("\n")
	}

	footer := renderKeyHelp([][2]string{
		{"↑/↓", "滚动"},
		{"Esc", "返回"},
		{"q", "退出"},
	})
	if m.controllerAvailable {
		body.WriteString("\n")
		body.WriteString(InfoStyle.Render("Controller API 可用，可以管理节点。"))
		body.WriteString("\n")
		footer = renderKeyHelp([][2]string{
			{"↑/↓", "滚动"},
			{"Enter", "进入节点管理"},
			{"Esc", "返回"},
			{"q", "退出"},
		})
	} else {
		if m.canImportFallback {
			body.WriteString("\n")
			body.WriteString(WarningStyle.Render("检测到订阅未成功加载，可切换为本地导入。"))
			body.WriteString("\n")
			if strings.TrimSpace(m.importHint) != "" {
				body.WriteString(InfoStyle.Render(m.importHint))
				body.WriteString("\n")
			}
			body.WriteString(InfoStyle.Render("先在本地下载/解码订阅文件，再输入文件路径。"))
			body.WriteString("\n")
			footer = renderKeyHelp([][2]string{
				{"↑/↓", "滚动"},
				{"Enter", "导入本地订阅"},
				{"Esc", "返回"},
				{"q", "退出"},
			})
			return m.renderScrollablePage("执行结果", body.String(), footer)
		}
		body.WriteString("\n")
		body.WriteString(InfoStyle.Render("使用 'clashctl service start' 启动服务"))
		body.WriteString("\n")
		body.WriteString(InfoStyle.Render("使用 'clashctl doctor' 检查环境"))
		body.WriteString("\n")
	}

	return m.renderScrollablePage("执行结果", body.String(), footer)
}

func (m WizardModel) viewImportLocal() string {
	m.importInput.Width = max(40, m.width-16)
	content := strings.Join([]string{
		InfoStyle.Render("检测到服务器无法成功拉取订阅，建议改用本地导入。"),
		InfoStyle.Render("支持两类文件：base64 原始订阅，或解码后的 vless:// / trojan:// / hysteria2:// 链接列表。"),
		"",
		TextStyle.Render("文件路径: ") + m.importInput.View(),
	}, "\n")
	return m.renderStaticCard("导入本地订阅文件", content, renderKeyHelp([][2]string{
		{"Enter", "开始导入"},
		{"Esc", "返回结果页"},
		{"q", "退出"},
	}))
}

func (m WizardModel) renderStaticCard(header, body, footer string) string {
	return renderStaticCard(m.viewportState, int(m.screen), m.baseViewportSize, header, m.feedback, body, footer)
}

func (m WizardModel) renderScrollablePage(header, body, footer string) string {
	return renderScrollablePage(m.viewportState, int(m.screen), m.baseViewportSize, header, m.feedback, body, footer)
}

func (m WizardModel) renderSelectablePage(header, body, footer string, selectedIndex int) string {
	return renderSelectablePage(m.viewportState, int(m.screen), m.baseViewportSize, header, m.feedback, body, footer, selectedIndex)
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func formatKV(label, value string, width int) string {
	labelW := runewidth.StringWidth(label)
	wrapped := wrapText(value, max(20, width-labelW-4))
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
			out.WriteString(strings.Repeat(" ", labelW+2))
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
		for runewidth.StringWidth(line) > width {
			cut := 0
			w := 0
			for i, r := range line {
				cw := runewidth.RuneWidth(r)
				if w+cw > width {
					break
				}
				cut = i + utf8.RuneLen(r)
				w += cw
			}
			if cut == 0 {
				cut = len(line)
			}
			spaceCut := cut
			for i := cut; i > cut/2 && i > 0; i-- {
				if line[i-1] == ' ' {
					spaceCut = i
					break
				}
			}
			out = append(out, strings.TrimSpace(line[:spaceCut]))
			line = strings.TrimSpace(line[spaceCut:])
			if line == "" {
				break
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func delayStyle(delay int) lipgloss.Style {
	switch {
	case delay == 0:
		return DelayUnknownStyle
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
	return mihomo.GroupTypeIcon(t)
}

func protocolBadge(p string) string {
	clean := strings.TrimSpace(p)
	switch strings.ToLower(clean) {
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
		if len(clean) > 10 {
			clean = clean[:10]
		}
		return ProtocolUnknownStyle.Render(clean)
	}
}

func boolToYesNo(b bool) string {
	if b {
		return "是"
	}
	return "否"
}

func parseYesNo(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "是" || s == "yes" || s == "true" || s == "1"
}

func (m WizardModel) viewQuitConfirm() string {
	var d strings.Builder
	d.WriteString(HeaderBarStyle.Render("确认退出"))
	d.WriteString("\n\n")
	d.WriteString(ConfirmStyle.Render("确定要退出配置向导吗？"))
	d.WriteString("\n")
	d.WriteString(InfoStyle.Render("已输入的内容将不会被保存。"))
	d.WriteString("\n\n")
	d.WriteString(renderKeyHelp([][2]string{
		{"y", "确认退出"},
		{"Enter", "确认退出"},
		{"n", "取消"},
		{"Esc", "取消"},
	}))
	return BoxWarningStyle.Render(d.String())
}

func (m WizardModel) viewWizardHelp() string {
	var help strings.Builder
	help.WriteString(HeaderBarStyle.Render("快捷键帮助"))
	help.WriteString("\n\n")

	help.WriteString(HelpSectionStyle.Render("全局操作:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("q", "退出向导") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Ctrl+C", "退出向导") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("?", "显示/隐藏此帮助") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("导航:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Enter", "确认/下一步") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Esc", "返回上一步 / 退出") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("订阅来源:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Tab", "切换下一个来源") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("→", "切换下一个来源") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Shift+Tab", "切换上一个来源") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("←", "切换上一个来源") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("模式选择:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("↑/↓", "上下选择") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("j/k", "上下选择") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("a", "进入高级设置") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("高级设置:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("↑/↓", "切换输入框") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("j/k", "切换输入框") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Enter", "保存并返回预览") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Esc", "放弃更改并返回预览") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpSectionStyle.Render("预览:") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("↑/↓", "滚动查看") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("j/k", "滚动查看") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("Enter", "开始执行") + "\n")
	help.WriteString(InfoStyle.Render("  ") + renderKeyBinding("a", "进入高级设置") + "\n")
	help.WriteString("\n")

	help.WriteString(HelpStyle.Render("按任意键返回"))
	return BoxStyle.Render(help.String())
}
