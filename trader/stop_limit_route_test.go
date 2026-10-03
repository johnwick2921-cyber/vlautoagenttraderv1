package trader

import (
	"os"
	"strings"
	"testing"
	"time"

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

// N12 predicate pin (PR B, 2026-10-03): an unfilled stop-limit is stale at the
// candle close; nothing else is. Remove the state or the knob guard and a
// filled/terminal row gets re-cancelled, or a knob-OFF row changes behaviour.
func TestStopLimitStaleAtCandleClose(t *testing.T) {
	lastClose := time.Date(2026, 10, 3, 9, 31, 0, 0, time.UTC)
	before := lastClose.Add(-time.Minute)
	after := lastClose.Add(time.Minute)
	row := store.ArmedOrderDB{
		Kind: "stop_entry", State: store.StateArmed, UpdatedAt: before,
	}
	t.Setenv("MENTOR_STOP_LIMIT", "1")
	if !stopLimitStaleAtCandleClose(row, lastClose) {
		t.Fatalf("an unfilled stop-limit placed before the close must be stale")
	}
	row.UpdatedAt = after
	if stopLimitStaleAtCandleClose(row, lastClose) {
		t.Fatalf("a stop-limit placed inside the current candle is never stale")
	}
	row.UpdatedAt = before
	row.State = store.StateWorking
	if stopLimitStaleAtCandleClose(row, lastClose) {
		t.Fatalf("a FILLED stop entry is never stale")
	}
	row.State = store.StateCancelPending
	if stopLimitStaleAtCandleClose(row, lastClose) {
		t.Fatalf("a cancel_pending row is owned by the settlement pass, never this sweep")
	}
	row.State = store.StateArmed
	row.Kind = "limit"
	if stopLimitStaleAtCandleClose(row, lastClose) {
		t.Fatalf("a limit arm is never swept by the stop-limit candle close")
	}
	t.Setenv("MENTOR_STOP_LIMIT", "")
	if stopLimitStaleAtCandleClose(row, lastClose) {
		t.Fatalf("the knob OFF must keep the behaviour byte-identical: never stale")
	}
}

// N12 call-site pin (PR B, 2026-10-03): the candle-close sweep is READ in the
// armed pass at the stop branch. Remove the branch and the predicate keeps
// passing while no row is ever cancelled at the candle close — the exact
// "built ≠ wired" class A29 named.
func TestStopLimitCandleCloseSweepIsWiredInTheArmedPass(t *testing.T) {
	src, err := os.ReadFile("armed_executor.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "stopLimitStaleAtCandleClose(r, time.UnixMilli(last.CloseTime))") {
		t.Fatal("the N12 sweep must call the stale predicate in the armed pass stop branch")
	}
}
