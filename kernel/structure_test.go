package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// hourMs is a CT wall-clock hour rendered as epoch ms.
func hourMs(h, m int) int64 {
	return time.Date(2026, 8, 21, h, m, 0, 0, CTLocation()).UnixMilli()
}

// upFixture is a clean k=2 fractal TRENDING_UP fixture: swings at
// L100(i=2) H110(i=5) L105(i=8) H116(i=11) L110(i=14); then bar 17 closes
// 118 → BOS-up; bar 18 collapses 118→102 (body 16 ≥ 1.5×ATR(10)) → MSS-down.
func upFixture() []market.KlineBar {
	t0 := hourMs(9, 0)
	step := int64(15 * 60_000)
	rows := [][4]float64{ // open, high, low, close
		{103.0, 104, 103, 103.5},
		{102.0, 103, 102, 102.5},
		{101.0, 102, 100, 100.5}, // low swing 100
		{102.0, 103, 102, 102.5},
		{107.0, 108, 107, 107.5},
		{110.0, 110, 109, 110.0}, // high swing 110
		{108.0, 108, 107, 107.5},
		{106.5, 107, 106, 106.5},
		{106.0, 106, 105, 105.5}, // low swing 105 (HL)
		{106.5, 107, 106, 106.5},
		{113.0, 114, 113, 113.5},
		{116.0, 116, 115, 116.0}, // high swing 116 (HH)
		{114.0, 114, 113, 113.5},
		{112.5, 113, 112, 112.5},
		{111.0, 112, 110, 110.5}, // low swing 110 (HL)
		{111.5, 112, 111, 111.5},
		{112.5, 113, 112, 112.5},
		{114.0, 118, 114, 118.0}, // BOS-up: close 118 > swing high 116
		{118.0, 118, 102, 102.0}, // MSS-down: body 16 ≥ 15, close < swing low 110
	}
	kb := make([]market.KlineBar, len(rows))
	for i, r := range rows {
		kb[i] = market.KlineBar{Time: t0 + int64(i)*step, Open: r[0], High: r[1], Low: r[2], Close: r[3]}
	}
	return kb
}

// mirrorDown mirrors the UP fixture vertically: highs become lows, so the same
// fractal geometry reads as TRENDING_DOWN with an MSS-up at the end.
func mirrorDown(up []market.KlineBar) []market.KlineBar {
	out := make([]market.KlineBar, len(up))
	const mirror = 240.0
	for i, b := range up {
		out[i] = market.KlineBar{Time: b.Time,
			Open: mirror - b.Open, High: mirror - b.Low, Low: mirror - b.High, Close: mirror - b.Close}
	}
	return out
}

func hasEvent(st StructureState, typ, dir string) bool {
	for _, e := range st.LastEvents {
		if e.Type == typ && e.Dir == dir {
			return true
		}
	}
	return false
}

func TestComputeStructureState_UptrendBOSAndMSS(t *testing.T) {
	bars := upFixture()
	now := bars[len(bars)-1].Time + 15*60_000
	st := ComputeStructureState(bars, 15, 10, now)
	if st.Trend != "TRENDING_UP" {
		t.Fatalf("want TRENDING_UP, got %s (swing %+v)", st.Trend, st.Swing)
	}
	if st.Swing == nil || st.Swing.Kind != "HL" {
		t.Fatalf("newest swing must be HL, got %+v", st.Swing)
	}
	if !hasEvent(st, "BOS", "up") {
		t.Fatalf("want BOS-up in events: %+v", st.LastEvents)
	}
	if !hasEvent(st, "MSS", "down") {
		t.Fatalf("want MSS-down (displaced CHoCH) in events: %+v", st.LastEvents)
	}
}

func TestComputeStructureState_DowntrendMSS(t *testing.T) {
	bars := mirrorDown(upFixture())
	now := bars[len(bars)-1].Time + 15*60_000
	st := ComputeStructureState(bars, 15, 10, now)
	if st.Trend != "TRENDING_DOWN" {
		t.Fatalf("want TRENDING_DOWN, got %s", st.Trend)
	}
	if !hasEvent(st, "MSS", "up") {
		t.Fatalf("want MSS-up (counter close through swing low): %+v", st.LastEvents)
	}
}

func TestComputeStructureState_Sweep(t *testing.T) {
	bars := upFixture()
	// Bar 17 wicks through the swing high 116 but closes back inside → SWEEP.
	bars[len(bars)-2] = market.KlineBar{Time: bars[len(bars)-2].Time, Open: 114, High: 121, Low: 113, Close: 115.5}
	// And the MSS bar becomes a quiet inside bar so no CHoCH/MSS competes.
	bars[len(bars)-1] = market.KlineBar{Time: bars[len(bars)-1].Time, Open: 115, High: 116, Low: 113, Close: 114.5}
	now := bars[len(bars)-1].Time + 15*60_000
	st := ComputeStructureState(bars, 15, 10, now)
	if !hasEvent(st, "SWEEP", "up") {
		t.Fatalf("want SWEEP of high: %+v", st.LastEvents)
	}
}

func TestComputeStructureState_RangingUnconfirmed(t *testing.T) {
	// Flat alternating bars never build four confirmed swings → RANGING.
	t0 := hourMs(9, 0)
	step := int64(15 * 60_000)
	kb := make([]market.KlineBar, 0, 14)
	for i := 0; i < 14; i++ {
		base := 100.0
		kb = append(kb, market.KlineBar{Time: t0 + int64(i)*step,
			Open: base, High: base + 3, Low: base - 3, Close: base + 0.5})
	}
	now := kb[len(kb)-1].Time + step
	st := ComputeStructureState(kb, 15, 10, now)
	if st.Trend != "RANGING" {
		t.Fatalf("want RANGING, got %s", st.Trend)
	}
}

// real15m0821 is the actual 15m table from the 2026-08-21 stored prompt
// (rec 31587, quoted in the G7 soak) — the shift-day series the detector
// replays.
func real15m0821() []market.KlineBar {
	rows := []struct {
		h, m         int
		o, hi, lo, c float64
	}{
		{6, 0, 29479.75, 29539.75, 29452.75, 29499.25},
		{6, 15, 29499.50, 29533.75, 29472.00, 29511.50},
		{6, 30, 29511.75, 29516.25, 29257.25, 29303.75},
		{6, 45, 29303.50, 29335.75, 29220.25, 29331.75},
		{7, 0, 29332.00, 29488.50, 29326.25, 29443.75},
		{7, 15, 29443.25, 29454.50, 29380.50, 29412.25},
		{7, 30, 29412.00, 29423.75, 29375.00, 29383.50},
		{7, 45, 29383.25, 29405.50, 29353.75, 29400.25},
		{8, 0, 29399.75, 29433.25, 29380.00, 29411.50},
		{9, 0, 29303.50, 29335.75, 29220.25, 29331.75},
		{10, 0, 29332.00, 29368.75, 29326.25, 29352.25},
		{10, 15, 29353.00, 29371.25, 29330.75, 29368.50},
		{10, 30, 29368.00, 29471.75, 29364.25, 29451.75},
		{10, 45, 29451.75, 29488.50, 29425.50, 29443.75},
		{11, 0, 29443.25, 29447.75, 29389.25, 29394.25},
		{11, 15, 29393.75, 29454.50, 29386.00, 29410.00},
		{11, 30, 29409.75, 29425.75, 29393.75, 29417.50},
		{11, 45, 29417.50, 29417.50, 29380.50, 29412.25},
		{12, 0, 29412.00, 29423.75, 29398.25, 29408.50},
		{12, 15, 29408.00, 29410.50, 29375.00, 29395.25},
		{12, 30, 29396.00, 29404.75, 29382.75, 29403.75},
		{12, 45, 29404.50, 29410.75, 29381.25, 29383.50},
		{13, 0, 29383.25, 29405.50, 29371.25, 29374.25},
		{13, 15, 29374.50, 29392.00, 29362.00, 29363.00},
		{13, 30, 29363.25, 29372.25, 29353.75, 29366.50},
		{13, 45, 29366.50, 29403.50, 29363.50, 29400.25},
		{14, 0, 29399.75, 29414.75, 29384.25, 29388.50},
		{14, 15, 29388.50, 29405.00, 29380.00, 29393.25},
		{14, 30, 29393.50, 29412.00, 29393.50, 29410.75},
		{14, 45, 29411.00, 29433.25, 29402.00, 29411.50},
	}
	kb := make([]market.KlineBar, len(rows))
	for i, r := range rows {
		kb[i] = market.KlineBar{Time: hourMs(r.h, r.m), Open: r.o, High: r.hi, Low: r.lo, Close: r.c}
	}
	return kb
}

// TestG2Replay_ShiftDay15m replays the shift-day through the detector and pins
// the HONEST machine truth:
//   - eval 11:30 CT (bars closed through 11:00, so the 10:45 fractal window is
//     complete): the 3-swing standard REFUSES TRENDING_DOWN — the 09:00 low
//     equals the 06:45 low (29220.25), so the low-pair is unconfirmed →
//     RANGING (fail-open). The 10:45 flush high 29488.50 IS the newest swing
//     (HH) with its timestamp — the detector SEES the shift the plan missed,
//     even while it refuses to grade the trend on an equal low.
//   - eval 15:00 CT (full day): RANGING (the day was two-sided: down, then the
//     flush, then drift) and NO false CHoCH/MSS — a close-through never
//     occurred on this day, so the engine must not invent one.
func TestG2Replay_ShiftDay15m(t *testing.T) {
	bars := real15m0821()
	st := ComputeStructureState(bars, 15, 0, hourMs(11, 30))
	if st.Trend != "RANGING" {
		t.Fatalf("11:30 eval: equal-low pair must refuse trend confirmation, got %s", st.Trend)
	}
	if st.Swing == nil || st.Swing.Kind != "HH" || st.Swing.Price != 29488.50 || st.Swing.TimeMs != hourMs(11, 0) {
		t.Fatalf("11:30 eval: newest swing must be the 10:45 flush HH 29488.50 (confirmed at its 11:00 close), got %+v", st.Swing)
	}
	full := ComputeStructureState(bars, 15, 0, hourMs(15, 0))
	if full.Trend != "RANGING" {
		t.Fatalf("15:00 eval: two-sided day must read RANGING, got %s", full.Trend)
	}
	for _, e := range full.LastEvents {
		if e.Type == "CHoCH" || e.Type == "MSS" || e.Type == "BOS" {
			t.Fatalf("15:00 eval: no close-through ever happened on 08-21 — engine must not invent %s: %+v", e.Type, full.LastEvents)
		}
	}
}

func TestStructureATRMatchesMarketWilder(t *testing.T) {
	// C-ATR1 conformance pin: the structure engine's ATR must equal nofx/market's
	// Wilder-smoothed calculateATR on the same series (research: nautilus ATR).
	bars := upFixture()
	klines := make([]market.Kline, len(bars))
	highs := make([]float64, len(bars))
	lows := make([]float64, len(bars))
	closes := make([]float64, len(bars))
	for i, b := range bars {
		klines[i] = market.Kline{OpenTime: b.Time, Open: b.Open, High: b.High, Low: b.Low, Close: b.Close}
		highs[i], lows[i], closes[i] = b.High, b.Low, b.Close
	}
	want := market.ExportCalculateATR(klines, 14)
	got := simpleATR14(highs, lows, closes)
	if got <= 0 || want <= 0 {
		t.Fatalf("ATR must be positive: got %v want %v", got, want)
	}
	if diff := got - want; diff < -0.0001 || diff > 0.0001 {
		t.Fatalf("structure ATR %.6f != market Wilder ATR %.6f", got, want)
	}
}

func TestStructurePromptLine(t *testing.T) {
	snap := map[string]StructureState{
		"15m": {Trend: "TRENDING_DOWN", Swing: &SwingRef{Kind: "LL", Price: 29346, TimeMs: hourMs(10, 15)},
			LastEvents: []StructureEvent{{Type: "CHoCH", Dir: "up", TimeMs: hourMs(10, 45), Price: 29470.25}}},
		"1h": {Trend: "RANGING"},
	}
	line := StructurePromptLine(snap)
	if !strings.HasPrefix(line, "STRUCTURE ") || !strings.Contains(line, "TRENDING_DOWN") || !strings.Contains(line, "LL 29346.00") {
		t.Fatalf("bad structure line: %q", line)
	}
	if !strings.Contains(line, "last event: CHoCH-up 15m") {
		t.Fatalf("missing last event in line: %q", line)
	}
}
