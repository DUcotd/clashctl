// Package ui defines the wizard state machine.
package ui

// Screen represents a wizard page.
type Screen int

const (
	ScreenWelcome     Screen = iota
	ScreenSubscription
	ScreenMode
	ScreenAdvanced
	ScreenPreview
	ScreenExecution // 正在执行配置
	ScreenResult
	ScreenGroupSelect // 选择代理组
	ScreenNodeSelect  // 选择节点
	ScreenDone
)

// StepLabel returns a human-readable step label.
func (s Screen) StepLabel() string {
	switch s {
	case ScreenWelcome:
		return "欢迎"
	case ScreenSubscription:
		return "步骤 1/8: 输入订阅 URL"
	case ScreenMode:
		return "步骤 2/8: 选择运行模式"
	case ScreenAdvanced:
		return "步骤 3/8: 高级设置"
	case ScreenPreview:
		return "步骤 4/8: 配置预览"
	case ScreenExecution:
		return "步骤 5/8: 正在配置..."
	case ScreenResult:
		return "步骤 6/8: 执行结果"
	case ScreenGroupSelect:
		return "步骤 7/8: 选择代理组"
	case ScreenNodeSelect:
		return "步骤 8/8: 选择节点"
	default:
		return ""
	}
}

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
	Protocol string // Vless, Hysteria2, Trojan, etc.
	Delay    int    // -1 = timeout, 0 = untested, >0 = delay in ms
	Selected bool
}

// nodeTestedMsg is sent when a batch of node tests completes.
type nodeTestedMsg struct {
	delays map[int]int // index -> delay
}

// executionDoneMsg is sent when executeFull completes.
type executionDoneMsg struct {
	steps             []ExecStep
	controllerReady   bool
}
