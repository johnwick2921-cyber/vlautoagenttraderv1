package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// mkBar builds a 1m Kline.
// tapeScope wraps a fixture tape in the resolved void window. For an
// intra-session tape the session-day start precedes bar 0, so the clamp makes
// the whole tape in-window — exactly what these fixtures assumed before the
// scope existed.
func tapeScope(bars []market.Kline) VoidScope {
	if len(bars) == 0 {
		return VoidScopeOf(bars, time.Now())
	}
	return VoidScopeOf(bars, time.UnixMilli(bars[len(bars)-1].CloseTime))
}

func mkTapeBar(t time.Time, o, h, l, c float64) market.Kline {
	return market.Kline{OpenTime: t.UnixMilli(), Open: o, High: h, Low: l, Close: c,
		CloseTime: t.UnixMilli() + 60_000}
}

// waterfallTape reproduces the 2026-08-28 10:25→11:24 CT -347pt crash shape
// around the broken VWAP level 29657.39 (the missed-200pt tape): a high base,
// a displacement leg through the level, and a no-reclaim waterfall.
func waterfallTape(level float64) ([]market.Kline, time.Time) {
	start := time.Date(2026, 8, 28, 10, 25, 0, 0, time.Local)
	seq := [][4]float64{ // open, high, low, close
		{29784.00, 29791.75, 29776.00, 29780.25}, // 10:25 base
		{29779.75, 29782.25, 29771.75, 29777.50},
		{29778.00, 29784.50, 29767.75, 29783.00},
		{29783.00, 29783.25, 29771.25, 29783.00},
		{29783.25, 29785.00, 29774.50, 29779.50},
		{29779.25, 29782.00, 29770.50, 29772.00}, // 10:30
		{29772.00, 29781.75, 29770.50, 29778.75},
		{29778.75, 29785.75, 29773.25, 29775.00},
		{29774.75, 29777.50, 29762.50, 29763.50},
		{29763.00, 29769.00, 29759.00, 29762.50},
		{29762.00, 29769.25, 29758.25, 29759.75}, // 10:35
		{29759.50, 29761.00, 29748.00, 29748.75},
		{29748.75, 29752.50, 29731.25, 29733.00},
		{29733.00, 29734.50, 29716.50, 29721.25},
		{29721.75, 29722.50, 29674.00, 29677.75},
		{29678.25, 29699.50, 29662.25, 29697.75}, // 10:40 — the write cut
		{29698.25, 29699.50, 29680.50, 29683.00},
		{29682.75, 29697.75, 29676.00, 29689.50},
		{29689.50, 29690.75, 29665.00, 29671.00},
		{29671.00, 29682.75, 29662.50, 29675.75},
		{29675.75, 29678.25, 29660.75, 29662.75}, // 10:45
		{29662.75, 29671.50, 29650.25, 29652.00},
		{29652.00, 29654.25, 29629.00, 29646.00},
		{29645.75, 29646.50, 29627.50, 29634.00},
		{29633.50, 29647.00, 29619.75, 29623.50},
		{29623.25, 29631.25, 29600.00, 29600.00}, // 10:50
		{29600.00, 29602.75, 29585.25, 29591.00},
		{29590.50, 29616.00, 29587.25, 29594.50},
		{29594.50, 29619.50, 29585.75, 29618.50},
		{29618.50, 29621.00, 29584.00, 29584.25},
		{29583.75, 29588.75, 29556.25, 29562.50}, // 10:55
		{29562.50, 29566.00, 29547.00, 29557.50},
		{29557.50, 29571.75, 29550.50, 29565.50},
		{29564.75, 29567.25, 29544.75, 29552.50},
		{29552.75, 29566.50, 29544.75, 29545.50}, // 11:00
		{29545.25, 29546.00, 29509.50, 29528.25},
		{29528.50, 29543.00, 29520.00, 29542.50},
		{29542.00, 29555.75, 29537.75, 29549.50},
		{29550.50, 29559.75, 29525.00, 29531.50},
		{29531.50, 29552.75, 29528.75, 29551.75},
		{29551.75, 29562.25, 29533.50, 29556.25}, // 11:05
		{29556.25, 29575.75, 29556.00, 29574.50},
		{29574.50, 29580.00, 29561.00, 29578.75},
		{29578.50, 29592.25, 29577.00, 29585.75},
		{29585.50, 29591.25, 29572.00, 29590.50},
		{29590.25, 29591.00, 29555.50, 29558.00}, // 11:10
		{29558.25, 29574.00, 29554.25, 29568.75},
		{29568.75, 29571.25, 29554.50, 29562.75},
		{29562.50, 29571.75, 29555.50, 29558.25},
		{29558.75, 29570.50, 29533.00, 29534.50},
		{29533.75, 29538.75, 29523.75, 29533.75}, // 11:15
		{29534.00, 29537.25, 29519.25, 29524.50},
		{29524.50, 29529.00, 29492.75, 29494.25},
		{29494.50, 29519.25, 29492.25, 29511.00},
		{29511.00, 29523.75, 29492.00, 29497.00},
		{29497.00, 29506.50, 29477.00, 29504.50}, // 11:20
		{29504.75, 29505.75, 29487.00, 29494.50},
		{29494.50, 29498.00, 29482.25, 29490.50},
		{29490.00, 29494.75, 29464.50, 29485.50},
		{29485.75, 29496.00, 29476.25, 29494.00}, // 11:24
	}
	bars := make([]market.Kline, 0, len(seq))
	for i, b := range seq {
		t := start.Add(time.Duration(i) * time.Minute)
		bars = append(bars, mkTapeBar(t, b[0], b[1], b[2], b[3]))
	}
	return bars, start
}

func waterfallPlan() PlanDoc {
	return PlanDoc{
		Bias: PlanBias{Direction: "short", Conviction: "medium"},
		Scenarios: []PlanScenario{{
			ID: "S1", Trigger: "waterfall through VWAP 29657.39 continues",
			Condition: "breakdown_continue", Direction: "short",
			TargetChain: []float64{29599.75, 29576.50},
			Quality:     "A",
			Confirm:     &PlanConfirm{Rule: "2x5m_close", RefPrice: 29657.39, Side: "below"},
			Confirm2:    &PlanConfirm{Rule: "1x5m_close", RefPrice: 29657.39, Side: "below"},
			Breakdown:   &PlanBreakdownContinue{Level: 29657.39, LevelLabel: "VWAP 29657.39", EntryMode: "pullback"},
			Arm: &PlanArmSpec{Enabled: true, Entry: 29657.64, Stop: 29677.64, Target: 29614.00,
				WaitConfirm: true},
		}},
	}
}

// TestBreakdownContinueValidatorRealTape — the missed-200pt fixture: today's
// 10:25-11:24 tape must produce a valid authored-and-triggerable instance.
func TestBreakdownContinueValidatorRealTape(t *testing.T) {
	bars, start := waterfallTape(29657.39)
	plan := waterfallPlan()
	plan.Scenarios[0].Arm.Stop = plan.Scenarios[0].Arm.Entry + 20.0 // ≥1×ATR15
	writeTime := start.Add(26 * time.Minute)                        // 10:51 cut — the real v4 birth
	price := 29600.0
	// Write-time validation: displacement ≥ BD_MIN_DISP_ATR×ATR, no reclaim.
	if err := ValidateBreakdownContinueScenarios(&plan, tapeScope(bars), 15.0, price, writeTime.UnixMilli()); err != nil {
		t.Fatalf("real-tape plan rejected at write: %v", err)
	}
	// Triggerable: at birth, leg 1 MET (breakdown), leg 2 pending (no retest).
	st := BreakdownContinueState(plan.Scenarios[0], bars, writeTime.UnixMilli(), bars[len(bars)-1].CloseTime)
	if !st.Leg1Met || st.Leg2Met || st.Reclaimed {
		t.Fatalf("want leg1 MET + leg2 pending + not reclaimed; got %+v", st)
	}
	if st.BreakLegPts < 1.0*15.0 {
		t.Fatalf("want displacement ≥ 1×ATR5m, got %.2f", st.BreakLegPts)
	}
	// Append a synthetic COMPLETE 5m retest after the unchanged real tape.
	extended := append([]market.Kline{}, bars...)
	retestStart := time.UnixMilli(bars[len(bars)-1].OpenTime).Add(time.Minute)
	for i := 0; i < 5; i++ {
		extended = append(extended, mkTapeBar(retestStart.Add(time.Duration(i)*time.Minute), 29640, 29658, 29639, 29650))
	}
	st2 := BreakdownContinueState(plan.Scenarios[0], extended, writeTime.UnixMilli(), retestStart.Add(5*time.Minute).UnixMilli())
	if !st2.Leg2Met {
		t.Fatalf("retest-that-fails must fire leg 2; got %+v", st2)
	}
}

// TestBreakdownContinueValidatorRejectsWeakDisplacement — a 0.5×ATR leg is not a
// displacement move.
func TestBreakdownContinueValidatorRejectsWeakDisplacement(t *testing.T) {
	start := time.Date(2026, 8, 28, 10, 0, 0, 0, time.Local)
	bars := []market.Kline{
		mkTapeBar(start, 29670, 29675, 29660, 29662),
		mkTapeBar(start.Add(time.Minute), 29662, 29666, 29650, 29654),
		mkTapeBar(start.Add(2*time.Minute), 29654, 29658, 29648, 29652),
	}
	for i := 3; i < 5; i++ {
		bars = append(bars, mkTapeBar(start.Add(time.Duration(i)*time.Minute), 29652, 29658, 29648, 29652))
	}
	plan := PlanDoc{Scenarios: []PlanScenario{{
		ID: "S1", Condition: "breakdown_continue", Direction: "short",
		Breakdown: &PlanBreakdownContinue{Level: 29655, EntryMode: "pullback"},
	}}}
	err := ValidateBreakdownContinueScenarios(&plan, tapeScope(bars), 15.0, 29652, bars[len(bars)-1].CloseTime)
	if err == nil || !strings.Contains(err.Error(), "measured displacement") {
		t.Fatalf("want displacement rejection, got %v", err)
	}
}

// TestBreakdownContinueValidatorRejectsReclaimed — a close back across the level
// voids the play.
func TestBreakdownContinueValidatorRejectsReclaimed(t *testing.T) {
	start := time.Date(2026, 8, 28, 10, 0, 0, 0, time.Local)
	bars := []market.Kline{
		mkTapeBar(start, 29670, 29675, 29640, 29645),
		mkTapeBar(start.Add(time.Minute), 29645, 29650, 29620, 29630),
		mkTapeBar(start.Add(2*time.Minute), 29630, 29662, 29625, 29658), // reclaims
		mkTapeBar(start.Add(3*time.Minute), 29658, 29660, 29630, 29640),
	}
	bars = completeFiveMinuteFixture(bars)
	plan := PlanDoc{Scenarios: []PlanScenario{{
		ID: "S1", Condition: "breakdown_continue", Direction: "short",
		Breakdown: &PlanBreakdownContinue{Level: 29655, EntryMode: "pullback"},
	}}}
	err := ValidateBreakdownContinueScenarios(&plan, tapeScope(bars), 15.0, 29640, bars[len(bars)-1].CloseTime)
	if err == nil || !strings.Contains(err.Error(), "breakdown is void") {
		t.Fatalf("want reclaim rejection, got nil")
	}
}

// TestBreakupContinueMirror — the long twin validates and fires.
// E3: under BD_MIN_CLOSES=1 the first beyond close completes leg 1, so the
// touch-back in bar 1 IS the retest-that-fails → both legs MET.
func TestBreakupContinueMirror(t *testing.T) {
	start := time.Date(2026, 8, 28, 10, 0, 0, 0, time.Local)
	lvl := 29464.50
	bars := []market.Kline{
		mkTapeBar(start, 29460, 29470, 29455, 29466),
		mkTapeBar(start.Add(time.Minute), 29466, 29490, 29462, 29484),
		mkTapeBar(start.Add(2*time.Minute), 29484, 29520, 29480, 29514),
		mkTapeBar(start.Add(3*time.Minute), 29514, 29550, 29510, 29542),
	}
	bars = completeFiveMinuteFixture(bars)
	plan := PlanDoc{Scenarios: []PlanScenario{{
		ID: "S1", Condition: "breakup_continue", Direction: "long",
		Breakdown: &PlanBreakdownContinue{Level: lvl, EntryMode: "pullback"},
	}}}
	if err := ValidateBreakdownContinueScenarios(&plan, tapeScope(bars), 15.0, 29484, bars[9].CloseTime); err != nil {
		t.Fatalf("mirror rejected: %v", err)
	}
	st := BreakdownContinueState(plan.Scenarios[0], bars, 0, bars[len(bars)-1].CloseTime)
	if !st.Leg1Met || !st.Leg2Met {
		t.Fatalf("mirror leg state wrong (E3: single confirming close completes leg 1): %+v", st)
	}
}

// TestBreakdownArmRules — arm requires pullback + wait_confirm + confirm.
func TestBreakdownArmRules(t *testing.T) {
	sc := PlanScenario{ID: "S1", Condition: "breakdown_continue", Direction: "short",
		Breakdown: &PlanBreakdownContinue{Level: 29657.39, EntryMode: "immediate"},
		Arm:       &PlanArmSpec{Enabled: true, Entry: 29657.64, Stop: 29677.64, Target: 29614.00, WaitConfirm: true},
		Confirm:   &PlanConfirm{Rule: "2x5m_close", RefPrice: 29657.39, Side: "below"}}
	if err := ArmSpecValid(sc); err == nil || !strings.Contains(err.Error(), "pullback") {
		t.Fatalf("immediate-mode arm must be rejected, got %v", err)
	}
	sc.Breakdown.EntryMode = "pullback"
	sc.Arm.WaitConfirm = false
	if err := ArmSpecValid(sc); err == nil || !strings.Contains(err.Error(), "wait_confirm") {
		t.Fatalf("no-chain arm must be rejected, got %v", err)
	}
	sc.Arm.WaitConfirm = true
	if err := ArmSpecValid(sc); err != nil {
		t.Fatalf("valid waterfall arm rejected: %v", err)
	}
	if px := ArmedEntryPx(sc, 0, 0.25); px != 29657.64 {
		t.Fatalf("short retest entry want 29657.64, got %.2f", px)
	}
	long := sc
	long.Condition = "breakup_continue"
	long.Direction = "long"
	if px := ArmedEntryPx(long, 0, 0.25); px != 29657.14 {
		t.Fatalf("long retest entry want 29657.14, got %.2f", px)
	}
}

// TestBreakdownImmediateAuthorableBeforeSecondClose — E3 (entry-mechanics
// 2026-08-30): BD_MIN_CLOSES=1 relaxes the breakdown floor. Immediate-mode
// authoring is legal as soon as the DISPLACEMENT exists; PULLBACK now needs
// only the SINGLE confirming close (the displacement + reclaim-check carry the
// quality gate — the double close is gone).
func TestBreakdownImmediateAuthorableBeforeSecondClose(t *testing.T) {
	start := time.Date(2026, 8, 28, 10, 46, 0, 0, time.Local)
	lvl := 29657.39
	bars := []market.Kline{
		mkTapeBar(start, 29670, 29675, 29660, 29666),
		mkTapeBar(start.Add(time.Minute), 29666, 29670, 29658, 29662),
		// close ABOVE the level — still pre-breakdown.
		mkTapeBar(start.Add(2*time.Minute), 29662, 29668, 29656, 29660),
		// ONE beyond close so far, with real displacement: low = lvl − 1.3×ATR.
		mkTapeBar(start.Add(3*time.Minute), 29655, 29658, lvl-19.5, lvl-10),
	}
	imm := PlanDoc{Scenarios: []PlanScenario{{
		ID: "S1", Condition: "breakdown_continue", Direction: "short",
		Breakdown: &PlanBreakdownContinue{Level: lvl, EntryMode: "immediate"},
	}}}
	if err := ValidateBreakdownContinueScenarios(&imm, tapeScope(bars), 15.0, lvl-10, bars[len(bars)-1].CloseTime); err != nil {
		t.Fatalf("immediate authoring before the confirming close must pass once displacement exists: %v", err)
	}
	pb := PlanDoc{Scenarios: []PlanScenario{{
		ID: "S1", Condition: "breakdown_continue", Direction: "short",
		Breakdown: &PlanBreakdownContinue{Level: lvl, EntryMode: "pullback"},
	}}}
	// E3 fixture #1 — 1 close + displacement PASSES for pullback now.
	if err := ValidateBreakdownContinueScenarios(&pb, tapeScope(bars), 15.0, lvl-10, bars[len(bars)-1].CloseTime); err != nil {
		t.Fatalf("E3: pullback with ONE confirming close + displacement must now PASS: %v", err)
	}
	// No displacement at all → immediate is still rejected.
	flat := PlanDoc{Scenarios: []PlanScenario{{
		ID: "S1", Condition: "breakdown_continue", Direction: "short",
		Breakdown: &PlanBreakdownContinue{Level: lvl, EntryMode: "immediate"},
	}}}
	if err := ValidateBreakdownContinueScenarios(&flat, tapeScope(bars[:3]), 15.0, 29655, bars[2].CloseTime); err == nil || !strings.Contains(err.Error(), "displacement") {
		t.Fatalf("immediate with zero displacement must be rejected, got %v", err)
	}
}

// TestBreakdownImmediateFixturePassesGateChain — the dispatch fixture: the
// 2026-08-28 10:48 leg of the −347pt crash. An immediate-mode plan-legal entry
// (2nd confirming close) passes the FULL market-entry gate chain in replay:
// min-SL ≥ 1.0×ATR5m, R:R ≥ 3.0 (min_risk_reward_ratio), confidence ≥ 60, and
// the target fills on the real tape. (HTF veto: the 1h was already
// TRENDING_DOWN that morning — a continuation short aligns, so cross-veto
// passes; not unit-asserted here [B].)
func TestBreakdownImmediateFixturePassesGateChain(t *testing.T) {
	bars, start := waterfallTape(29657.39)
	lvl, atr := 29657.39, 15.0
	sc := PlanScenario{
		ID: "S1", Trigger: "waterfall through VWAP 29657.39 continues",
		Condition: "breakdown_continue", Direction: "short",
		TargetChain: []float64{29524.50},
		Quality:     "A",
		Confirm:     &PlanConfirm{Rule: "2x5m_close", RefPrice: lvl, Side: "below"},
		Breakdown:   &PlanBreakdownContinue{Level: lvl, LevelLabel: "VWAP 29657.39", EntryMode: "immediate"},
	}
	plan := PlanDoc{Bias: PlanBias{Direction: "short", Conviction: "medium"}, Scenarios: []PlanScenario{sc}}
	writeTime := start.Add(26 * time.Minute) // 10:51 — the real v4 birth cut
	if err := ValidateBreakdownContinueScenarios(&plan, tapeScope(bars), atr, 29600.0, writeTime.UnixMilli()); err != nil {
		t.Fatalf("immediate plan rejected at the write cut: %v", err)
	}
	// Entry = the 2nd confirming close beyond the level (bar 22, 10:47).
	entry := 29646.00
	if bars[22].Close != entry || bars[22].Close >= lvl || bars[21].Close >= lvl {
		t.Fatalf("fixture assumption broken: bars[21]=%.2f bars[22]=%.2f vs level %.2f", bars[21].Close, bars[22].Close, lvl)
	}
	// Pullback extreme = the high of the beyond-run through entry.
	extreme := bars[21].High // 29671.50 — the run high before the 2nd close
	for _, b := range bars[22:26] {
		if b.High > extreme {
			extreme = b.High
		}
	}
	sl := extreme + 1.0*atr // beyond the pullback extreme by 1×ATR5m
	tp := entry - 3.0*(sl-entry)
	if sl-entry < 1.0*atr {
		t.Fatalf("min-SL gate: risk %.2f < 1.0×ATR5m", sl-entry)
	}
	if rr := (entry - tp) / (sl - entry); rr < 3.0 {
		t.Fatalf("R:R gate: %.2f < 3.0", rr)
	}
	if conf := 65; conf < 60 {
		t.Fatalf("min-conf gate: %d < 60", conf)
	}
	// The real tape fills the target and never touches the stop.
	hitTP, hitSL := false, false
	for _, b := range bars[23:] {
		if b.Low <= tp {
			hitTP = true
			break
		}
		if b.High >= sl {
			hitSL = true
			break
		}
	}
	if hitSL || !hitTP {
		t.Fatalf("replay: hitTP=%v hitSL=%v (SL %.2f TP %.2f) — the 10:48 leg must fill the target", hitTP, hitSL, sl, tp)
	}
}

// completeFiveMinuteFixture gives each synthetic OHLC stage a complete 5m
// bucket. It is never applied to the measured waterfallTape data.
func completeFiveMinuteFixture(stages []market.Kline) []market.Kline {
	var out []market.Kline
	base := stages[0].OpenTime
	for i, b := range stages {
		for j := 0; j < 5; j++ {
			b.OpenTime = base + int64(i*5+j)*60000
			b.CloseTime = b.OpenTime + 60000
			out = append(out, b)
		}
	}
	return out
}
