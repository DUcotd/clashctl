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
		return "步骤 1/5: 输入订阅 URL"
	case ScreenMode:
		return "步骤 2/5: 选择运行模式"
	case ScreenAdvanced:
		return "步骤 3/5: 高级设置"
	case ScreenPreview:
		return "步骤 4/5: 配置预览"
	case ScreenResult:
		return "步骤 5/5: 执行结果"
	case ScreenGroupSelect:
		return "代理组管理"
	case ScreenNodeSelect:
		return "节点选择"
	default:
		return ""
	}
}

// GroupItem represents a proxy group in the TUI list.
type GroupItem struct {
	Name     string
	Type     string
	Now      string
	NodeCount int
}

// NodeItem represents a proxy node in the TUI list.
type NodeItem struct {
	Name     string
	Delay    int  // -1 = timeout, 0 = untested, >0 = delay in ms
	Selected bool
}

// nodeTestedMsg is sent when a batch of node tests completes.
type nodeTestedMsg struct {
	delays map[int]int // index -> delay
}
