package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"clashctl/internal/core"
)

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
	return newNodeManagerWithService(appCfg, newDefaultNodeService(), false)
}

func newNodeManagerWithService(appCfg *core.AppConfig, nodeSvc NodeService, completed bool) NodeManagerModel {
	if appCfg == nil {
		appCfg = core.DefaultAppConfig()
	}
	if nodeSvc == nil {
		nodeSvc = newDefaultNodeService()
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return NodeManagerModel{
		screen:      ScreenGroupSelect,
		appCfg:      appCfg,
		title:       "📡 clashctl 节点管理",
		completed:   completed,
		spinner:     s,
		nodeService: nodeSvc,
		nodeInteractionState: nodeInteractionState{
			loading:    true,
			loadingMsg: "正在加载代理组...",
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
	if isQuitKey(msg) {
		m.quitting = true
		return m, tea.Quit
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
	if m.vpReady {
		m.screenOffsets[m.screen] = m.vp.YOffset
	}
	m.feedback.clear()
	m.screen = screen
	if m.vpReady {
		m.ensureViewport()
	}
}

func (m NodeManagerModel) baseViewportSize() (int, int) {
	innerWidth := max(24, m.width-BoxStyle.GetHorizontalFrameSize()-4)
	innerHeight := max(6, m.height-4-BoxStyle.GetVerticalFrameSize()-2)
	return innerWidth, innerHeight
}

func (m *NodeManagerModel) scrollViewport(key string) {
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

func (m NodeManagerModel) updateGroupSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.feedback.clear()
			m.nodeIndex = 0
			return m, tea.Batch(m.spinner.Tick, m.loadNodes(m.selectedGroup))
		}
	case "r":
		m.loading = true
		m.loadingMsg = "正在刷新代理组..."
		m.feedback.clear()
		return m, tea.Batch(m.spinner.Tick, m.loadGroups())
	case "esc":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m NodeManagerModel) updateNodeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.feedback.clear()
			return m, tea.Batch(m.spinner.Tick, m.switchNode(m.selectedGroup, nodeName))
		}
	case "t":
		if len(m.nodes) > 0 {
			m.testing = true
			m.testDone = 0
			m.testTotal = len(m.nodes)
			m.feedback.clear()
			stream := m.nodeService.StartNodeTest(m.appCfg.ControllerAddr, m.selectedGroup, append([]NodeItem(nil), m.nodes...), 10)
			m.testStream = stream
			return m, tea.Batch(m.spinner.Tick, waitForNodeTestProgress(stream))
		}
	case "r":
		m.loading = true
		m.loadingMsg = "正在刷新节点..."
		m.feedback.clear()
		return m, tea.Batch(m.spinner.Tick, m.loadNodes(m.selectedGroup))
	case "esc":
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
		for i := range m.nodes {
			m.nodes[i].Selected = (i == m.nodeIndex)
		}
	} else {
		m.feedback.setError("切换失败: " + msg.err)
	}
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

	switch m.screen {
	case ScreenGroupSelect:
		b.WriteString(m.viewGroupSelect())
	case ScreenNodeSelect:
		b.WriteString(m.viewNodeSelect())
	}

	return b.String()
}

// Completed reports whether the model comes from a completed setup flow.
func (m NodeManagerModel) Completed() bool {
	return m.completed
}
