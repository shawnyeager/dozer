package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shawnyeager/dozer/internal/inhibitor"
)

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(RenderStatusLine(m.rows))
	b.WriteString("\n\n")
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// One optional line below the table: confirm prompt, kill feedback, or
	// error. Only one ever shows; rendering is conditional so the layout stays
	// flat when there's nothing to say.
	switch {
	case m.state == stateConfirmKill:
		if sel := m.selected(); sel != nil {
			b.WriteString("\n")
			b.WriteString(confirmStyle.Render(
				fmt.Sprintf("Send SIGTERM to pid %d (%s)?  [y/N]", sel.PID, sel.Comm),
			))
			b.WriteString("\n")
		}
	case m.killMessage != "":
		b.WriteString("\n")
		b.WriteString(muted.Render(m.killMessage))
		b.WriteString("\n")
	case m.lastErr != nil:
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("error: " + m.lastErr.Error()))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.help.View(m.keys))
	return b.String()
}

// RenderStatusLine produces the one-line headline that answers "is anything
// actually blocking?". Exported so --once reuses it; lipgloss/termenv will
// strip color automatically when stdout isn't a TTY.
func RenderStatusLine(rows []inhibitor.Inhibitor) string {
	if len(rows) == 0 {
		return idleDot.Render("○") + " " + muted.Render("no inhibitors")
	}

	blocks, delays := 0, 0
	blockWhats := map[string]int{}
	for _, r := range rows {
		if r.Mode == inhibitor.ModeBlock {
			blocks++
			blockWhats[string(r.What)]++
		} else {
			delays++
		}
	}

	if blocks == 0 {
		return okDot.Render("●") + " " +
			muted.Render(fmt.Sprintf("nothing blocking · %d delay locks", delays))
	}

	parts := make([]string, 0, len(blockWhats))
	for k, v := range blockWhats {
		parts = append(parts, fmt.Sprintf("%s×%d", k, v))
	}
	sort.Strings(parts)

	line := alertDot.Render("●") + " " +
		alertText.Render(fmt.Sprintf("%d blocking", blocks)) +
		": " + strings.Join(parts, ", ")
	if delays > 0 {
		line += muted.Render(fmt.Sprintf("  ·  %d delay locks", delays))
	}
	return line
}
