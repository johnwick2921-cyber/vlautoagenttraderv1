// E5 — THE WARNING THAT FIRES BY CONSTRUCTION.
//
// backoffWhileClosed sleeps a deliberate 3 minutes (trader/auto_trader_loop.go:
// cmeClosedBackoff) and the overrun check compares the cycle against
// ScanInterval, which the live trader resolves to 2 minutes. A 3-minute sleep
// can never fit inside a 2-minute interval, so on a closed market the warning
// was GUARANTEED — 165 of them in the 2026-09-06 boot log, every single one
// reading "3m0.0XXs > 2m0s".
//
// Owner ruling 2026-09-07: KEEP the sleep, EXEMPT the closed path. A warning
// that cannot indicate a fault must not fire; a real overrun on an open day
// still must.
package trader

import (
	"testing"
	"time"
)

func TestE5_ClosedPathNeverWarns(t *testing.T) {
	const interval = 2 * time.Minute
	// The exact shape the log recorded 165 times: the deliberate 3m backoff.
	if shouldWarnOverrun(3*time.Minute+9*time.Millisecond, interval, true) {
		t.Errorf("E5: the closed path must NOT warn — the 3m backoff is deliberate and cannot indicate a fault")
	}
	// Even an absurd closed-path duration is not a fault worth this warning.
	if shouldWarnOverrun(59*time.Minute, interval, true) {
		t.Errorf("E5: no closed-path duration should raise the scan-interval warning")
	}
}

func TestE5_OpenDayStillWarnsOnARealOverrun(t *testing.T) {
	const interval = 2 * time.Minute
	if !shouldWarnOverrun(3*time.Minute, interval, false) {
		t.Errorf("E5: a REAL overrun on an open day must still warn — the fix must not blanket-silence the check")
	}
	if !shouldWarnOverrun(interval+time.Millisecond, interval, false) {
		t.Errorf("E5: one millisecond over the interval on an open day must warn")
	}
}

func TestE5_OpenDayUnderIntervalIsSilent(t *testing.T) {
	const interval = 2 * time.Minute
	if shouldWarnOverrun(interval, interval, false) {
		t.Errorf("E5: exactly at the interval is not an overrun")
	}
	if shouldWarnOverrun(time.Second, interval, false) {
		t.Errorf("E5: a fast cycle must be silent")
	}
}
