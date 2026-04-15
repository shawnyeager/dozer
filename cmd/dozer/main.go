package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shawnyeager/dozer/internal/inhibitor"
	"github.com/shawnyeager/dozer/internal/tui"
)

func main() {
	var (
		once     = flag.Bool("once", false, "print inhibitors once and exit")
		asJSON   = flag.Bool("json", false, "emit inhibitors as JSON and exit")
		interval = flag.Duration("interval", time.Second, "TUI refresh interval")
	)
	flag.Parse()

	logind, err := inhibitor.NewLogindSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dozer: %v\n", err)
		fmt.Fprintln(os.Stderr, "dozer requires systemd-logind; is systemd running?")
		os.Exit(1)
	}
	defer logind.Close()

	sources := []inhibitor.Source{logind}

	if *once || *asJSON {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		rows, errs := inhibitor.Collect(ctx, sources)
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "dozer: %v\n", e)
		}
		if *asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(rows); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
		rows = tui.SortBlocksFirst(rows)
		fmt.Fprintln(os.Stdout, tui.RenderStatusLine(rows))
		fmt.Fprintln(os.Stdout)
		printOnce(os.Stdout, rows)
		return
	}

	p := tea.NewProgram(tui.New(sources, *interval), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "dozer: %v\n", err)
		os.Exit(1)
	}
}

func printOnce(w io.Writer, rows []inhibitor.Inhibitor) {
	if len(rows) == 0 {
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PID\tUSER\tPROCESS\tBLOCKING\tLOCK\tREASON")
	for _, r := range rows {
		lock := "delay"
		if r.Mode == inhibitor.ModeBlock {
			lock = "BLOCK"
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n",
			r.PID, r.User, r.Comm, r.What, lock, r.Why)
	}
	tw.Flush()
}
