package inhibitor

import (
	"context"

	"github.com/shawnyeager/dozer/internal/proc"
)

// Collect queries every source in order and returns the merged, enriched list.
// Errors are collected — a failing source does not prevent others from running.
func Collect(ctx context.Context, sources []Source) ([]Inhibitor, []error) {
	var all []Inhibitor
	var errs []error
	for _, s := range sources {
		list, err := s.List(ctx)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		all = append(all, list...)
	}
	return enrich(all), errs
}

// enrich fills in Comm and User from /proc and /etc/passwd for any row that
// came out of a source without them.
func enrich(in []Inhibitor) []Inhibitor {
	out := make([]Inhibitor, 0, len(in))
	for _, i := range in {
		if i.Comm == "" {
			i.Comm = proc.Comm(i.PID)
		}
		if i.User == "" {
			i.User = proc.Username(i.UID)
		}
		out = append(out, i)
	}
	return out
}
