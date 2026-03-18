package ui

import (
	"testing"
)

func TestScreenStepLabel(t *testing.T) {
	tests := []struct {
		screen Screen
		want   string
	}{
		{ScreenWelcome, "欢迎"},
		{ScreenSubscription, "步骤 1/7: 输入订阅 URL"},
		{ScreenMode, "步骤 2/7: 选择运行模式"},
		{ScreenAdvanced, "步骤 3/7: 高级设置"},
		{ScreenPreview, "步骤 4/7: 配置预览"},
		{ScreenResult, "步骤 5/7: 执行结果"},
		{ScreenGroupSelect, "步骤 6/7: 选择代理组"},
		{ScreenNodeSelect, "步骤 7/7: 选择节点"},
		{ScreenDone, ""},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.screen.StepLabel()
			if got != tt.want {
				t.Errorf("StepLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGroupIcon(t *testing.T) {
	tests := []struct {
		typ  string
		want string
	}{
		{"select", "🔀"},
		{"url-test", "⚡"},
		{"fallback", "🔄"},
		{"load-balance", "⚖️"},
		{"unknown", "📦"},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			got := groupIcon(tt.typ)
			if got != tt.want {
				t.Errorf("groupIcon(%q) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestBoolToYesNo(t *testing.T) {
	if got := boolToYesNo(true); got != "是" {
		t.Errorf("boolToYesNo(true) = %q, want 是", got)
	}
	if got := boolToYesNo(false); got != "否" {
		t.Errorf("boolToYesNo(false) = %q, want 否", got)
	}
}

func TestFormatNodeDelay(t *testing.T) {
	tests := []struct {
		name  string
		delay int
		want  string
	}{
		{"timeout", -1, "超时"},
		{"untested", 0, ""},
		{"fast", 50, "50ms ✨"},
		{"ok", 150, "150ms"},
		{"slow", 500, "500ms ⚠️"},
		{"very slow", 1500, "1.5s 🔴"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNodeDelay(tt.delay)
			if got != tt.want {
				t.Errorf("formatNodeDelay(%d) = %q, want %q", tt.delay, got, tt.want)
			}
		})
	}
}

func TestNewWizardDefaults(t *testing.T) {
	wizard := NewWizard()

	if wizard.screen != ScreenWelcome {
		t.Errorf("initial screen = %d, want %d", wizard.screen, ScreenWelcome)
	}
	if wizard.appCfg == nil {
		t.Fatal("appCfg should not be nil")
	}
	if wizard.urlInput.Value() != "" {
		t.Errorf("urlInput should be empty, got %q", wizard.urlInput.Value())
	}
	if len(wizard.advancedFields) != 7 {
		t.Errorf("advancedFields count = %d, want 7", len(wizard.advancedFields))
	}
}

func TestWizardCompleted(t *testing.T) {
	wizard := NewWizard()
	if wizard.Completed() {
		t.Error("new wizard should not be completed")
	}

	wizard.execSteps = []ExecStep{{Label: "test", Success: true}}
	if !wizard.Completed() {
		t.Error("wizard with execSteps should be completed")
	}
}

func TestBoolToInt(t *testing.T) {
	if got := boolToInt(true); got != 1 {
		t.Errorf("boolToInt(true) = %d, want 1", got)
	}
	if got := boolToInt(false); got != 0 {
		t.Errorf("boolToInt(false) = %d, want 0", got)
	}
}
