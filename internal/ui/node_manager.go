package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"clashctl/internal/core"
)

const defaultNodeTestConcurrency = 10

type nodeInteractionState struct {
	groups        []GroupItem
	groupIndex    int
	nodes         []NodeItem
	nodeIndex     int
	selectedGroup string
	loading       bool
	loadingMsg    string
	testing       bool
	testDone      int
	testTotal     int
	testStream    <-chan nodeTestProgressMsg

	searchInput   textinput.Model
	searchQuery   string
	filteredNodes []NodeItem

	confirmAction ConfirmAction
	confirmTarget string

	showHelp bool

	sortMode NodeSortMode

	groupSearchInput textinput.Model
	groupSearchQuery string
	groupFiltered    []GroupItem

	showNodeDetail  bool
	detailNodeIndex int
	quitConfirm     bool
}

// NodeManagerModel is the standalone node management TUI state.
type NodeManagerModel struct {
	screen    Screen
	appCfg    *core.AppConfig
	width     int
	height    int
	quitting  bool
	title     string
	completed bool

	spinner     spinner.Model
	nodeService NodeService
	feedback    pageFeedbackState
	nodeInteractionState
	viewportState
}

// NewNodeManager creates a standalone node-management TUI starting from group selection.
func NewNodeManager(appCfg *core.AppConfig) NodeManagerModel {
	if appCfg == nil {
		appCfg = core.DefaultAppConfig()
	}
	return newNodeManagerWithService(appCfg, newDefaultNodeService(appCfg.ControllerSecret), false)
}

func newNodeManagerWithService(appCfg *core.AppConfig, nodeSvc NodeService, completed bool) NodeManagerModel {
	if appCfg == nil {
		appCfg = core.DefaultAppConfig()
	}
	if nodeSvc == nil {
		nodeSvc = newDefaultNodeService(appCfg.ControllerSecret)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	searchInput := textinput.New()
	searchInput.Placeholder = "搜索节点..."
	searchInput.Width = 30
	searchInput.Prompt = "› "
	searchInput.PromptStyle = InputStyle
	searchInput.TextStyle = InputStyle

	groupSearchInput := textinput.New()
	groupSearchInput.Placeholder = "搜索代理组..."
	groupSearchInput.Width = 30
	groupSearchInput.Prompt = "› "
	groupSearchInput.PromptStyle = InputStyle
	groupSearchInput.TextStyle = InputStyle

	return NodeManagerModel{
		screen:      ScreenGroupSelect,
		appCfg:      appCfg,
		title:       "📡 clashctl 节点管理",
		completed:   completed,
		spinner:     s,
		nodeService: nodeSvc,
		nodeInteractionState: nodeInteractionState{
			loading:          true,
			loadingMsg:       "正在加载代理组...",
			searchInput:      searchInput,
			groupSearchInput: groupSearchInput,
		},
		viewportState: viewportState{
			screenOffsets: make(map[Screen]int),
		},
	}
}

func (m NodeManagerModel) Init() tea.Cmd {
	if m.loading {
		return tea.Batch(m.spinner.Tick, m.loadGroups())
	}
	return nil
}

func (m NodeManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureViewport()
		return m, nil
	case tea.MouseMsg:
		if !m.vpReady {
			m.ensureViewport()
		}
		if msg.Type == tea.MouseWheelUp {
			m.vp.LineUp(3)
			m.screenOffsets[m.screen] = m.vp.YOffset
			return m, nil
		}
		if msg.Type == tea.MouseWheelDown {
			m.vp.LineDown(3)
			m.screenOffsets[m.screen] = m.vp.YOffset
			return m, nil
		}
		if msg.Type == tea.MouseRelease && msg.Action == tea.MouseActionPress {
			return m.handleMouseClick(msg)
		}
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		m.screenOffsets[m.screen] = m.vp.YOffset
		return m, cmd
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case groupsLoadedMsg:
		return m.handleGroupsLoaded(msg)
	case nodesLoadedMsg:
		return m.handleNodesLoaded(msg)
	case nodeSwitchedMsg:
		return m.handleNodeSwitched(msg)
	case nodeTestProgressMsg:
		return m.handleNodeTestProgress(msg)
	}
	return m, nil
}

func (m NodeManagerModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.quitConfirm {
		if quit, _ := handleQuitConfirm(msg.String(), &m.quitConfirm); quit {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	if isQuitKey(msg) {
		if m.testing {
			m.testing = false
			m.testStream = nil
		}
		if m.screen == ScreenGroupSelect {
			m.quitConfirm = true
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit
	}

	if m.showHelp {
		if shouldDismissHelp(msg.String()) {
			m.showHelp = false
			return m, nil
		}
		return m, nil
	}

	if m.showNodeDetail {
		if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "i" {
			m.showNodeDetail = false
			return m, nil
		}
		if msg.String() == "c" && len(m.getDisplayNodes()) > 0 {
			nodeName := m.getDisplayNodes()[m.detailNodeIndex].Name
			_ = clipboard.WriteAll(nodeName)
			m.feedback.setInfo("已复制节点名: " + nodeName)
		}
		return m, nil
	}

	if m.confirmAction != ConfirmNone {
		return m.handleConfirmDialog(msg)
	}

	if m.screen == ScreenGroupSelect && m.groupSearchInput.Focused() {
		return m.handleGroupSearchInput(msg)
	}

	if m.screen == ScreenNodeSelect && m.searchInput.Focused() {
		return m.handleSearchInput(msg)
	}

	switch m.screen {
	case ScreenGroupSelect:
		return m.updateGroupSelect(msg)
	case ScreenNodeSelect:
		return m.updateNodeSelect(msg)
	default:
		return m, nil
	}
}

func (m *NodeManagerModel) ensureViewport() {
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

func (m *NodeManagerModel) setScreen(screen Screen) {
	m.viewportState.switchScreen(m.screen, screen, m.width, m.height, screen.topChrome())
	m.feedback.clear()
	m.screen = screen
}

func (m NodeManagerModel) baseViewportSize() (int, int) {
	return calcViewportSize(m.width, m.height, m.screen.topChrome())
}

func (m *NodeManagerModel) scrollViewport(key string) {
	m.viewportState.scroll(key)
	m.screenOffsets[m.screen] = m.vp.YOffset
}

func (m NodeManagerModel) handleMouseClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Y < 0 || m.showHelp || m.confirmAction != ConfirmNone || !m.vpReady {
		return m, nil
	}

	footer := ""
	if m.screen == ScreenNodeSelect {
		footer = "↑/↓ 选择 │ Enter 切换 │ t 测速 │ s 排序 │ / 搜索 │ i 详情 │ g 跳到选中 │ c 复制 │ ? 帮助 │ Esc 返回"
	} else {
		footer = "↑/↓ 选择 │ Enter 查看节点 │ / 搜索 │ r 刷新 │ ? 帮助 │ Esc 退出"
	}
	headerText := ""
	if m.screen == ScreenGroupSelect {
		headerText = fmt.Sprintf("选择代理组 (%d)", len(m.groups))
	} else {
		headerText = fmt.Sprintf("代理组: %s (%d 节点)", m.selectedGroup, len(m.nodes))
	}
	extraLines := 0
	if m.screen == ScreenGroupSelect && (m.groupSearchInput.Focused() || m.groupSearchQuery != "") {
		extraLines++
	}
	if m.screen == ScreenNodeSelect && (m.searchInput.Focused() || m.searchQuery != "") {
		extraLines++
	}
	chromeHeight := cardChromeHeight(headerText, m.feedback, footer, extraLines)

	if m.screen == ScreenGroupSelect && !m.loading {
		yOffset := m.vp.YOffset
		idx := msg.Y - chromeHeight + yOffset
		displayGroups := m.getDisplayGroups()
		if idx >= 0 && idx < len(displayGroups) {
			m.groupIndex = idx
		}
		return m, nil
	}

	if m.screen == ScreenNodeSelect && !m.loading && !m.testing {
		yOffset := m.vp.YOffset
		displayNodes := m.getDisplayNodes()
		idx := msg.Y - chromeHeight + yOffset
		if idx >= 0 && idx < len(displayNodes) {
			m.nodeIndex = idx
		}
		return m, nil
	}

	return m, nil
}

func (m NodeManagerModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.searchInput.Blur()
		if m.searchQuery != m.searchInput.Value() {
			m.searchQuery = m.searchInput.Value()
			m.applyFilter()
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}
}

func (m *NodeManagerModel) applyFilter() {
	prevIndex := m.nodeIndex
	if m.searchQuery == "" {
		m.filteredNodes = nil
		if m.sortMode != NodeSortDefault {
			m.filteredNodes = append([]NodeItem(nil), m.nodes...)
			m.applySort()
		}
		m.nodeIndex = min(prevIndex, len(m.getDisplayNodes())-1)
		if m.nodeIndex < 0 {
			m.nodeIndex = 0
		}
		return
	}
	q := strings.ToLower(m.searchQuery)
	m.filteredNodes = make([]NodeItem, 0, len(m.nodes))
	for _, n := range m.nodes {
		if strings.Contains(strings.ToLower(n.Name), q) || strings.Contains(strings.ToLower(n.Protocol), q) {
			m.filteredNodes = append(m.filteredNodes, n)
		}
	}
	if m.sortMode != NodeSortDefault {
		m.applySort()
	}
	m.nodeIndex = 0
}

func (m *NodeManagerModel) applySort() {
	nodes := m.getDisplayNodes()
	if len(nodes) <= 1 {
		return
	}
	switch m.sortMode {
	case NodeSortDelay:
		sort.SliceStable(nodes, func(i, j int) bool {
			a, b := nodes[i].Delay, nodes[j].Delay
			if a == 0 && b == 0 {
				return nodes[i].Name < nodes[j].Name
			}
			if a == 0 {
				return false
			}
			if b == 0 {
				return true
			}
			return a < b
		})
	case NodeSortName:
		sort.SliceStable(nodes, func(i, j int) bool {
			return nodes[i].Name < nodes[j].Name
		})
	case NodeSortProtocol:
		sort.SliceStable(nodes, func(i, j int) bool {
			if nodes[i].Protocol == nodes[j].Protocol {
				return nodes[i].Name < nodes[j].Name
			}
			return nodes[i].Protocol < nodes[j].Protocol
		})
	}
}

func (m *NodeManagerModel) cycleSortMode() {
	m.sortMode = NodeSortMode((int(m.sortMode) + 1) % int(nodeSortCount))
	if m.sortMode == NodeSortDefault {
		m.filteredNodes = nil
		if m.searchQuery != "" {
			m.applyFilter()
		}
	} else {
		if m.filteredNodes == nil {
			m.filteredNodes = append([]NodeItem(nil), m.nodes...)
		} else if m.searchQuery != "" {
			m.applyFilter()
			m.applySort()
			return
		}
		m.applySort()
	}
	m.nodeIndex = 0
}

func (m *NodeManagerModel) getDisplayGroups() []GroupItem {
	if m.groupFiltered != nil {
		return m.groupFiltered
	}
	return m.groups
}

func (m *NodeManagerModel) applyGroupFilter() {
	if m.groupSearchQuery == "" {
		m.groupFiltered = nil
		m.groupIndex = 0
		return
	}
	q := strings.ToLower(m.groupSearchQuery)
	m.groupFiltered = make([]GroupItem, 0, len(m.groups))
	for _, g := range m.groups {
		if strings.Contains(strings.ToLower(g.Name), q) || strings.Contains(strings.ToLower(g.Type), q) {
			m.groupFiltered = append(m.groupFiltered, g)
		}
	}
	m.groupIndex = 0
}

func (m NodeManagerModel) getDisplayNodes() []NodeItem {
	if m.filteredNodes != nil {
		return m.filteredNodes
	}
	return m.nodes
}

func (m NodeManagerModel) handleConfirmDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		action := m.confirmAction
		target := m.confirmTarget
		m.confirmAction = ConfirmNone
		m.confirmTarget = ""
		switch action {
		case ConfirmSwitchNode:
			m.loading = true
			m.loadingMsg = "正在切换节点..."
			m.feedback.clear()
			return m, tea.Batch(m.spinner.Tick, m.switchNode(m.selectedGroup, target))
		}
		return m, nil
	case "n", "esc":
		m.confirmAction = ConfirmNone
		m.confirmTarget = ""
		m.feedback.setInfo("已取消操作")
		return m, nil
	}
	return m, nil
}

func (m NodeManagerModel) updateGroupSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	if m.groupSearchInput.Focused() {
		return m.handleGroupSearchInput(msg)
	}

	displayGroups := m.getDisplayGroups()

	switch msg.String() {
	case "up", "k":
		if m.groupIndex > 0 {
			m.groupIndex--
		}
	case "down", "j":
		if m.groupIndex < len(displayGroups)-1 {
			m.groupIndex++
		}
	case "pgup":
		m.groupIndex -= m.vp.Height
		if m.groupIndex < 0 {
			m.groupIndex = 0
		}
	case "pgdown":
		m.groupIndex += m.vp.Height
		if m.groupIndex >= len(displayGroups) {
			m.groupIndex = len(displayGroups) - 1
		}
	case "home":
		m.groupIndex = 0
	case "end":
		m.groupIndex = len(displayGroups) - 1
	case "enter":
		if len(displayGroups) > 0 {
			m.selectedGroup = displayGroups[m.groupIndex].Name
			m.loading = true
			m.loadingMsg = "正在加载节点..."
			m.feedback.clear()
			m.nodeIndex = 0
			return m, tea.Batch(m.spinner.Tick, m.loadNodes(m.selectedGroup))
		}
	case "r":
		m.loading = true
		m.loadingMsg = "正在刷新代理组..."
		m.feedback.clear()
		return m, tea.Batch(m.spinner.Tick, m.loadGroups())
	case "/":
		m.groupSearchInput.SetValue("")
		m.groupSearchInput.Focus()
		return m, nil
	case "?":
		m.showHelp = true
		return m, nil
	}
	return m, nil
}

func (m NodeManagerModel) handleGroupSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.groupSearchInput.Blur()
		if m.groupSearchQuery != m.groupSearchInput.Value() {
			m.groupSearchQuery = m.groupSearchInput.Value()
			m.applyGroupFilter()
		}
		return m, nil
	case "esc":
		m.groupSearchInput.Blur()
		return m, nil
	default:
		var cmd tea.Cmd
		m.groupSearchInput, cmd = m.groupSearchInput.Update(msg)
		return m, cmd
	}
}

func (m NodeManagerModel) updateNodeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.testing {
		switch msg.String() {
		case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
			m.scrollViewport(msg.String())
			return m, nil
		}
		return m, nil
	}
	if m.loading {
		return m, nil
	}

	displayNodes := m.getDisplayNodes()

	switch msg.String() {
	case "up", "k":
		if m.nodeIndex > 0 {
			m.nodeIndex--
		}
	case "down", "j":
		if m.nodeIndex < len(displayNodes)-1 {
			m.nodeIndex++
		}
	case "pgup":
		m.nodeIndex -= m.vp.Height
		if m.nodeIndex < 0 {
			m.nodeIndex = 0
		}
	case "pgdown":
		m.nodeIndex += m.vp.Height
		if m.nodeIndex >= len(displayNodes) {
			m.nodeIndex = len(displayNodes) - 1
		}
	case "home":
		m.nodeIndex = 0
	case "end":
		m.nodeIndex = len(displayNodes) - 1
	case "enter":
		if len(displayNodes) > 0 {
			nodeName := displayNodes[m.nodeIndex].Name
			m.confirmAction = ConfirmSwitchNode
			m.confirmTarget = nodeName
			return m, nil
		}
	case "t":
		if len(m.nodes) > 0 {
			m.testing = true
			m.testDone = 0
			m.testTotal = len(m.nodes)
			m.feedback.clear()
			stream := m.nodeService.StartNodeTest(m.appCfg.ControllerAddr, m.selectedGroup, append([]NodeItem(nil), m.nodes...), defaultNodeTestConcurrency)
			m.testStream = stream
			return m, tea.Batch(m.spinner.Tick, waitForNodeTestProgress(stream))
		}
	case "r":
		m.loading = true
		m.loadingMsg = "正在刷新节点..."
		m.feedback.clear()
		return m, tea.Batch(m.spinner.Tick, m.loadNodes(m.selectedGroup))
	case "/":
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, nil
	case "s":
		m.cycleSortMode()
		return m, nil
	case "c":
		if len(displayNodes) > 0 {
			nodeName := displayNodes[m.nodeIndex].Name
			_ = clipboard.WriteAll(nodeName)
			m.feedback.setInfo("已复制节点名: " + nodeName)
		}
		return m, nil
	case "i":
		if len(displayNodes) > 0 {
			m.showNodeDetail = true
			m.detailNodeIndex = m.nodeIndex
		}
		return m, nil
	case "g", "*":
		for i, node := range displayNodes {
			if node.Selected {
				m.nodeIndex = i
				break
			}
		}
		return m, nil
	case "?":
		m.showHelp = true
		return m, nil
	case "esc":
		if m.searchInput.Focused() {
			m.searchInput.Blur()
			return m, nil
		}
		if m.showNodeDetail {
			m.showNodeDetail = false
			return m, nil
		}
		m.setScreen(ScreenGroupSelect)
		return m, nil
	}
	return m, nil
}

func waitForNodeTestProgress(stream <-chan nodeTestProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-stream
		if !ok {
			return nodeTestProgressMsg{done: true}
		}
		return msg
	}
}

func (m NodeManagerModel) loadGroups() tea.Cmd {
	nodeService := m.nodeService
	controllerAddr := m.appCfg.ControllerAddr
	return func() tea.Msg {
		groups, err := nodeService.LoadGroups(controllerAddr)
		if err != nil {
			return groupsLoadedMsg{err: err.Error()}
		}
		return groupsLoadedMsg{groups: groups}
	}
}

func (m NodeManagerModel) loadNodes(groupName string) tea.Cmd {
	nodeService := m.nodeService
	controllerAddr := m.appCfg.ControllerAddr
	return func() tea.Msg {
		nodes, err := nodeService.LoadNodes(controllerAddr, groupName)
		if err != nil {
			return nodesLoadedMsg{err: err.Error()}
		}
		return nodesLoadedMsg{nodes: nodes}
	}
}

func (m NodeManagerModel) switchNode(groupName, nodeName string) tea.Cmd {
	nodeService := m.nodeService
	controllerAddr := m.appCfg.ControllerAddr
	return func() tea.Msg {
		err := nodeService.SwitchNode(controllerAddr, groupName, nodeName)
		if err != nil {
			return nodeSwitchedMsg{success: false, err: err.Error()}
		}
		return nodeSwitchedMsg{success: true}
	}
}

func (m NodeManagerModel) handleGroupsLoaded(msg groupsLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != "" {
		m.feedback.setError("加载代理组失败: " + msg.err)
		return m, nil
	}
	m.feedback.clear()
	m.groups = msg.groups
	m.groupIndex = 0
	return m, nil
}

func (m NodeManagerModel) handleNodesLoaded(msg nodesLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != "" {
		m.feedback.setError("加载节点失败: " + msg.err)
		return m, nil
	}
	m.feedback.clear()
	m.nodes = msg.nodes
	m.nodeIndex = 0
	for i, node := range m.nodes {
		if node.Selected {
			m.nodeIndex = i
			break
		}
	}
	m.setScreen(ScreenNodeSelect)
	return m, nil
}

func (m NodeManagerModel) handleNodeSwitched(msg nodeSwitchedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.success {
		m.feedback.setSuccess("✅ 节点切换成功")
		switchedName := ""
		displayNodes := m.getDisplayNodes()
		if m.nodeIndex < len(displayNodes) {
			switchedName = displayNodes[m.nodeIndex].Name
		}
		for i := range m.nodes {
			m.nodes[i].Selected = (m.nodes[i].Name == switchedName)
		}
		if m.filteredNodes != nil {
			for i := range m.filteredNodes {
				m.filteredNodes[i].Selected = (m.filteredNodes[i].Name == switchedName)
			}
		}
		m.quitting = true
		return m, tea.Quit
	}
	m.feedback.setError("切换失败: " + msg.err)
	return m, nil
}

func (m NodeManagerModel) handleNodeTestProgress(msg nodeTestProgressMsg) (tea.Model, tea.Cmd) {
	if msg.err != "" {
		m.testing = false
		m.testStream = nil
		m.feedback.setError("测速失败: " + msg.err)
		return m, nil
	}
	if msg.index >= 0 && msg.index < len(m.nodes) {
		m.nodes[msg.index].Delay = msg.delay
	}
	if msg.total > 0 {
		m.testTotal = msg.total
	}
	if msg.tested > 0 {
		m.testDone = msg.tested
	}
	if msg.done {
		m.testing = false
		m.testStream = nil
		m.feedback.setSuccess(fmt.Sprintf("✅ 延迟测试完成 (%d 个节点)", m.testDone))
		return m, nil
	}
	if m.testStream != nil {
		return m, waitForNodeTestProgress(m.testStream)
	}
	return m, nil
}

// View renders the node manager.
func (m NodeManagerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render(m.title))
	b.WriteString("\n")

	if m.quitConfirm {
		b.WriteString(m.viewQuitConfirm())
		b.WriteString("\n")
		b.WriteString(m.renderStatusBar())
		return b.String()
	}

	if m.showHelp {
		b.WriteString(m.viewHelp())
		return b.String()
	}

	if m.showNodeDetail {
		b.WriteString(m.viewNodeDetail())
		b.WriteString("\n")
		b.WriteString(m.renderStatusBar())
		return b.String()
	}

	if m.confirmAction != ConfirmNone {
		b.WriteString(m.viewConfirmDialog())
		return b.String()
	}

	switch m.screen {
	case ScreenGroupSelect:
		b.WriteString(m.viewGroupSelect())
	case ScreenNodeSelect:
		b.WriteString(m.viewNodeSelect())
	}

	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// Completed reports whether the model comes from a completed setup flow.
func (m NodeManagerModel) Completed() bool {
	return m.completed
}
