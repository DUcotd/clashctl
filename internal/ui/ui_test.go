package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

func TestScreenStepLabel(t *testing.T) {
	tests := []struct {
		screen Screen
		want   string
	}{
		{ScreenWelcome, "欢迎"},
		{ScreenSubscription, "步骤 1/7: 选择订阅来源"},
		{ScreenMode, "步骤 2/7: 选择运行模式"},
		{ScreenAdvanced, "可选设置: 高级参数"},
		{ScreenPreview, "步骤 3/7: 配置预览"},
		{ScreenExecution, "步骤 4/7: 正在配置..."},
		{ScreenResult, "步骤 5/7: 执行结果"},
		{ScreenImportLocal, "步骤 5/7: 导入本地订阅"},
		{ScreenGroupSelect, "步骤 6/7: 选择代理组"},
		{ScreenNodeSelect, "步骤 7/7: 选择节点"},
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

	if wizard.screen != ScreenSubscription {
		t.Errorf("initial screen = %d, want %d", wizard.screen, ScreenSubscription)
	}
	if wizard.appCfg == nil {
		t.Fatal("appCfg should not be nil")
	}
	if wizard.urlInput.Value() != "" {
		t.Errorf("urlInput should be empty, got %q", wizard.urlInput.Value())
	}
	if wizard.sourceMode != SubscriptionSourceURL {
		t.Errorf("sourceMode = %v, want URL", wizard.sourceMode)
	}
	if len(wizard.advancedFields) != 7 {
		t.Errorf("advancedFields count = %d, want 7", len(wizard.advancedFields))
	}
}

func TestNewNodeManagerDefaults(t *testing.T) {
	manager := NewNodeManager(core.DefaultAppConfig())

	if manager.screen != ScreenGroupSelect {
		t.Fatalf("screen = %v, want ScreenGroupSelect", manager.screen)
	}
	if !manager.loading {
		t.Fatal("loading should be true")
	}
	if manager.title != "📡 clashctl 节点管理" {
		t.Fatalf("title = %q", manager.title)
	}
}

func TestStandaloneNodeManagerViewUsesNodeTitle(t *testing.T) {
	manager := NewNodeManager(core.DefaultAppConfig())
	manager.loading = false
	manager.groups = []GroupItem{{Name: "PROXY", Type: "select", NodeCount: 3}}
	manager.width = 80
	manager.height = 24
	manager.ensureViewport()

	view := manager.View()
	if !strings.Contains(view, "📡 clashctl 节点管理") {
		t.Fatalf("view missing node manager title:\n%s", view)
	}
	if strings.Contains(view, "步骤 ") {
		t.Fatalf("view should not include wizard step label:\n%s", view)
	}
}

func TestNodeManagerEscReturnsToGroupSelect(t *testing.T) {
	manager := NewNodeManager(core.DefaultAppConfig())
	manager.screen = ScreenNodeSelect
	manager.loading = false
	manager.selectedGroup = "PROXY"
	manager.nodes = []NodeItem{{Name: "Node A"}}

	updated, _ := manager.updateNodeSelect(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(NodeManagerModel)
	if got.screen != ScreenGroupSelect {
		t.Fatalf("screen = %v, want ScreenGroupSelect", got.screen)
	}
}

func TestNodeManagerQQuits(t *testing.T) {
	manager := NewNodeManager(core.DefaultAppConfig())

	updated, cmd := manager.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := updated.(NodeManagerModel)
	if !got.quitting {
		t.Fatal("quitting should be true")
	}
	if cmd == nil {
		t.Fatal("quit command should be returned")
	}
}

func TestNodeManagerSwitchSuccessSetsSuccessFeedback(t *testing.T) {
	manager := NewNodeManager(core.DefaultAppConfig())
	manager.screen = ScreenNodeSelect
	manager.nodes = []NodeItem{{Name: "Node A"}, {Name: "Node B"}}
	manager.nodeIndex = 1

	updated, _ := manager.handleNodeSwitched(nodeSwitchedMsg{success: true})
	got := updated.(NodeManagerModel)
	if got.feedback.messageText == "" {
		t.Fatal("success feedback should be set")
	}
	if !got.nodes[1].Selected {
		t.Fatal("selected node should be updated")
	}
}

func TestWizardCompleted(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	if wizard.Completed() {
		t.Error("new wizard should not be completed")
	}

	wizard.completed = true
	if !wizard.Completed() {
		t.Error("wizard with completed flag should be completed")
	}
}

func TestNodeManagerCompletedDefaultsFalse(t *testing.T) {
	manager := NewNodeManager(core.DefaultAppConfig())
	if manager.Completed() {
		t.Fatal("standalone node manager should not mark setup completed")
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

func TestSubscriptionEnterStartsExecutionForURL(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenSubscription
	wizard.urlInput.SetValue("https://example.com/sub")

	updated, _ := wizard.updateSubscription(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(WizardModel)
	if got.screen != ScreenMode {
		t.Fatalf("screen = %v, want ScreenMode", got.screen)
	}
	if got.appCfg.SubscriptionURL != "https://example.com/sub" {
		t.Fatalf("SubscriptionURL = %q", got.appCfg.SubscriptionURL)
	}
	if got.inlineContent != "" || got.localImportPath != "" {
		t.Fatalf("unexpected alternate source state: inline=%q file=%q", got.inlineContent, got.localImportPath)
	}
}

func TestModeEnterUsesQuickPathToPreview(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenMode
	wizard.modeIndex = 1

	updated, _ := wizard.updateMode(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(WizardModel)
	if got.screen != ScreenPreview {
		t.Fatalf("screen = %v, want ScreenPreview", got.screen)
	}
	if got.appCfg.Mode != "mixed" {
		t.Fatalf("Mode = %q, want mixed", got.appCfg.Mode)
	}
}

func TestModeAOpensAdvancedWithoutExtraStep(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenMode
	wizard.modeIndex = 0

	updated, _ := wizard.updateMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got := updated.(WizardModel)
	if got.screen != ScreenAdvanced {
		t.Fatalf("screen = %v, want ScreenAdvanced", got.screen)
	}
	if got.appCfg.Mode != "tun" {
		t.Fatalf("Mode = %q, want tun", got.appCfg.Mode)
	}
}

func TestAdvancedEscDiscardsDraftChanges(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenAdvanced
	wizard.appCfg.ConfigDir = "/etc/mihomo"
	wizard.advancedInputs[0].SetValue("/tmp/custom")

	updated, _ := wizard.updateAdvanced(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(WizardModel)
	if got.screen != ScreenPreview {
		t.Fatalf("screen = %v, want ScreenPreview", got.screen)
	}
	if got.appCfg.ConfigDir != "/etc/mihomo" {
		t.Fatalf("ConfigDir = %q, want original value", got.appCfg.ConfigDir)
	}
	if got.advancedInputs[0].Value() != "/etc/mihomo" {
		t.Fatalf("advanced input = %q, want reset to committed value", got.advancedInputs[0].Value())
	}
}

func TestSubscriptionEnterTracksLocalImportPath(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenSubscription
	wizard.setSubscriptionSource(SubscriptionSourceFile)
	wizard.fileInput.SetValue("/tmp/sub.txt")

	updated, _ := wizard.updateSubscription(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(WizardModel)
	if got.screen != ScreenMode {
		t.Fatalf("screen = %v, want ScreenMode", got.screen)
	}
	if got.localImportPath != "/tmp/sub.txt" {
		t.Fatalf("localImportPath = %q", got.localImportPath)
	}
	if got.appCfg.SubscriptionURL != "" {
		t.Fatalf("SubscriptionURL = %q, want empty for local import", got.appCfg.SubscriptionURL)
	}
}

func TestSubscriptionEnterUsesInlineContent(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenSubscription
	wizard.setSubscriptionSource(SubscriptionSourceInline)
	wizard.inlineInput.SetValue("vless://abc@example.com:443?security=tls#NodeA")

	updated, _ := wizard.updateSubscription(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(WizardModel)
	if got.screen != ScreenMode {
		t.Fatalf("screen = %v, want ScreenMode", got.screen)
	}
	if got.inlineContent == "" {
		t.Fatal("inlineContent should be captured")
	}
	if got.appCfg.SubscriptionURL != "" || got.localImportPath != "" {
		t.Fatalf("unexpected alternate source state: url=%q file=%q", got.appCfg.SubscriptionURL, got.localImportPath)
	}
}

func TestSubscriptionEnterWithEmptyInputShowsInlineError(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenSubscription

	updated, _ := wizard.updateSubscription(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(WizardModel)
	if got.screen != ScreenSubscription {
		t.Fatalf("screen = %v, want ScreenSubscription", got.screen)
	}
	if got.feedback.errorText == "" {
		t.Fatal("expected validation error on empty subscription input")
	}
}

func TestSubscriptionTabCyclesSources(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenSubscription

	updated, _ := wizard.updateSubscription(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(WizardModel)
	if got.sourceMode != SubscriptionSourceInline {
		t.Fatalf("sourceMode after tab = %v, want inline", got.sourceMode)
	}

	updated, _ = got.updateSubscription(tea.KeyMsg{Type: tea.KeyTab})
	got = updated.(WizardModel)
	if got.sourceMode != SubscriptionSourceFile {
		t.Fatalf("sourceMode after second tab = %v, want file", got.sourceMode)
	}
}

func TestSetupProgressDoneCarriesImportFallbackState(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())

	updated, _ := wizard.Update(setupProgressMsg{done: true, canImport: true, importHint: "provider 拉取失败"})

	got := updated.(WizardModel)
	if !got.canImportFallback {
		t.Fatal("canImportFallback should be true")
	}
	if got.importHint != "provider 拉取失败" {
		t.Fatalf("importHint = %q", got.importHint)
	}
	if !got.Completed() {
		t.Fatal("wizard should be marked completed after done progress")
	}
}

func TestResultEnterOpensImportFallback(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenResult
	wizard.canImportFallback = true

	updated, _ := wizard.updateResult(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(WizardModel)
	if got.screen != ScreenImportLocal {
		t.Fatalf("screen = %v, want ScreenImportLocal", got.screen)
	}
}

func TestResultEscReturnsToPreview(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenResult

	updated, _ := wizard.updateResult(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(WizardModel)
	if got.screen != ScreenPreview {
		t.Fatalf("screen = %v, want ScreenPreview", got.screen)
	}
}

func TestImportLocalEmptyPathShowsError(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenImportLocal

	updated, _ := wizard.updateImportLocal(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(WizardModel)
	if got.screen != ScreenImportLocal {
		t.Fatalf("screen = %v, want ScreenImportLocal", got.screen)
	}
	if got.feedback.errorText == "" {
		t.Fatal("expected validation error on empty import path")
	}
}

func TestSubscriptionViewDoesNotPanicWithPlaceholder(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.screen = ScreenSubscription
	wizard.width = 80
	wizard.height = 24

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View() should not panic on subscription screen: %v", r)
		}
	}()

	view := wizard.View()
	if !strings.Contains(view, "选择订阅来源") {
		t.Fatalf("subscription view missing source header: %s", view)
	}
	if !strings.Contains(view, "订阅 URL") || !strings.Contains(view, "直接粘贴") || !strings.Contains(view, "本地文件") {
		t.Fatalf("subscription view missing source selector: %s", view)
	}
}

func TestPreviewSourceForInlineContent(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.inlineContent = "dmxlc3M6Ly9hYmNAZXhhbXBsZS5jb206NDQzP3NlY3VyaXR5PXRscyNOb2RlQQ=="

	label, value := wizard.previewSource()
	if label != "直接粘贴内容" {
		t.Fatalf("label = %q", label)
	}
	if !strings.Contains(value, "Base64 节点订阅") {
		t.Fatalf("value = %q", value)
	}
}

func TestProtocolBadgeNormalizesCase(t *testing.T) {
	if got := protocolBadge("vless"); !strings.Contains(got, "Vless") {
		t.Fatalf("protocolBadge(vless) = %q", got)
	}
}
