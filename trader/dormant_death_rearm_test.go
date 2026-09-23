package trader

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// W-DORMANT-DEATH-REARM (2026-09-17) — a plan parked by its DEATH line must be
// judged for re-arm by its DEATH condition, never its flip condition. Since D3
// the dormant kind lives in the lifecycle log (dormant:death:…/dormant:flip:…),
// but describeDormantCleared keyed off plans.trigger_reason — the AUTHORING
// trigger — so every D3+ dormant row fell to the flip predicate: a death-dormant
// plan re-armed when its FLIP line cleared while the death line stayed breached,
// or (no flip line) re-armed immediately on "no machine condition".
//
// All tests run through the PRODUCTION call site maybeRunSessionReadsAt with a
// real store row parked by the real write path (UpdatePlanLifecycle).

// dormantDeathTape builds 24 one-minute bars: flat minutes at `flat`, then
// `drift` minutes at `driftPx`, ending 6 minutes before now.
func dormantDeathTape(now time.Time, flatPx, driftPx float64) {
	t0 := now.Add(-30 * time.Minute).UnixMilli()
	bars := make([]market.Kline, 0, 24)
	for i := 0; i < 14; i++ {
		ot := t0 + int64(i)*60_000
		bars = append(bars, market.Kline{OpenTime: ot, CloseTime: ot + 60_000 - 1, Open: flatPx, High: flatPx, Low: flatPx, Close: flatPx})
	}
	for i := 0; i < 10; i++ { // two 5m buckets on the drift side
		ot := t0 + int64(14+i)*60_000
		bars = append(bars, market.Kline{OpenTime: ot, CloseTime: ot + 60_000 - 1, Open: driftPx, High: driftPx, Low: driftPx, Close: driftPx})
	}
	barsAt(bars)
}

// dormantDeathFixture parks (via the real write path) a dormant plan with the
// given doc and marker, inside the NY read window at 09:00 CT.
func dormantDeathFixture(t *testing.T, marker string, doc kernel.PlanDoc, legacyTrigger string) (*AutoTrader, *store.Store, *store.PlanDB, time.Time) {
	t.Helper()
	t.Setenv("FLIP_ATR_BUFFER", "0")
	t.Setenv("DORMANT_MIN_HOLD_MIN", "0")
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true, ReplanCap: store.IntPtr(4), SessionsEnabled: []string{"NY"}}}
	at, st := resetTrader(t, cfg)
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, time.UTC) // 09:00 CT, inside NY
	testNow = func() time.Time { return now }
	t.Cleanup(func() { testNow = nil })
	t.Cleanup(func() { market.FuturesBarsProvider = nil; traderTestBarsInstalled = false })

	td := "2026-08-18"
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	pid := store.MakePlanIDForTrader(at.id, td, "NY")
	birth := now.Add(-40 * time.Minute)
	if legacyTrigger != "" {
		// Pre-D3 legacy row: the marker lived in trigger_reason and there is no
		// lifecycle-log event (the log did not exist before 2026-09-03).
		if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: pid, TradeDate: td, Session: "NY", StrategyID: at.id, Lifecycle: "dormant", TriggerReason: legacyTrigger, Doc: string(blob), CreatedAt: birth}); err != nil {
			t.Fatal(err)
		}
	} else {
		if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: pid, TradeDate: td, Session: "NY", StrategyID: at.id, Lifecycle: "active", Doc: string(blob), CreatedAt: birth}); err != nil {
			t.Fatal(err)
		}
		if err := st.Plan().UpdatePlanLifecycle(pid, 1, "dormant", marker); err != nil {
			t.Fatal(err)
		}
	}
	row, err := st.Plan().GetLatestPlanForTraderSession(td, "NY", at.id)
	if err != nil || row == nil || row.Lifecycle != "dormant" {
		t.Fatalf("parked row: %+v err=%v", row, err)
	}
	return at, st, row, now
}

func dormantDeathRow(t *testing.T, st *store.Store, row *store.PlanDB) *store.PlanDB {
	t.Helper()
	fresh, err := st.Plan().GetLatestPlanForTraderSession(row.TradeDate, row.Session, row.StrategyID)
	if err != nil || fresh == nil {
		t.Fatalf("read back: %v", err)
	}
	return fresh
}

func lastLifecycleEvent(t *testing.T, st *store.Store, row *store.PlanDB) store.PlanLifecycleEvent {
	t.Helper()
	events, err := st.Plan().LifecycleLog(row.PlanID, row.Version)
	if err != nil || len(events) == 0 {
		t.Fatalf("lifecycle log: %v (%d events)", err, len(events))
	}
	return events[len(events)-1]
}

// RED today: a death-dormant plan with NO flip line re-arms immediately on
// "no machine condition" because the death marker is never read.
func TestDeathDormantNoFlipStaysDormantWhileDeathBreached(t *testing.T) {
	doc := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "long"}, DeathStructured: &kernel.PlanCondition{Price: 100, Side: "below", Rule: "2x5m"}}
	at, st, row, now := dormantDeathFixture(t, "dormant:death:death-condition: 2x5m close below 100.00", doc, "")
	dormantDeathTape(now, 96, 96) // death line (below 100) still breached

	at.maybeRunSessionReadsAt(now)
	defer drainReReads(t) // CTO M4: join the async re-read before the seam resets

	fresh := dormantDeathRow(t, st, row)
	if fresh.Lifecycle != "dormant" {
		t.Fatalf("death-dormant must NOT re-arm while the death line stays breached, got %s", fresh.Lifecycle)
	}
}

// RED today: with BOTH lines, the flip line clearing re-arms a plan whose death
// line is still breached — the wrong predicate. Price 96: above the flip (90),
// below the death (100).
func TestDeathDormantFlipClearedDeathBreachedStaysDormant(t *testing.T) {
	doc := kernel.PlanDoc{
		Bias:            kernel.PlanBias{Direction: "long"},
		DeathStructured: &kernel.PlanCondition{Price: 100, Side: "below", Rule: "2x5m"},
		FlipStructured:  &kernel.PlanCondition{Price: 90, Side: "below", Rule: "2x5m", FlipTo: "short"},
	}
	at, st, row, now := dormantDeathFixture(t, "dormant:death:death-condition: 2x5m close below 100.00", doc, "")
	dormantDeathTape(now, 96, 96) // above the flip line, below the death line

	at.maybeRunSessionReadsAt(now)
	defer drainReReads(t) // CTO M4: join the async re-read before the seam resets

	fresh := dormantDeathRow(t, st, row)
	if fresh.Lifecycle != "dormant" {
		t.Fatalf("death-dormant must be judged by its DEATH line (still breached), got %s", fresh.Lifecycle)
	}
}

// The death line clearing re-arms — and the re-arm reason must name the DEATH
// line (100.00), not the flip line (90.00), proving the death predicate ran.
func TestDeathDormantRearmsWhenDeathClears(t *testing.T) {
	doc := kernel.PlanDoc{
		Bias:            kernel.PlanBias{Direction: "long"},
		DeathStructured: &kernel.PlanCondition{Price: 100, Side: "below", Rule: "2x5m"},
		FlipStructured:  &kernel.PlanCondition{Price: 90, Side: "below", Rule: "2x5m", FlipTo: "short"},
	}
	at, st, row, now := dormantDeathFixture(t, "dormant:death:death-condition: 2x5m close below 100.00", doc, "")
	dormantDeathTape(now, 96, 104) // two 5m buckets close back above 100

	at.maybeRunSessionReadsAt(now)
	defer drainReReads(t) // CTO M4: join the async re-read before the seam resets

	fresh := dormantDeathRow(t, st, row)
	if fresh.Lifecycle != "active" {
		t.Fatalf("death-dormant must re-arm when the death line clears, got %s", fresh.Lifecycle)
	}
	last := lastLifecycleEvent(t, st, row)
	if !strings.HasPrefix(last.Reason, "rearmed:") || !strings.Contains(last.Reason, "above 100.00") {
		t.Fatalf("re-arm must be judged by the death line, got %q", last.Reason)
	}
}

// Regression: a flip-dormant plan still re-arms on its FLIP line, and the
// reason names the flip line (90.00) — the flip predicate is preserved.
func TestFlipDormantRearmsWhenFlipClears(t *testing.T) {
	doc := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "long"}, FlipStructured: &kernel.PlanCondition{Price: 90, Side: "below", Rule: "2x5m", FlipTo: "short"}}
	at, st, row, now := dormantDeathFixture(t, "dormant:flip:flip-condition: 2x5m close below 90.00 → bias short", doc, "")
	dormantDeathTape(now, 96, 104) // closes back above 90

	at.maybeRunSessionReadsAt(now)
	defer drainReReads(t) // CTO M4: join the async re-read before the seam resets

	fresh := dormantDeathRow(t, st, row)
	if fresh.Lifecycle != "active" {
		t.Fatalf("flip-dormant must re-arm when its flip line clears, got %s", fresh.Lifecycle)
	}
	last := lastLifecycleEvent(t, st, row)
	if !strings.HasPrefix(last.Reason, "rearmed:") || !strings.Contains(last.Reason, "above 90.00") {
		t.Fatalf("re-arm must be judged by the flip line, got %q", last.Reason)
	}
}

// Pre-D3 legacy rows carry the marker in trigger_reason and have no log event;
// the fallback keeps them judged by the death line.
func TestLegacyTriggerReasonDeathDormantStillJudgedByDeath(t *testing.T) {
	doc := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "long"}, DeathStructured: &kernel.PlanCondition{Price: 100, Side: "below", Rule: "2x5m"}}

	// breached → stays dormant
	at, st, row, now := dormantDeathFixture(t, "", doc, "dormant:death:death-condition: 2x5m close below 100.00")
	dormantDeathTape(now, 96, 96)
	at.maybeRunSessionReadsAt(now)
	defer drainReReads(t) // CTO M4: join the async re-read before the seam resets
	if fresh := dormantDeathRow(t, st, row); fresh.Lifecycle != "dormant" {
		t.Fatalf("legacy death-dormant must stay dormant while breached, got %s", fresh.Lifecycle)
	}

	// cleared → re-arms on the death line
	at2, st2, row2, now2 := dormantDeathFixture(t, "", doc, "dormant:death:death-condition: 2x5m close below 100.00")
	dormantDeathTape(now2, 96, 104)
	at2.maybeRunSessionReadsAt(now2)
	if fresh := dormantDeathRow(t, st2, row2); fresh.Lifecycle != "active" {
		t.Fatalf("legacy death-dormant must re-arm when the death line clears, got %s", fresh.Lifecycle)
	}
	last := lastLifecycleEvent(t, st2, row2)
	if !strings.HasPrefix(last.Reason, "rearmed:") || !strings.Contains(last.Reason, "above 100.00") {
		t.Fatalf("legacy re-arm must be judged by the death line, got %q", last.Reason)
	}
}
