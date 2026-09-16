package trader

import (
	"strings"
	"testing"

	"nofx/kernel"
)

// F6 (2026-08-30) — clock-hold fixtures: injected drift > tolerance must defer
// plan authoring with the exact "authoring DEFERRED" line, and write nothing.

func TestClockHoldAuthoringDeferredByInjectedDrift(t *testing.T) {
	at := plannerTestTrader(t)
	orig := clockHoldDriftFn
	clockHoldDriftFn = func(symbol string) (int64, bool) { return -61_000, true }
	defer func() { clockHoldDriftFn = orig }()

	deferred, widen, drift, have := at.clockHoldAuthoring()
	if !deferred || !have {
		t.Fatalf("61s drift must defer authoring: deferred=%v have=%v", deferred, have)
	}
	if widen != 61_000 || drift != -61_000 {
		t.Fatalf("widen must carry the absolute drift and drift the signed one: widen=%d drift=%d", widen, drift)
	}

	// The journal line the operator will grep for:
	line := clockHoldDeferLine("2026-08-30", "ASIA", widen, kernel.C2ToleranceMs())
	for _, want := range []string{"clock-hold", "authoring DEFERRED", "2026-08-30 ASIA", "no plan written", "61000", "60000"} {
		if !strings.Contains(line, want) {
			t.Fatalf("defer line %q missing %q", line, want)
		}
	}
}

func TestRunPlannerReadDeferredByClockHold(t *testing.T) {
	at := plannerTestTrader(t)
	orig := clockHoldDriftFn
	clockHoldDriftFn = func(symbol string) (int64, bool) { return -61_000, true }
	defer func() { clockHoldDriftFn = orig }()

	ok := at.runPlannerReadWithTriggerClaimed("ASIA", "2026-08-30", "")
	if ok {
		t.Fatal("a broken clock must not claim/run a planner read")
	}
	got, _ := at.store.Plan().GetLatestPlanForSession("2026-08-30", "ASIA")
	if got != nil {
		t.Fatalf("no plan row may be written under clock-hold, got %+v", got)
	}
}

func TestClockHoldWarnBandDoesNotDefer(t *testing.T) {
	at := plannerTestTrader(t)
	orig := clockHoldDriftFn
	clockHoldDriftFn = func(symbol string) (int64, bool) { return 41_000, true }
	defer func() { clockHoldDriftFn = orig }()

	deferred, widen, _, _ := at.clockHoldAuthoring()
	if deferred {
		t.Fatal("warn band (41s < 60s tolerance) must NOT defer authoring")
	}
	if widen != 41_000 {
		t.Fatalf("warn band must widen news windows by the measured drift, got %d", widen)
	}
}

func TestClockHoldNoMeasurementFailsOpen(t *testing.T) {
	at := plannerTestTrader(t)
	orig := clockHoldDriftFn
	clockHoldDriftFn = func(symbol string) (int64, bool) { return 0, false }
	defer func() { clockHoldDriftFn = orig }()

	deferred, widen, _, _ := at.clockHoldAuthoring()
	if deferred || widen != 0 {
		t.Fatalf("no measurement must fail open: deferred=%v widen=%d", deferred, widen)
	}
}

func TestClockHoldPositiveDriftNeverDefers(t *testing.T) {
	at := plannerTestTrader(t)
	orig := clockHoldDriftFn
	clockHoldDriftFn = func(symbol string) (int64, bool) { return 600_000, true }
	defer func() { clockHoldDriftFn = orig }()

	// A closed market's 10-min-old bars produce the same positive drift —
	// the 16:55 Sunday ASIA read must still fire (P0B contract).
	deferred, widen, _, _ := at.clockHoldAuthoring()
	if deferred {
		t.Fatal("positive drift must NOT defer authoring (ambiguous with a closed market)")
	}
	if widen != 600_000 {
		t.Fatalf("positive drift must still widen news windows, got %d", widen)
	}
}
