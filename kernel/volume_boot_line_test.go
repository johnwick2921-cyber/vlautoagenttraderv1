// (2) The volume-wave boot line must be READ, not typed (A24).
//
// It printed `seats=8` from a package default while the bound strategy's
// max_levels is 12, and `proximity=cfg(resolved per-trader; retuned 0.3)` —
// prose naming a retune that the resolver does not produce (ResolveProximityK
// falls back to ActivationWindowK = 1.5, and the strategy sets no proximity_k
// at all). A boot line that states a number the process does not use is worse
// than one that says nothing.

package kernel

import (
	"fmt"
	"strings"
	"testing"
)

func TestVolumeWaveBootLineIsRead(t *testing.T) {
	line := VolumeWaveBootLine()

	// Every number must be derivable from a constant or resolver.
	for _, want := range []string{
		fmt.Sprintf("%d", DefaultMaxLevels),
		fmt.Sprintf("%d", PlanHardMaxLevels),
		fmt.Sprintf("%.1f", ActivationWindowK),
	} {
		if !strings.Contains(line, want) {
			t.Errorf("boot line must quote the resolved value %q: %s", want, line)
		}
	}

	// The two literals this fixture exists to kill.
	if strings.Contains(line, "retuned 0.3") {
		t.Error("the typed 'retuned 0.3' is still present — the resolver does not produce it")
	}
	if strings.Contains(line, "seats=8") {
		t.Error("seats printed as a bare default; the per-trader cap is what governs and must be labelled as such")
	}

	// A24: a per-trader value the boot process cannot know yet must be
	// labelled, never printed as if it were the live figure.
	if !strings.Contains(line, "per-trader") {
		t.Errorf("per-trader values must be labelled as such: %s", line)
	}
}
