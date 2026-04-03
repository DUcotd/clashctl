package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"clashctl/internal/core"
	"clashctl/internal/system"
)

func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

type fakeSetupService struct {
	remoteEvents  []setupProgressMsg
	importEvents  []setupProgressMsg
	inlineEvents  []setupProgressMsg
	remoteCalls   int
	importCalls   int
	inlineCalls   int
	lastImport    string
	lastInline    string
	lastRemoteCfg *core.AppConfig
}

func streamSetupEvents(events []setupProgressMsg) <-chan setupProgressMsg {
	stream := make(chan setupProgressMsg, len(events))
	for _, event := range events {
		stream <- event
	}
	close(stream)
	return stream
}

func streamNodeEvents(events []nodeTestProgressMsg) <-chan nodeTestProgressMsg {
	stream := make(chan nodeTestProgressMsg, len(events))
	for _, event := range events {
		stream <- event
	}
	close(stream)
	return stream
}

func (s *fakeSetupService) StartRemote(appCfg *core.AppConfig) <-chan setupProgressMsg {
	s.remoteCalls++
	s.lastRemoteCfg = cloneAppConfig(appCfg)
	return streamSetupEvents(s.remoteEvents)
}

func (s *fakeSetupService) StartImport(appCfg *core.AppConfig, filePath string) <-chan setupProgressMsg {
	s.importCalls++
	s.lastImport = filePath
	return streamSetupEvents(s.importEvents)
}

func (s *fakeSetupService) StartInline(appCfg *core.AppConfig, content string) <-chan setupProgressMsg {
	s.inlineCalls++
	s.lastInline = content
	return streamSetupEvents(s.inlineEvents)
}

type fakeNodeService struct {
	groups             []GroupItem
	nodes              []NodeItem
	testEvents         []nodeTestProgressMsg
	loadGroupsErr      error
	loadNodesErr       error
	switchErr          error
	loadGroupsCalls    int
	loadNodesCalls     int
	switchCalls        int
	testCalls          int
	lastControllerAddr string
	lastGroup          string
	lastNode           string
}

func (s *fakeNodeService) LoadGroups(controllerAddr string) ([]GroupItem, error) {
	s.loadGroupsCalls++
	s.lastControllerAddr = controllerAddr
	if s.loadGroupsErr != nil {
		return nil, s.loadGroupsErr
	}
	return append([]GroupItem(nil), s.groups...), nil
}

func (s *fakeNodeService) LoadNodes(controllerAddr, groupName string) ([]NodeItem, error) {
	s.loadNodesCalls++
	s.lastControllerAddr = controllerAddr
	s.lastGroup = groupName
	if s.loadNodesErr != nil {
		return nil, s.loadNodesErr
	}
	return append([]NodeItem(nil), s.nodes...), nil
}

func (s *fakeNodeService) SwitchNode(controllerAddr, groupName, nodeName string) error {
	s.switchCalls++
	s.lastControllerAddr = controllerAddr
	s.lastGroup = groupName
	s.lastNode = nodeName
	return s.switchErr
}

func (s *fakeNodeService) StartNodeTest(controllerAddr, groupName string, nodes []NodeItem, maxConcurrent int) <-chan nodeTestProgressMsg {
	s.testCalls++
	s.lastControllerAddr = controllerAddr
	s.lastGroup = groupName
	return streamNodeEvents(s.testEvents)
}

func TestUpdatePreviewStartsStreamingExecution(t *testing.T) {
	setupSvc := &fakeSetupService{
		remoteEvents: []setupProgressMsg{{currentStep: "检查 Mihomo 可执行文件"}, {done: true}},
	}
	wizard := newWizardWithServices(core.DefaultAppConfig(), setupSvc, &fakeNodeService{})
	wizard.screen = ScreenPreview
	wizard.appCfg.SubscriptionURL = "https://example.com/sub"

	updated, cmd := wizard.updatePreview(keyMsg("enter"))
	got := updated.(WizardModel)
	if got.screen != ScreenExecution {
		t.Fatalf("screen = %v, want ScreenExecution", got.screen)
	}
	if got.setupStream == nil {
		t.Fatal("setupStream should be initialized")
	}
	if cmd == nil {
		t.Fatal("cmd should wait for setup progress")
	}
	if setupSvc.remoteCalls != 1 {
		t.Fatalf("remoteCalls = %d, want 1", setupSvc.remoteCalls)
	}
}

func TestNodeManagerLoadGroupsUsesNodeService(t *testing.T) {
	nodeSvc := &fakeNodeService{groups: []GroupItem{{Name: "PROXY", Type: "select", NodeCount: 2}}}
	cfg := core.DefaultAppConfig()
	cfg.ControllerAddr = "127.0.0.1:9090"
	manager := newNodeManagerWithService(cfg, nodeSvc, false)

	msg := manager.loadGroups()()
	loaded, ok := msg.(groupsLoadedMsg)
	if !ok {
		t.Fatalf("msg type = %T, want groupsLoadedMsg", msg)
	}
	if nodeSvc.loadGroupsCalls != 1 {
		t.Fatalf("loadGroupsCalls = %d, want 1", nodeSvc.loadGroupsCalls)
	}
	if nodeSvc.lastControllerAddr != cfg.ControllerAddr {
		t.Fatalf("controllerAddr = %q, want %q", nodeSvc.lastControllerAddr, cfg.ControllerAddr)
	}
	if len(loaded.groups) != 1 || loaded.groups[0].Name != "PROXY" {
		t.Fatalf("unexpected groups: %#v", loaded.groups)
	}
}

func TestNodeManagerStartsStreamingLatencyTest(t *testing.T) {
	nodeSvc := &fakeNodeService{
		testEvents: []nodeTestProgressMsg{{index: 0, delay: 88, tested: 1, total: 1}, {done: true, tested: 1, total: 1}},
	}
	manager := newNodeManagerWithService(core.DefaultAppConfig(), nodeSvc, false)
	manager.screen = ScreenNodeSelect
	manager.loading = false
	manager.selectedGroup = "PROXY"
	manager.nodes = []NodeItem{{Name: "Node A"}}

	updated, cmd := manager.updateNodeSelect(keyMsg("t"))
	got := updated.(NodeManagerModel)
	if !got.testing {
		t.Fatal("testing should be true")
	}
	if got.testStream == nil {
		t.Fatal("testStream should be initialized")
	}
	if cmd == nil {
		t.Fatal("cmd should wait for node test progress")
	}
	if nodeSvc.testCalls != 1 {
		t.Fatalf("testCalls = %d, want 1", nodeSvc.testCalls)
	}
}

func TestStartInlineRejectsOversizedContent(t *testing.T) {
	svc := &defaultSetupService{}
	stream := svc.StartInline(core.DefaultAppConfig(), strings.Repeat("a", int(system.MaxPreparedSubscriptionBytes)+1))

	_, ok := <-stream
	if !ok {
		t.Fatal("expected progress message")
	}
	result, ok := <-stream
	if !ok {
		t.Fatal("expected failed step message")
	}
	if result.step == nil || result.step.Success {
		t.Fatalf("result message = %#v, want failed step", result)
	}
	if !strings.Contains(result.step.Detail, "粘贴订阅内容过大") {
		t.Fatalf("detail = %q", result.step.Detail)
	}
}

func TestStartImportRejectsOversizedContent(t *testing.T) {
	svc := &defaultSetupService{
		readFile: func(string) ([]byte, error) {
			return []byte(strings.Repeat("a", int(system.MaxPreparedSubscriptionBytes)+1)), nil
		},
	}
	stream := svc.StartImport(core.DefaultAppConfig(), "/tmp/sub.txt")

	_, ok := <-stream
	if !ok {
		t.Fatal("expected progress message")
	}
	result, ok := <-stream
	if !ok {
		t.Fatal("expected failed step message")
	}
	if result.step == nil || result.step.Success {
		t.Fatalf("result message = %#v, want failed step", result)
	}
	if !strings.Contains(result.step.Detail, "本地订阅文件过大") {
		t.Fatalf("detail = %q", result.step.Detail)
	}
}

func TestDefaultSetupServiceImportUsesLimitedFileReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub.txt")
	data := make([]byte, system.MaxPreparedSubscriptionBytes+1)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	svc := newDefaultSetupService()
	stream := svc.StartImport(core.DefaultAppConfig(), path)

	_, ok := <-stream
	if !ok {
		t.Fatal("expected progress message")
	}
	result, ok := <-stream
	if !ok {
		t.Fatal("expected failed step message")
	}
	if result.step == nil || result.step.Success {
		t.Fatalf("result message = %#v, want failed step", result)
	}
	if !strings.Contains(result.step.Detail, "过大") {
		t.Fatalf("detail = %q", result.step.Detail)
	}
}
