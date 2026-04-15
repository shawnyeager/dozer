package inhibitor

import (
	"context"
	"errors"
	"testing"
)

type fakeSource struct {
	name string
	list []Inhibitor
	err  error
}

func (f *fakeSource) Name() string                              { return f.name }
func (f *fakeSource) List(context.Context) ([]Inhibitor, error) { return f.list, f.err }

func TestCollectMergesSources(t *testing.T) {
	a := &fakeSource{name: "a", list: []Inhibitor{{PID: 1, What: WhatSleep, Source: "a"}}}
	b := &fakeSource{name: "b", list: []Inhibitor{{PID: 2, What: WhatIdle, Source: "b"}}}
	rows, errs := Collect(context.Background(), []Source{a, b})
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
}

func TestCollectCollectsErrorsAndContinues(t *testing.T) {
	a := &fakeSource{name: "a", err: errors.New("boom")}
	b := &fakeSource{name: "b", list: []Inhibitor{{PID: 2, What: WhatIdle}}}
	rows, errs := Collect(context.Background(), []Source{a, b})
	if len(errs) != 1 {
		t.Fatalf("want 1 err, got %d", len(errs))
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row after recoverable error, got %d", len(rows))
	}
}

func TestScreensaverSourceIsStub(t *testing.T) {
	s := NewScreensaverSource()
	if _, err := s.List(context.Background()); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
