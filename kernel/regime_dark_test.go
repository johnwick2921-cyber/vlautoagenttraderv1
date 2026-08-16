package kernel

import "strings"
import "testing"

// P2 — dark-field naming, the DEGRADED threshold, and the alert body.
func TestDarkRegimeFields(t *testing.T) {
	// a fully dark block (nothing computed) → all 7 named
	if got := DarkRegimeFields(RegimeBlock{TrendDaily: "n/a", Trend1h: "n/a", ATRRegime: "n/a"}); len(got) != RegimeFieldCount {
		t.Fatalf("a blind read must name all %d fields, got %d: %v", RegimeFieldCount, len(got), got)
	}
	// the real-world shape TODAY: VIX has no feed + no gap + RV warming-at-zero
	live := RegimeBlock{
		TrendDaily: "up", Trend1h: "up", ATR14: 595.8, ATRRegime: "NORMAL",
		RealizedVolPct: 1.7, ExpectedRangePts: 595.8, HasGap: true,
	}
	dark := DarkRegimeFields(live)
	if len(dark) != 1 || dark[0] != "vix_level" {
		t.Fatalf("a healthy live block should be dark only on vix_level, got %v", dark)
	}
	h := AssessRegime(live, 0)
	if h.Degraded {
		t.Fatal("1 dark field must NOT degrade the plan (threshold 3)")
	}
	// the livelock shape: the watchdog starves timeframes → 4 dark
	starved := RegimeBlock{TrendDaily: "n/a", Trend1h: "n/a", ATRRegime: "n/a", VIXRegime: "unavailable"}
	h2 := AssessRegime(starved, 0)
	if h2.DarkCount < 4 || !h2.Degraded {
		t.Fatalf("a starved read (4+ dark) must flag DEGRADED, got count=%d degraded=%v", h2.DarkCount, h2.Degraded)
	}
	if !strings.Contains(h2.AlertBody(), "DEGRADED") || !strings.Contains(h2.AlertBody(), "trend_state_daily") {
		t.Fatalf("the alert must name the dark fields + the verdict: %q", h2.AlertBody())
	}
	// nothing dark → no alert body at all
	full := RegimeBlock{TrendDaily: "up", Trend1h: "up", ATR14: 1, ATRRegime: "LOW",
		RealizedVolPct: 1, VIXLevel: 18, ExpectedRangePts: 1, HasGap: true}
	if AssessRegime(full, 0).AlertBody() != "" {
		t.Fatal("a full map must produce no alert")
	}
}
