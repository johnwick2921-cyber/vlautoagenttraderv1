package trader

import (
	"os"
	"strings"
	"testing"

	"vl/store"
)

// Routing pin (PR B, 2026-10-03) at the production call site: placeOneStopEntry
// must route the stop entry through PlaceStopEntryWithLimit when
// MENTOR_STOP_LIMIT is ON and through PlaceStopEntry when it is OFF. Remove the
// branch and the knob silently keeps sending stop-MARKET frames — the exact
// "built ≠ wired ≠ used" failure class A29 / N5 named.
func TestStopLimitKnobRoutesStopEntriesThroughTheLimitVariant(t *testing.T) {
	for _, tc := range []struct {
		name      string
		env       string
		wantLimit bool
	}{
		{"knob-off", "", false},
		{"knob-on", "1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MENTOR_STOP_LIMIT", tc.env)
			at, _ := resetTrader(t, store.StrategyConfig{})
			at.id = "route-" + tc.name
			r := store.ArmedOrderDB{
				ID: 8, TraderID: at.id, PlanID: "2026-09-23:NY", Version: 1,
				Session: "TEST-R", Scenario: "TEST-R", Side: "long",
				EntryPx: 29650, StopPx: 29640, TargetPx: 29670,
			}
			d := decideStopEntry("LONG", r.EntryPx, testOffset(), testTick, 99)
			if d.Action != stopEntryPlace {
				t.Fatalf("fixture must be an otherwise-placeable arm, got %q", d.Action)
			}
			pl := &fakePlacer{}
			if got := at.placeOneStopEntry(pl, &fakeLedger{}, r, d, 99, rthInstant(), freeSlot()); got != stopPlaceCommitted {
				t.Fatalf("a real send must be COMMITTED, got %d", got)
			}
			if pl.stopLimitCalls != map[bool]int{true: 1, false: 0}[tc.wantLimit] {
				t.Fatalf("knob %q routed %d stop entries through the limit variant, want %d",
					tc.env, pl.stopLimitCalls, map[bool]int{true: 1, false: 0}[tc.wantLimit])
			}
		})
	}
}

// N12 expiry pin (PR B, 2026-10-03): an unfilled order with a stored expiry is
// due at now >= expiry_ms; nothing else is. A missing expiry (0) is never
// swept — no intent expired it. Remove the state guard and a filled/terminal
// row gets re-cancelled.
func TestArmExpired(t *testing.T) {
	nowMs := int64(1760000000000)
	row := store.ArmedOrderDB{State: store.StateArmed}
	if armExpired(row, nowMs) {
		t.Fatalf("no stored expiry → never due (a blanket sweep is exactly what the CTO vetoed)")
	}
	row.ExpiryMs = nowMs + 60_000
	if armExpired(row, nowMs) {
		t.Fatalf("an expiry in the future must be kept")
	}
	row.ExpiryMs = nowMs
	if !armExpired(row, nowMs) {
		t.Fatalf("an unfilled order at its expiry is due")
	}
	row.ExpiryMs = nowMs - 1
	if !armExpired(row, nowMs) {
		t.Fatalf("a lapsed unfilled order is due")
	}
	row.State = store.StatePlacePending
	if !armExpired(row, nowMs) {
		t.Fatalf("a place_pending order at its expiry is due")
	}
	row.State = store.StateWorking
	row.FillQuantity = 0
	if !armExpired(row, nowMs) {
		t.Fatalf("a working, UNFILLED order is due — a stop-limit resting at NT8 IS the order the expiry exists to cancel")
	}
	row.FillQuantity = 1
	if armExpired(row, nowMs) {
		t.Fatalf("a working order with a partial fill is a trade in progress — never due")
	}
	row.FillQuantity = 0
	row.State = store.StateCancelPending
	if armExpired(row, nowMs) {
		t.Fatalf("a cancel_pending row is owned by the settlement pass, never this sweep")
	}
	row.State = store.StateCancelled
	if armExpired(row, nowMs) {
		t.Fatalf("a terminal row is never swept by its expiry")
	}
}

// N12 call-site pin (PR B, 2026-10-03): the expiry sweep is READ in the armed
// pass before any placement branch, and the partial-fill guard (a working row
// with qty > 0 at its expiry is KEPT, logged, never cancelled) is wired beside
// it. Remove either and armExpired keeps passing while live rows are missed —
// the exact "built ≠ wired" class A29 named.
func TestExpirySweepIsWiredInTheArmedPass(t *testing.T) {
	src, err := os.ReadFile("armed_executor.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "armExpired(r, now.UnixMilli())") {
		t.Fatal("the expiry sweep must call the due predicate in the armed pass row loop")
	}
	if !strings.Contains(string(src), "expiry_partial_fill:") {
		t.Fatal("the partial-fill guard must be wired beside the expiry sweep")
	}
}
