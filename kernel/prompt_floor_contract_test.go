// (1) The prompt's stop-floor sentence must READ the resolver the gate uses.
//
// A typed multiple in the prompt is a promise the gate does not keep: set
// MIN_SL_ATR_MULT=1.5 and the model keeps authoring 1.0×ATR stops that the
// arm gate then refuses, every cycle, learning nothing. Prompt text == gate
// floor, or the two are arguing.

package kernel

import (
	"fmt"
	"strings"
	"testing"
)

func TestPromptStopFloorReadsTheGateResolver(t *testing.T) {
	// Default (1.0) is the easy case and would pass even against a literal, so
	// the pin moves the resolver and demands the prompt move with it.
	t.Setenv("MIN_SL_ATR_MULT", "1.5")
	if got := MinSLATRMult(); got != 1.5 {
		t.Fatalf("test premise wrong: resolver = %v", got)
	}

	p := plannerOutputContract(8, 3, true, true)
	want := fmt.Sprintf("%.1f×", MinSLATRMult())
	if !strings.Contains(p, want) {
		t.Fatalf("prompt must quote the RESOLVED stop floor %q — a typed multiple promises what the gate will not keep", want)
	}
	if strings.Contains(p, "1.0× the current 5m ATR") {
		t.Error("the typed 1.0× is still present; it must be interpolated from MinSLATRMult()")
	}
}

// And at the default the sentence must still read correctly.
func TestPromptStopFloorAtDefault(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "")
	p := plannerOutputContract(8, 3, true, true)
	if !strings.Contains(p, fmt.Sprintf("%.1f×", MinSLATRMult())) {
		t.Errorf("default floor %v not quoted in the prompt", MinSLATRMult())
	}
}
