// Package ui provides the main Bubble Tea wizard model.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

// boolToInt converts a bool to int (false=0, true=1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// WizardModel is the main TUI state.
type WizardModel struct {
	screen    Screen
	appCfg    *core.AppConfig
	width     int
	height    int
	quitting  bool

	// Subscription URL input
	urlInput textinput.Model

	// Mode selection
	modeIndex int // 0 = TUN, 1 = mixed-port

	// Advanced settings
	advancedIndex  int
	advancedFields []string
	advancedInputs []textinput.Model

	// Result
	execSteps []ExecStep
	execError string

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
	vp       viewport.Model
	vpReady  bool
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

// NewWizard creates a new WizardModel with defaults.
func NewWizard() WizardModel {
	// URL input
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/subscription"
	urlInput.Focus()
	urlInput.Width = 60
	urlInput.Prompt = "› "
	urlInput.PromptStyle = InputStyle
	urlInput.TextStyle = InputStyle

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
			ti.SetValue("/etc/mihomo")
		case "控制器地址":
			ti.SetValue("127.0.0.1:9090")
		case "mixed-port":
			ti.SetValue("7890")
		case "Provider 路径":
			ti.SetValue("./providers/airport.yaml")
		case "健康检查":
			ti.SetValue("是")
		case "systemd 服务":
			ti.SetValue("是")
		case "自动启动":
			ti.SetValue("是")
		}
		advInputs[i] = ti
	}

	// Spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	// Auto-detect TUN capability — default to mixed-port for reliability
	appCfg := core.DefaultAppConfig()
	// Default to mixed-port (modeIndex=1) unless user explicitly wants TUN
	appCfg.Mode = "mixed"

	return WizardModel{
		screen:         ScreenWelcome,
		appCfg:         appCfg,
		modeIndex:      1, // default to mixed-port for reliability
		urlInput:       urlInput,
		advancedFields: fields,
		advancedInputs: advInputs,
		spinner:        s,
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

		// Initialize or update viewport for scrollable lists
		headerHeight := 5 // title + step indicator + padding
		footerHeight := 3 // help text + padding
		vpHeight := msg.Height - headerHeight - footerHeight
		if vpHeight < 5 {
			vpHeight = 5
		}

		if !m.vpReady {
			m.vp = viewport.New(msg.Width, vpHeight)
			m.vpReady = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = vpHeight
		}

		return m, nil
	case tea.MouseMsg:
		// Forward mouse events (scroll wheel) to viewport
		if m.vpReady && (m.screen == ScreenGroupSelect || m.screen == ScreenNodeSelect) {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
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
		m.screen = ScreenResult
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
	case ScreenGroupSelect:
		return m.updateGroupSelect(msg)
	case ScreenNodeSelect:
		return m.updateNodeSelect(msg)
	}
	return m, nil
}

func (m WizardModel) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.screen = ScreenSubscription
		return m, nil
	case "q", "esc":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m WizardModel) updateSubscription(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			return m, nil
		}
		m.appCfg.SubscriptionURL = url
		m.screen = ScreenMode
		return m, nil
	case "esc":
		m.screen = ScreenWelcome
		return m, nil
	default:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}
}

func (m WizardModel) updateMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
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
		m.screen = ScreenAdvanced
		return m, nil
	case "esc":
		m.screen = ScreenSubscription
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
		m.screen = ScreenPreview
		return m, nil
	case "esc":
		m.screen = ScreenMode
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
	case "enter":
		m.screen = ScreenExecution
		return m, tea.Batch(m.spinner.Tick, m.runExecution())
	case "esc":
		m.screen = ScreenAdvanced
		return m, nil
	}
	return m, nil
}

// runExecution runs executeFull as an async background command.
func (m WizardModel) runExecution() tea.Cmd {
	return func() tea.Msg {
		steps := m.executeFull()
		// Check if controller is available by looking at the last step
		controllerReady := false
		if len(steps) > 0 {
			last := steps[len(steps)-1]
			controllerReady = last.Success && last.Label == "检查 Controller API"
		}
		return executionDoneMsg{steps: steps, controllerReady: controllerReady}
	}
}

func (m WizardModel) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "q":
		if m.controllerAvailable {
			// Go to node selection instead of quitting
			m.loading = true
			m.loadingMsg = "正在加载代理组..."
			m.screen = ScreenGroupSelect
			return m, tea.Batch(m.spinner.Tick, m.loadGroups())
		}
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.quitting = true
		return m, tea.Quit
	case "n":
		if m.controllerAvailable {
			m.loading = true
			m.loadingMsg = "正在加载代理组..."
			m.screen = ScreenGroupSelect
			return m, tea.Batch(m.spinner.Tick, m.loadGroups())
		}
	}
	return m, nil
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
			m.nodeIndex = 0
			return m, tea.Batch(m.spinner.Tick, m.loadNodes(m.selectedGroup))
		}
	case "r":
		// Refresh groups
		m.loading = true
		m.loadingMsg = "正在刷新代理组..."
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
		return m, tea.Batch(m.spinner.Tick, m.loadNodes(m.selectedGroup))
	case "esc":
		// Back to group list
		m.screen = ScreenGroupSelect
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

		// Fetch all proxies to get protocol types
		allProxies, _ := client.GetAllProxies()
		typeMap := make(map[string]string)
		if allProxies != nil {
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
		m.groups = nil
		return m, nil
	}
	m.groups = msg.groups
	m.groupIndex = 0
	return m, nil
}

func (m WizardModel) handleNodesLoaded(msg nodesLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != "" {
		m.nodes = nil
		return m, nil
	}
	m.nodes = msg.nodes
	m.nodeIndex = 0
	// Move to the currently selected node
	for i, n := range m.nodes {
		if n.Selected {
			m.nodeIndex = i
			break
		}
	}
	m.screen = ScreenNodeSelect
	return m, nil
}

func (m WizardModel) handleNodeSwitched(msg nodeSwitchedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.switchSuccess = msg.success
	if msg.success {
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
	m.switchResult = fmt.Sprintf("✅ 延迟测试完成 (%d 个节点)", len(msg.delays))
	return m, nil
}

// testAllNodes tests latency for all nodes with bounded concurrency.
func (m WizardModel) testAllNodes() tea.Cmd {
	return func() tea.Msg {
		client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)
		delays := make(map[int]int)

		// Limit concurrent requests to avoid overwhelming the subscription server
		const maxConcurrent = 10
		sem := make(chan struct{}, maxConcurrent)
		ch := make(chan struct {
			index int
			delay int
		}, len(m.nodes))

		for i, node := range m.nodes {
			sem <- struct{}{} // acquire slot
			go func(idx int, name string) {
				defer func() { <-sem }() // release slot
				delay := client.TestNode(m.selectedGroup, name)
				ch <- struct {
					index int
					delay int
				}{idx, delay}
			}(i, node.Name)
		}

		for range m.nodes {
			r := <-ch
			delays[r.index] = r.delay
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
