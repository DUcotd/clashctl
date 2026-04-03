// Package ui provides the Bubble Tea setup wizard model.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"clashctl/internal/core"
)

type subscriptionState struct {
	sourceMode      SubscriptionSource
	urlInput        textinput.Model
	fileInput       textinput.Model
	inlineInput     textarea.Model
	localImportPath string
	inlineContent   string
}

type advancedState struct {
	modeIndex      int
	advancedIndex  int
	advancedFields []string
	advancedInputs []textinput.Model
}

type executionViewState struct {
	execSteps         []ExecStep
	currentStep       string
	canImportFallback bool
	importHint        string
	importInput       textinput.Model
	setupStream       <-chan setupProgressMsg
}

type viewportState struct {
	vp            viewport.Model
	vpReady       bool
	screenOffsets map[Screen]int
}

// WizardModel is the setup TUI state.
type WizardModel struct {
	screen   Screen
	appCfg   *core.AppConfig
	width    int
	height   int
	quitting bool
	title    string

	controllerAvailable bool
	completed           bool
	spinner             spinner.Model
	setupService        SetupService
	nodeService         NodeService
	feedback            pageFeedbackState
	subscriptionState
	advancedState
	executionViewState
	viewportState
}

// ExecStep represents a single execution step result.
type ExecStep struct {
	Label   string
	Success bool
	Detail  string
}

// NewWizard creates a new setup wizard with defaults or persisted values.
func NewWizard(appCfg *core.AppConfig) WizardModel {
	if appCfg == nil {
		appCfg = core.DefaultAppConfig()
	}
	return newWizardWithServices(appCfg, newDefaultSetupService(), newDefaultNodeService(appCfg.ControllerSecret))
}

func newWizardWithServices(appCfg *core.AppConfig, setupSvc SetupService, nodeSvc NodeService) WizardModel {
	if appCfg == nil {
		appCfg = core.DefaultAppConfig()
	}
	if setupSvc == nil {
		setupSvc = newDefaultSetupService()
	}
	if nodeSvc == nil {
		nodeSvc = newDefaultNodeService(appCfg.ControllerSecret)
	}

	modeIndex := 1
	if appCfg.Mode == "tun" {
		modeIndex = 0
	}

	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/sub or /path/to/sub.txt"
	urlInput.SetValue(appCfg.SubscriptionURL)
	urlInput.Focus()
	urlInput.Width = 60
	urlInput.Prompt = "› "
	urlInput.PromptStyle = InputStyle
	urlInput.TextStyle = InputStyle

	fileInput := textinput.New()
	fileInput.Placeholder = "/path/to/sub.txt"
	fileInput.Width = 60
	fileInput.Prompt = "› "
	fileInput.PromptStyle = InputStyle
	fileInput.TextStyle = InputStyle

	inlineInput := textarea.New()
	inlineInput.Placeholder = "Paste base64, raw vless:// links, trojan:// links, hysteria2:// links, or Clash/Mihomo YAML here"
	inlineInput.Prompt = "› "
	inlineInput.ShowLineNumbers = false
	inlineInput.SetWidth(60)
	inlineInput.SetHeight(8)
	inlineInput.CharLimit = 0
	inlineInput.MaxHeight = 12

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
			ti.SetValue(appCfg.ConfigDir)
		case "控制器地址":
			ti.SetValue(appCfg.ControllerAddr)
		case "mixed-port":
			ti.SetValue(fmt.Sprintf("%d", appCfg.MixedPort))
		case "Provider 路径":
			ti.SetValue(appCfg.ProviderPath)
		case "健康检查":
			ti.SetValue(boolToYesNo(appCfg.EnableHealthCheck))
		case "systemd 服务":
			ti.SetValue(boolToYesNo(appCfg.EnableSystemd))
		case "自动启动":
			ti.SetValue(boolToYesNo(appCfg.AutoStart))
		}
		advInputs[i] = ti
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	importInput := textinput.New()
	importInput.Placeholder = "/path/to/sub.txt"
	importInput.Width = 60
	importInput.Prompt = "› "
	importInput.PromptStyle = InputStyle
	importInput.TextStyle = InputStyle

	return WizardModel{
		screen:       ScreenSubscription,
		appCfg:       appCfg,
		title:        "🧙 clashctl 配置向导",
		spinner:      s,
		setupService: setupSvc,
		nodeService:  nodeSvc,
		subscriptionState: subscriptionState{
			sourceMode:  SubscriptionSourceURL,
			urlInput:    urlInput,
			fileInput:   fileInput,
			inlineInput: inlineInput,
		},
		advancedState: advancedState{
			modeIndex:      modeIndex,
			advancedFields: fields,
			advancedInputs: advInputs,
		},
		executionViewState: executionViewState{
			importInput: importInput,
		},
		viewportState: viewportState{
			screenOffsets: make(map[Screen]int),
		},
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
		m.ensureViewport()
		return m, nil
	case tea.MouseMsg:
		if m.usesViewport() {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			m.screenOffsets[m.screen] = m.vp.YOffset
			return m, cmd
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case setupProgressMsg:
		return m.handleSetupProgress(msg)
	}
	return m, nil
}

func (m WizardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isQuitKey(msg) {
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
		return m, nil
	case ScreenResult:
		return m.updateResult(msg)
	case ScreenImportLocal:
		return m.updateImportLocal(msg)
	default:
		return m, nil
	}
}

func (m *WizardModel) ensureViewport() {
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

func (m *WizardModel) setScreen(screen Screen) {
	if m.vpReady {
		m.screenOffsets[m.screen] = m.vp.YOffset
	}
	m.feedback.clear()
	m.screen = screen
	if m.vpReady {
		m.ensureViewport()
	}
}

func (m WizardModel) usesViewport() bool {
	switch m.screen {
	case ScreenWelcome, ScreenMode, ScreenPreview, ScreenExecution, ScreenResult:
		return true
	default:
		return false
	}
}

func (m WizardModel) baseViewportSize() (int, int) {
	innerWidth := max(24, m.width-BoxStyle.GetHorizontalFrameSize()-4)
	topChrome := 4
	if m.screen != ScreenWelcome {
		topChrome = 6
	}
	innerHeight := max(6, m.height-topChrome-BoxStyle.GetVerticalFrameSize()-2)
	return innerWidth, innerHeight
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m WizardModel) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.focusSubscriptionInput()
		m.setScreen(ScreenSubscription)
		return m, nil
	case "q", "esc":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m WizardModel) updateSubscription(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "shift+tab":
		m.setSubscriptionSource((m.sourceMode + 2) % 3)
		m.feedback.clear()
		return m, nil
	case "right", "tab":
		m.setSubscriptionSource((m.sourceMode + 1) % 3)
		m.feedback.clear()
		return m, nil
	case "shift+enter":
		if m.sourceMode == SubscriptionSourceInline {
			var cmd tea.Cmd
			m.inlineInput, cmd = m.inlineInput.Update(msg)
			return m, cmd
		}
	case "enter":
		if err := m.commitSubscriptionSelection(); err != "" {
			m.feedback.setError(err)
			return m, nil
		}
		m.setScreen(ScreenMode)
		return m, nil
	case "esc":
		m.quitting = true
		return m, tea.Quit
	default:
		return m.updateSubscriptionInput(msg)
	}
	return m, nil
}

func (m *WizardModel) setSubscriptionSource(source SubscriptionSource) {
	m.sourceMode = source
	m.focusSubscriptionInput()
}

func (m *WizardModel) focusSubscriptionInput() {
	m.urlInput.Blur()
	m.fileInput.Blur()
	m.inlineInput.Blur()
	switch m.sourceMode {
	case SubscriptionSourceURL:
		m.urlInput.Focus()
	case SubscriptionSourceFile:
		m.fileInput.Focus()
	case SubscriptionSourceInline:
		_ = m.inlineInput.Focus()
	}
}

func (m *WizardModel) updateSubscriptionInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.feedback.clear()
	switch m.sourceMode {
	case SubscriptionSourceURL:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	case SubscriptionSourceFile:
		var cmd tea.Cmd
		m.fileInput, cmd = m.fileInput.Update(msg)
		return m, cmd
	case SubscriptionSourceInline:
		var cmd tea.Cmd
		m.inlineInput, cmd = m.inlineInput.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m *WizardModel) commitSubscriptionSelection() string {
	m.appCfg.SubscriptionURL = ""
	m.localImportPath = ""
	m.inlineContent = ""

	switch m.sourceMode {
	case SubscriptionSourceURL:
		input := strings.TrimSpace(m.urlInput.Value())
		if input == "" {
			return "请输入订阅 URL 或本地路径"
		}
		m.appCfg.SubscriptionURL = input
		return ""
	case SubscriptionSourceFile:
		input := strings.TrimSpace(m.fileInput.Value())
		if input == "" {
			return "请输入本地订阅文件路径"
		}
		m.localImportPath = input
		return ""
	case SubscriptionSourceInline:
		input := strings.TrimSpace(m.inlineInput.Value())
		if input == "" {
			return "请粘贴订阅内容后再继续"
		}
		m.inlineContent = input
		return ""
	default:
		return "请选择订阅来源"
	}
}

func (m WizardModel) updateMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "pgup", "pgdown", "home", "end":
		m.scrollViewport(msg.String())
		return m, nil
	case "up", "k":
		if m.modeIndex > 0 {
			m.modeIndex--
		}
	case "down", "j":
		if m.modeIndex < 1 {
			m.modeIndex++
		}
	case "enter":
		m.applyModeSelection()
		m.setScreen(ScreenPreview)
		return m, nil
	case "a":
		m.applyModeSelection()
		m.resetAdvancedInputsFromConfig()
		m.focusAdvancedInput()
		m.setScreen(ScreenAdvanced)
		return m, nil
	case "esc":
		m.setScreen(ScreenSubscription)
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
		m.focusAdvancedInput()
	case "down", "j":
		if m.advancedIndex < len(m.advancedFields)-1 {
			m.advancedIndex++
		}
		m.focusAdvancedInput()
	case "enter":
		m.collectAdvancedValues()
		m.blurAdvancedInputs()
		m.setScreen(ScreenPreview)
		return m, nil
	case "esc":
		m.resetAdvancedInputsFromConfig()
		m.blurAdvancedInputs()
		m.setScreen(ScreenPreview)
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

func (m *WizardModel) resetAdvancedInputsFromConfig() {
	for i, field := range m.advancedFields {
		switch field {
		case "配置目录":
			m.advancedInputs[i].SetValue(m.appCfg.ConfigDir)
		case "控制器地址":
			m.advancedInputs[i].SetValue(m.appCfg.ControllerAddr)
		case "mixed-port":
			m.advancedInputs[i].SetValue(fmt.Sprintf("%d", m.appCfg.MixedPort))
		case "Provider 路径":
			m.advancedInputs[i].SetValue(m.appCfg.ProviderPath)
		case "健康检查":
			m.advancedInputs[i].SetValue(boolToYesNo(m.appCfg.EnableHealthCheck))
		case "systemd 服务":
			m.advancedInputs[i].SetValue(boolToYesNo(m.appCfg.EnableSystemd))
		case "自动启动":
			m.advancedInputs[i].SetValue(boolToYesNo(m.appCfg.AutoStart))
		}
	}
}

func (m *WizardModel) focusAdvancedInput() {
	for i := range m.advancedInputs {
		if i == m.advancedIndex {
			m.advancedInputs[i].Focus()
			continue
		}
		m.advancedInputs[i].Blur()
	}
}

func (m *WizardModel) blurAdvancedInputs() {
	for i := range m.advancedInputs {
		m.advancedInputs[i].Blur()
	}
}

func (m *WizardModel) applyModeSelection() {
	if m.modeIndex == 0 {
		m.appCfg.Mode = "tun"
		return
	}
	m.appCfg.Mode = "mixed"
}

func (m WizardModel) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
		m.scrollViewport(msg.String())
		return m, nil
	case "a":
		m.resetAdvancedInputsFromConfig()
		m.focusAdvancedInput()
		m.setScreen(ScreenAdvanced)
		return m, nil
	case "enter":
		m.setScreen(ScreenExecution)
		m.completed = false
		m.execSteps = nil
		m.currentStep = ""
		m.canImportFallback = false
		m.importHint = ""
		m.feedback.clear()
		if strings.TrimSpace(m.localImportPath) != "" {
			stream := m.setupService.StartImport(cloneAppConfig(m.appCfg), m.localImportPath)
			m.setupStream = stream
			return m, tea.Batch(m.spinner.Tick, waitForSetupProgress(stream))
		}
		if strings.TrimSpace(m.inlineContent) != "" {
			stream := m.setupService.StartInline(cloneAppConfig(m.appCfg), m.inlineContent)
			m.setupStream = stream
			return m, tea.Batch(m.spinner.Tick, waitForSetupProgress(stream))
		}
		stream := m.setupService.StartRemote(cloneAppConfig(m.appCfg))
		m.setupStream = stream
		return m, tea.Batch(m.spinner.Tick, waitForSetupProgress(stream))
	case "esc":
		m.setScreen(ScreenMode)
		return m, nil
	}
	return m, nil
}

func waitForSetupProgress(stream <-chan setupProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-stream
		if !ok {
			return setupProgressMsg{done: true}
		}
		return msg
	}
}

func (m WizardModel) handleSetupProgress(msg setupProgressMsg) (tea.Model, tea.Cmd) {
	if msg.currentStep != "" {
		m.currentStep = msg.currentStep
	}
	if msg.step != nil {
		m.execSteps = append(m.execSteps, *msg.step)
		m.currentStep = ""
	}
	if msg.done {
		m.setupStream = nil
		m.currentStep = ""
		m.controllerAvailable = msg.controllerReady
		m.canImportFallback = msg.canImport
		m.importHint = msg.importHint
		m.completed = true
		m.setScreen(ScreenResult)
		return m, nil
	}
	if m.setupStream != nil {
		return m, waitForSetupProgress(m.setupStream)
	}
	return m, nil
}

func (m WizardModel) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
		m.scrollViewport(msg.String())
		return m, nil
	case "enter", "n":
		if m.controllerAvailable {
			manager := newNodeManagerWithService(cloneAppConfig(m.appCfg), m.nodeService, true)
			manager.width = m.width
			manager.height = m.height
			if manager.width > 0 && manager.height > 0 {
				manager.ensureViewport()
			}
			return manager, tea.Batch(manager.spinner.Tick, manager.loadGroups())
		}
		if m.canImportFallback {
			m.importInput.Focus()
			m.setScreen(ScreenImportLocal)
			return m, nil
		}
	case "esc":
		m.setScreen(ScreenPreview)
		return m, nil
	}
	return m, nil
}

func (m *WizardModel) scrollViewport(key string) {
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

func (m WizardModel) updateImportLocal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.importInput.Value())
		if path == "" {
			m.feedback.setError("请输入本地订阅文件路径")
			return m, nil
		}
		m.setScreen(ScreenExecution)
		m.completed = false
		m.execSteps = nil
		m.currentStep = ""
		m.importInput.Blur()
		m.feedback.clear()
		stream := m.setupService.StartImport(cloneAppConfig(m.appCfg), path)
		m.setupStream = stream
		return m, tea.Batch(m.spinner.Tick, waitForSetupProgress(stream))
	case "esc":
		m.importInput.Blur()
		m.setScreen(ScreenResult)
		return m, nil
	default:
		m.feedback.clear()
		var cmd tea.Cmd
		m.importInput, cmd = m.importInput.Update(msg)
		return m, cmd
	}
}

// View renders the current screen.
func (m WizardModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render(m.title))
	b.WriteString("\n")

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
	case ScreenImportLocal:
		b.WriteString(m.viewImportLocal())
	}

	return b.String()
}

// Completed returns true if the setup flow finished.
func (m WizardModel) Completed() bool {
	return m.completed
}
