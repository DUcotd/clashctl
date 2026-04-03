package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
	"clashctl/internal/system"
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
		{ScreenImportLocal, "附加步骤: 导入本地订阅"},
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
	if got.quitConfirm != true {
		t.Fatal("quitConfirm should be true on first q")
	}
	if cmd != nil {
		t.Fatal("no command should be returned on quit confirm")
	}

	updated2, cmd2 := got.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	got2 := updated2.(NodeManagerModel)
	if !got2.quitting {
		t.Fatal("quitting should be true after confirming")
	}
	if cmd2 == nil {
		t.Fatal("quit command should be returned after confirming")
	}
}

func TestNodeManagerQCancel(t *testing.T) {
	manager := NewNodeManager(core.DefaultAppConfig())

	updated, _ := manager.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := updated.(NodeManagerModel)
	if got.quitConfirm != true {
		t.Fatal("quitConfirm should be true")
	}

	updated2, cmd2 := got.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	got2 := updated2.(NodeManagerModel)
	if got2.quitConfirm != false {
		t.Fatal("quitConfirm should be false after canceling")
	}
	if got2.quitting {
		t.Fatal("quitting should not be set after canceling")
	}
	if cmd2 != nil {
		t.Fatal("no command should be returned on cancel")
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

func TestInlineInputUsesPreparedSubscriptionLimit(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	if wizard.inlineInput.CharLimit != int(system.MaxPreparedSubscriptionBytes) {
		t.Fatalf("CharLimit = %d, want %d", wizard.inlineInput.CharLimit, system.MaxPreparedSubscriptionBytes)
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

func TestParseYesNo(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"是", true},
		{"yes", true},
		{"Yes", true},
		{"YES", true},
		{"true", true},
		{"True", true},
		{"1", true},
		{"否", false},
		{"no", false},
		{"false", false},
		{"0", false},
		{"", false},
		{" maybe ", false},
	}
	for _, tt := range tests {
		if got := parseYesNo(tt.input); got != tt.want {
			t.Errorf("parseYesNo(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestDelayStyle(t *testing.T) {
	tests := []struct {
		delay int
		name  string
	}{
		{0, "unknown"},
		{-1, "bad"},
		{50, "good"},
		{99, "good"},
		{100, "ok"},
		{299, "ok"},
		{300, "ok"},
		{999, "ok"},
		{1000, "bad"},
		{5000, "bad"},
	}
	for _, tt := range tests {
		got := delayStyle(tt.delay).Render("test")
		if got == "" {
			t.Errorf("delayStyle(%d) returned empty", tt.delay)
		}
	}
}

func TestProtocolBadgeEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"VLESS", "Vless"},
		{"  hy2  ", "Hy2"},
		{"Trojan", "Trojan"},
		{"vmess", "VMess"},
		{"SS", "SS"},
		{"unknown-protocol-very-long-name", "unknown-pr"},
		{"", ""},
	}
	for _, tt := range tests {
		got := protocolBadge(tt.input)
		if tt.contains != "" && !strings.Contains(got, tt.contains) {
			t.Errorf("protocolBadge(%q) = %q, want to contain %q", tt.input, got, tt.contains)
		}
	}
}

func TestWrapTextEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		check func([]string) bool
	}{
		{"empty", "", 20, func(lines []string) bool { return len(lines) == 1 && lines[0] == "" }},
		{"short", "hi", 20, func(lines []string) bool { return len(lines) == 1 && lines[0] == "hi" }},
		{"narrow width", "hello world foo bar baz", 10, func(lines []string) bool { return len(lines) >= 2 }},
		{"width less than 8", "hello world", 4, func(lines []string) bool { return len(lines) == 1 }},
		{"cjk text", "你好世界", 6, func(lines []string) bool { return len(lines) >= 1 }},
		{"only newlines", "\n\n", 20, func(lines []string) bool { return len(lines) == 3 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.text, tt.width)
			if !tt.check(got) {
				t.Errorf("wrapText(%q, %d) = %#v, check failed", tt.text, tt.width, got)
			}
		})
	}
}

func TestFormatKV(t *testing.T) {
	got := formatKV("订阅 URL", "https://example.com/sub", 60)
	if !strings.Contains(got, "订阅 URL") {
		t.Fatalf("formatKV should contain label, got %q", got)
	}
	if !strings.Contains(got, "https://example.com/sub") {
		t.Fatalf("formatKV should contain value, got %q", got)
	}
}

func TestFormatKVEmptyValue(t *testing.T) {
	got := formatKV("Label", "", 40)
	if !strings.Contains(got, "Label:") {
		t.Fatalf("formatKV with empty value should contain label, got %q", got)
	}
}

func TestApplySort(t *testing.T) {
	m := newNodeManagerWithService(core.DefaultAppConfig(), &fakeNodeService{}, false)
	m.nodes = []NodeItem{
		{Name: "Zebra", Protocol: "vless", Delay: 200},
		{Name: "Alpha", Protocol: "trojan", Delay: 50},
		{Name: "Beta", Protocol: "vless", Delay: 0},
	}
	m.filteredNodes = append([]NodeItem(nil), m.nodes...)

	m.sortMode = NodeSortDelay
	m.applySort()
	if m.filteredNodes[0].Name != "Alpha" {
		t.Errorf("sort by delay: first = %q, want Alpha", m.filteredNodes[0].Name)
	}
	if m.filteredNodes[2].Name != "Beta" {
		t.Errorf("sort by delay: last (delay=0) = %q, want Beta", m.filteredNodes[2].Name)
	}

	m.sortMode = NodeSortName
	m.filteredNodes = append([]NodeItem(nil), m.nodes...)
	m.applySort()
	if m.filteredNodes[0].Name != "Alpha" {
		t.Errorf("sort by name: first = %q, want Alpha", m.filteredNodes[0].Name)
	}

	m.sortMode = NodeSortProtocol
	m.filteredNodes = append([]NodeItem(nil), m.nodes...)
	m.applySort()
	if m.filteredNodes[0].Protocol != "trojan" {
		t.Errorf("sort by protocol: first = %q, want trojan", m.filteredNodes[0].Protocol)
	}
}

func TestCycleSortMode(t *testing.T) {
	m := newNodeManagerWithService(core.DefaultAppConfig(), &fakeNodeService{}, false)
	m.nodes = []NodeItem{
		{Name: "B", Protocol: "vless", Delay: 200},
		{Name: "A", Protocol: "trojan", Delay: 50},
	}

	expected := []NodeSortMode{NodeSortDelay, NodeSortName, NodeSortProtocol, NodeSortDefault}
	for i, want := range expected {
		m.cycleSortMode()
		if m.sortMode != want {
			t.Errorf("cycle %d: sortMode = %v, want %v", i+1, m.sortMode, want)
		}
	}
	if m.filteredNodes != nil {
		t.Error("after cycling back to default, filteredNodes should be nil")
	}
}

func TestHandleMouseClick(t *testing.T) {
	m := newNodeManagerWithService(core.DefaultAppConfig(), &fakeNodeService{
		groups: []GroupItem{
			{Name: "PROXY", Type: "select", NodeCount: 2},
			{Name: "AUTO", Type: "url-test", NodeCount: 3},
		},
	}, false)
	m.width = 80
	m.height = 24
	m.screen = ScreenGroupSelect
	m.loading = false
	m.ensureViewport()

	msg := tea.MouseMsg{Y: 5, Type: tea.MouseRelease, Action: tea.MouseActionPress}
	updated, _ := m.handleMouseClick(msg)
	got := updated.(NodeManagerModel)
	if got.groupIndex != 0 {
		t.Errorf("click on first group: groupIndex = %d, want 0", got.groupIndex)
	}

	negMsg := tea.MouseMsg{Y: -1, Type: tea.MouseRelease, Action: tea.MouseActionPress}
	updated2, _ := got.handleMouseClick(negMsg)
	got2 := updated2.(NodeManagerModel)
	if got2.groupIndex != 0 {
		t.Errorf("negative Y click should not change groupIndex, got %d", got2.groupIndex)
	}
}

func TestRenderStatusBar(t *testing.T) {
	m := newNodeManagerWithService(core.DefaultAppConfig(), &fakeNodeService{
		groups: []GroupItem{{Name: "PROXY", Type: "select", NodeCount: 2}},
	}, false)
	m.width = 80
	m.height = 24
	m.screen = ScreenGroupSelect
	m.loading = false
	m.groups = []GroupItem{{Name: "PROXY", Type: "select", NodeCount: 2}}
	m.ensureViewport()

	got := m.renderStatusBar()
	if !strings.Contains(got, "PROXY") {
		t.Fatalf("status bar missing group name: %s", got)
	}

	m.screen = ScreenNodeSelect
	m.selectedGroup = "PROXY"
	m.nodes = []NodeItem{{Name: "Node A", Delay: 50}}
	m.nodeIndex = 0
	got2 := m.renderStatusBar()
	if !strings.Contains(got2, "节点: 1/1") {
		t.Fatalf("status bar missing node info: %s", got2)
	}
}

func TestRenderStatusBarNarrowTerminal(t *testing.T) {
	m := newNodeManagerWithService(core.DefaultAppConfig(), &fakeNodeService{}, false)
	m.width = 2
	m.height = 10
	m.screen = ScreenGroupSelect

	got := m.renderStatusBar()
	if got != "" {
		t.Fatalf("status bar on narrow terminal should be empty, got %q", got)
	}
}

func TestRenderStatusBarEmptyNodes(t *testing.T) {
	m := newNodeManagerWithService(core.DefaultAppConfig(), &fakeNodeService{}, false)
	m.width = 80
	m.height = 24
	m.screen = ScreenNodeSelect
	m.selectedGroup = "PROXY"
	m.nodes = []NodeItem{}

	got := m.renderStatusBar()
	if strings.Contains(got, "1/0") {
		t.Fatalf("status bar should not show 1/0 for empty nodes: %s", got)
	}
	if !strings.Contains(got, "节点: 0 个") {
		t.Fatalf("status bar should show 0 nodes: %s", got)
	}
}

func TestViewportFollowSelected(t *testing.T) {
	v := viewportState{
		vpReady:       true,
		screenOffsets: make(map[Screen]int),
	}
	v.vp.Width = 40
	v.vp.Height = 5
	v.vp.SetContent("line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10")

	v.followSelected(8)
	if v.vp.YOffset != 4 {
		t.Errorf("followSelected(8) with height 5: YOffset = %d, want 4", v.vp.YOffset)
	}

	v.followSelected(0)
	if v.vp.YOffset != 0 {
		t.Errorf("followSelected(0): YOffset = %d, want 0", v.vp.YOffset)
	}
}

func TestViewportScroll(t *testing.T) {
	v := viewportState{
		vpReady:       true,
		screenOffsets: make(map[Screen]int),
	}
	v.vp.Width = 40
	v.vp.Height = 5
	v.vp.SetContent("line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10")

	v.scroll("down")
	if v.vp.YOffset != 1 {
		t.Errorf("scroll down: YOffset = %d, want 1", v.vp.YOffset)
	}

	v.scroll("up")
	if v.vp.YOffset != 0 {
		t.Errorf("scroll up: YOffset = %d, want 0", v.vp.YOffset)
	}

	v.scroll("end")
	if v.vp.YOffset+v.vp.Height < v.vp.TotalLineCount() {
		t.Errorf("scroll end: YOffset+Height = %d, should reach bottom", v.vp.YOffset+v.vp.Height)
	}

	v.scroll("home")
	if v.vp.YOffset != 0 {
		t.Errorf("scroll home: YOffset = %d, want 0", v.vp.YOffset)
	}
}

func TestNodeManagerQuitConfirm(t *testing.T) {
	m := newNodeManagerWithService(core.DefaultAppConfig(), &fakeNodeService{}, false)

	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := updated.(NodeManagerModel)
	if !got.quitConfirm {
		t.Fatal("quitConfirm should be true on first q")
	}
	if cmd != nil {
		t.Fatal("no command on quit confirm")
	}

	updated2, cmd2 := got.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	got2 := updated2.(NodeManagerModel)
	if got2.quitConfirm {
		t.Fatal("quitConfirm should be false after cancel")
	}
	if cmd2 != nil {
		t.Fatal("no command on cancel")
	}

	updated3, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got3 := updated3.(NodeManagerModel)
	updated4, cmd4 := got3.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	got4 := updated4.(NodeManagerModel)
	if !got4.quitting {
		t.Fatal("quitting should be true after confirm")
	}
	if cmd4 == nil {
		t.Fatal("quit command expected after confirm")
	}
}

func TestNodeSortModeLabel(t *testing.T) {
	tests := []struct {
		mode NodeSortMode
		want string
	}{
		{NodeSortDefault, "默认"},
		{NodeSortDelay, "延迟"},
		{NodeSortName, "名称"},
		{NodeSortProtocol, "协议"},
	}
	for _, tt := range tests {
		if got := tt.mode.Label(); got != tt.want {
			t.Errorf("NodeSortMode(%d).Label() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestWizardQuitConfirmView(t *testing.T) {
	wizard := NewWizard(core.DefaultAppConfig())
	wizard.quitConfirm = true

	view := wizard.View()
	if !strings.Contains(view, "确认退出") {
		t.Fatalf("wizard quit confirm view missing '确认退出': %s", view)
	}
}

func TestNodeManagerQuitConfirmView(t *testing.T) {
	m := newNodeManagerWithService(core.DefaultAppConfig(), &fakeNodeService{}, false)
	m.quitConfirm = true

	view := m.View()
	if !strings.Contains(view, "确认退出") {
		t.Fatalf("node manager quit confirm view missing '确认退出': %s", view)
	}
}

func TestNodeManagerHelpView(t *testing.T) {
	m := newNodeManagerWithService(core.DefaultAppConfig(), &fakeNodeService{}, false)
	m.showHelp = true

	view := m.View()
	if !strings.Contains(view, "快捷键帮助") {
		t.Fatalf("node manager help view missing '快捷键帮助': %s", view)
	}
	if !strings.Contains(view, "搜索/过滤节点") {
		t.Fatalf("node manager help view missing '搜索/过滤节点': %s", view)
	}
}

func TestPageFeedbackSetInfo(t *testing.T) {
	f := &pageFeedbackState{}
	f.setInfo("test info message")
	if f.messageText != "test info message" {
		t.Errorf("messageText = %q, want %q", f.messageText, "test info message")
	}
	if f.errorText != "" {
		t.Errorf("errorText should be empty, got %q", f.errorText)
	}
	if f.messageTone != feedbackInfo {
		t.Errorf("messageTone = %v, want %v", f.messageTone, feedbackInfo)
	}
}
