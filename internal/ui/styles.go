// Package ui provides TUI styles using lipgloss.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B9D")).
			MarginBottom(1)

	// Section header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	// Normal text
	TextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E2E8F0"))

	// Description/info text
	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8"))

	// Success message
	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981"))

	// Error message
	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EF4444"))

	// Warning message
	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F59E0B"))

	// Input field
	InputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")).
			Bold(true)

	// Selected item in list
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B9D")).
			PaddingLeft(2)

	// Unselected item
	UnselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			PaddingLeft(2)

	// Key binding hints
	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			MarginTop(1)

	// YAML preview block
	YAMLStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399")).
			Background(lipgloss.Color("#1E293B")).
			Padding(1, 2).
			MarginTop(1)

	// Step indicator
	StepStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F472B6"))

	// Border style for content areas
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2).
			MarginTop(1)

	// Node delay styles
	DelayGoodStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	DelayOkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B"))

	DelayBadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	DelayUnknownStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	// Current selection marker
	ActiveMarkerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#10B981"))

	// Group type badge
	BadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Background(lipgloss.Color("#1E1B4B")).
			Padding(0, 1)

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B9D"))

	// Protocol badges
	ProtocolVlessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#60A5FA")).
				Background(lipgloss.Color("#1E293B")).
				Padding(0, 1)

	ProtocolHy2Style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#34D399")).
				Background(lipgloss.Color("#064E3B")).
				Padding(0, 1)

	ProtocolTrojanStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B")).
				Background(lipgloss.Color("#451A03")).
				Padding(0, 1)

	ProtocolVMessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A78BFA")).
				Background(lipgloss.Color("#2E1065")).
				Padding(0, 1)

	ProtocolSSStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F472B6")).
				Background(lipgloss.Color("#500724")).
				Padding(0, 1)

	ProtocolUnknownStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94A3B8")).
				Background(lipgloss.Color("#1E293B")).
				Padding(0, 1)
)
