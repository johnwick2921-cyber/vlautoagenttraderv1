package trader

import (
	"os"
	"strings"
	"testing"
)

// E5 (ledger-close 2026-08-19) — interaction edge: when BOTH the stop_until
// pause and the contract-roll window are active, the refusal must name
// stop_until (gate-order contract 2.4: first among owner/policy gates). Pinned
// on the source like the other ordering guards.
func TestOwnerGateOrderPauseBeforeRoll(t *testing.T) {
	b, err := os.ReadFile("auto_trader_orders.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	pause := strings.Index(s, "stop_until OWNER PAUSE")
	roll := strings.Index(s, "CONTRACT-ROLL gate")
	loss := strings.Index(s, "consecutive-loss halt")
	if pause < 0 || roll < 0 || loss < 0 {
		t.Fatalf("gate anchors missing: pause=%d roll=%d loss=%d", pause, roll, loss)
	}
	if !(pause < roll && roll < loss) {
		t.Fatalf("owner-gate order violated: stop_until(%d) < contract-roll(%d) < consecutive-loss(%d) required — a paused refusal must NAME the pause", pause, roll, loss)
	}
}
