package trader

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nofx/calendar"
	"nofx/kernel"
	"nofx/store"
)

// W-T1-CURRENCIES (2026-09-18) — PRODUCTION CALL SITES (class 53). The
// 2026-09-17 slice (calendar_slices, src forexfactory) carried "BOJ Policy
// Rate" 2026-09-18T02:54:00Z JPY T1 = 21:54 CT in ASIA, and the MNQ bot sat in
// a HARD window for a Japanese rate decision. These tests seed that event next
// to a USD red event and read the three consumers the way production does:
// the arm gate (currentT1Windows → sessionGateDecision), the plan write (the
// real write core with plannerT1Lines as the extra no_trade lines, then the
// stored doc's no_trade + no_trade_windows) and the fade facts (fadeFactsAt).

const t1CcyDate = "2026-09-17"

func t1CcyTrader(t *testing.T, ccy []string) (*AutoTrader, *store.Store) {
	t.Helper()
	// ASIA is enabled through the per-session override — the explicit toggle
	// the production resolver (sessionRunnable) honours first.
	asiaOn := true
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{
		PlanEnabled: true, SessionsEnabled: []string{"ASIA", "NY"}, T1Currencies: ccy,
		Sessions: []store.DayPlanSessionOverride{{Session: "ASIA", Enable: &asiaOn}},
	}}
	at, st := resetTrader(t, cfg)
	orig := clockHoldDriftFn
	clockHoldDriftFn = func(string) (int64, bool) { return 0, false }
	t.Cleanup(func() { clockHoldDriftFn = orig })
	slice := &store.CalendarSliceDB{
		TradeDate: t1CcyDate, Source: "forexfactory",
		EventsJSON: `[{"time":"2026-09-18T01:00:00Z","currency":"USD","title":"Fed Chair Powell Speaks","impact":"T1"},
		              {"time":"2026-09-18T02:54:00Z","currency":"JPY","title":"BOJ Policy Rate","impact":"T1"}]`,
		CreatedAt: time.Now().UnixMilli(),
	}
	if _, err := st.Calendar().SaveSliceIfAbsent(slice); err != nil {
		t.Fatalf("save slice: %v", err)
	}
	return at, st
}

func t1CcyAt(h, m int) time.Time { return time.Date(2026, 9, 17, h, m, 0, 0, chicagoLoc()) }

func t1CcyEvents(t *testing.T, st *store.Store) []kernel.PlannerCalendarEvent {
	t.Helper()
	slice, err := st.Calendar().GetSlice(t1CcyDate)
	if err != nil || slice == nil {
		t.Fatalf("slice: %v", err)
	}
	var evs []calendar.Event
	if err := json.Unmarshal([]byte(slice.EventsJSON), &evs); err != nil {
		t.Fatal(err)
	}
	return sessionPlannerEvents(evs, "ASIA")
}

// writeASIAPlan runs the REAL write core for ASIA with plannerT1Lines as the
// extra no_trade lines — exactly what runPlannerReadWithTriggerClaimed does —
// and returns the stored doc.
func writeASIAPlan(t *testing.T, at *AutoTrader, st *store.Store) kernel.PlanDoc {
	t.Helper()
	t1Lines := at.plannerT1Lines(t1CcyEvents(t, st), false, 0, 0, t1CcyDate, "ASIA")
	facts := kernel.PlanFacts{Price: 15550, DATR: 300}
	machine := map[float64]string{15480: "PWL", 15700: "RN 15700"}
	// The authoring clock is the test's (class 60/113): 16:55 CT on the trade
	// date, the scheduled ASIA read.
	ver, lc, err := at.runPlannerReadCoreWithFactsGradesClock(func() time.Time { return t1CcyAt(16, 55) }, "ASIA", t1CcyDate, "owner_reset",
		"deepseek-v4-pro", "hashT1ccy", "", "", "", "PROMPT", facts, nil, machine, nil, true,
		func(string) (string, error) { return class39LegsPlanJSON("15550"), nil }, t1Lines...)
	if err != nil || lc != "active" {
		t.Fatalf("write must land: ver=%d lc=%q err=%v", ver, lc, err)
	}
	row, err := st.Plan().GetLatestPlanForSession(t1CcyDate, "ASIA")
	if err != nil || row == nil {
		t.Fatalf("stored plan row: %v", err)
	}
	var doc kernel.PlanDoc
	if err := json.Unmarshal([]byte(row.Doc), &doc); err != nil {
		t.Fatalf("stored doc: %v", err)
	}
	return doc
}

func t1WindowsOfKind(doc kernel.PlanDoc) []kernel.NoTradeWindow {
	var out []kernel.NoTradeWindow
	for _, w := range doc.NoTradeWindows {
		if w.Kind == kernel.KindT1 {
			out = append(out, w)
		}
	}
	return out
}

// THE PIN: under the shipped default the BOJ (JPY) event gates nothing on any
// path and is visible as an advisory line; the USD event hard-blocks on every
// path; all three consumers agree.
func TestT1DefaultUSD_BOJIsAdvisoryOnEveryProductionPath(t *testing.T) {
	at, st := t1CcyTrader(t, nil)
	const wantBOJ = "🟠 BOJ Policy Rate 21:54 CT (JPY) — red news, advisory only (t1_currencies=USD)"

	// (1) ARM GATE — currentT1Windows feeds sessionGateDecision.
	windows := at.currentT1Windows(t1CcyAt(20, 0))
	if len(windows) != 1 || !strings.HasPrefix(windows[0].Label, "Fed Chair Powell Speaks 20:00 CT ±15m") {
		t.Fatalf("arm gate: only the USD event may open a window, got %+v", windows)
	}
	reg := at.sessionRegistry(t1CcyAt(20, 0))
	if why, blocked := sessionGateDecision(reg, t1CcyAt(20, 0), at.currentT1Windows(t1CcyAt(20, 0)), at.sessionRunnable); !blocked || !strings.Contains(why, "red-news blackout") {
		t.Fatalf("20:00 CT (USD red) must be blocked: blocked=%v why=%q", blocked, why)
	}
	if why, blocked := sessionGateDecision(reg, t1CcyAt(21, 54), at.currentT1Windows(t1CcyAt(21, 54)), at.sessionRunnable); blocked {
		t.Fatalf("21:54 CT (BOJ, JPY) must be TRADEABLE under the USD default, got blocked: %q", why)
	}

	// (2) PLAN WRITE — the real write core: band + no_trade lines.
	doc := writeASIAPlan(t, at, st)
	t1 := t1WindowsOfKind(doc)
	if len(t1) != 1 || t1[0].StartMin != 19*60+45 || t1[0].EndMin != 20*60+15 {
		t.Fatalf("the doc's machine band must carry ONLY the USD window (19:45–20:15): %+v", t1)
	}
	if t1[0].StartMin != windows[0].Start || t1[0].EndMin != windows[0].End {
		t.Fatalf("band %d–%d disagrees with the arm gate %d–%d", t1[0].StartMin, t1[0].EndMin, windows[0].Start, windows[0].End)
	}
	joined := strings.Join(doc.NoTrade, "\n")
	if !strings.Contains(joined, "🔴 Fed Chair Powell Speaks 20:00 CT ±15m — HARD no-trade (red news)") {
		t.Fatalf("the USD hard line is missing from no_trade: %v", doc.NoTrade)
	}
	if !strings.Contains(joined, wantBOJ) {
		t.Fatalf("the BOJ advisory line must be VISIBLE in no_trade: %v", doc.NoTrade)
	}
	if strings.Contains(joined, "🔴 BOJ") {
		t.Fatalf("the BOJ event must never be a HARD line under the USD default: %v", doc.NoTrade)
	}
	for _, w := range doc.NoTradeWindows {
		if strings.Contains(w.Label, "BOJ") {
			t.Fatalf("an advisory must NEVER reach the machine band: %+v", w)
		}
	}

	// (3) FADE FACTS — the InT1Blackout fact reads the same split.
	if f := at.fadeFactsAt(t1CcyAt(21, 54), "MNQ", 15550, nil, "short"); !f.CalendarHasSlice || f.InT1Blackout {
		t.Fatalf("fade facts at 21:54 (BOJ): slice=%v inT1=%v — must be known and NOT in blackout", f.CalendarHasSlice, f.InT1Blackout)
	}
	if f := at.fadeFactsAt(t1CcyAt(20, 0), "MNQ", 15550, nil, "short"); !f.InT1Blackout {
		t.Fatal("fade facts at 20:00 (USD red) must read InT1Blackout")
	}
}

// ["ALL"] restores the pre-wave behaviour on every path: the BOJ event is a
// hard window, no advisory line exists.
func TestT1All_BOJIsHardOnEveryProductionPath(t *testing.T) {
	at, st := t1CcyTrader(t, []string{"ALL"})
	windows := at.currentT1Windows(t1CcyAt(21, 54))
	if len(windows) != 2 {
		t.Fatalf("ALL: both events open windows, got %+v", windows)
	}
	if why, blocked := sessionGateDecision(at.sessionRegistry(t1CcyAt(21, 54)), t1CcyAt(21, 54), windows, at.sessionRunnable); !blocked || !strings.Contains(why, "BOJ") {
		t.Fatalf("ALL: 21:54 CT must be red-news-blocked by BOJ: blocked=%v why=%q", blocked, why)
	}
	doc := writeASIAPlan(t, at, st)
	if t1 := t1WindowsOfKind(doc); len(t1) != 2 {
		t.Fatalf("ALL: the band carries both windows, got %+v", t1)
	}
	joined := strings.Join(doc.NoTrade, "\n")
	if !strings.Contains(joined, "🔴 BOJ Policy Rate 21:54 CT ±15m — HARD no-trade (red news)") || strings.Contains(joined, "🟠") {
		t.Fatalf("ALL: BOJ is a hard line and nothing is advisory: %v", doc.NoTrade)
	}
	if f := at.fadeFactsAt(t1CcyAt(21, 54), "MNQ", 15550, nil, "short"); !f.InT1Blackout {
		t.Fatal("ALL: fade facts at 21:54 must read InT1Blackout")
	}
}

// A saved multi-currency list is case-folded and honoured by the gate.
func TestT1SavedListCaseFolded(t *testing.T) {
	at, _ := t1CcyTrader(t, []string{"usd", " jpy "})
	if got := at.t1Currencies(); strings.Join(got, ",") != "USD,JPY" {
		t.Fatalf("resolved set = %v", got)
	}
	if _, blocked := sessionGateDecision(at.sessionRegistry(t1CcyAt(21, 54)), t1CcyAt(21, 54), at.currentT1Windows(t1CcyAt(21, 54)), at.sessionRunnable); !blocked {
		t.Fatal("usd+jpy: BOJ must hard-block")
	}
}

// The boot line is READ from the resolver, never a literal.
func TestT1CurrenciesBootLine(t *testing.T) {
	cases := map[string]*store.DayPlanConfig{
		"🔴 t1_blackout=USD(default) (W-T1-CURRENCIES)":   nil,
		"🔴 t1_blackout=USD,EUR(saved) (W-T1-CURRENCIES)": {T1Currencies: []string{"usd", "EUR"}},
		"🔴 t1_blackout=ALL(saved) (W-T1-CURRENCIES)":     {T1Currencies: []string{"*"}},
	}
	for want, dp := range cases {
		if got := T1CurrenciesBootLine(dp); got != want {
			t.Errorf("boot line:\n got %q\nwant %q", got, want)
		}
	}
	if got := T1CurrenciesBootLine(&store.DayPlanConfig{T1Currencies: []string{" "}}); got != "🔴 t1_blackout=USD(default) (W-T1-CURRENCIES)" {
		t.Errorf("a blank-only list is the default, not saved: %q", got)
	}
	if got := T1CurrenciesBootLine(&store.DayPlanConfig{T1Currencies: []string{"USD"}}); got != "🔴 t1_blackout=USD(saved) (W-T1-CURRENCIES)" {
		t.Errorf("an explicit USD prints saved: %q", got)
	}
}
