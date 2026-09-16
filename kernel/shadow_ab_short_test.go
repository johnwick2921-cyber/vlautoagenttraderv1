package kernel

import (
	"math"
	"testing"

	"nofx/market"
)

// E8 ONE PRICE SPACE (data-integrity wave) — E1, THE PIN.
//
// shadow_ab.go mirrors stop/target/ref into a negative price space for shorts
// ("mirror signs so MFE is always favorable-positive"), but the close-rule fill
// returns the REAL close and row.StopPx/TargetPx are stored REAL. Everything
// downstream then mixes two spaces:
//
//	risk := row.FillPx - stop      →  29204.50 − (−29226.00)  =  58 430.50
//	row.RR = (target - FillPx)/risk → (−29132.50 − 29204.50)/58 430.50 = −0.9984
//
// Measured on the live store (188 rows, direction taken from the PLAN, never
// inferred): 121 short rows, 109 with RR < −0.9, 46 with |MAE| > 1000, and all
// 67 long rows clean. Row 166 is the type specimen:
//
//	id 166 · S1 · 1x5m_close · fill 29204.50 · stop 29226.00 · target 29132.50
//	stored: rr −0.998399808319285 · mae 58409.25 · net_pnl −1.0
//
// Correct short arithmetic on the same inputs:
//
//	risk   = stop − fill   = 29226.00 − 29204.50 = 21.50
//	reward = fill − target = 29204.50 − 29132.50 = 72.00
//	RR     = 72.00 / 21.50 = 3.348837…
const (
	row166Fill   = 29204.50
	row166Stop   = 29226.00
	row166Target = 29132.50
	row166Risk   = row166Stop - row166Fill   // 21.50
	row166Reward = row166Fill - row166Target // 72.00
)

// shortScenario builds the row-166 scenario: a short armed at 29204.50.
func shortScenario(rule string) PlanScenario {
	return PlanScenario{
		ID: "S1", Condition: "reject", Direction: "short",
		Confirm: &PlanConfirm{Rule: rule, RefPrice: row166Fill, Side: "below"},
		Arm: &PlanArmSpec{
			Enabled: true, Entry: row166Fill, Stop: row166Stop, Target: row166Target,
		},
	}
}

// shortBars: five 5m buckets of 1m bars. Price closes below the ref (the short
// confirms), drifts adversely to 29215 (never reaching the 29226 stop), then
// runs to the 29132.50 target.
func shortBars(startMs int64) []market.Kline {
	mk := func(i int, o, h, l, c float64) market.Kline {
		open := startMs + int64(i)*60_000
		return market.Kline{OpenTime: open, CloseTime: open + 59_999, Open: o, High: h, Low: l, Close: c}
	}
	var bars []market.Kline
	// 5 bars closing below the ref → 1x and 2x 5m_close both confirm
	for i := 0; i < 10; i++ {
		bars = append(bars, mk(i, 29204, 29206, 29200, 29203))
	}
	// adverse excursion to 29215 — inside the 29226 stop
	for i := 10; i < 15; i++ {
		bars = append(bars, mk(i, 29205, 29215, 29203, 29210))
	}
	// run to target
	for i := 15; i < 25; i++ {
		bars = append(bars, mk(i, 29200, 29202, 29130, 29131))
	}
	return bars
}

func TestShadowABShortUsesOnePriceSpace(t *testing.T) {
	start := int64(1_700_000_000_000)
	start -= start % 300_000
	bars := shortBars(start)
	nowMs := bars[len(bars)-1].CloseTime + 1

	rows := ShadowABForScenario(shortScenario("1x5m_close"), bars, "MNQ", start, nowMs)
	if len(rows) == 0 {
		t.Fatal("no shadow row produced for the short scenario")
	}
	r := rows[0]

	wantRR := row166Reward / row166Risk // 3.348837…
	if math.Abs(r.RR-wantRR) > 1e-6 {
		t.Errorf("RR = %.9f, want %.9f (reward %.2f / risk %.2f) — the stored row 166 reads −0.998399808319285, which is a real fill divided by a mirrored stop",
			r.RR, wantRR, row166Reward, row166Risk)
	}
	// MAE is an ADVERSE excursion in points and must sit inside the stop.
	if r.MAE < 0 || r.MAE > row166Risk {
		t.Errorf("MAE = %.4f, want within [0, %.2f] — the stored row reads 58409.25, which is two price spaces subtracted",
			r.MAE, row166Risk)
	}
	// A short that reaches its target MAKES money.
	if r.Outcome == "target" && r.NetPnL <= 0 {
		t.Errorf("a short that hit its target has net_pnl %.2f — a target hit is a profit", r.NetPnL)
	}
	if r.MFE < 0 {
		t.Errorf("MFE = %.4f, want >= 0", r.MFE)
	}
}

// The LONG side must not move: all 67 stored long rows are clean, so this
// wave's diff on them has to be empty.
func TestShadowABLongSideUnchanged(t *testing.T) {
	start := int64(1_700_000_000_000)
	start -= start % 300_000
	mk := func(i int, o, h, l, c float64) market.Kline {
		open := start + int64(i)*60_000
		return market.Kline{OpenTime: open, CloseTime: open + 59_999, Open: o, High: h, Low: l, Close: c}
	}
	var bars []market.Kline
	for i := 0; i < 10; i++ {
		bars = append(bars, mk(i, 29200, 29206, 29198, 29205)) // closes above ref
	}
	for i := 10; i < 25; i++ {
		bars = append(bars, mk(i, 29210, 29290, 29208, 29285)) // runs to target
	}
	nowMs := bars[len(bars)-1].CloseTime + 1

	long := PlanScenario{
		ID: "S1", Condition: "reclaim", Direction: "long",
		Confirm: &PlanConfirm{Rule: "1x5m_close", RefPrice: 29204, Side: "above"},
		Arm:     &PlanArmSpec{Enabled: true, Entry: 29204, Stop: 29180, Target: 29280},
	}
	rows := ShadowABForScenario(long, bars, "MNQ", start, nowMs)
	if len(rows) == 0 {
		t.Fatal("no shadow row for the long scenario")
	}
	r := rows[0]
	wantRR := (29280.0 - r.FillPx) / (r.FillPx - 29180.0)
	if math.Abs(r.RR-wantRR) > 1e-6 {
		t.Errorf("long RR = %.9f, want %.9f — long-side math must be untouched", r.RR, wantRR)
	}
	if r.MAE < 0 || r.MFE < 0 {
		t.Errorf("long MAE/MFE must stay non-negative: %.4f / %.4f", r.MAE, r.MFE)
	}
}
