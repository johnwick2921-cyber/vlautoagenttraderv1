package trader

import (
	"testing"

	"nofx/kernel"
)

// P1 — the ENTRY GATE honors the boot-integrity refusal: a refused process opens
// NO new position, while closes stay allowed (an existing position must still be
// manageable out). This is the behavioral half of the Knight-Capital control.
func TestW14BootIntegrityBlocksEntriesNotCloses(t *testing.T) {
	t.Cleanup(func() { kernel.SetTradingRefusedForTest(false, "") })

	// healthy boot → gate open
	kernel.SetTradingRefusedForTest(false, "")
	if _, refused := kernel.TradingRefused(); refused {
		t.Fatal("healthy boot must allow entries")
	}

	// failed boot → gate closed, with a reason the log/alert can carry
	kernel.SetTradingRefusedForTest(true, "binary is revision \"aaa\" but the intended release is \"bbb\" — a stale binary is running")
	reason, refused := kernel.TradingRefused()
	if !refused {
		t.Fatal("a failed boot assertion must refuse entries")
	}
	if reason == "" {
		t.Fatal("the refusal must explain itself (it drives the P0 alert body)")
	}

	// the gate is entry-only by construction: the switch in executeDecisionWithRecord
	// lists open_long/open_short only. Assert that contract here so a future edit
	// that adds close_* to the boot-integrity switch fails loudly.
	for _, act := range []string{"close_long", "close_short", "hold", "wait"} {
		if isBootIntegrityGatedAction(act) {
			t.Fatalf("%s must NEVER be blocked by boot integrity (closes manage risk down)", act)
		}
	}
	for _, act := range []string{"open_long", "open_short"} {
		if !isBootIntegrityGatedAction(act) {
			t.Fatalf("%s must be blocked when boot integrity failed", act)
		}
	}
}
