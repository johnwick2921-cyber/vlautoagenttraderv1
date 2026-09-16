// D6 — the arms boot line is READ from the resolvers, never a literal (A24).

package trader

import (
	"strings"
	"testing"
)

func TestArmsBootLineIsRead(t *testing.T) {
	t.Setenv("ARM_FAR_ATR_MULT", "4.5")
	line := ArmsBootLine()

	// The far threshold must come from the resolver, so changing it changes
	// the line. A literal would keep printing 3.0.
	if !strings.Contains(line, "4.5") {
		t.Fatalf("far threshold not read from the resolver: %s", line)
	}
	for _, want := range []string{"🎯 arms:", "bias-coherent=warn", "stop-entry=", "far-arm counter=on", "ledger append-only=on"} {
		if !strings.Contains(line, want) {
			t.Errorf("boot line missing %q: %s", want, line)
		}
	}
	// D2 ships WARN-first — the line must not claim a reject it does not do.
	if strings.Contains(line, "bias-coherent=on") {
		t.Errorf("D2 is WARN-first; the line must not read as an enforced gate: %s", line)
	}
	// stop-entry availability is the reclaim gate change — it must name reclaim.
	if !strings.Contains(line, "reclaim") {
		t.Errorf("the line must name the condition that gained a stop entry: %s", line)
	}
}
