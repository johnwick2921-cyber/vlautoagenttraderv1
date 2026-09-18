package kernel

import (
	"strings"
	"testing"
)

// W-WRITE-TIME-FEASIBILITY (2026-09-18) — (d) byte parity at the rendering seam.
//
// KNOB OFF must leave the output contract byte-identical to before the wave:
// the ON rendering is exactly the OFF rendering plus the contract sentence, at
// the same seam the class-38 boot guard renders every boot.

const writeFeasibilitySentenceParity = "WRITE-TIME FEASIBILITY: a scenario whose arm would be refused by the gate-at-arm chain " +
	"(stop too close to the min-SL floor, arm R:R below the arm minimum, a level the structural-geometry gate refuses, " +
	"or a stop-entry trigger already through price) " +
	"is repaired first; after the last repair attempt it is written with arm.enabled=false " +
	"and arm_disabled_reason naming the refusal. "

func TestWriteTimeFeasibilityOffIsByteIdenticalToOnMinusSentence(t *testing.T) {
	off := plannerOutputContract(8, 5, true, true, false)
	on := plannerOutputContract(8, 5, true, true, true)
	if !strings.HasSuffix(on, writeFeasibilitySentenceParity) {
		t.Fatalf("ON rendering must end with the contract sentence")
	}
	if on[:len(on)-len(writeFeasibilitySentenceParity)] != off {
		t.Fatalf("KNOB OFF is not byte-identical to ON-minus-sentence — the wave changed the OFF rendering")
	}
}
