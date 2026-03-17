// Package ui contains screen view rendering.
package ui

import "fmt"

func (m WizardModel) viewWelcome() string {
	content := `
欢迎使用 clashctl — Mihomo TUN 交互式配置工具

这个向导将帮助你：

  • 输入机场订阅 URL
  • 选择代理运行模式 (TUN / mixed-port)
  • 调整高级设置（可选）
  • 自动生成并写入 Mihomo 配置
  • 启动 Mihomo 服务

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

	content += "\n" + HelpStyle.Render("Enter 确认并写入配置 │ Esc 返回修改")

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
		content += InfoStyle.Render("使用 'clashctl start' 启动服务") + "\n"
		content += InfoStyle.Render("使用 'clashctl doctor' 检查环境") + "\n"
	} else {
		content += ErrorStyle.Render("⚠️ 部分步骤失败，请检查上述错误信息。") + "\n"
	}

	content += "\n" + HelpStyle.Render("按 Enter 退出")

	return BoxStyle.Render(content)
}

// Completed returns true if the wizard finished all steps.
func (m WizardModel) Completed() bool {
	return len(m.execSteps) > 0
}

func boolToYesNo(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
