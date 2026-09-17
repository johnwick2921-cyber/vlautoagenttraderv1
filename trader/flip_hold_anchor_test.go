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

// W-FLIP-HOLD-ANCHOR (2026-09-17) — tests at the PRODUCTION call site
// (describeActivePlanDeath, the function that logs flip_eval_skipped). The
// condition window is the version's birth (2×5m confirm closes need ≥10 min
// of post-version bars, so "re-read N minutes ago" is chosen so two 5m
// buckets close inside the window); the HOLD reads the chain anchor.

// flipHoldTape builds a 1m tape ending at now: flat at 100 until breakMinute
// minutes before now, then closes at 104 (touching 100 on the first up bar so
// the P1c touch gate passes). All bars are closed at now.
func flipHoldTape(now time.Time, minutes, breakMinutesBeforeNow int) []market.Kline {
	start := now.Truncate(5 * time.Minute).Add(-time.Duration(minutes) * time.Minute)
	brk := now.Truncate(5 * time.Minute).Add(-time.Duration(breakMinutesBeforeNow) * time.Minute)
	out := make([]market.Kline, 0, minutes)
	for t := start; t.Before(now.Truncate(time.Minute)); t = t.Add(time.Minute) {
		ot := t.UnixMilli()
		b := market.Kline{OpenTime: ot, CloseTime: ot + 60_000 - 1, Open: 100, High: 100, Low: 100, Close: 100}
		if !t.Before(brk) {
			b = market.Kline{OpenTime: ot, CloseTime: ot + 60_000 - 1, Open: 100, High: 104, Low: 100, Close: 104}
		}
		out = append(out, b)
	}
	return out
}

func flipHoldTrader(t *testing.T) (*AutoTrader, *store.Store, time.Time) {
	t.Helper()
	t.Setenv("FLIP_ATR_BUFFER", "0")
	t.Setenv("FLIP_MIN_HOLD_MIN", "")
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true, ReplanCap: 4, SessionsEnabled: []string{"NY"}}}
	at, st := resetTrader(t, cfg)
	// 14:35:30 UTC = 09:35:30 CT, 30s past a 5m boundary so the 09:30 bucket is
	// the newest CLOSED bucket (fresh for the G7 gate).
	now := time.Date(2026, 8, 18, 14, 35, 30, 0, time.UTC)
	testNow = func() time.Time { return now }
	t.Cleanup(func() { testNow = nil })
	t.Cleanup(func() { market.FuturesBarsProvider = nil; traderTestBarsInstalled = false })
	return at, st, now
}

func shortPlanDoc(t *testing.T) string {
	t.Helper()
	doc := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "short", FlipCondition: "flips long on 2x5m above 100"}, FlipStructured: &kernel.PlanCondition{Price: 100, Side: "above", Rule: "2x5m", FlipTo: "long"}}
	blob, _ := json.Marshal(doc)
	return string(blob)
}

func appendVersion(t *testing.T, st *store.Store, at *AutoTrader, td, trigger, doc string, at0 time.Time) *store.PlanDB {
	t.Helper()
	pid := store.MakePlanIDForTrader(at.id, td, "NY")
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: pid, TradeDate: td, Session: "NY", StrategyID: at.id, TriggerReason: trigger, Lifecycle: "active", Doc: doc, CreatedAt: at0}); err != nil {
		t.Fatal(err)
	}
	row, err := st.Plan().GetLatestPlanForTraderSession(td, "NY", at.id)
	if err != nil || row == nil {
		t.Fatalf("read back: %v", err)
	}
	return row
}

// insertTransitionAt appends a lifecycle-log row with an explicit timestamp
// (UpdatePlanLifecycle stamps time.Now(), which the frozen clock cannot use).
func insertTransitionAt(t *testing.T, st *store.Store, pid string, version int, event, reason string, at0 time.Time) {
	t.Helper()
	if _, err := st.DB().Exec("INSERT INTO plan_lifecycle_log (plan_id, version, event, reason, at) VALUES (?, ?, ?, ?, ?)", pid, version, event, reason, at0); err != nil {
		t.Fatal(err)
	}
}

// (a) v3 re-read 11 minutes ago, session plan born 45 minutes ago → EVALUATED.
func TestFlipHoldAnchorReReadDoesNotRestartHold(t *testing.T) {
	at, st, now := flipHoldTrader(t)
	td := "2026-08-18"
	doc := shortPlanDoc(t)
	appendVersion(t, st, at, td, "NY_scheduled_read", doc, now.Add(-45*time.Minute))
	appendVersion(t, st, at, td, "level_event", doc, now.Add(-25*time.Minute))
	row := appendVersion(t, st, at, td, "level_event", doc, now.Add(-11*time.Minute))
	barsAt(flipHoldTape(now, 40, 10)) // break 10 min ago: two 5m closes at 104 inside v3's window
	buf := captureTraderLog(t)

	detail, dead := at.describeActivePlanDeath(row)

	if !dead || !strings.Contains(detail.Killer, "flip-condition") {
		t.Fatalf("flip must be EVALUATED and fire (chain 45 min old): dead=%v killer=%q log=%s", dead, detail.Killer, buf.String())
	}
	if strings.Contains(buf.String(), "flip=hold") {
		t.Fatalf("a same-bias re-read must not restart the hold:\n%s", buf.String())
	}
	if a := at.flipHoldAnchor(row); a.Source != kernel.FlipHoldAnchorBirth || a.SinceMs != now.Add(-45*time.Minute).UnixMilli() {
		t.Fatalf("anchor must be the chain birth, got %+v", a)
	}
}

// (b) v1 born 12 minutes ago → held (a fresh session plan still cannot flip
// in its first 30 minutes).
func TestFlipHoldAnchorFirstVersionStillHeld(t *testing.T) {
	at, st, now := flipHoldTrader(t)
	td := "2026-08-18"
	row := appendVersion(t, st, at, td, "NY_scheduled_read", shortPlanDoc(t), now.Add(-12*time.Minute))
	barsAt(flipHoldTape(now, 40, 10))
	buf := captureTraderLog(t)

	_, dead := at.describeActivePlanDeath(row)

	if dead {
		t.Fatalf("v1 inside its first 30 min must be HELD:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "flip=hold") || !strings.Contains(buf.String(), kernel.FlipHoldAnchorBirth) {
		t.Fatalf("want flip=hold since %s on the skip line:\n%s", kernel.FlipHoldAnchorBirth, buf.String())
	}
}

// (c) a flip 20 minutes ago (then a re-arm), followed by a same-bias re-read
// → held: the anti double-flip chop survives; the re-read neither restarts
// nor shortens it.
func TestFlipHoldAnchorAfterFlipThenReReadHeld(t *testing.T) {
	at, st, now := flipHoldTrader(t)
	td := "2026-08-18"
	doc := shortPlanDoc(t)
	v1 := appendVersion(t, st, at, td, "NY_scheduled_read", doc, now.Add(-60*time.Minute))
	insertTransitionAt(t, st, v1.PlanID, 1, "dormant", "dormant:flip:flip-condition: 2x5m close above 100 → bias long", now.Add(-20*time.Minute))
	insertTransitionAt(t, st, v1.PlanID, 1, "active", "rearmed:2x5m close back below 100", now.Add(-19*time.Minute))
	row := appendVersion(t, st, at, td, "level_event", doc, now.Add(-15*time.Minute))
	barsAt(flipHoldTape(now, 40, 14)) // three 5m closes at 104 inside v2's window
	buf := captureTraderLog(t)

	_, dead := at.describeActivePlanDeath(row)

	if dead {
		t.Fatalf("flip 20 min ago + re-read must still HOLD:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "flip=hold") || !strings.Contains(buf.String(), "since "+kernel.FlipHoldAnchorRearm) {
		t.Fatalf("want flip=hold since %s:\n%s", kernel.FlipHoldAnchorRearm, buf.String())
	}
	// flip WITHOUT a re-arm anchors on the flip itself.
	if _, err := st.DB().Exec("DELETE FROM plan_lifecycle_log WHERE reason LIKE 'rearmed:%'"); err != nil {
		t.Fatal(err)
	}
	if a := at.flipHoldAnchor(row); a.Source != kernel.FlipHoldAnchorFlip || a.SinceMs != now.Add(-20*time.Minute).UnixMilli() {
		t.Fatalf("anchor must be the flip, got %+v", a)
	}
	// and once the hold has elapsed (flip 40 min ago) the same tape flips.
	if _, err := st.DB().Exec("UPDATE plan_lifecycle_log SET at = ?", now.Add(-40*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, dead := at.describeActivePlanDeath(row); !dead {
		t.Fatalf("flip 40 min ago must be past the hold")
	}
}

// (d) REPLAY 2026-09-16 ASIA from the live store: v1 16:35:56 … v13 01:21:25
// (all level_event re-reads, bias short except v5 neutral), live MNQ 1m tape
// 00:00–01:40 CT, evaluation at 01:35:00 CT (the journal's "plan age 815s").
// Pre-fix: held. With the fix: the flip is EVALUATED and fires.
func TestFlipHoldAnchorReplayASIA0916(t *testing.T) {
	t.Setenv("FLIP_MIN_HOLD_MIN", "")
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true, ReplanCap: 4, SessionsEnabled: []string{"ASIA"}}}
	at, st := resetTrader(t, cfg)
	ct := kernel.CTLocation()
	now := time.Date(2026, 9, 17, 1, 35, 0, 0, ct)
	testNow = func() time.Time { return now }
	t.Cleanup(func() { testNow = nil })
	t.Cleanup(func() { market.FuturesBarsProvider = nil; traderTestBarsInstalled = false })

	td := "2026-09-16"
	pid := store.MakePlanIDForTrader(at.id, td, "ASIA")
	type ver struct {
		trigger    string
		bias       string
		d, h, m, s int
	}
	chain := []ver{
		{"ASIA_scheduled_read", "short", 16, 16, 35, 56},
		{"level_event", "short", 16, 17, 19, 47},
		{"level_event", "short", 16, 19, 2, 59},
		{"level_event", "short", 16, 19, 50, 25},
		{"level_event", "neutral", 16, 20, 28, 20},
		{"level_event", "short", 16, 21, 0, 40},
		{"level_event", "short", 16, 21, 39, 5},
		{"level_event", "short", 16, 22, 19, 36},
		{"level_event", "short", 16, 22, 58, 4},
		{"level_event", "short", 16, 23, 36, 2},
		{"level_event", "short", 17, 0, 13, 28},
		{"level_event", "short", 17, 0, 39, 28},
		{"level_event", "short", 17, 1, 21, 25},
	}
	v13Doc := kernel.PlanDoc{
		Bias:            kernel.PlanBias{Direction: "short", Conviction: "low", FlipCondition: "2x5m closes above 29418.80 VWAP flip to long"},
		FlipStructured:  &kernel.PlanCondition{Price: 29418.8, Side: "above", Rule: "2x5m", FlipTo: "long"},
		DeathStructured: &kernel.PlanCondition{Price: 29450.5, Side: "above", Rule: "5m_close"},
	}
	v13Blob, _ := json.Marshal(v13Doc)
	for i, v := range chain {
		doc := `{"bias":{"direction":"` + v.bias + `"}}`
		if i == len(chain)-1 {
			doc = string(v13Blob)
		}
		born := time.Date(2026, 9, v.d, v.h, v.m, v.s, 0, ct)
		if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: pid, TradeDate: td, Session: "ASIA", StrategyID: at.id, TriggerReason: v.trigger, Lifecycle: "active", Doc: doc, CreatedAt: born}); err != nil {
			t.Fatal(err)
		}
	}
	row, err := st.Plan().GetLatestPlanForTraderSession(td, "ASIA", at.id)
	if err != nil || row == nil || row.Version != 13 {
		t.Fatalf("v13 read back: %+v err=%v", row, err)
	}
	bars := make([]market.Kline, 0, len(asia0916OneMinuteBars))
	for _, r := range asia0916OneMinuteBars {
		ot := int64(r[0])
		bars = append(bars, market.Kline{OpenTime: ot, CloseTime: ot + 60_000 - 1, Open: r[1], High: r[2], Low: r[3], Close: r[4], Volume: r[5]})
	}
	barsAt(bars)

	// The defect, pinned to the journal: on the version clock the plan is
	// 815s old at 01:35:00 and the flip is held.
	sinceMs := row.CreatedAt.UnixMilli()
	_, firedOld, skippedOld := kernel.PlanDeathOrFlipSinceFreshHold(v13Doc, bars, at.acceptanceRuleFor("ASIA"), sinceMs, now.UnixMilli(), kernel.FlipHoldAnchor{SinceMs: sinceMs, Source: kernel.FlipHoldAnchorVersion})
	if firedOld || !strings.Contains(strings.Join(skippedOld, ";"), "flip=hold (hold age 815s") {
		t.Fatalf("pre-fix replay must reproduce the 815s hold: fired=%v skipped=%v", firedOld, skippedOld)
	}

	buf := captureTraderLog(t)
	detail, dead := at.describeActivePlanDeath(row)
	if strings.Contains(buf.String(), "flip=hold") {
		t.Fatalf("01:35 evaluation must NOT be skipped for hold:\n%s", buf.String())
	}
	if !dead || !strings.Contains(detail.Killer, "flip-condition") || !strings.Contains(detail.Killer, "bias long") {
		t.Fatalf("replay must flip to long at 01:35: dead=%v killer=%q", dead, detail.Killer)
	}
	a := at.flipHoldAnchor(row)
	// v6 (21:00:40) is the last bias change (v5 neutral → v6 short); v7–v13 are
	// same-bias re-reads and move nothing.
	if a.Source != kernel.FlipHoldAnchorBiasChange || a.SinceMs != time.Date(2026, 9, 16, 21, 0, 40, 0, ct).UnixMilli() {
		t.Fatalf("anchor must be v6's bias change, got %+v", a)
	}
}
