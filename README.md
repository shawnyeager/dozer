# dozer

> See what's keeping your Linux box awake, and kill it.

dozer is a terminal UI for systemd-logind inhibitor locks: the things that quietly keep your machine from sleeping, your screen from idling, and your laptop lid from suspending. It gives you a one-line answer to "why won't this thing sleep?", shows you who's responsible, and lets you SIGTERM the offender without leaving the terminal.

I wrote it because `systemd-inhibit --list` tells you what's happening but won't let you do anything about it, and because my screen kept refusing to lock on Hyprland and I wanted a faster way to find out who to blame.

---

## Demo

```
● 1 blocking: idle×1  ·  6 delay locks

PID      USER   PROCESS          BLOCKING  LOCK   REASON
1582120  shawn  systemd-inhibit  idle      BLOCK  video call
3237     root   upowerd          sleep     delay  Pause device polling
3647     shawn  hypridle         sleep     delay  Hypridle wants to delay sleep…
3785     root   rtkit-daemon     sleep     delay  Demote realtime scheduling…
156269   shawn  signal-desktop   sleep     delay  Application cleanup before suspend
3436     shawn  hypridle         sleep     delay  Hypridle wants to delay sleep…
3469     shawn  electron         sleep     delay  Application cleanup before suspend

  ↑/k up  ↓/j down  r refresh  x SIGTERM  q quit
```

Green dot: nothing is hard-blocking. The delay locks are harmless, more on that in a minute. Red dot: something is actually keeping the machine awake. It sorts to the top of the table; you select it with `j`/`k`, press `x`, confirm with `y`.

## Why

`systemd-inhibit --list` dumps six lines of delay locks and one real blocker in the same monospace font, with no way to tell which is which and no way to kill any of them. If you're on Hyprland or hypridle you've probably already run it twice this week. dozer is basically that command with a live refresh and a kill button.

## What it does

At the top of the screen there's a single status line. A red dot means something is hard-blocking sleep right now, with a count and a breakdown by type. A green dot means only harmless delay locks are present. A gray dot means nothing is inhibiting at all, which on a busy Linux box is more unusual than you'd think.

Block rows always sort to the top of the table, and the LOCK column uses uppercase `BLOCK` versus lowercase `delay` so the difference still lands on a monochrome terminal.

The table refreshes every second from `org.freedesktop.login1.Manager.ListInhibitors` over the system D-Bus, so new inhibitors show up without you touching anything. If you select a row you own, `x` sends SIGTERM with a `[y/N]` confirmation. If you select a row owned by another user, you get a `cannot signal pid N: owned by <user>` message instead of a failed syscall.

For scripting, `dozer --once` prints a tab-aligned table and `dozer --json` emits machine-readable output. Color gets stripped automatically when stdout isn't a TTY, so output pipes cleanly. The binary is static, has no runtime dependencies beyond systemd-logind, and talks to D-Bus directly through [godbus/dbus/v5](https://github.com/godbus/dbus). Nothing shells out to `systemd-inhibit`.

## Install

```sh
go install github.com/shawnyeager/dozer/cmd/dozer@latest
```

Or build from source:

```sh
git clone https://github.com/shawnyeager/dozer
cd dozer
go build ./cmd/dozer
./dozer
```

Requires Go 1.22+ and a systemd-based Linux host.

## Usage

Interactive TUI (the default):

```sh
dozer
```

One-shot, non-interactive:

```sh
dozer --once              # tab-aligned table, suitable for shell pipelines
dozer --json              # JSON array with full field names
dozer --interval 500ms    # custom TUI refresh cadence
```

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | move selection |
| `r` | refresh now |
| `x` | SIGTERM selected process (with confirmation) |
| `y` / `n` / `esc` | confirm / cancel pending kill |
| `q`, `Ctrl+C` | quit |

Only SIGTERM is wired up. If something genuinely ignores a polite signal you can reach for `kill -9` in another pane, and that's rare enough not to deserve its own keybind.

## Block vs. delay locks

systemd-logind has two inhibitor strengths and they behave very differently.

A `BLOCK` lock is a hard stop. The action (sleep, idle, etc.) cannot proceed at all until the process releases the lock. This is what actually keeps your machine awake. dozer surfaces these at the top of the table and flips the status line red when any exist.

A `delay` lock is a grace period. When you request sleep, logind waits up to `InhibitDelayMaxUSec` (default 5 seconds) for the process to clean up, then sleeps regardless. Delay locks are almost always harmless; they're how daemons like `upower`, `rtkit`, `hypridle`, and Electron apps ask for a moment to flush state on the way down. Every row in the demo above is a delay lock, and none of them are keeping that machine awake.

If dozer says "nothing blocking" and your machine still won't sleep, the culprit is outside systemd-logind. See the coverage gap below.

## Data source

dozer reads inhibitors from systemd-logind via the system D-Bus:

```
org.freedesktop.login1.Manager.ListInhibitors  →  a(ssssuu)
                                                  (what, who, why, mode, uid, pid)
```

That catches every process holding a `sleep`, `idle`, `shutdown`, `handle-power-key`, `handle-suspend-key`, `handle-hibernate-key`, or `handle-lid-switch` lock, which is everything `systemd-inhibit --list` shows. D-Bus calls go through [godbus/dbus/v5](https://github.com/godbus/dbus) directly. No subprocesses, no text parsing.

### Coverage gap

Apps that inhibit idle only via the legacy `org.freedesktop.ScreenSaver` session D-Bus interface (typically browsers playing video, Zoom, and some Electron apps) are not currently listed. The D-Bus spec defines `Inhibit` and `UnInhibit` but no standard `ListInhibitors`. Cookies are opaque, and implementations like hypridle don't expose a vendor extension either. A `--experimental-screensaver` source is stubbed but unimplemented.

If your screen won't lock during video playback on Hyprland + hypridle, dozer probably can't see the culprit. Sorry. Open an issue if you know a compositor that exposes idle inhibitors enumerably and I'll have a look.

## Architecture

```
cmd/dozer/            entry point: flag parsing, --once, --json, TUI launch
internal/inhibitor/   Source interface, logind source, Collect() merger
internal/proc/        /proc/<pid> reads for comm, cmdline, username
internal/killer/      syscall.Kill wrapper with EPERM/ESRCH preservation
internal/tui/         bubbletea model: keys, styles, model, update, view
```

Sources implement a small interface:

```go
type Source interface {
    Name() string
    List(ctx context.Context) ([]Inhibitor, error)
}
```

so adding a GNOME SessionManager, KDE PowerManagement, or compositor-specific source later is a new file, not a refactor.

## Development

```sh
go test ./...    # decoder, Collect merge, killer, tui status line + sort
go vet ./...
go build -o dozer ./cmd/dozer
```

Tests are pure. No D-Bus, no subprocesses, no `/proc` reads from the test binary. The logind decoder is extracted from the D-Bus call path so the colon-packed `what` field (`sleep:idle`) can be unit tested without a live bus.

## Prior art

- [`systemd-inhibit --list`](https://www.freedesktop.org/software/systemd/man/systemd-inhibit.html), the one-shot non-interactive version that ships with systemd. dozer wraps it in a live UI with a kill action.
- [`htop`](https://htop.dev/), spiritual ancestor of every interactive `top`-style TUI.
- [Bubble Tea](https://github.com/charmbracelet/bubbletea), the TUI runtime.

## License

MIT © Shawn Yeager. See [LICENSE](LICENSE).
