package kernel

import (
	"math"
	"testing"
	"time"

	"nofx/market"
)

// TestSVPValueArea_MandatoryVector is the LOCKED spec vector (Part B2). The
// classic two-row method MUST enclose rows -2..+2 (volume 61). The greedy
// one-row method would stop at -2..+1 (57) — which is WRONG. This test is the
// contract; do not "fix" the algorithm to satisfy a different result.
func TestSVPValueArea_MandatoryVector(t *testing.T) {
	hist := map[int]float64{
		4: 3, 3: 14, 2: 4, 1: 5, 0: 30, -1: 6, -2: 16, -3: 2,
	}
	poc, vaLow, vaHigh, vaVol, total := svpValueArea(hist, 0.70)

	if total != 80 {
		t.Fatalf("total = %v, want 80", total)
	}
	if poc != 0 {
		t.Errorf("POC = %d, want 0", poc)
	}
	if vaLow != -2 || vaHigh != 2 {
		t.Errorf("VA span = [%d,%d], want [-2,+2] (classic). Greedy's wrong answer is [-2,+1].", vaLow, vaHigh)
	}
	if vaVol != 61 {
		t.Errorf("VA volume = %v, want 61 (greedy would give 57)", vaVol)
	}
}

// TestSVPValueArea_PriceLevels checks the row-index → price conversions off the
// mandatory vector: POC = row midpoint, VAH = top edge of top row, VAL = bottom
// edge of bottom row (rowHeight = 1.25).
func TestSVPValueArea_PriceLevels(t *testing.T) {
	hist := map[int]float64{4: 3, 3: 14, 2: 4, 1: 5, 0: 30, -1: 6, -2: 16, -3: 2}
	poc, vaLow, vaHigh, _, _ := svpValueArea(hist, 0.70)
	gotPOC := (float64(poc) + 0.5) * SVPRowHeight
	gotVAH := float64(vaHigh+1) * SVPRowHeight
	gotVAL := float64(vaLow) * SVPRowHeight
	if gotPOC != 0.625 { // (0+0.5)*1.25
		t.Errorf("POC price = %v, want 0.625", gotPOC)
	}
	if gotVAH != 3.75 { // (2+1)*1.25
		t.Errorf("VAH = %v, want 3.75", gotVAH)
	}
	if gotVAL != -2.5 { // (-2)*1.25
		t.Errorf("VAL = %v, want -2.5", gotVAL)
	}
}

// TestSVPHistogram_Distribution covers uniform bar-range distribution, a doji,
// and a boundary-aligned high (no double count).
func TestSVPHistogram_Distribution(t *testing.T) {
	// Uniform split across two rows: [100, 102.5] over rowHeight 1.25 → bins 80,81
	// each get half.
	h := map[int]float64{}
	svpAddBar(h, 100.0, 102.5, 100)
	if math.Abs(h[80]-50) > 1e-9 || math.Abs(h[81]-50) > 1e-9 {
		t.Errorf("uniform split: bin80=%v bin81=%v, want 50/50", h[80], h[81])
	}
	if len(h) != 2 {
		t.Errorf("uniform split touched %d bins, want 2 (no boundary leak): %v", len(h), h)
	}

	// Doji: low==high drops everything in one row.
	d := map[int]float64{}
	svpAddBar(d, 100.4, 100.4, 30)
	if d[80] != 30 || len(d) != 1 {
		t.Errorf("doji: %v, want {80:30}", d)
	}

	// Conservation: total distributed volume == bar volume.
	c := map[int]float64{}
	svpAddBar(c, 99.1, 103.7, 250)
	var sum float64
	for _, v := range c {
		sum += v
	}
	if math.Abs(sum-250) > 1e-9 {
		t.Errorf("volume not conserved: sum=%v, want 250", sum)
	}
}

// TestSVPPocTieBreak: equal max-volume rows → the row nearest the middle of the
// occupied range wins; a true tie on distance goes to the LOWER row.
func TestSVPPocTieBreak(t *testing.T) {
	// Rows 0 and 2 both max (10); middle of [0,4] is 2 → row 2 is nearer.
	h1 := map[int]float64{0: 10, 1: 5, 2: 10, 3: 5, 4: 5}
	if poc, _, _, _, _ := svpValueArea(h1, 0.70); poc != 2 {
		t.Errorf("nearest-middle POC = %d, want 2", poc)
	}
	// Rows 0 and 4 both max (10), equidistant from middle 2 → LOWER wins (0).
	h2 := map[int]float64{0: 10, 1: 1, 2: 1, 3: 1, 4: 10}
	if poc, _, _, _, _ := svpValueArea(h2, 0.70); poc != 0 {
		t.Errorf("equidistant tie POC = %d, want 0 (lower)", poc)
	}
}

// mk1mBar builds a closed 1-minute UP bar (Close==high>=Open==low) at the given
// Chicago wall-clock minute.
func mk1mBar(loc *time.Location, y int, mo time.Month, d, h, mi int, low, high, vol float64) market.Kline {
	return mk1mBarDir(loc, y, mo, d, h, mi, low, high, vol, true)
}

// mk1mBarDir builds a closed 1-minute bar with an explicit direction: up==true
// → close>=open (up candle), up==false → close<open (down candle). Range stays
// [low,high] either way so the histogram distribution is identical; only the
// up/down split changes.
func mk1mBarDir(loc *time.Location, y int, mo time.Month, d, h, mi int, low, high, vol float64, up bool) market.Kline {
	open := time.Date(y, mo, d, h, mi, 0, 0, loc).UnixMilli()
	o, c := low, high // up candle: open at low, close at high
	if !up {
		o, c = high, low // down candle: open at high, close at low
	}
	return market.Kline{
		OpenTime:  open,
		Open:      o,
		High:      high,
		Low:       low,
		Close:     c,
		Volume:    vol,
		CloseTime: open + 60_000 - 1,
	}
}

// findSession returns the session with the given date (17:00 CT anchor) or nil.
func findSession(p SVPProfile, date string) *SVPSession {
	for i := range p.Sessions {
		if p.Sessions[i].Date == date {
			return &p.Sessions[i]
		}
	}
	return nil
}

// TestSVPMultiSession exercises the multi-session grouping: bars spanning TWO
// different CME session-days produce TWO sessions, each anchored to its own
// 17:00 CT open, ascending by SessionStart; the live (current) session is not
// frozen while a prior session is; the partial flag; and the up/down split (an
// up candle's volume lands in UpVol, a down candle's in DownVol).
func TestSVPMultiSession(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")
	// now = Mon 2026-06-15 12:00 CT → live futures session opened Sun 2026-06-14
	// 17:00 CT. The prior session opened Sat... no — the session before is the one
	// that opened Fri 2026-06-12 17:00 CT (Sat is closed), but to keep the test
	// deterministic we use two consecutive session-days that both have bars:
	//   session A: opened Sat 2026-06-13 17:00 CT (label 2026-06-13) — FROZEN
	//   session B (live): opened Sun 2026-06-14 17:00 CT (label 2026-06-14).
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, loc)
	const priorDate = "2026-06-13"
	const liveDate = "2026-06-14"

	var bars []market.Kline
	// Prior session (opened Sat 06-13 17:00): one UP bar at its open, one DOWN bar.
	bars = append(bars, mk1mBarDir(loc, 2026, 6, 13, 17, 0, 100.0, 101.25, 30, true))  // up
	bars = append(bars, mk1mBarDir(loc, 2026, 6, 13, 20, 0, 100.0, 101.25, 10, false)) // down
	// Live session (opened Sun 06-14 17:00): first bar AT the open (not partial),
	// an UP and a DOWN bar so both split channels are populated.
	bars = append(bars, mk1mBarDir(loc, 2026, 6, 14, 17, 0, 200.0, 201.25, 40, true))  // up
	bars = append(bars, mk1mBarDir(loc, 2026, 6, 14, 20, 0, 200.0, 201.25, 15, false)) // down
	bars = append(bars, mk1mBarDir(loc, 2026, 6, 15, 8, 30, 200.0, 201.25, 5, true))   // up (RTH)

	p := BuildSVPProfile(bars, now)
	if p.RowHeight != 1.25 {
		t.Fatalf("rowHeight = %v, want 1.25", p.RowHeight)
	}
	if p.Sessions == nil {
		t.Fatal("Sessions must never be nil")
	}
	if len(p.Sessions) != 2 {
		t.Fatalf("got %d sessions, want 2 (%+v)", len(p.Sessions), p.Sessions)
	}

	// Ascending by SessionStart, and each anchored to its own 17:00 CT open.
	if p.Sessions[0].SessionStart >= p.Sessions[1].SessionStart {
		t.Error("sessions must be ascending by SessionStart")
	}
	wantPriorStart := time.Date(2026, 6, 13, 17, 0, 0, 0, loc).Unix()
	wantLiveStart := time.Date(2026, 6, 14, 17, 0, 0, 0, loc).Unix()
	prior := findSession(p, priorDate)
	live := findSession(p, liveDate)
	if prior == nil || live == nil {
		t.Fatalf("expected sessions %q and %q, got %+v", priorDate, liveDate, p.Sessions)
	}
	if prior.SessionStart != wantPriorStart {
		t.Errorf("prior SessionStart = %d, want %d (17:00 CT anchor)", prior.SessionStart, wantPriorStart)
	}
	if live.SessionStart != wantLiveStart {
		t.Errorf("live SessionStart = %d, want %d (17:00 CT anchor)", live.SessionStart, wantLiveStart)
	}

	// Frozen: only the live (current) session is developing.
	if !prior.Frozen {
		t.Error("prior session must be frozen")
	}
	if live.Frozen {
		t.Error("live session must NOT be frozen (market open Mon 12:00 CT)")
	}
	// Partial: live's first bar is AT the 17:00 CT open → not partial.
	if live.Partial {
		t.Error("live session should not be partial (first bar at the open)")
	}

	// Up/down split: the live session's up bars (40+5=45) and down bar (15) fall
	// in the SAME price row [200,201.25] → one bin with UpVol=45, DownVol=15.
	if len(live.Bins) != 1 {
		t.Fatalf("live bins = %d, want 1 (all bars share the [200,201.25] row): %+v", len(live.Bins), live.Bins)
	}
	lb := live.Bins[0]
	if lb.UpVol != 45 {
		t.Errorf("live UpVol = %v, want 45 (two up candles)", lb.UpVol)
	}
	if lb.DownVol != 15 {
		t.Errorf("live DownVol = %v, want 15 (one down candle)", lb.DownVol)
	}
	if lb.Vol != 60 {
		t.Errorf("live Vol = %v, want 60 (up+down)", lb.Vol)
	}
}

// TestSVPUpDownConservation proves the per-bin invariant Vol == UpVol + DownVol
// across a realistic mixed session (bars spanning several rows, both directions).
func TestSVPUpDownConservation(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, loc)

	var bars []market.Kline
	bars = append(bars, mk1mBarDir(loc, 2026, 6, 14, 17, 0, 100.0, 104.0, 120, true))
	bars = append(bars, mk1mBarDir(loc, 2026, 6, 14, 18, 0, 101.0, 103.5, 80, false))
	bars = append(bars, mk1mBarDir(loc, 2026, 6, 14, 19, 0, 99.5, 102.0, 60, true))
	bars = append(bars, mk1mBarDir(loc, 2026, 6, 14, 20, 0, 100.5, 101.5, 25, false)) // down candle

	p := BuildSVPProfile(bars, now)
	if len(p.Sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(p.Sessions))
	}
	var totalUp, totalDown, totalVol float64
	for _, b := range p.Sessions[0].Bins {
		if math.Abs(b.Vol-(b.UpVol+b.DownVol)) > 1e-9 {
			t.Errorf("bin %v: Vol=%v != UpVol+DownVol=%v", b.Price, b.Vol, b.UpVol+b.DownVol)
		}
		totalUp += b.UpVol
		totalDown += b.DownVol
		totalVol += b.Vol
	}
	// Session totals must match the input volumes (120+60 up, 80+25 down).
	if math.Abs(totalUp-180) > 1e-9 {
		t.Errorf("total UpVol = %v, want 180", totalUp)
	}
	if math.Abs(totalDown-105) > 1e-9 {
		t.Errorf("total DownVol = %v, want 105", totalDown)
	}
	if math.Abs(totalVol-285) > 1e-9 {
		t.Errorf("total Vol = %v, want 285", totalVol)
	}
}

// TestSVPSession_PartialAndReset exercises the partial flag (session open missing)
// and that a session-day with no bars simply does not appear in Sessions.
func TestSVPSession_PartialAndReset(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, loc)

	// Partial: earliest bar Mon 08:30 — hours after the Sun 06-14 17:00 open.
	var late []market.Kline
	late = append(late, mk1mBar(loc, 2026, 6, 15, 8, 30, 100.0, 101.25, 30))
	late = append(late, mk1mBar(loc, 2026, 6, 15, 8, 31, 101.0, 102.0, 10))
	lp := BuildSVPProfile(late, now)
	live := findSession(lp, "2026-06-14")
	if live == nil || !live.Partial {
		t.Errorf("live session should be partial when the 17:00 CT open is missing, got %+v", lp.Sessions)
	}

	// A session-day with no bars in range never appears. Feed only Sun-session
	// bars but evaluate on the NEXT session (Tue 12:00, opened Mon 17:00 CT):
	// the live session-day has no bars, so it is absent; the prior Sun session is
	// still present (frozen).
	var full []market.Kline
	full = append(full, mk1mBar(loc, 2026, 6, 14, 17, 0, 100.0, 101.25, 30))
	full = append(full, mk1mBar(loc, 2026, 6, 14, 20, 0, 101.0, 102.0, 10))
	nextSession := time.Date(2026, 6, 16, 12, 0, 0, 0, loc)
	np := BuildSVPProfile(full, nextSession)
	if findSession(np, "2026-06-15") != nil {
		t.Error("the live (Mon 17:00) session has no bars → must be absent from Sessions")
	}
	prior := findSession(np, "2026-06-14")
	if prior == nil {
		t.Fatalf("prior Sun session should still be present, got %+v", np.Sessions)
	}
	if !prior.Frozen {
		t.Error("prior Sun session must be frozen when evaluating a later session")
	}
}
