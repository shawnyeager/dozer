package inhibitor

import "testing"

func TestDecodeLogindSplitsColonSeparatedWhat(t *testing.T) {
	raw := []logindEntry{
		{What: "sleep:idle", Who: "hypridle", Why: "waiting", Mode: "block", UID: 1000, PID: 123},
	}
	got := decodeLogind(raw)
	if len(got) != 2 {
		t.Fatalf("want 2 rows, got %d", len(got))
	}
	if got[0].What != WhatSleep || got[1].What != WhatIdle {
		t.Errorf("unexpected what fields: %v, %v", got[0].What, got[1].What)
	}
	for _, g := range got {
		if g.PID != 123 || g.UID != 1000 || g.Source != "logind" {
			t.Errorf("unexpected row: %+v", g)
		}
	}
}

func TestDecodeLogindSkipsBlankTokens(t *testing.T) {
	raw := []logindEntry{{What: "::sleep::", Mode: "delay", PID: 1}}
	got := decodeLogind(raw)
	if len(got) != 1 || got[0].What != WhatSleep {
		t.Fatalf("want single sleep row, got %+v", got)
	}
}

func TestDecodeLogindEmpty(t *testing.T) {
	if got := decodeLogind(nil); got != nil {
		t.Fatalf("want nil for empty input, got %v", got)
	}
}
