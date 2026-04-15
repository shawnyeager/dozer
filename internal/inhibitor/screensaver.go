package inhibitor

import (
	"context"
	"errors"
)

// ErrNotImplemented is returned by sources that exist only as stubs.
var ErrNotImplemented = errors.New("screensaver source is not implemented: org.freedesktop.ScreenSaver exposes no standard ListInhibitors and hypridle does not implement a vendor extension")

// ScreensaverSource is a placeholder for the session-bus org.freedesktop.ScreenSaver
// interface. It exists so the plumbing is in place; the session-bus probe is
// intentionally unimplemented for v1 — see README "Coverage gap".
type ScreensaverSource struct{}

func NewScreensaverSource() *ScreensaverSource { return &ScreensaverSource{} }

func (s *ScreensaverSource) Name() string { return "screensaver" }

func (s *ScreensaverSource) List(_ context.Context) ([]Inhibitor, error) {
	return nil, ErrNotImplemented
}
