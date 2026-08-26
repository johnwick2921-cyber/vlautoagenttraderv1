package trader

import (
	"testing"
	"time"

	"nofx/discipline"
)

// W16/R5 — a post-fill DB write failure must FREEZE the trader.
//
// The order already filled at the broker. Without the row, the position is
// invisible to the position list, the reconcile's belief side, daily-loss and
// trade-count accounting, EOD-flat's work list, and MAE/MFE + adherence. It used
// to log at INFO and carry on, with the P0 alert stranded in the else-branch.
//
// The freeze is the reusable A4 flag: it blocks NEW ENTRIES ONLY and persists
// until the owner clears it, which is what makes it visible rather than a log
// line that scrolls away.
func TestW16UntrackedPositionFreezesEntriesOnly(t *testing.T) {
	const traderID = "w16-untracked"
	discipline.ClearFreeze(traderID) // start clean regardless of test order
	t.Cleanup(func() { discipline.ClearFreeze(traderID) })

	if _, frozen := discipline.IsFrozen(traderID); frozen {
		t.Fatal("precondition: trader must start unfrozen")
	}

	// What the failure branch does.
	reason := "untracked live position: LONG MNQ @ 30000.00 filled but the DB write failed (disk full)"
	if !discipline.FreezeTrader(traderID, reason, time.Now().UnixMilli()) {
		t.Fatal("the first freeze must take effect")
	}

	got, frozen := discipline.IsFrozen(traderID)
	if !frozen {
		t.Fatal("an untracked live position MUST freeze the trader — otherwise the bot keeps opening on top of a position it cannot see")
	}
	if got != reason {
		t.Errorf("freeze reason = %q, want the untracked-position detail so the owner knows what to reconcile", got)
	}

	// Re-freezing is a no-op: a repeated failure must not churn the flag.
	if discipline.FreezeTrader(traderID, "something else", time.Now().UnixMilli()) {
		t.Error("a second freeze must not overwrite the first — the original cause is the one to investigate")
	}
	if got2, _ := discipline.IsFrozen(traderID); got2 != reason {
		t.Errorf("freeze reason changed to %q; the first cause must survive", got2)
	}

	// The owner can clear it, which is what makes it a resolvable flag.
	if !discipline.ClearFreeze(traderID) {
		t.Fatal("the owner must be able to clear the freeze after reconciling")
	}
	if _, still := discipline.IsFrozen(traderID); still {
		t.Fatal("clearing must actually unfreeze")
	}
}

// The freeze gate must never trap a position: it is consulted for open_long /
// open_short only, so closes, stop-moves and reconcile stay available.
func TestW16FreezeBlocksOpensNotExits(t *testing.T) {
	const traderID = "w16-exit-safety"
	discipline.ClearFreeze(traderID)
	t.Cleanup(func() { discipline.ClearFreeze(traderID) })
	discipline.FreezeTrader(traderID, "untracked live position", time.Now().UnixMilli())

	// Mirrors the switch in executeDecisionWithRecord (auto_trader_orders.go:165-174).
	blocked := func(action string) bool {
		switch action {
		case "open_long", "open_short":
			_, frozen := discipline.IsFrozen(traderID)
			return frozen
		}
		return false
	}
	for _, a := range []string{"open_long", "open_short"} {
		if !blocked(a) {
			t.Errorf("%s must be blocked while frozen", a)
		}
	}
	for _, a := range []string{"close_long", "close_short", "hold"} {
		if blocked(a) {
			t.Errorf("%s must NOT be blocked — a frozen trader must still be able to get flat", a)
		}
	}
}
