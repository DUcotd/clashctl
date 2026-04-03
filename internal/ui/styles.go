// Package ui provides TUI styles using lipgloss.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme colors
const (
	colorBgBase    = "#0F172A"
	colorBgCard    = "#1E293B"
	colorBgSurface = "#334155"

	colorPrimary   = "#818CF8"
	colorSecondary = "#A78BFA"
	colorAccent    = "#F472B6"
	colorSuccess   = "#34D399"
	colorWarning   = "#FBBF24"
	colorError     = "#F87171"
	colorInfo      = "#60A5FA"
	colorText      = "#E2E8F0"
	colorMuted     = "#94A3B8"
	colorDim       = "#64748B"

	colorBorderPrimary = "#6366F1"
	colorBorderSuccess = "#10B981"
	colorBorderWarning = "#F59E0B"
	colorBorderError   = "#EF4444"
)

var (
	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorAccent)).
			MarginBottom(1)

	TitleDividerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorDim)).
				MarginBottom(1)

	// Section header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorSecondary)).
			MarginBottom(1)

	HeaderBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorPrimary)).
			Background(lipgloss.Color(colorBgSurface)).
			Padding(0, 2)

	// Normal text
	TextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText))

	// Description/info text
	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))

	// Success message
	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorSuccess))

	// Error message
	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorError))

	// Warning message
	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorWarning))

	// Input field
	InputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorInfo)).
			Bold(true)

	InputFocusedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorPrimary)).
				Bold(true)

	// Selected item in list
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorAccent)).
			PaddingLeft(2)

	SelectedBarStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(colorBgSurface)).
				Border(lipgloss.Border{Left: "▋", Top: " ", Right: " ", Bottom: " "}, true, false, false, false).
				BorderForeground(lipgloss.Color(colorAccent)).
				PaddingLeft(1)

	// Unselected item
	UnselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted)).
			PaddingLeft(2)

	// Key binding hints
	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim)).
			MarginTop(1)

	// Key style for shortcuts
	KeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBgBase)).
			Background(lipgloss.Color(colorDim)).
			Padding(0, 1).
			Bold(true)

	KeySepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim))

	// YAML preview block
	YAMLStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSuccess)).
			Background(lipgloss.Color(colorBgCard)).
			Padding(1, 2).
			MarginTop(1)

	// Step indicator
	StepStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorAccent))

	// Step progress dots
	StepDotActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorAccent)).
				Bold(true)

	StepDotDoneStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorSuccess)).
				Bold(true)

	StepDotInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorDim))

	// Border style for content areas
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorderPrimary)).
			Padding(1, 2).
			MarginTop(1)

	BoxSuccessStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorderSuccess)).
			Padding(1, 2).
			MarginTop(1)

	BoxWarningStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorderWarning)).
			Padding(1, 2).
			MarginTop(1)

	BoxErrorStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorderError)).
			Padding(1, 2).
			MarginTop(1)

	// Node delay styles
	DelayGoodStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSuccess))

	DelayOkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWarning))

	DelayBadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError))

	DelayUnknownStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorDim))

	// Current selection marker
	ActiveMarkerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(colorSuccess))

	// Group type badge
	BadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSecondary)).
			Background(lipgloss.Color("#1E1B4B")).
			Padding(0, 1)

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent))

	// Protocol badges
	ProtocolVlessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorInfo)).
				Background(lipgloss.Color(colorBgCard)).
				Padding(0, 1)

	ProtocolHy2Style = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorSuccess)).
				Background(lipgloss.Color("#064E3B")).
				Padding(0, 1)

	ProtocolTrojanStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorWarning)).
				Background(lipgloss.Color("#451A03")).
				Padding(0, 1)

	ProtocolVMessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorSecondary)).
				Background(lipgloss.Color("#2E1065")).
				Padding(0, 1)

	ProtocolSSStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Background(lipgloss.Color("#500724")).
			Padding(0, 1)

	ProtocolUnknownStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorMuted)).
				Background(lipgloss.Color(colorBgCard)).
				Padding(0, 1)

	// Progress bar
	ProgressBarFullStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorSuccess))

	ProgressTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorMuted)).
				Bold(true)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Background(lipgloss.Color(colorBgCard))

	StatusHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim)).
			Background(lipgloss.Color(colorBgCard))

	StatusDividerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorBorderPrimary))

	// Detail view
	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorSecondary)).
				Bold(true).
				Width(8)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorText))

	// Card header
	CardHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorPrimary))

	// Confirmation dialog
	ConfirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWarning)).
			Bold(true)

	// List item separator
	SeparatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBgSurface))

	// Source selector
	SelectorActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorAccent)).
				Background(lipgloss.Color(colorBgSurface)).
				Bold(true).
				Padding(0, 2)

	SelectorInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorMuted)).
				Padding(0, 2)

	// Help section headers
	HelpSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorPrimary)).
				Bold(true).
				MarginTop(1)

	// Empty state
	EmptyStateIconStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorDim)).
				Bold(true)

	EmptyStateTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorMuted))
)
