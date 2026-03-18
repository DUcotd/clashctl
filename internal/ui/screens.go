// Package ui contains screen view rendering.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"clashctl/internal/mihomo"
)

func (m WizardModel) viewWelcome() string {
	content := `
欢迎使用 clashctl — Mihomo TUN 交互式配置工具

这个向导将帮助你：

  • 输入机场订阅 URL
  • 选择代理运行模式 (TUN / mixed-port)
  • 调整高级设置（可选）
  • 自动生成并写入 Mihomo 配置
  • 启动 Mihomo 服务
  • 选择代理组和节点

  你只需要一个订阅链接，剩下的我们来搞定。

` + HelpStyle.Render("按 Enter 开始 │ 按 Esc / q 退出")

	return BoxStyle.Render(content)
}

func (m WizardModel) viewSubscription() string {
	content := HeaderStyle.Render("请输入你的机场订阅 URL") + "\n\n" +
		InfoStyle.Render("订阅链接通常以 https:// 开头，由你的机场服务商提供") + "\n\n" +
		m.urlInput.View() + "\n\n" +
		HelpStyle.Render("按 Enter 确认 │ 按 Esc 返回")

	return BoxStyle.Render(content)
}

func (m WizardModel) viewMode() string {
	options := []string{"TUN 模式（推荐，全局代理）", "mixed-port 模式（兼容，本地端口代理）"}

	content := HeaderStyle.Render("选择代理运行模式") + "\n\n"

	for i, opt := range options {
		if i == m.modeIndex {
			content += SelectedStyle.Render("▸ "+opt) + "\n"
		} else {
			content += UnselectedStyle.Render("  "+opt) + "\n"
		}
	}

	content += "\n" + InfoStyle.Render("TUN 模式：接管系统全部流量，无需配置应用代理")
	content += "\n" + InfoStyle.Render("mixed-port 模式：提供本地代理端口，需手动配置应用")

	// Show warning if TUN was auto-detected as unavailable
	if m.modeIndex == 1 && m.appCfg.Mode == "mixed" {
		content += "\n\n" + WarningStyle.Render("⚠ 检测到 /dev/net/tun 或 iptables 不可用，已自动切换到 mixed-port 模式")
	}

	content += "\n\n" + HelpStyle.Render("↑/↓ 选择 │ Enter 确认 │ Esc 返回")

	return BoxStyle.Render(content)
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

	content := HeaderStyle.Render("请确认以下配置") + "\n\n"
	content += TextStyle.Render("订阅 URL:    ") + InputStyle.Render(cfg.SubscriptionURL) + "\n"
	content += TextStyle.Render("运行模式:    ") + InputStyle.Render(cfg.Mode) + "\n"
	content += TextStyle.Render("配置目录:    ") + InputStyle.Render(cfg.ConfigDir) + "\n"
	content += TextStyle.Render("控制器地址:  ") + InputStyle.Render(cfg.ControllerAddr) + "\n"
	content += TextStyle.Render("mixed-port:  ") + InputStyle.Render(fmt.Sprintf("%d", cfg.MixedPort)) + "\n"
	content += TextStyle.Render("Provider:    ") + InputStyle.Render(cfg.ProviderPath) + "\n"
	content += TextStyle.Render("健康检查:    ") + InputStyle.Render(boolToYesNo(cfg.EnableHealthCheck)) + "\n"
	content += TextStyle.Render("systemd:     ") + InputStyle.Render(boolToYesNo(cfg.EnableSystemd)) + "\n"
	content += TextStyle.Render("自动启动:    ") + InputStyle.Render(boolToYesNo(cfg.AutoStart)) + "\n"

	content += "\n" + InfoStyle.Render("首次启动可能需要下载 GeoSite/GeoIP 数据（~33MB）")
	content += "\n" + HelpStyle.Render("Enter 确认并开始配置 │ Esc 返回修改")

	return BoxStyle.Render(content)
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
	content := HeaderStyle.Render("执行结果") + "\n\n"

	allSuccess := true
	for _, step := range m.execSteps {
		if step.Success {
			content += SuccessStyle.Render("✅ "+step.Label) + "\n"
		} else {
			content += ErrorStyle.Render("❌ "+step.Label) + "\n"
			allSuccess = false
		}
		if step.Detail != "" {
			content += InfoStyle.Render("   "+step.Detail) + "\n"
		}
	}

	content += "\n"
	if allSuccess {
		content += SuccessStyle.Render("🎉 配置完成！Mihomo 已配置就绪。") + "\n"
	} else {
		content += WarningStyle.Render("⚠️ 部分步骤失败，请检查上述错误信息。") + "\n"
	}

	if m.controllerAvailable {
		content += "\n" + InfoStyle.Render("Controller API 可用，可以管理节点。") + "\n"
		content += HelpStyle.Render("按 Enter/n 进入节点管理 │ 按 Esc 退出")
	} else {
		content += "\n" + InfoStyle.Render("使用 'clashctl start' 启动服务") + "\n"
		content += InfoStyle.Render("使用 'clashctl doctor' 检查环境") + "\n"
		content += HelpStyle.Render("按 Enter 退出")
	}

	return BoxStyle.Render(content)
}

func (m WizardModel) viewGroupSelect() string {
	if m.loading {
		content := m.spinner.View() + " " + m.loadingMsg
		return BoxStyle.Render(content)
	}

	if len(m.groups) == 0 {
		content := WarningStyle.Render("未找到任何代理组") + "\n\n"
		content += InfoStyle.Render("请确认 Mihomo 已启动并且有可用的代理组") + "\n"
		content += HelpStyle.Render("按 Esc 退出")
		return BoxStyle.Render(content)
	}

	content := HeaderStyle.Render("选择代理组") + "\n\n"

	for i, g := range m.groups {
		typeIcon := groupIcon(g.Type)
		line := fmt.Sprintf("%s %s", typeIcon, g.Name)

		if g.Now != "" {
			line += InfoStyle.Render(fmt.Sprintf(" → %s", g.Now))
		}
		line += InfoStyle.Render(fmt.Sprintf(" (%d)", g.NodeCount))

		if i == m.groupIndex {
			content += SelectedStyle.Render("▸ "+line) + "\n"
		} else {
			content += UnselectedStyle.Render("  "+line) + "\n"
		}
	}

	content += "\n" + BadgeStyle.Render("🔀 select") + " 选择  "
	content += BadgeStyle.Render("⚡ url-test") + " 测试  "
	content += BadgeStyle.Render("🔄 fallback") + " 故障转移"
	content += "\n"
	content += HelpStyle.Render("↑/↓ 选择 │ Enter 查看节点 │ r 刷新 │ Esc 退出")

	return BoxStyle.Render(content)
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

	content := HeaderStyle.Render(fmt.Sprintf("代理组: %s", m.selectedGroup)) + "\n\n"

	if len(m.nodes) == 0 {
		content += WarningStyle.Render("该组没有可用节点") + "\n"
		content += HelpStyle.Render("按 Esc 返回")
		return BoxStyle.Render(content)
	}

	for i, node := range m.nodes {
		marker := "  "
		if node.Selected {
			marker = ActiveMarkerStyle.Render("✓ ")
		}

		line := marker + node.Name

		// Show delay if tested (using shared FormatDelay from mihomo package)
		if node.Delay != 0 {
			delayStr := mihomo.FormatDelay(node.Delay)
			line += " " + delayStyle(node.Delay).Render(delayStr)
		}

		if i == m.nodeIndex {
			content += SelectedStyle.Render("▸ "+strings.TrimSpace(line)) + "\n"
		} else {
			content += UnselectedStyle.Render("  "+strings.TrimSpace(line)) + "\n"
		}
	}

	if m.switchResult != "" {
		content += "\n" + m.switchResult + "\n"
	}

	content += "\n" + HelpStyle.Render("↑/↓ 选择 │ Enter 切换 │ t 测试延迟 │ r 刷新 │ Esc 返回")

	return BoxStyle.Render(content)
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

func boolToYesNo(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
