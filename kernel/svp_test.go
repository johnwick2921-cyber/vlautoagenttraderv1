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

// mk1mBar builds a closed 1-minute bar at the given Chicago wall-clock minute.
func mk1mBar(loc *time.Location, y int, mo time.Month, d, h, mi int, low, high, vol float64) market.Kline {
	open := time.Date(y, mo, d, h, mi, 0, 0, loc).UnixMilli()
	return market.Kline{
		OpenTime:  open,
		Open:      low,
		High:      high,
		Low:       low,
		Close:     high,
		Volume:    vol,
		CloseTime: open + 60_000 - 1,
	}
}

// TestSVPSession_LivePartialReset exercises the session windowing: a live
// developing session, the partial flag when early bars are missing, and that a
// bar outside the RTH window is ignored.
func TestSVPSession_LivePartialReset(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")
	// 2026-06-15 is a Monday (RTH day). Session 08:30–15:00 CT.
	// now = 12:00 CT → developing session is live.
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, loc)

	// Full-session-start case: first bar at the 08:30 open → not partial.
	var full []market.Kline
	full = append(full, mk1mBar(loc, 2026, 6, 15, 8, 30, 100.0, 101.25, 30)) // POC-ish
	full = append(full, mk1mBar(loc, 2026, 6, 15, 8, 31, 101.0, 102.0, 10))
	full = append(full, mk1mBar(loc, 2026, 6, 15, 9, 0, 100.5, 101.0, 5))
	// A bar OUTSIDE the RTH window (07:00 CT pre-open) must be ignored.
	full = append(full, mk1mBar(loc, 2026, 6, 15, 7, 0, 90.0, 91.0, 999))

	p := BuildSVPProfile(full, now)
	if p.RowHeight != 1.25 {
		t.Fatalf("rowHeight = %v, want 1.25", p.RowHeight)
	}
	if p.Dev == nil {
		t.Fatal("dev session nil")
	}
	if p.Dev.Frozen {
		t.Error("dev should be live (not frozen) at 12:00 CT")
	}
	if p.Dev.Partial {
		t.Error("dev should NOT be partial: first bar is at the 08:30 open")
	}
	if p.Dev.Date != "2026-06-15" {
		t.Errorf("dev date = %q, want 2026-06-15", p.Dev.Date)
	}
	// The pre-open 999-volume bar must not appear (no giant POC at ~90.5).
	for _, b := range p.Dev.Bins {
		if b.Vol >= 999 {
			t.Errorf("pre-open bar leaked into RTH profile: bin %v vol %v", b.Price, b.Vol)
		}
	}

	// Partial case: earliest bar an hour after the open → partial.
	var late []market.Kline
	late = append(late, mk1mBar(loc, 2026, 6, 15, 9, 30, 100.0, 101.25, 30))
	late = append(late, mk1mBar(loc, 2026, 6, 15, 9, 31, 101.0, 102.0, 10))
	lp := BuildSVPProfile(late, now)
	if lp.Dev == nil || !lp.Dev.Partial {
		t.Error("dev should be partial when the session start is missing")
	}

	// Reset: on the next RTH day, the same bars fall in the PRIOR window, and the
	// new day's dev has no bars.
	nextDay := time.Date(2026, 6, 16, 12, 0, 0, 0, loc)
	np := BuildSVPProfile(full, nextDay)
	if np.Dev == nil || len(np.Dev.Bins) != 0 {
		t.Errorf("dev on the next day should be empty (reset), got %d bins", len(np.Dev.Bins))
	}
	if np.Prior == nil || np.Prior.Date != "2026-06-15" || !np.Prior.Frozen {
		t.Errorf("prior should be the frozen 2026-06-15 session, got %+v", np.Prior)
	}
}
