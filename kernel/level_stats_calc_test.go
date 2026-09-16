package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// TestEvaluateLevelOutcome covers the B4 spec verdicts: untouched → all false;
// touched → Touched; a ≥8pt move within 3 bars of the touch → Reacted;
// a close-through that holds → BrokeClean; ≥3 touches without a clean move →
// Chopped.
func TestEvaluateLevelOutcome(t *testing.T) {
	mk := func(prices []float64) []market.Kline {
		out := make([]market.Kline, len(prices))
		base := time.Date(2026, 8, 25, 18, 0, 0, 0, time.UTC)
		for i, p := range prices {
			t := base.Add(time.Duration(i) * time.Minute)
			out[i] = market.Kline{OpenTime: t.UnixMilli(), Open: p, High: p + 1, Low: p - 1, Close: p, Volume: 10, CloseTime: t.Add(time.Minute).UnixMilli() - 1}
		}
		return out
	}
	// Untouched.
	if o := EvaluateLevelOutcome(mk([]float64{100, 110, 120}), 200, 0, 0); o.Touched || o.Reacted || o.BrokeClean || o.Chopped {
		t.Fatalf("untouched level must be all-false, got %+v", o)
	}
	// Touch + reaction: touch at 100, then 112 (>8pts away) within 3 bars.
	if o := EvaluateLevelOutcome(mk([]float64{100, 101, 112, 115}), 100, 0, 0); !o.Touched || !o.Reacted {
		t.Fatalf("touch + reaction expected, got %+v", o)
	}
	// Broke-clean: close through by >8pts and stays beyond (no return within 4).
	if o := EvaluateLevelOutcome(mk([]float64{100, 101, 111, 112, 113, 114}), 100, 0, 0); !o.BrokeClean {
		t.Fatalf("broke-clean expected, got %+v", o)
	}
	// Broke then returned → NOT clean.
	if o := EvaluateLevelOutcome(mk([]float64{100, 101, 111, 99, 112, 113}), 100, 0, 0); o.BrokeClean {
		t.Fatalf("return-into-level must void broke-clean, got %+v", o)
	}
	// Chopped: 3+ touches, no clean move.
	if o := EvaluateLevelOutcome(mk([]float64{100, 102, 100, 103, 100, 102}), 100, 0, 0); !o.Chopped {
		t.Fatalf("3 touches without a clean move must be chopped, got %+v", o)
	}
}
