package tui

import (
	"strings"
	"testing"

	"github.com/shawnyeager/dozer/internal/inhibitor"
)

func TestSortBlocksFirst(t *testing.T) {
	in := []inhibitor.Inhibitor{
		{PID: 1, Mode: inhibitor.ModeDelay, What: inhibitor.WhatSleep},
		{PID: 2, Mode: inhibitor.ModeBlock, What: inhibitor.WhatIdle},
		{PID: 3, Mode: inhibitor.ModeDelay, What: inhibitor.WhatSleep},
		{PID: 4, Mode: inhibitor.ModeBlock, What: inhibitor.WhatSleep},
	}
	out := SortBlocksFirst(in)
	if out[0].PID != 2 || out[1].PID != 4 {
		t.Fatalf("blocks should be first, got PIDs %d, %d", out[0].PID, out[1].PID)
	}
	if out[2].PID != 1 || out[3].PID != 3 {
		t.Fatalf("delay ordering should be stable, got PIDs %d, %d", out[2].PID, out[3].PID)
	}
}

func TestRenderStatusLineEmpty(t *testing.T) {
	got := stripANSI(RenderStatusLine(nil))
	if !strings.Contains(got, "no inhibitors") {
		t.Errorf("want idle status, got %q", got)
	}
}

func TestRenderStatusLineOnlyDelays(t *testing.T) {
	rows := []inhibitor.Inhibitor{
		{Mode: inhibitor.ModeDelay, What: inhibitor.WhatSleep},
		{Mode: inhibitor.ModeDelay, What: inhibitor.WhatSleep},
	}
	got := stripANSI(RenderStatusLine(rows))
	if !strings.Contains(got, "nothing blocking") {
		t.Errorf("want 'nothing blocking', got %q", got)
	}
	if !strings.Contains(got, "2 delay locks") {
		t.Errorf("want delay count, got %q", got)
	}
}

func TestRenderStatusLineWithBlocks(t *testing.T) {
	rows := []inhibitor.Inhibitor{
		{Mode: inhibitor.ModeBlock, What: inhibitor.WhatSleep},
		{Mode: inhibitor.ModeBlock, What: inhibitor.WhatIdle},
		{Mode: inhibitor.ModeDelay, What: inhibitor.WhatSleep},
	}
	got := stripANSI(RenderStatusLine(rows))
	if !strings.Contains(got, "2 blocking") {
		t.Errorf("want '2 blocking', got %q", got)
	}
	if !strings.Contains(got, "idle×1") || !strings.Contains(got, "sleep×1") {
		t.Errorf("want per-what counts in status, got %q", got)
	}
	if !strings.Contains(got, "1 delay locks") {
		t.Errorf("want delay suffix when delays co-exist, got %q", got)
	}
}

func TestRenderStatusLineBlocksOnly(t *testing.T) {
	rows := []inhibitor.Inhibitor{
		{Mode: inhibitor.ModeBlock, What: inhibitor.WhatSleep},
	}
	got := stripANSI(RenderStatusLine(rows))
	if strings.Contains(got, "delay") {
		t.Errorf("should not mention delays when there are none, got %q", got)
	}
}

// stripANSI crudely removes CSI escape sequences so assertions can match on
// visible text without hard-coding the styling.
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if inEscape {
			if (r >= '@' && r <= '~') || r == 'm' {
				inEscape = false
			}
			continue
		}
		if r == 0x1b {
			inEscape = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
