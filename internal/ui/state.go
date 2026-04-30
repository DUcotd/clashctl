// Package ui defines the wizard and node manager states.
package ui

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

// ConfirmAction represents confirmation dialog states.
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
