package tui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/shawnyeager/dozer/internal/inhibitor"
)

type state int

const (
	stateBrowsing state = iota
	stateConfirmKill
)

// Model is the Bubble Tea model for the dozer TUI.
type Model struct {
	sources     []inhibitor.Source
	table       table.Model
	help        help.Model
	keys        keyMap
	rows        []inhibitor.Inhibitor
	state       state
	killMessage string
	lastErr     error
	interval    time.Duration
	ownUID      uint32
}

func New(sources []inhibitor.Source, interval time.Duration) Model {
	columns := []table.Column{
		{Title: "PID", Width: 7},
		{Title: "USER", Width: 10},
		{Title: "PROCESS", Width: 18},
		{Title: "BLOCKING", Width: 10},
		{Title: "LOCK", Width: 6},
		{Title: "REASON", Width: 58},
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(ts)

	h := help.New()

	return Model{
		sources:  sources,
		table:    t,
		help:     h,
		keys:     defaultKeys(),
		interval: interval,
		ownUID:   uint32(os.Getuid()),
	}
}
