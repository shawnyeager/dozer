---
title: "feat: Idle/Screensaver Inhibitor TUI (dozer)"
type: feat
status: completed
date: 2026-04-15
---

# feat: Idle/Screensaver Inhibitor TUI (`dozer`)

> **Rename history:** `inhibctl` (planning) → `uninhibit` (v1 build) → `dozer` (final). Repo lives at `github.com/shawnyeager/dozer`, binary is `dozer`. Tagline: *"See what's keeping your Linux box awake, and kill it."*

## Overview

A small Go CLI — working name `inhibctl` — that lists every process currently inhibiting system idle, sleep, or screensaver on a Linux/systemd box, renders them in an `htop`-style interactive table built with [Charm Bubble Tea](https://github.com/charmbracelet/bubbletea), and lets the user browse and kill offenders without leaving the terminal. Target environment is Arch + Hyprland + systemd-logind (the author's daily driver), but the tool should work anywhere `systemd-logind` and an optional `org.freedesktop.ScreenSaver` session service are present.

## Problem Statement / Motivation

On Wayland/Hyprland, the screen refuses to lock or the machine refuses to sleep for reasons that are hard to diagnose. `systemd-inhibit --list` prints a one-shot table, but:

- It's not interactive — you can't sort, filter, or act on a row.
- It only covers `org.freedesktop.login1` inhibitors. Apps that inhibit via the legacy `org.freedesktop.ScreenSaver` session interface (Zoom, browsers playing video, Signal, Electron apps) are missed.
- Killing an offender means copy-pasting the PID into another terminal.
- The output does not refresh, so you can't watch inhibitors come and go.

The goal is a single binary that answers **"what is keeping my machine awake right now, and let me kill it"** in under five seconds.

## Proposed Solution

A Bubble Tea program with three panes stacked vertically:

1. **Header** — summary line (`N inhibitors • sleep:3 idle:2 shutdown:1 • refresh 1s`) plus a legend for keybindings.
2. **Table** — sortable table (via `github.com/charmbracelet/bubbles/table`) of inhibitors with columns: `PID`, `USER`, `COMM`, `WHAT`, `MODE`, `SOURCE`, `WHY`.
3. **Detail / confirm strip** — shows the full `WHY` text for the highlighted row; when the user presses `k` or `K`, swaps to a confirmation prompt before signalling.

Data is pulled directly from D-Bus (no shelling out to `systemd-inhibit`) using [`github.com/godbus/dbus/v5`](https://github.com/godbus/dbus). Two sources are merged:

| Source | Bus | Object | Method |
|---|---|---|---|
| systemd-logind | system | `/org/freedesktop/login1` | `org.freedesktop.login1.Manager.ListInhibitors` → `a(ssssuu)` (what, who, why, mode, uid, pid) |
| Screensaver (session) | session | `/org/freedesktop/ScreenSaver` or `/ScreenSaver` | Enumerate cookies heuristically — see "System-Wide Impact" |

Refresh cadence is a 1-second `tea.Tick`. Killing uses `syscall.Kill(pid, SIGTERM)` with an optional `--force` / `K` binding that escalates to `SIGKILL`.

## Technical Considerations

- **Language/runtime:** Go 1.22+. Single static binary. No CGO.
- **Dependencies (all MIT/BSD):**
  - `github.com/charmbracelet/bubbletea` — TUI runtime
  - `github.com/charmbracelet/bubbles` — reusable `table`, `help`, `key` components
  - `github.com/charmbracelet/lipgloss` — styling
  - `github.com/godbus/dbus/v5` — D-Bus client
- **Privilege model:** Reading logind inhibitors requires no root. Killing PIDs owned by other users does — surface this in the UI by greying out rows you can't signal and showing `permission denied` in the detail strip rather than failing the whole refresh.
- **Wayland note:** The Wayland `zwp_idle_inhibit_manager_v1` protocol is compositor-internal; we cannot enumerate surface-level inhibitors without compositor help. Hyprland does not currently expose them via `hyprctl`. Document this gap; most offenders also hold a logind or ScreenSaver inhibitor, so the practical coverage is high.
- **Cross-desktop:** GNOME uses `/org/gnome/SessionManager`, KDE uses `org.freedesktop.PowerManagement.Inhibit`. Design the source layer as a pluggable `Source` interface so these can be added later without touching the Bubble Tea model.

## System-Wide Impact

- **Interaction graph:** TUI → `Source.List()` (logind + screensaver) → merged `[]Inhibitor` → `table.Model` rows. Kill action → `syscall.Kill` → next tick re-queries and the row disappears (or doesn't, which is itself a useful signal).
- **Error propagation:** D-Bus errors from either source must not crash the program. The renderer shows a red footer (`logind: connection refused`) and keeps the last good snapshot. `killFailed` is a `tea.Msg` that surfaces in the detail strip, not a panic.
- **State lifecycle risks:** The screensaver DBus interface has no standard `ListInhibitors`. Different implementations (hypridle, xss-lock, KDE, GNOME) expose different methods — some only track cookies returned from `Inhibit`. Strategy: call the implementation-specific method when present (`ListInhibitors` on freedesktop, `GetInhibitors` on GNOME); fall back to listing every session-bus name that *holds* the `org.freedesktop.ScreenSaver` well-known name and any peer that has called `Inhibit` in this session (requires `org.freedesktop.DBus.Monitoring` or subscribing to `NameOwnerChanged`). Ship v1 with logind-only + a best-effort screensaver probe, and document the limitation.
- **API surface parity:** One `Source` interface, two initial implementations. Adding GNOME/KDE later is a new file, not a refactor.
- **Integration test scenarios:**
  1. Start the TUI with zero inhibitors → empty-state message, not a crash.
  2. Launch `systemd-inhibit --what=idle sleep 300` in another terminal → appears within one tick, `WHAT=idle`.
  3. Kill that process from inside the TUI → row disappears on next tick, exit code of child is `SIGTERM`.
  4. Try to kill a `root`-owned inhibitor as an unprivileged user → row stays, detail strip shows `EPERM`, program keeps running.
  5. Stop the system bus mid-session (`sudo systemctl stop dbus` in a VM) → footer shows error, program doesn't crash, reconnects when bus returns.

## Acceptance Criteria

- [x] `go build ./cmd/uninhibit` produces a single static binary. (Project renamed `inhibctl` → `uninhibit` per user.)
- [x] Running `uninhibit` with no flags opens the TUI and populates from logind within 500ms.
- [x] Table columns: `PID`, `USER`, `COMM`, `WHAT`, `MODE`, `SOURCE`, `WHY`.
- [x] Keybindings: `j/k/↑/↓` navigate, `r` refresh now, `x` SIGTERM, `X` SIGKILL, `y/n` confirm/cancel, `q`/`Ctrl+C` quit, `?` toggle help. (Kill binding is `x`/`X` rather than `K` to avoid collision with vim-style `k`-nav. Sort cycle and filter dropped for v1 — logind usually returns <10 rows, so not needed yet.)
- [x] Auto-refresh every 1s via `tea.Tick`; confirm prompt swallows non-y/n keys so table nav doesn't fire mid-confirm.
- [x] Rows the current UID cannot signal surface `cannot signal pid N: owned by <user> (uid N)` in the status strip instead of attempting the kill.
- [x] `--once` flag prints a non-interactive tab-aligned table and exits 0.
- [x] `--json` flag emits the merged inhibitor list as JSON and exits 0.
- [x] Unit tests for the logind decoder (`decodeLogind`) covering colon-packed `what` fields, blank tokens, and empty input.
- [x] Unit tests for `Collect` (merge + error isolation) and the `ScreensaverSource` stub.
- [ ] Golden-file test for the `--once` renderer. (Deferred — real `/proc` varies per host, making the golden flaky. Revisit with a fake source injected into `printOnce`.)
- [x] README documents the Wayland/screensaver coverage gap honestly.

## Success Metrics

- Time from "screen won't lock" to "identified and killed offender" under 10 seconds.
- Zero crashes over a week of daily use on the author's machine.
- Binary size under 15 MB.

## Dependencies & Risks

- **Risk — screensaver enumeration is non-standard.** Mitigation: ship v1 logind-only, mark screensaver as experimental behind `--experimental-screensaver`, collect real-world offenders before hardening.
- **Risk — Bubble Tea `table` component has had API churn.** Mitigation: pin `bubbles` version in `go.mod`, gate upgrades on a snapshot test.
- **Risk — killing the wrong process.** Mitigation: confirm prompt defaults to `No`, shows PID + COMM + full cmdline (read from `/proc/<pid>/cmdline`) before signalling.
- **Dependency — systemd-logind.** The tool is a no-op on non-systemd distros; detect at startup and exit with a clear message rather than an empty table.

## Project Layout

```
inhibctl/
├── go.mod
├── go.sum
├── README.md
├── cmd/
│   └── inhibctl/
│       └── main.go           # flag parsing, mode dispatch (tui / --once / --json)
├── internal/
│   ├── inhibitor/
│   │   ├── inhibitor.go      # type Inhibitor struct{...}; Source interface
│   │   ├── logind.go         # logind source (ListInhibitors over system bus)
│   │   ├── logind_test.go
│   │   ├── screensaver.go    # session-bus screensaver probe (experimental)
│   │   └── merge.go          # dedupe by (pid, what, source)
│   ├── proc/
│   │   └── proc.go           # /proc/<pid>/{comm,cmdline,status} lookups, owner UID
│   ├── tui/
│   │   ├── model.go          # tea.Model: table, help, confirm states
│   │   ├── update.go         # Update() — keybindings, ticks, kill flow
│   │   ├── view.go           # View() — lipgloss layout
│   │   ├── keys.go           # key.Binding map
│   │   └── styles.go         # lipgloss styles
│   └── killer/
│       ├── killer.go         # Kill(pid, signal) with EPERM surfacing
│       └── killer_test.go
└── testdata/
    └── once_golden.txt
```

## MVP Pseudocode

### internal/inhibitor/inhibitor.go

```go
package inhibitor

type What string // "sleep" | "idle" | "shutdown" | "handle-power-key" | "screensaver"
type Mode string // "block" | "delay"

type Inhibitor struct {
    PID    int32
    UID    uint32
    User   string
    Comm   string
    What   What
    Mode   Mode
    Who    string
    Why    string
    Source string // "logind" | "screensaver"
}

type Source interface {
    Name() string
    List(ctx context.Context) ([]Inhibitor, error)
}
```

### internal/inhibitor/logind.go

```go
// Connects to system bus, calls
//   org.freedesktop.login1.Manager.ListInhibitors
// which returns a(ssssuu) = (what, who, why, mode, uid, pid).
// Splits `what` on ":" (logind packs "sleep:idle" in one entry) and emits
// one Inhibitor per token so the UI shows them as distinct rows.
```

### internal/tui/model.go

```go
type state int
const (
    stateBrowsing state = iota
    stateConfirmKill
    stateError
)

type Model struct {
    sources  []inhibitor.Source
    table    table.Model
    help     help.Model
    keys     keyMap
    rows     []inhibitor.Inhibitor
    state    state
    lastErr  error
    killing  *inhibitor.Inhibitor
    interval time.Duration
}

// Init returns tea.Batch(tickCmd(m.interval), refreshCmd(m.sources))
// Update handles tea.KeyMsg, tickMsg, refreshedMsg, killResultMsg.
// View composes header + table + (detail | confirm | error) using lipgloss.JoinVertical.
```

### internal/killer/killer.go

```go
func Kill(pid int32, sig syscall.Signal) error {
    if err := syscall.Kill(int(pid), sig); err != nil {
        return fmt.Errorf("kill %d: %w", pid, err) // wraps EPERM, ESRCH
    }
    return nil
}
```

## Sources & References

### Internal

- `systemd-inhibit --list --no-pager` — reference output format, verified on this machine.
- `busctl introspect org.freedesktop.login1 /org/freedesktop/login1` — confirms `ListInhibitors` signature `a(ssssuu)` and `NCurrentInhibitors` property for change detection.

### External

- Bubble Tea — https://github.com/charmbracelet/bubbletea
- Bubbles (table component) — https://github.com/charmbracelet/bubbles
- Lip Gloss — https://github.com/charmbracelet/lipgloss
- godbus — https://github.com/godbus/dbus
- systemd Inhibitor Locks spec — https://systemd.io/INHIBITOR_LOCKS/
- `org.freedesktop.login1` D-Bus API — https://www.freedesktop.org/software/systemd/man/org.freedesktop.login1.html
- `org.freedesktop.ScreenSaver` spec — https://people.freedesktop.org/~hadess/idle-inhibition-spec/re01.html
- Wayland `zwp_idle_inhibit_manager_v1` — https://wayland.app/protocols/idle-inhibit-unstable-v1 (coverage gap reference)
