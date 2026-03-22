// Package ui defines the wizard and node manager states.
package ui

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
		return "步骤 5/7: 导入本地订阅"
	case ScreenGroupSelect:
		return "步骤 6/7: 选择代理组"
	case ScreenNodeSelect:
		return "步骤 7/7: 选择节点"
	default:
		return ""
	}
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
