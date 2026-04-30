package ui

import "strings"

// WizardScreen represents a wizard setup page.
type WizardScreen int

const (
	WizardScreenWelcome WizardScreen = iota
	WizardScreenSubscription
	WizardScreenMode
	WizardScreenAdvanced
	WizardScreenPreview
	WizardScreenExecution
	WizardScreenResult
	WizardScreenImportLocal
)

func (s WizardScreen) StepLabel() string {
	switch s {
	case WizardScreenWelcome:
		return "欢迎"
	case WizardScreenSubscription:
		return "步骤 1/5: 选择订阅来源"
	case WizardScreenMode:
		return "步骤 2/5: 选择运行模式"
	case WizardScreenAdvanced:
		return "可选设置: 高级参数"
	case WizardScreenPreview:
		return "步骤 3/5: 配置预览"
	case WizardScreenExecution:
		return "步骤 4/5: 正在配置..."
	case WizardScreenResult:
		return "步骤 5/5: 执行结果"
	case WizardScreenImportLocal:
		return "附加步骤: 导入本地订阅"
	default:
		return ""
	}
}

func (s WizardScreen) topChrome() int {
	if s == WizardScreenWelcome {
		return 4
	}
	return 6
}

func (s WizardScreen) StepIndex() int {
	switch s {
	case WizardScreenWelcome:
		return 0
	case WizardScreenSubscription:
		return 1
	case WizardScreenMode:
		return 2
	case WizardScreenAdvanced:
		return 3
	case WizardScreenPreview:
		return 4
	case WizardScreenExecution:
		return 5
	case WizardScreenResult:
		return 6
	case WizardScreenImportLocal:
		return 7
	default:
		return 0
	}
}

func (s WizardScreen) TotalSteps() int {
	return 8
}

func (s WizardScreen) StepDots() string {
	idx := s.StepIndex()
	total := s.TotalSteps()
	parts := make([]string, 0, total)
	for i := 0; i < total; i++ {
		if i < idx {
			parts = append(parts, StepDotDoneStyle.Render("●"))
		} else if i == idx {
			parts = append(parts, StepDotActiveStyle.Render("●"))
		} else {
			parts = append(parts, StepDotInactiveStyle.Render("○"))
		}
	}
	return strings.Join(parts, " ")
}

// NodeScreen represents a node manager page.
type NodeScreen int

const (
	NodeScreenGroupSelect NodeScreen = iota
	NodeScreenNodeSelect
)

func (s NodeScreen) topChrome() int {
	return 6
}
