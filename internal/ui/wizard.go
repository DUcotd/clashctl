// Package ui provides the main Bubble Tea wizard model.
package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"

	"clashctl/internal/core"
)

// WizardModel is the main TUI state.
type WizardModel struct {
	screen    Screen
	appCfg    *core.AppConfig
	width     int
	height    int
	quitting  bool

	// Subscription URL input
	urlInput textinput.Model

	// Mode selection
	modeIndex int // 0 = TUN, 1 = mixed-port

	// Advanced settings
	advancedIndex   int
	advancedFields  []string
	advancedInputs  []textinput.Model

	// Result
	execSteps   []ExecStep
	execError   string
}

// ExecStep represents a single execution step result.
type ExecStep struct {
	Label   string
	Success bool
	Detail  string
}

// NewWizard creates a new WizardModel with defaults.
func NewWizard() WizardModel {
	// URL input
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/subscription"
	urlInput.Focus()
	urlInput.Width = 60
	urlInput.Prompt = "› "
	urlInput.PromptStyle = InputStyle
	urlInput.TextStyle = InputStyle

	// Advanced fields
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
			ti.SetValue("/etc/mihomo")
		case "控制器地址":
			ti.SetValue("127.0.0.1:9090")
		case "mixed-port":
			ti.SetValue("7890")
		case "Provider 路径":
			ti.SetValue("./providers/airport.yaml")
		case "健康检查":
			ti.SetValue("是")
		case "systemd 服务":
			ti.SetValue("是")
		case "自动启动":
			ti.SetValue("是")
		}
		advInputs[i] = ti
	}

	return WizardModel{
		screen:         ScreenWelcome,
		appCfg:         core.DefaultAppConfig(),
		urlInput:       urlInput,
		advancedFields: fields,
		advancedInputs: advInputs,
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
		return m, nil
	}
	return m, nil
}

func (m WizardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if msg.String() == "ctrl+c" {
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
	case ScreenResult:
		return m.updateResult(msg)
	}
	return m, nil
}

func (m WizardModel) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.screen = ScreenSubscription
		return m, nil
	case "q", "esc":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m WizardModel) updateSubscription(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			return m, nil // don't proceed without URL
		}
		m.appCfg.SubscriptionURL = url
		m.screen = ScreenMode
		return m, nil
	case "esc":
		m.screen = ScreenWelcome
		return m, nil
	default:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}
}

func (m WizardModel) updateMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.modeIndex > 0 {
			m.modeIndex--
		}
	case "down", "j":
		if m.modeIndex < 1 {
			m.modeIndex++
		}
	case "enter":
		if m.modeIndex == 0 {
			m.appCfg.Mode = "tun"
		} else {
			m.appCfg.Mode = "mixed"
		}
		m.screen = ScreenAdvanced
		return m, nil
	case "esc":
		m.screen = ScreenSubscription
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
	case "down", "j":
		if m.advancedIndex < len(m.advancedFields)-1 {
			m.advancedIndex++
		}
	case "enter":
		// Collect advanced values
		m.collectAdvancedValues()
		m.screen = ScreenPreview
		return m, nil
	case "esc":
		m.screen = ScreenMode
		return m, nil
	default:
		// Update the focused input
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

func (m WizardModel) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Execute full pipeline!
		m.execSteps = m.executeFull()
		m.screen = ScreenResult
		return m, nil
	case "esc":
		m.screen = ScreenAdvanced
		return m, nil
	}
	return m, nil
}

func (m WizardModel) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "q":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

// View renders the current screen.
func (m WizardModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(TitleStyle.Render("🧙 clashctl 配置向导"))
	b.WriteString("\n")

	// Step indicator (except welcome)
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
	case ScreenResult:
		b.WriteString(m.viewResult())
	}

	return b.String()
}
