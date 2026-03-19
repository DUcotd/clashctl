// Package ui provides the main Bubble Tea wizard model.
package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

// WizardModel is the main TUI state.
type WizardModel struct {
	screen   Screen
	appCfg   *core.AppConfig
	width    int
	height   int
	quitting bool

	// Subscription URL input
	sourceMode  SubscriptionSource
	urlInput    textinput.Model
	fileInput   textinput.Model
	inlineInput textarea.Model

	// Mode selection
	modeIndex int // 0 = TUN, 1 = mixed-port

	// Advanced settings
	advancedIndex  int
	advancedFields []string
	advancedInputs []textinput.Model

	// Result
	execSteps         []ExecStep
	execError         string
	canImportFallback bool
	importHint        string
	localImportPath   string
	inlineContent     string
	importInput       textinput.Model
	loadError         string

	// Controller availability (set after execution)
	controllerAvailable bool

	// Node selection state
	spinner       spinner.Model
	groups        []GroupItem
	groupIndex    int
	nodes         []NodeItem
	nodeIndex     int
	selectedGroup string
	loading       bool
	loadingMsg    string
	switchResult  string
	switchSuccess bool

	// Node testing state
	testing   bool
	testTotal int
	testDone  int

	// Viewport for scrollable lists
	vp            viewport.Model
	vpReady       bool
	screenOffsets map[Screen]int
}

// ExecStep represents a single execution step result.
type ExecStep struct {
	Label   string
	Success bool
	Detail  string
}

// --- Async messages ---

type groupsLoadedMsg struct {
	groups []GroupItem
	err    string
}

type nodesLoadedMsg struct {
	nodes []NodeItem
	err   string
}

type nodeSwitchedMsg struct {
	success bool
	err     string
}

// NewWizard creates a new WizardModel with defaults or persisted values.
func NewWizard(appCfg *core.AppConfig) WizardModel {
	if appCfg == nil {
		appCfg = core.DefaultAppConfig()
	}

	modeIndex := 1
	if appCfg.Mode == "tun" {
		modeIndex = 0
	}

	// URL input
	urlInput := textinput.New()
	// bubbles/textinput v0.18.0 can panic when Width is set and the placeholder
	// contains wide runes because it slices by display width instead of rune count.
	// Keep placeholders ASCII-only until that upstream behavior is fixed.
	urlInput.Placeholder = "https://example.com/sub or /path/to/sub.txt"
	urlInput.SetValue(appCfg.SubscriptionURL)
	urlInput.Focus()
	urlInput.Width = 60
	urlInput.Prompt = "› "
	urlInput.PromptStyle = InputStyle
	urlInput.TextStyle = InputStyle

	fileInput := textinput.New()
	fileInput.Placeholder = "/path/to/sub.txt"
	fileInput.Width = 60
	fileInput.Prompt = "› "
	fileInput.PromptStyle = InputStyle
	fileInput.TextStyle = InputStyle

	inlineInput := textarea.New()
	inlineInput.Placeholder = "Paste base64, raw vless:// links, trojan:// links, hysteria2:// links, or Clash/Mihomo YAML here"
	inlineInput.Prompt = "› "
	inlineInput.ShowLineNumbers = false
	inlineInput.SetWidth(60)
	inlineInput.SetHeight(8)
	inlineInput.CharLimit = 0
	inlineInput.MaxHeight = 12

	// Advanced fields
	fields := []string{
		"配置目录",
		"控制器地址",
		"mixed-port",
		"Provider 路径",
		"健康检查",
		"systemd 服务",
		"自动启动",
	}
	advInputs := make([]textinput.Model, len(fields))
	for i, label := range fields {
		ti := textinput.New()
		ti.Width = 40
		ti.Prompt = "› "
		ti.PromptStyle = InputStyle
		ti.TextStyle = InputStyle
		switch label {
		case "配置目录":
			ti.SetValue(appCfg.ConfigDir)
		case "控制器地址":
			ti.SetValue(appCfg.ControllerAddr)
		case "mixed-port":
			ti.SetValue(fmt.Sprintf("%d", appCfg.MixedPort))
		case "Provider 路径":
			ti.SetValue(appCfg.ProviderPath)
		case "健康检查":
			ti.SetValue(boolToYesNo(appCfg.EnableHealthCheck))
		case "systemd 服务":
			ti.SetValue(boolToYesNo(appCfg.EnableSystemd))
		case "自动启动":
			ti.SetValue(boolToYesNo(appCfg.AutoStart))
		}
		advInputs[i] = ti
	}

	// Spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	importInput := textinput.New()
	importInput.Placeholder = "/path/to/sub.txt"
	importInput.Width = 60
	importInput.Prompt = "› "
	importInput.PromptStyle = InputStyle
	importInput.TextStyle = InputStyle

	return WizardModel{
		screen:         ScreenWelcome,
		appCfg:         appCfg,
		sourceMode:     SubscriptionSourceURL,
		modeIndex:      modeIndex,
		urlInput:       urlInput,
		fileInput:      fileInput,
		inlineInput:    inlineInput,
		advancedFields: fields,
		advancedInputs: advInputs,
		spinner:        s,
		importInput:    importInput,
		screenOffsets:  make(map[Screen]int),
	}
}

func (m WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureViewport()
		return m, nil
	case tea.MouseMsg:
		// Forward mouse events (scroll wheel) to viewport
		if m.usesViewport() {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			m.screenOffsets[m.screen] = m.vp.YOffset
			return m, cmd
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case executionDoneMsg:
		m.execSteps = msg.steps
		m.controllerAvailable = msg.controllerReady
		m.canImportFallback = msg.canImport
		m.importHint = msg.importHint
		m.setScreen(ScreenResult)
		return m, nil
	case groupsLoadedMsg:
		return m.handleGroupsLoaded(msg)
	case nodesLoadedMsg:
		return m.handleNodesLoaded(msg)
	case nodeSwitchedMsg:
		return m.handleNodeSwitched(msg)
	case nodeTestedMsg:
		return m.handleNodeTested(msg)
	}
	return m, nil
}

func (m WizardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	switch m.screen {
	case ScreenWelcome:
		return m.updateWelcome(msg)
	case ScreenSubscription:
		return m.updateSubscription(msg)
	case ScreenMode:
		return m.updateMode(msg)
	case ScreenAdvanced:
		return m.updateAdvanced(msg)
	case ScreenPreview:
		return m.updatePreview(msg)
	case ScreenExecution:
		// Block all keys during execution — just wait
		return m, nil
	case ScreenResult:
		return m.updateResult(msg)
	case ScreenImportLocal:
		return m.updateImportLocal(msg)
	case ScreenGroupSelect:
		return m.updateGroupSelect(msg)
	case ScreenNodeSelect:
		return m.updateNodeSelect(msg)
	}
	return m, nil
}

func (m *WizardModel) ensureViewport() {
	if !m.vpReady {
		m.vp = viewport.New(1, 1)
		m.vpReady = true
	}
	innerWidth, innerHeight := m.baseViewportSize()
	m.vp.Width = innerWidth
	m.vp.Height = innerHeight
	if off, ok := m.screenOffsets[m.screen]; ok {
		m.vp.SetYOffset(off)
	}
}

func (m *WizardModel) setScreen(screen Screen) {
	if m.vpReady {
		m.screenOffsets[m.screen] = m.vp.YOffset
	}
	m.screen = screen
	if m.vpReady {
		m.ensureViewport()
	}
}

func (m WizardModel) usesViewport() bool {
	switch m.screen {
	case ScreenWelcome, ScreenMode, ScreenPreview, ScreenResult, ScreenGroupSelect, ScreenNodeSelect:
		return true
	default:
		return false
	}
}

func (m WizardModel) baseViewportSize() (int, int) {
	innerWidth := max(24, m.width-BoxStyle.GetHorizontalFrameSize()-4)
	topChrome := 4
	if m.screen != ScreenWelcome {
		topChrome = 6
	}
	innerHeight := max(6, m.height-topChrome-BoxStyle.GetVerticalFrameSize()-2)
	return innerWidth, innerHeight
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m WizardModel) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.focusSubscriptionInput()
		m.setScreen(ScreenSubscription)
		return m, nil
	case "q", "esc":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m WizardModel) updateSubscription(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "shift+tab":
		m.setSubscriptionSource((m.sourceMode + 2) % 3)
		return m, nil
	case "right", "tab":
		m.setSubscriptionSource((m.sourceMode + 1) % 3)
		return m, nil
	case "enter":
		if m.sourceMode == SubscriptionSourceInline {
			var cmd tea.Cmd
			m.inlineInput, cmd = m.inlineInput.Update(msg)
			return m, cmd
		}
		if !m.commitSubscriptionSelection() {
			return m, nil
		}
		m.setScreen(ScreenMode)
		return m, nil
	case "ctrl+s":
		if m.sourceMode == SubscriptionSourceInline {
			if !m.commitSubscriptionSelection() {
				return m, nil
			}
			m.setScreen(ScreenMode)
			return m, nil
		}
	case "esc":
		m.setScreen(ScreenWelcome)
		return m, nil
	default:
		return m.updateSubscriptionInput(msg)
	}
	return m, nil
}

func (m *WizardModel) setSubscriptionSource(source SubscriptionSource) {
	m.sourceMode = source
	m.focusSubscriptionInput()
}

func (m *WizardModel) focusSubscriptionInput() {
	m.urlInput.Blur()
	m.fileInput.Blur()
	m.inlineInput.Blur()
	switch m.sourceMode {
	case SubscriptionSourceURL:
		m.urlInput.Focus()
	case SubscriptionSourceFile:
		m.fileInput.Focus()
	case SubscriptionSourceInline:
		_ = m.inlineInput.Focus()
	}
}

func (m *WizardModel) updateSubscriptionInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.sourceMode {
	case SubscriptionSourceURL:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	case SubscriptionSourceFile:
		var cmd tea.Cmd
		m.fileInput, cmd = m.fileInput.Update(msg)
		return m, cmd
	case SubscriptionSourceInline:
		var cmd tea.Cmd
		m.inlineInput, cmd = m.inlineInput.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m *WizardModel) commitSubscriptionSelection() bool {
	m.appCfg.SubscriptionURL = ""
	m.localImportPath = ""
	m.inlineContent = ""

	switch m.sourceMode {
	case SubscriptionSourceURL:
		input := strings.TrimSpace(m.urlInput.Value())
		if input == "" {
			return false
		}
		m.appCfg.SubscriptionURL = input
		return true
	case SubscriptionSourceFile:
		input := strings.TrimSpace(m.fileInput.Value())
		if input == "" {
			return false
		}
		m.localImportPath = input
		return true
	case SubscriptionSourceInline:
		input := strings.TrimSpace(m.inlineInput.Value())
		if input == "" {
			return false
		}
		m.inlineContent = input
		return true
	default:
		return false
	}
}

func (m WizardModel) updateMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "pgup", "pgdown", "home", "end":
		m.scrollViewport(msg.String())
		return m, nil
	case "up", "k":
		if m.modeIndex > 0 {
			m.modeIndex--
		}
	case "down", "j":
		if m.modeIndex < 1 {
			m.modeIndex++
		}
	case "enter":
		if m.modeIndex == 0 {
			m.appCfg.Mode = "tun"
		} else {
			m.appCfg.Mode = "mixed"
		}
		m.setScreen(ScreenAdvanced)
		return m, nil
	case "esc":
		m.setScreen(ScreenSubscription)
		return m, nil
	}
	return m, nil
}

func (m WizardModel) updateAdvanced(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.advancedIndex > 0 {
			m.advancedIndex--
		}
	case "down", "j":
		if m.advancedIndex < len(m.advancedFields)-1 {
			m.advancedIndex++
		}
	case "enter":
		m.collectAdvancedValues()
		m.setScreen(ScreenPreview)
		return m, nil
	case "esc":
		m.setScreen(ScreenMode)
		return m, nil
	default:
		if m.advancedIndex < len(m.advancedInputs) {
			var cmd tea.Cmd
			m.advancedInputs[m.advancedIndex], cmd = m.advancedInputs[m.advancedIndex].Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *WizardModel) collectAdvancedValues() {
	for i, field := range m.advancedFields {
		val := m.advancedInputs[i].Value()
		switch field {
		case "配置目录":
			m.appCfg.ConfigDir = val
		case "控制器地址":
			m.appCfg.ControllerAddr = val
		case "mixed-port":
			var port int
			if _, err := fmt.Sscanf(val, "%d", &port); err == nil && port > 0 {
				m.appCfg.MixedPort = port
			}
		case "Provider 路径":
			m.appCfg.ProviderPath = val
		case "健康检查":
			m.appCfg.EnableHealthCheck = (val == "是" || val == "yes" || val == "true" || val == "1")
		case "systemd 服务":
			m.appCfg.EnableSystemd = (val == "是" || val == "yes" || val == "true" || val == "1")
		case "自动启动":
			m.appCfg.AutoStart = (val == "是" || val == "yes" || val == "true" || val == "1")
		}
	}
}

func (m WizardModel) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
		m.scrollViewport(msg.String())
		return m, nil
	case "enter":
		m.setScreen(ScreenExecution)
		if strings.TrimSpace(m.localImportPath) != "" {
			return m, tea.Batch(m.spinner.Tick, m.runImportExecution(m.localImportPath))
		}
		if strings.TrimSpace(m.inlineContent) != "" {
			return m, tea.Batch(m.spinner.Tick, m.runInlineExecution(m.inlineContent))
		}
		return m, tea.Batch(m.spinner.Tick, m.runExecution())
	case "esc":
		m.setScreen(ScreenAdvanced)
		return m, nil
	}
	return m, nil
}

// runExecution runs executeFull as an async background command.
func (m WizardModel) runExecution() tea.Cmd {
	return func() tea.Msg {
		steps := m.executeFull()
		canImport, importHint := detectImportFallback(steps)
		return executionDoneMsg{
			steps:           steps,
			controllerReady: m.controllerAvailable,
			canImport:       canImport,
			importHint:      importHint,
		}
	}
}

func (m WizardModel) runImportExecution(filePath string) tea.Cmd {
	return func() tea.Msg {
		steps := m.executeImport(filePath)
		canImport, importHint := detectImportFallback(steps)
		return executionDoneMsg{
			steps:           steps,
			controllerReady: m.controllerAvailable,
			canImport:       canImport,
			importHint:      importHint,
		}
	}
}

func (m WizardModel) runInlineExecution(content string) tea.Cmd {
	return func() tea.Msg {
		steps := m.executeInlineContent(content)
		canImport, importHint := detectImportFallback(steps)
		return executionDoneMsg{
			steps:           steps,
			controllerReady: m.controllerAvailable,
			canImport:       canImport,
			importHint:      importHint,
		}
	}
}

func detectImportFallback(steps []ExecStep) (bool, string) {
	for _, step := range steps {
		if step.Label == "验证代理节点加载" && !step.Success {
			return true, step.Detail
		}
	}
	return false, ""
}

func (m WizardModel) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
		m.scrollViewport(msg.String())
		return m, nil
	case "enter", "q":
		if m.controllerAvailable {
			// Go to node selection instead of quitting
			m.loading = true
			m.loadingMsg = "正在加载代理组..."
			m.loadError = ""
			m.setScreen(ScreenGroupSelect)
			return m, tea.Batch(m.spinner.Tick, m.loadGroups())
		}
		m.quitting = true
		return m, tea.Quit
	case "i":
		if m.canImportFallback {
			m.importInput.Focus()
			m.setScreen(ScreenImportLocal)
			return m, nil
		}
	case "esc":
		m.quitting = true
		return m, tea.Quit
	case "n":
		if m.controllerAvailable {
			m.loading = true
			m.loadingMsg = "正在加载代理组..."
			m.loadError = ""
			m.setScreen(ScreenGroupSelect)
			return m, tea.Batch(m.spinner.Tick, m.loadGroups())
		}
	}
	return m, nil
}

func (m *WizardModel) scrollViewport(key string) {
	if !m.vpReady {
		return
	}
	switch key {
	case "up", "k":
		m.vp.LineUp(1)
	case "down", "j":
		m.vp.LineDown(1)
	case "pgup":
		m.vp.HalfViewUp()
	case "pgdown":
		m.vp.HalfViewDown()
	case "home":
		m.vp.GotoTop()
	case "end":
		m.vp.GotoBottom()
	}
	m.screenOffsets[m.screen] = m.vp.YOffset
}

func (m WizardModel) updateImportLocal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.importInput.Value())
		if path == "" {
			return m, nil
		}
		m.setScreen(ScreenExecution)
		m.importInput.Blur()
		return m, tea.Batch(m.spinner.Tick, m.runImportExecution(path))
	case "esc":
		m.importInput.Blur()
		m.setScreen(ScreenResult)
		return m, nil
	default:
		var cmd tea.Cmd
		m.importInput, cmd = m.importInput.Update(msg)
		return m, cmd
	}
}

func (m WizardModel) updateGroupSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.groupIndex > 0 {
			m.groupIndex--
		}
	case "down", "j":
		if m.groupIndex < len(m.groups)-1 {
			m.groupIndex++
		}
	case "pgup":
		m.groupIndex -= m.vp.Height
		if m.groupIndex < 0 {
			m.groupIndex = 0
		}
	case "pgdown":
		m.groupIndex += m.vp.Height
		if m.groupIndex >= len(m.groups) {
			m.groupIndex = len(m.groups) - 1
		}
	case "home":
		m.groupIndex = 0
	case "end":
		m.groupIndex = len(m.groups) - 1
	case "enter":
		if len(m.groups) > 0 {
			m.selectedGroup = m.groups[m.groupIndex].Name
			m.loading = true
			m.loadingMsg = "正在加载节点..."
			m.loadError = ""
			m.nodeIndex = 0
			return m, tea.Batch(m.spinner.Tick, m.loadNodes(m.selectedGroup))
		}
	case "r":
		// Refresh groups
		m.loading = true
		m.loadingMsg = "正在刷新代理组..."
		m.loadError = ""
		return m, tea.Batch(m.spinner.Tick, m.loadGroups())
	case "esc":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m WizardModel) updateNodeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading || m.testing {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.nodeIndex > 0 {
			m.nodeIndex--
		}
	case "down", "j":
		if m.nodeIndex < len(m.nodes)-1 {
			m.nodeIndex++
		}
	case "pgup":
		m.nodeIndex -= m.vp.Height
		if m.nodeIndex < 0 {
			m.nodeIndex = 0
		}
	case "pgdown":
		m.nodeIndex += m.vp.Height
		if m.nodeIndex >= len(m.nodes) {
			m.nodeIndex = len(m.nodes) - 1
		}
	case "home":
		m.nodeIndex = 0
	case "end":
		m.nodeIndex = len(m.nodes) - 1
	case "enter":
		if len(m.nodes) > 0 {
			nodeName := m.nodes[m.nodeIndex].Name
			m.loading = true
			m.loadingMsg = "正在切换节点..."
			return m, tea.Batch(m.spinner.Tick, m.switchNode(m.selectedGroup, nodeName))
		}
	case "t":
		// Test all nodes
		if len(m.nodes) > 0 {
			m.testing = true
			m.testTotal = len(m.nodes)
			m.testDone = 0
			m.switchResult = ""
			return m, tea.Batch(m.spinner.Tick, m.testAllNodes())
		}
	case "r":
		// Refresh nodes
		m.loading = true
		m.loadingMsg = "正在刷新节点..."
		m.loadError = ""
		return m, tea.Batch(m.spinner.Tick, m.loadNodes(m.selectedGroup))
	case "esc":
		// Back to group list
		m.setScreen(ScreenGroupSelect)
		m.switchResult = ""
		return m, nil
	}
	return m, nil
}

// --- Async commands ---

func (m WizardModel) loadGroups() tea.Cmd {
	return func() tea.Msg {
		client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)
		groups, err := client.GetAllProxyGroups()
		if err != nil {
			return groupsLoadedMsg{err: err.Error()}
		}

		items := make([]GroupItem, 0, len(groups))
		for name, g := range groups {
			items = append(items, GroupItem{
				Name:      name,
				Type:      g.Type,
				Now:       g.Now,
				NodeCount: len(g.All),
			})
		}
		sort.Slice(items, func(i, j int) bool {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		})

		return groupsLoadedMsg{groups: items}
	}
}

func (m WizardModel) loadNodes(groupName string) tea.Cmd {
	return func() tea.Msg {
		client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)
		detail, err := client.GetProxyGroupDetail(groupName)
		if err != nil {
			return nodesLoadedMsg{err: err.Error()}
		}

		// Fetch all proxies to get protocol types (optional, best-effort)
		typeMap := make(map[string]string)
		if allProxies, err := client.GetAllProxies(); err == nil {
			for name, info := range allProxies {
				typeMap[name] = info.Type
			}
		}

		items := make([]NodeItem, 0, len(detail.All))
		for _, name := range detail.All {
			node := NodeItem{
				Name:     name,
				Protocol: typeMap[name],
				Selected: name == detail.Now,
			}
			items = append(items, node)
		}

		return nodesLoadedMsg{nodes: items}
	}
}

func (m WizardModel) switchNode(groupName, nodeName string) tea.Cmd {
	return func() tea.Msg {
		client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)
		err := client.SwitchProxy(groupName, nodeName)
		if err != nil {
			return nodeSwitchedMsg{success: false, err: err.Error()}
		}
		return nodeSwitchedMsg{success: true}
	}
}

func (m WizardModel) handleGroupsLoaded(msg groupsLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != "" {
		m.loadError = "加载代理组失败: " + msg.err
		return m, nil
	}
	m.loadError = ""
	m.groups = msg.groups
	m.groupIndex = 0
	return m, nil
}

func (m WizardModel) handleNodesLoaded(msg nodesLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != "" {
		m.loadError = "加载节点失败: " + msg.err
		return m, nil
	}
	m.loadError = ""
	m.nodes = msg.nodes
	m.nodeIndex = 0
	// Move to the currently selected node
	for i, n := range m.nodes {
		if n.Selected {
			m.nodeIndex = i
			break
		}
	}
	m.setScreen(ScreenNodeSelect)
	return m, nil
}

func (m WizardModel) handleNodeSwitched(msg nodeSwitchedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.switchSuccess = msg.success
	if msg.success {
		m.loadError = ""
		m.switchResult = "✅ 节点切换成功！"
		// Update the selected state in node list
		for i := range m.nodes {
			m.nodes[i].Selected = (i == m.nodeIndex)
		}
	} else {
		m.switchResult = "❌ 切换失败: " + msg.err
	}
	return m, nil
}

func (m WizardModel) handleNodeTested(msg nodeTestedMsg) (tea.Model, tea.Cmd) {
	for idx, delay := range msg.delays {
		if idx < len(m.nodes) {
			m.nodes[idx].Delay = delay
		}
	}
	m.testing = false
	m.loadError = ""
	m.switchResult = fmt.Sprintf("✅ 延迟测试完成 (%d 个节点)", len(msg.delays))
	return m, nil
}

// testAllNodes tests latency for all nodes with bounded concurrency.
func (m WizardModel) testAllNodes() tea.Cmd {
	return func() tea.Msg {
		client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)
		delays := make(map[int]int)
		detail, err := client.TestProxyGroupNodes(m.selectedGroup, 10)
		if err != nil {
			return nodeTestedMsg{delays: delays}
		}
		for _, tested := range detail.Nodes {
			for i, node := range m.nodes {
				if node.Name == tested.Name {
					delays[i] = tested.Delay
					break
				}
			}
		}

		return nodeTestedMsg{delays: delays}
	}
}

// View renders the current screen.
func (m WizardModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(TitleStyle.Render("🧙 clashctl 配置向导"))
	b.WriteString("\n")

	// Step indicator (except welcome)
	if m.screen != ScreenWelcome {
		b.WriteString(StepStyle.Render(m.screen.StepLabel()))
		b.WriteString("\n\n")
	}

	switch m.screen {
	case ScreenWelcome:
		b.WriteString(m.viewWelcome())
	case ScreenSubscription:
		b.WriteString(m.viewSubscription())
	case ScreenMode:
		b.WriteString(m.viewMode())
	case ScreenAdvanced:
		b.WriteString(m.viewAdvanced())
	case ScreenPreview:
		b.WriteString(m.viewPreview())
	case ScreenExecution:
		b.WriteString(m.viewExecution())
	case ScreenResult:
		b.WriteString(m.viewResult())
	case ScreenImportLocal:
		b.WriteString(m.viewImportLocal())
	case ScreenGroupSelect:
		b.WriteString(m.viewGroupSelect())
	case ScreenNodeSelect:
		b.WriteString(m.viewNodeSelect())
	}

	return b.String()
}

// Completed returns true if the wizard finished all steps.
func (m WizardModel) Completed() bool {
	return len(m.execSteps) > 0
}
