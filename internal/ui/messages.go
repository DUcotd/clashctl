package ui

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
	success  bool
	err      string
	nodeName string // the node that was switched to
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
