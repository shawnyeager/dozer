package inhibitor

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	logindBus    = "org.freedesktop.login1"
	logindPath   = "/org/freedesktop/login1"
	logindIface  = "org.freedesktop.login1.Manager"
	logindMethod = logindIface + ".ListInhibitors"
)

// LogindSource reads inhibitors from systemd-logind via the system D-Bus.
// ListInhibitors returns a(ssssuu) = (what, who, why, mode, uid, pid).
type LogindSource struct {
	conn *dbus.Conn
}

func NewLogindSource() (*LogindSource, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect to system bus: %w", err)
	}
	return &LogindSource{conn: conn}, nil
}

func (s *LogindSource) Name() string { return "logind" }

func (s *LogindSource) Close() error {
	if s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

// logindEntry matches the D-Bus signature a(ssssuu).
type logindEntry struct {
	What string
	Who  string
	Why  string
	Mode string
	UID  uint32
	PID  uint32
}

func (s *LogindSource) List(ctx context.Context) ([]Inhibitor, error) {
	obj := s.conn.Object(logindBus, dbus.ObjectPath(logindPath))
	var raw []logindEntry
	if err := obj.CallWithContext(ctx, logindMethod, 0).Store(&raw); err != nil {
		return nil, fmt.Errorf("logind ListInhibitors: %w", err)
	}
	return decodeLogind(raw), nil
}

// decodeLogind is pulled out of List so it can be unit tested without D-Bus.
// logind packs multiple "what" values as colon-separated ("sleep:idle"); we
// emit one Inhibitor per token so the UI treats them as distinct rows.
func decodeLogind(raw []logindEntry) []Inhibitor {
	var out []Inhibitor
	for _, e := range raw {
		for _, w := range strings.Split(e.What, ":") {
			w = strings.TrimSpace(w)
			if w == "" {
				continue
			}
			out = append(out, Inhibitor{
				PID:    int32(e.PID),
				UID:    e.UID,
				Who:    e.Who,
				Why:    e.Why,
				What:   What(w),
				Mode:   Mode(e.Mode),
				Source: "logind",
			})
		}
	}
	return out
}
