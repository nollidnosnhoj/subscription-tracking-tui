package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#5A4FCF")
	successColor   = lipgloss.Color("#04B575")
	warningColor   = lipgloss.Color("#FFBE0B")
	errorColor     = lipgloss.Color("#FF6B6B")
	mutedColor     = lipgloss.Color("#626262")
	whiteColor     = lipgloss.Color("#FFFFFF")

	// Base styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginBottom(1)

	// List styles
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(whiteColor).
				Background(primaryColor).
				Padding(0, 1)

	NormalItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Status styles
	MonthlyStyle = lipgloss.NewStyle().
			Foreground(successColor)

	YearlyStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	// Input styles
	FocusedInputStyle = lipgloss.NewStyle().
				Foreground(primaryColor)

	BlurredInputStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	// Error/Success messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(mutedColor)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Amount styles
	AmountStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(successColor)
)
