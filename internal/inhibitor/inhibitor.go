package inhibitor

import "context"

type What string
type Mode string

const (
	WhatSleep           What = "sleep"
	WhatIdle            What = "idle"
	WhatShutdown        What = "shutdown"
	WhatHandlePower     What = "handle-power-key"
	WhatHandleSuspend   What = "handle-suspend-key"
	WhatHandleHibernate What = "handle-hibernate-key"
	WhatHandleLid       What = "handle-lid-switch"
	WhatScreensaver     What = "screensaver"
)

const (
	ModeBlock Mode = "block"
	ModeDelay Mode = "delay"
)

type Inhibitor struct {
	PID    int32  `json:"pid"`
	UID    uint32 `json:"uid"`
	User   string `json:"user"`
	Comm   string `json:"comm"`
	What   What   `json:"what"`
	Mode   Mode   `json:"mode"`
	Who    string `json:"who"`
	Why    string `json:"why"`
	Source string `json:"source"`
}

type Source interface {
	Name() string
	List(ctx context.Context) ([]Inhibitor, error)
}
