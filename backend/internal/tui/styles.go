package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Brand Colors (OzyAssist Electric Lime & Dark Surface)
	ColorPrimary   = lipgloss.Color("#d1f107") // Electric Neon Lime
	ColorDarkBg    = lipgloss.Color("#131313") // Deep Surface Dark
	ColorContainer = lipgloss.Color("#1e1e1e") // Card/Container Dark
	ColorChatBg    = lipgloss.Color("#141613") // Dark background with subtle lime tint
	ColorMessageBg = lipgloss.Color("#181b14") // Message container background
	ColorBorder    = lipgloss.Color("#2f2f2f")
	ColorText      = lipgloss.Color("#f0f0f0")
	ColorMuted     = lipgloss.Color("#777777")
	ColorDim       = lipgloss.Color("#444444")
	ColorSuccess   = lipgloss.Color("#2ecc71")
	ColorWarning   = lipgloss.Color("#f39c12")
	ColorError     = lipgloss.Color("#e74c3c")
	ColorAccent    = lipgloss.Color("#00d2d3")
	ColorQueue     = lipgloss.Color("#f1c40f") // Amber/Yellow for queue indication

	// Header Styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#181e00")).
			Background(ColorPrimary).
			Padding(0, 1)

	TagStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorContainer).
			Padding(0, 1)

	TagActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorContainer).
			Padding(0, 1)

	QueueBadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#181e00")).
			Background(ColorQueue).
			Padding(0, 1)

	// Chat & Roles
	UserStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	AssistantStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	SystemStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Tool Calls & Execution
	ToolBadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#131313")).
			Background(ColorWarning).
			Padding(0, 1)

	ToolResultSuccessStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#131313")).
				Background(ColorSuccess).
				Padding(0, 1)

	ToolResultErrorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffffff")).
				Background(ColorError).
				Padding(0, 1)

	ToolContentStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				PaddingLeft(2)

	// Status & Prompt
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(ColorContainer).
			Padding(0, 1)

	InputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary)

	InputInactiveBorderStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(ColorBorder)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Menu & Interactive Navigation
	MenuItemSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#181e00")).
				Background(ColorPrimary).
				Padding(0, 1)

	MenuItemNormalStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Padding(0, 1)

	MenuItemDescStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// Top Header Reference Style
	TopHeaderTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary)

	TopHeaderMetaStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// Interactive Card Reference Style (Plan / Decision UI)
	CardContainerStyle = lipgloss.NewStyle().
				Background(ColorMessageBg).
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderLeftForeground(ColorPrimary).
				Padding(0, 1)

	CardTabActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary)

	CardTabInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	CardQuestionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorText)

	CardOptionSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffffff")).
				Background(lipgloss.Color("#252820")).
				Padding(0, 1)

	CardOptionNormalStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Padding(0, 1)

	CardHintsStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	InputBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#181e00")).
				Background(ColorPrimary).
				Padding(0, 1)
)
