// Package ui defines the wizard and node manager states.
package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
)

// Screen represents a TUI page.
type Screen int

const (
	ScreenWelcome Screen = iota
	ScreenSubscription
	ScreenMode
	ScreenAdvanced
	ScreenPreview
	ScreenExecution
	ScreenResult
	ScreenImportLocal
	ScreenGroupSelect
	ScreenNodeSelect
)

// StepLabel returns a human-readable step label.
func (s Screen) StepLabel() string {
	switch s {
	case ScreenWelcome:
		return "欢迎"
	case ScreenSubscription:
		return "步骤 1/7: 选择订阅来源"
	case ScreenMode:
		return "步骤 2/7: 选择运行模式"
	case ScreenAdvanced:
		return "可选设置: 高级参数"
	case ScreenPreview:
		return "步骤 3/7: 配置预览"
	case ScreenExecution:
		return "步骤 4/7: 正在配置..."
	case ScreenResult:
		return "步骤 5/7: 执行结果"
	case ScreenImportLocal:
		return "附加步骤: 导入本地订阅"
	case ScreenGroupSelect:
		return "步骤 6/7: 选择代理组"
	case ScreenNodeSelect:
		return "步骤 7/7: 选择节点"
	default:
		return ""
	}
}

func (s Screen) topChrome() int {
	if s == ScreenWelcome {
		return 4
	}
	return 6
}

const (
	minViewportWidth  = 24
	minViewportHeight = 6
	viewportPadX      = 4
	viewportPadY      = 2
)

func calcViewportSize(width, height, topChrome int) (int, int) {
	innerWidth := max(minViewportWidth, width-BoxStyle.GetHorizontalFrameSize()-viewportPadX)
	innerHeight := max(minViewportHeight, height-topChrome-BoxStyle.GetVerticalFrameSize()-viewportPadY)
	return innerWidth, innerHeight
}

// SubscriptionSource represents how the wizard receives subscription content.
type SubscriptionSource int

const (
	SubscriptionSourceURL SubscriptionSource = iota
	SubscriptionSourceInline
	SubscriptionSourceFile
)

func (s SubscriptionSource) Title() string {
	switch s {
	case SubscriptionSourceURL:
		return "订阅 URL"
	case SubscriptionSourceInline:
		return "直接粘贴"
	case SubscriptionSourceFile:
		return "本地文件"
	default:
		return "未知"
	}
}

const subscriptionSourceCount = 3

// GroupItem represents a proxy group in the TUI list.
type GroupItem struct {
	Name      string
	Type      string
	Now       string
	NodeCount int
}

// NodeItem represents a proxy node in the TUI list.
type NodeItem struct {
	Name     string
	Protocol string
	Delay    int
	Selected bool
}

// setupProgressMsg carries streaming setup progress.
type setupProgressMsg struct {
	currentStep     string
	step            *ExecStep
	done            bool
	controllerReady bool
	canImport       bool
	importHint      string
}

// groupsLoadedMsg is sent when proxy groups load.
type groupsLoadedMsg struct {
	groups []GroupItem
	err    string
}

// nodesLoadedMsg is sent when nodes for a group load.
type nodesLoadedMsg struct {
	nodes []NodeItem
	err   string
}

// nodeSwitchedMsg is sent when a node switch finishes.
type nodeSwitchedMsg struct {
	success bool
	err     string
}

// nodeTestProgressMsg carries streaming node latency results.
type nodeTestProgressMsg struct {
	index  int
	delay  int
	tested int
	total  int
	done   bool
	err    string
}

// Screen represents confirmation dialog states.
type ConfirmAction int

const (
	ConfirmNone ConfirmAction = iota
	ConfirmSwitchNode
)

// NodeSortMode represents how nodes are sorted.
type NodeSortMode int

const (
	NodeSortDefault NodeSortMode = iota
	NodeSortDelay
	NodeSortName
	NodeSortProtocol
	nodeSortCount
)

func (s NodeSortMode) Label() string {
	switch s {
	case NodeSortDefault:
		return "默认"
	case NodeSortDelay:
		return "延迟"
	case NodeSortName:
		return "名称"
	case NodeSortProtocol:
		return "协议"
	default:
		return "默认"
	}
}

// viewportState holds shared scrollable viewport state.
type viewportState struct {
	vp            viewport.Model
	vpReady       bool
	screenOffsets map[Screen]int
}

func (v *viewportState) initViewport() {
	if !v.vpReady {
		v.vp = viewport.New(1, 1)
		v.vpReady = true
	}
}

func (v *viewportState) switchScreen(from Screen, to Screen, width, height int, topChrome int) {
	if v.vpReady {
		v.screenOffsets[from] = v.vp.YOffset
	}
	v.initViewport()
	innerWidth, innerHeight := calcViewportSize(width, height, topChrome)
	v.vp.Width = innerWidth
	v.vp.Height = innerHeight
	if off, ok := v.screenOffsets[to]; ok {
		v.vp.SetYOffset(off)
	}
}

func (v *viewportState) scroll(key string) {
	if !v.vpReady {
		return
	}
	switch key {
	case "up", "k":
		v.vp.LineUp(1)
	case "down", "j":
		v.vp.LineDown(1)
	case "pgup":
		v.vp.HalfViewUp()
	case "pgdown":
		v.vp.HalfViewDown()
	case "home":
		v.vp.GotoTop()
	case "end":
		v.vp.GotoBottom()
	}
}

func (v *viewportState) followSelected(selectedIndex int) {
	if !v.vpReady {
		return
	}
	if selectedIndex < v.vp.YOffset {
		v.vp.SetYOffset(selectedIndex)
	} else if selectedIndex >= v.vp.YOffset+v.vp.Height {
		v.vp.SetYOffset(selectedIndex - v.vp.Height + 1)
	}
}
