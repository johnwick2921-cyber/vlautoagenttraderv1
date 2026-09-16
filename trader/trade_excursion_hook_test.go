package trader

import (
	"errors"
	"testing"
)

// F5 — CLASS 23 PIN. This wave is telemetry. A bad bar, a busy database or a
// nil dependency must produce a WARN and nothing else; it may never panic and
// it may never stop the trading loop.
func TestExcursionTelemetryNeverTakesTheLoopDown(t *testing.T) {
	at := plannerTestTrader(t)

	cases := map[string]func() error{
		"panic in the body":  func() error { var p *int; _ = *p; return nil },
		"error from the DB":  func() error { return errors.New("database is locked") },
		"index out of range": func() error { s := []int{}; _ = s[3]; return nil },
		"clean":              func() error { return nil },
	}
	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("excursion telemetry escaped as a panic (%v) — the loop would die", r)
				}
			}()
			at.safeExcursion("test:"+name, fn) // must swallow everything
		})
	}
}

// The hooks themselves must survive a trader with no store and no bars —
// the shape every unit test and every crypto trader has.
func TestExcursionHooksAreInertWithoutDeps(t *testing.T) {
	at := &AutoTrader{} // no store, no trader, no provider
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a bare AutoTrader panicked: %v", r)
		}
	}()
	at.excursionOnOpen(nil, 0, 0, 0)
	at.excursionOnBarTick()
	at.excursionOnClose(nil)
}
