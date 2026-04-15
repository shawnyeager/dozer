package tui

import "github.com/charmbracelet/lipgloss"

// All ambient state uses inline foreground color only. Background fills are
// reserved for the single modal moment in the UI — the SIGTERM confirm prompt.
var (
	okDot     = lipgloss.NewStyle().Foreground(lipgloss.Color("#5FD068")).Bold(true)
	alertDot  = lipgloss.NewStyle().Foreground(lipgloss.Color("#D7263D")).Bold(true)
	idleDot   = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	alertText = lipgloss.NewStyle().Foreground(lipgloss.Color("#D7263D")).Bold(true)
	muted     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#D7263D")).
			Bold(true).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E06C75"))
)
