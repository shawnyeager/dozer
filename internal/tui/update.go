package tui

import (
	"context"
	"fmt"
	"sort"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/shawnyeager/dozer/internal/inhibitor"
	"github.com/shawnyeager/dozer/internal/killer"
)

type tickMsg time.Time

type refreshedMsg struct {
	rows []inhibitor.Inhibitor
	errs []error
}

type killResultMsg struct {
	pid  int32
	comm string
	err  error
}

type clearKillMsgMsg struct{}

const killMsgTTL = 3 * time.Second

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), tickCmd(m.interval))
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) refreshCmd() tea.Cmd {
	sources := m.sources
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		rows, errs := inhibitor.Collect(ctx, sources)
		return refreshedMsg{rows: rows, errs: errs}
	}
}

func killCmd(inh inhibitor.Inhibitor) tea.Cmd {
	return func() tea.Msg {
		err := killer.Kill(inh.PID, syscall.SIGTERM)
		return killResultMsg{pid: inh.PID, comm: inh.Comm, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h := msg.Height - 8
		if h < 5 {
			h = 5
		}
		m.table.SetHeight(h)
		m.help.Width = msg.Width

	case tickMsg:
		cmds = append(cmds, m.refreshCmd(), tickCmd(m.interval))

	case refreshedMsg:
		m.rows = SortBlocksFirst(msg.rows)
		if len(msg.errs) > 0 {
			m.lastErr = msg.errs[0]
		} else {
			m.lastErr = nil
		}
		m.table.SetRows(buildRows(m.rows))
		if cursor := m.table.Cursor(); cursor >= len(m.rows) && len(m.rows) > 0 {
			m.table.SetCursor(len(m.rows) - 1)
		}

	case killResultMsg:
		if msg.err != nil {
			m.killMessage = fmt.Sprintf("kill %d (%s) failed: %v", msg.pid, msg.comm, msg.err)
		} else {
			m.killMessage = fmt.Sprintf("sent SIGTERM to %d (%s)", msg.pid, msg.comm)
		}
		cmds = append(cmds,
			m.refreshCmd(),
			tea.Tick(killMsgTTL, func(time.Time) tea.Msg { return clearKillMsgMsg{} }),
		)

	case clearKillMsgMsg:
		m.killMessage = ""

	case tea.KeyMsg:
		if m.state == stateConfirmKill {
			switch {
			case key.Matches(msg, m.keys.Confirm):
				if sel := m.selected(); sel != nil {
					cmds = append(cmds, killCmd(*sel))
				}
				m.state = stateBrowsing
				return m, tea.Batch(cmds...)
			case key.Matches(msg, m.keys.Cancel):
				m.state = stateBrowsing
				m.killMessage = "kill cancelled"
				cmds = append(cmds, tea.Tick(killMsgTTL, func(time.Time) tea.Msg { return clearKillMsgMsg{} }))
				return m, tea.Batch(cmds...)
			}
			// Swallow all other keys while the confirm prompt is up.
			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Refresh):
			cmds = append(cmds, m.refreshCmd())
		case key.Matches(msg, m.keys.Kill):
			m.prepareKill()
			return m, tea.Batch(cmds...)
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) prepareKill() {
	sel := m.selected()
	if sel == nil {
		return
	}
	if sel.UID != m.ownUID && m.ownUID != 0 {
		m.killMessage = fmt.Sprintf("cannot signal pid %d: owned by %s (uid %d)", sel.PID, sel.User, sel.UID)
		return
	}
	m.state = stateConfirmKill
}

func (m Model) selected() *inhibitor.Inhibitor {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.rows) {
		return nil
	}
	return &m.rows[idx]
}

func buildRows(in []inhibitor.Inhibitor) []table.Row {
	out := make([]table.Row, 0, len(in))
	for _, i := range in {
		// Visual weight for the LOCK column: blocks shout in uppercase so the
		// row jumps out even without per-row ANSI coloring, which bubbles/table
		// does not support cleanly.
		lock := "delay"
		if i.Mode == inhibitor.ModeBlock {
			lock = "BLOCK"
		}
		out = append(out, table.Row{
			fmt.Sprintf("%d", i.PID),
			i.User,
			i.Comm,
			string(i.What),
			lock,
			i.Why,
		})
	}
	return out
}

// SortBlocksFirst returns rows ordered so hard-blocking inhibitors appear
// before delay locks. Stable within each group so the natural logind order
// is preserved.
func SortBlocksFirst(in []inhibitor.Inhibitor) []inhibitor.Inhibitor {
	out := make([]inhibitor.Inhibitor, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		ai := out[i].Mode == inhibitor.ModeBlock
		aj := out[j].Mode == inhibitor.ModeBlock
		if ai != aj {
			return ai
		}
		return false
	})
	return out
}
