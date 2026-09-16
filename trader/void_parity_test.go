package trader

import (
	"os"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// ── VOID PARITY (2026-09-02) — THE PROMPT MUST LIST WHAT THE VALIDATOR REJECTS ─
//
// The 20:58 CT read on rev 575e9c05 was rejected "S2 breakdown_continue: a close
// came back across 29141.25 — the breakdown is void". 29141.25 IS a seated level
// (ONL, on the ranked list at 0.00 pts) and it was NOT in the prompt's VOID list.
//
// Cause: the two sides read DIFFERENT tape.
//   render   auto_trader_planner.go:2304 — sinceMs = CMESessionDayStart, 12,000 bars
//   validate breakdown_continue.go:248   — sinceMs = 0 (whole slice), 2,000 bars
// A level broken and reclaimed BEFORE the session-day boundary is void to the
// validator and invisible to the prompt. ONL is exactly that: an overnight low,
// broken overnight.
//
// The class-45 parity fixture passed 40/40 because it fed BOTH sides the same
// sinceMs. It pinned the functions; it never pinned the CALL SITES.

// onlTape reproduces the shape: a break-and-reclaim of `level` BEFORE the 17:00
// CT session-day boundary, then quiet bars above it up to `now`.
func onlTape(level float64, now time.Time) []market.Kline {
	ct := now.Location()
	start := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, ct) // overnight, pre-boundary
	var bars []market.Kline
	add := func(t time.Time, o, h, l, c float64) {
		bars = append(bars, market.Kline{OpenTime: t.UnixMilli(), Open: o, High: h, Low: l, Close: c,
			CloseTime: t.UnixMilli() + 60_000})
	}
	t := start
	// base ABOVE the level
	for i := 0; i < 20; i++ {
		add(t, level+18, level+22, level+14, level+17)
		t = t.Add(time.Minute)
	}
	// displacement DOWN through the level (the break) — several closes beyond
	for i := 0; i < 14; i++ {
		p := level - float64(4+3*i)
		add(t, p+3, p+4, p-4, p)
		t = t.Add(time.Minute)
	}
	// the RECLAIM: a close back across, still before 17:00 CT
	add(t, level-6, level+9, level-7, level+6)
	t = t.Add(time.Minute)
	// quiet bars above the level all the way to now (crosses the 17:00 boundary)
	for t.Before(now) {
		add(t, level+8, level+12, level+5, level+9)
		t = t.Add(time.Minute)
	}
	return bars
}

// wholeSliceScope is the REJECTED alternative (the first cut of this wave),
// kept only so the mechanism test can show what the ruling avoids: on the real
// tape it voided 20 entries across 12 levels — noise by construction.
func wholeSliceScope(bars []market.Kline) kernel.VoidScope {
	sc := kernel.ResolveVoidScopeOf(bars, time.Now())
	sc.SinceMs = 0
	return sc
}

// inSessionTape breaks and reclaims `level` AFTER the 17:00 CT session-day
// start, i.e. inside today's news.
func inSessionTape(level float64, now time.Time) []market.Kline {
	ct := now.Location()
	start := time.Date(now.Year(), now.Month(), now.Day(), 17, 30, 0, 0, ct)
	var bars []market.Kline
	add := func(t time.Time, o, h, l, c float64) {
		bars = append(bars, market.Kline{OpenTime: t.UnixMilli(), Open: o, High: h, Low: l, Close: c,
			CloseTime: t.UnixMilli() + 60_000})
	}
	t := start
	for i := 0; i < 15; i++ {
		add(t, level+18, level+22, level+14, level+17)
		t = t.Add(time.Minute)
	}
	for i := 0; i < 14; i++ {
		p := level - float64(4+3*i)
		add(t, p+3, p+4, p-4, p)
		t = t.Add(time.Minute)
	}
	add(t, level-6, level+9, level-7, level+6) // the reclaim, in-session
	t = t.Add(time.Minute)
	for t.Before(now) {
		add(t, level+8, level+12, level+5, level+9)
		t = t.Add(time.Minute)
	}
	return bars
}

func voidListHas(v []kernel.VoidBreakdownLevel, price float64, short bool) bool {
	for _, x := range v {
		if x.Price == price && x.Short == short {
			return true
		}
	}
	return false
}

// OWNER-RULED FIXTURE (2026-09-02): a break-and-reclaim that completed BEFORE
// the CME session-day start is NOT today's news — it must be void for NEITHER
// caller. Before this wave it was void for the VALIDATOR only (sinceMs=0) and
// invisible to the prompt, which is the asymmetry that burned the 20:58 read.
func TestVoidParityPreSessionReclaimIsVoidForNeither(t *testing.T) {
	ct := kernel.CTLocation()
	now := time.Date(2026, 9, 2, 20, 58, 0, 0, ct)
	const onl = 29141.25
	bars := onlTape(onl, now)
	nowMs := now.UnixMilli()

	// --- the VALIDATOR's verdict, through the production entry point ---
	doc := &kernel.PlanDoc{Scenarios: []kernel.PlanScenario{{
		ID: "S2", Condition: "breakdown_continue", Direction: "short",
		Breakdown: &kernel.PlanBreakdownContinue{Level: onl, EntryMode: "pullback"},
	}}}
	scope := kernel.ResolveVoidScopeOf(bars, now)
	verr := kernel.ValidateBreakdownContinueScenarios(doc, scope, 20, onl-5, nowMs)
	levels := []kernel.ScoredLevel{{DetectedLevel: kernel.DetectedLevel{Price: onl, Label: "ONL"}}}
	rendered := kernel.ComputeVoidBreakdownLevels(levels, scope, nowMs)

	voidToValidator := verr != nil && strings.Contains(verr.Error(), "the breakdown is void")
	voidToPrompt := voidListHas(rendered, onl, true)

	if voidToValidator {
		t.Errorf("a reclaim BEFORE the session-day start is not today's news — the validator must NOT void it: %v", verr)
	}
	if voidToPrompt {
		t.Errorf("the prompt must not list a pre-session reclaim as void (%d entries)", len(rendered))
	}
	if voidToValidator != voidToPrompt {
		t.Fatalf("PARITY BROKEN: validator=%v prompt=%v for the same level, tape and instant", voidToValidator, voidToPrompt)
	}
	t.Logf("pre-session reclaim: void to neither (validator=%v prompt=%v) · window=%s CT",
		voidToValidator, voidToPrompt, time.UnixMilli(scope.SinceMs).In(ct).Format("01-02 15:04"))
}

// The positive half: a reclaim INSIDE the session day is void for BOTH.
func TestVoidParityInSessionReclaimIsVoidForBoth(t *testing.T) {
	ct := kernel.CTLocation()
	now := time.Date(2026, 9, 2, 20, 58, 0, 0, ct)
	const lvl = 29141.25
	bars := inSessionTape(lvl, now)
	nowMs := now.UnixMilli()
	scope := kernel.ResolveVoidScopeOf(bars, now)

	doc := &kernel.PlanDoc{Scenarios: []kernel.PlanScenario{{
		ID: "S2", Condition: "breakdown_continue", Direction: "short",
		Breakdown: &kernel.PlanBreakdownContinue{Level: lvl, EntryMode: "pullback"},
	}}}
	verr := kernel.ValidateBreakdownContinueScenarios(doc, scope, 20, lvl-5, nowMs)
	levels := []kernel.ScoredLevel{{DetectedLevel: kernel.DetectedLevel{Price: lvl, Label: "ONL"}}}
	rendered := kernel.ComputeVoidBreakdownLevels(levels, scope, nowMs)

	voidToValidator := verr != nil && strings.Contains(verr.Error(), "the breakdown is void")
	if !voidToValidator {
		t.Errorf("an in-session reclaim must void the play: %v", verr)
	}
	if !voidListHas(rendered, lvl, true) {
		t.Errorf("PARITY BROKEN: the validator voids %.2f but the prompt omits it", lvl)
	}
	t.Logf("in-session reclaim: void to both · validator=%v", voidToValidator)
}

// The session-day window is the mechanism, isolated: identical bars, identical
// level, only sinceMs differs.
func TestVoidParityWindowIsTheMechanism(t *testing.T) {
	ct := kernel.CTLocation()
	now := time.Date(2026, 9, 2, 20, 58, 0, 0, ct)
	const onl = 29141.25
	bars := onlTape(onl, now)
	levels := []kernel.ScoredLevel{{DetectedLevel: kernel.DetectedLevel{Price: onl, Label: "ONL"}}}

	whole := kernel.ComputeVoidBreakdownLevels(levels, wholeSliceScope(bars), now.UnixMilli())
	ruled := kernel.ComputeVoidBreakdownLevels(levels, kernel.ResolveVoidScopeOf(bars, now), now.UnixMilli())

	if !voidListHas(whole, onl, true) {
		t.Fatalf("fixture is wrong: a whole-slice scan must see this reclaim")
	}
	if voidListHas(ruled, onl, true) {
		t.Fatalf("the ruled session-day window must NOT report a pre-boundary reclaim as today's news")
	}
	t.Logf("whole-slice=%d entries · ruled session-day=%d entries — the window is the ruling, applied to BOTH sides now", len(whole), len(ruled))
}

// SOURCE PIN — neither call site may choose its own window or bar slice again.
// The class-45 parity fixture pinned the two FUNCTIONS and passed 40/40 while
// production fed them different tape; this pins the CALL SITES.
func TestVoidParityCallSitesUseTheResolver(t *testing.T) {
	for _, f := range []string{"auto_trader_planner.go", "rootfix_shadow_ab.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)
		for _, banned := range []string{
			`ComputeVoidBreakdownLevels(scored, bars1m,`,
			`ValidateBreakdownContinueScenarios(d, bdBars,`,
			`ValidateBreakdownContinueScenarios(d, bars,`,
			`ValidateBreakdownContinueScenarios(d, scope.Bars,`,
			`ComputeVoidBreakdownLevels(scored, bars`,
		} {
			if strings.Contains(src, banned) {
				t.Errorf("%s still picks its own void tape: %q", f, banned)
			}
		}
	}
	// Positively: every void call site names the resolver.
	pb, _ := os.ReadFile("auto_trader_planner.go")
	if n := strings.Count(string(pb), "kernel.ResolveVoidScope("); n < 2 {
		t.Errorf("both void call sites must resolve the scope, found %d in auto_trader_planner.go", n)
	}
	sb, _ := os.ReadFile("rootfix_shadow_ab.go")
	if !strings.Contains(string(sb), "kernel.ResolveVoidScope(") {
		t.Error("the shadow A/B validator must read the same resolved scope")
	}

	// The deleted session-day window must not come back into production code.
	b, _ := os.ReadFile("auto_trader_planner.go")
	if strings.Contains(string(b), "func voidWindowStartMs(") {
		t.Error("voidWindowStartMs is deleted — the window belongs to the resolver")
	}
	// The ruled scope is the CME session day, for BOTH sides.
	ct := kernel.CTLocation()
	at := time.Date(2026, 9, 2, 20, 58, 0, 0, ct)
	if got, want := kernel.VoidScopeSinceMs(nil, at), kernel.CMESessionDayStart(at).UnixMilli(); got != want {
		t.Errorf("the ruled window is the session day: got %d want %d", got, want)
	}
	if kernel.VoidScopeBarCount() != kernel.AISVPBarCount {
		t.Errorf("bar depth must match the validator's historical slice, got %d", kernel.VoidScopeBarCount())
	}
	line := kernel.VoidScopeBootLine()
	for _, want := range []string{"void scope:", "session-day window", "1m×2000", "parity"} {
		if !strings.Contains(line, want) {
			t.Errorf("boot line missing %q: %s", want, line)
		}
	}
	t.Logf("boot: %s", line)
}
