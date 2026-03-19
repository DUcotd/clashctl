package ui

import (
	"strings"
	"testing"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

func TestScreenStepLabel(t *testing.T) {
	tests := []struct {
		screen Screen
		want   string
	}{
		{ScreenWelcome, "欢迎"},
		{ScreenSubscription, "步骤 1/8: 输入订阅 URL"},
		{ScreenMode, "步骤 2/8: 选择运行模式"},
		{ScreenAdvanced, "步骤 3/8: 高级设置"},
		{ScreenPreview, "步骤 4/8: 配置预览"},
		{ScreenExecution, "步骤 5/8: 正在配置..."},
		{ScreenResult, "步骤 6/8: 执行结果"},
		{ScreenImportLocal, "步骤 6/8: 导入本地订阅"},
		{ScreenGroupSelect, "步骤 7/8: 选择代理组"},
		{ScreenNodeSelect, "步骤 8/8: 选择节点"},
		{ScreenDone, ""},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.screen.StepLabel()
			if got != tt.want {
				t.Errorf("StepLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	got := wrapText("Controller API 已就绪，但订阅节点未成功加载 provider 文件不存在或为空", 16)
	if len(got) < 3 {
		t.Fatalf("wrapText should split long text, got %#v", got)
	}
}

func TestRenderScrollablePage(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.width = 80
	wizard.height = 24
	wizard.ensureViewport()
	wizard.screen = ScreenResult
	view := wizard.renderScrollablePage("测试标题", strings.Repeat("一二三四五六七八九十\n", 20), "帮助")
	if !strings.Contains(view, "测试标题") || !strings.Contains(view, "帮助") {
		t.Fatalf("renderScrollablePage missing header/footer: %s", view)
	}
}

func TestGroupIcon(t *testing.T) {
	tests := []struct {
		typ  string
		want string
	}{
		{"select", "🔀"},
		{"url-test", "⚡"},
		{"fallback", "🔄"},
		{"load-balance", "⚖️"},
		{"unknown", "📦"},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			got := groupIcon(tt.typ)
			if got != tt.want {
				t.Errorf("groupIcon(%q) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestBoolToYesNo(t *testing.T) {
	if got := boolToYesNo(true); got != "是" {
		t.Errorf("boolToYesNo(true) = %q, want 是", got)
	}
	if got := boolToYesNo(false); got != "否" {
		t.Errorf("boolToYesNo(false) = %q, want 否", got)
	}
}

func TestFormatNodeDelay(t *testing.T) {
	tests := []struct {
		name  string
		delay int
		want  string
	}{
		{"timeout", -1, "超时"},
		{"untested", 0, "未测试"},
		{"fast", 50, "50ms ✨"},
		{"ok", 150, "150ms"},
		{"slow", 500, "500ms ⚠️"},
		{"very slow", 1500, "1.5s 🔴"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mihomo.FormatDelay(tt.delay)
			if got != tt.want {
				t.Errorf("FormatDelay(%d) = %q, want %q", tt.delay, got, tt.want)
			}
		})
	}
}

func TestNewWizardDefaults(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())

	if wizard.screen != ScreenWelcome {
		t.Errorf("initial screen = %d, want %d", wizard.screen, ScreenWelcome)
	}
	if wizard.appCfg == nil {
		t.Fatal("appCfg should not be nil")
	}
	if wizard.urlInput.Value() != "" {
		t.Errorf("urlInput should be empty, got %q", wizard.urlInput.Value())
	}
	if len(wizard.advancedFields) != 7 {
		t.Errorf("advancedFields count = %d, want 7", len(wizard.advancedFields))
	}
}

func TestWizardCompleted(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	if wizard.Completed() {
		t.Error("new wizard should not be completed")
	}

	wizard.execSteps = []ExecStep{{Label: "test", Success: true}}
	if !wizard.Completed() {
		t.Error("wizard with execSteps should be completed")
	}
}

func TestNewWizardUsesPersistedValues(t *testing.T) {
	cfg := core.DefaultAppConfig()
	cfg.SubscriptionURL = "https://example.com/sub"
	cfg.Mode = "tun"
	cfg.ConfigDir = "/tmp/mihomo-test"
	cfg.ControllerAddr = "127.0.0.1:9191"
	cfg.MixedPort = 9090
	cfg.ProviderPath = "./providers/custom.yaml"
	cfg.EnableHealthCheck = false
	cfg.EnableSystemd = false
	cfg.AutoStart = false

	wizard := NewWizard(cfg)

	if wizard.urlInput.Value() != cfg.SubscriptionURL {
		t.Errorf("urlInput = %q, want %q", wizard.urlInput.Value(), cfg.SubscriptionURL)
	}
	if wizard.modeIndex != 0 {
		t.Errorf("modeIndex = %d, want 0 for tun", wizard.modeIndex)
	}
	if wizard.advancedInputs[0].Value() != cfg.ConfigDir {
		t.Errorf("config dir input = %q, want %q", wizard.advancedInputs[0].Value(), cfg.ConfigDir)
	}
	if wizard.advancedInputs[1].Value() != cfg.ControllerAddr {
		t.Errorf("controller input = %q, want %q", wizard.advancedInputs[1].Value(), cfg.ControllerAddr)
	}
}
