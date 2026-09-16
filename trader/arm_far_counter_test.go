// D4 — the far-arm counter. WARN-first, no refusal in this wave.
//
// 71% of arm-enabled scenarios (34/48) were authored at a price the tape never
// reached in that version's life; on 09-02 six versions carried a short arm
// 62.25 points above the day's high. Nothing counted it, so nothing could tell
// an unlucky level from a habit. A week of counts decides the threshold.

package trader

import (
	"math"
	"testing"
)

func TestFarArmFactorIsDistanceInATRs(t *testing.T) {
	// 62.25 points away with ATR5m 13.0 → 4.788…×ATR (the 09-02 shape).
	got := farArmFactor(29543.75+62.25, 29543.75, 13)
	if math.Abs(got-4.788) > 0.01 {
		t.Errorf("farArmFactor = %.3f, want ≈4.788", got)
	}
	// Direction must not matter — distance is distance.
	if below := farArmFactor(29543.75-62.25, 29543.75, 13); math.Abs(below-got) > 1e-9 {
		t.Errorf("a far arm below price must count the same: %.3f vs %.3f", below, got)
	}
}

// An unknown ATR must not manufacture a factor. A plausible zero here would
// read as "right at the money" (A24).
func TestFarArmFactorRefusesToGuess(t *testing.T) {
	for _, atr := range []float64{0, -1} {
		if got := farArmFactor(29600, 29500, atr); got != 0 {
			t.Errorf("atr=%v must yield 0 (unknown), got %v", atr, got)
		}
	}
	if got := farArmFactor(0, 29500, 13); got != 0 {
		t.Errorf("a zero entry is not a distance, got %v", got)
	}
}

// The threshold is resolved, not literal (A11), and defaults to 3.0.
func TestFarArmThresholdResolves(t *testing.T) {
	if got := farArmThreshold(); got != 3.0 {
		t.Errorf("default threshold = %v, want 3.0", got)
	}
	t.Setenv("ARM_FAR_ATR_MULT", "4.5")
	if got := farArmThreshold(); got != 4.5 {
		t.Errorf("env override = %v, want 4.5", got)
	}
	t.Setenv("ARM_FAR_ATR_MULT", "nonsense")
	if got := farArmThreshold(); got != 3.0 {
		t.Errorf("an unparseable override must fall back to the default, got %v", got)
	}
}

// WARN-first: the decision is "count it", never "refuse it".
func TestFarArmIsCountedNotRefused(t *testing.T) {
	// 62.25 pts at ATR 13 is 4.79× — far by any threshold in play.
	if !armIsFar(29543.75+62.25, 29543.75, 13) {
		t.Error("a 4.79×ATR arm must be counted as far")
	}
	// A 2×ATR arm is not far under the 3.0 default.
	if armIsFar(29543.75+26, 29543.75, 13) {
		t.Error("a 2×ATR arm must not be flagged under the 3.0 default")
	}
	// Unknown ATR must not be flagged — unknown is not far.
	if armIsFar(29600, 29500, 0) {
		t.Error("an unknown ATR must never flag; unknown is not far")
	}
}
