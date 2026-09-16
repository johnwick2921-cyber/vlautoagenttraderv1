package trader

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"nofx/store"
)

func class33Trader(t *testing.T) *AutoTrader {
	t.Helper()
	ResetBootSweepForTest()
	at := plannerTestTrader(t)
	if err := at.store.ArmedOrders().Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return at
}

func class33Seed(t *testing.T, at *AutoTrader, scenario, signal, bootID, state string) {
	t.Helper()
	now := time.Now()
	row := &store.ArmedOrderDB{
		TraderID: at.id, PlanID: "p1", Version: 1, Session: "ASIA", Scenario: scenario,
		Side: "long", EntryPx: 29044, StopPx: 29020, TargetPx: 29100,
		State: state, SignalID: signal, BootID: bootID, CreatedAt: now, UpdatedAt: now,
	}
	if err := at.store.ArmedOrders().UpsertArm(row); err != nil {
		t.Fatalf("seed %s: %v", scenario, err)
	}
	// UpsertArm stamps the CURRENT boot on create; force the foreign stamp.
	if bootID != "" {
		if err := at.store.ArmedOrders().DB().Model(&store.ArmedOrderDB{}).
			Where("plan_id = ? AND scenario = ?", "p1", scenario).
			Update("boot_id", bootID).Error; err != nil {
			t.Fatalf("stamp %s: %v", scenario, err)
		}
	}
}

// E4 — the 00:16 CT shape: two WORKING arms placed by a process that is now
// dead. Both are cancelled at the broker AND in the ledger, with the
// boot_sweep reason, and the recorded counter moves.
func TestClass33BootSweepCancelsPreBootArms(t *testing.T) {
	at := class33Trader(t)
	class33Seed(t, at, "S1", "sig-S1", "999-1", "working")
	class33Seed(t, at, "S3", "sig-S3", "999-1", "working")

	var cancelled []string
	n := at.sweepPreBootArmsWith(at.store.ArmedOrders(), func(sid string) error {
		cancelled = append(cancelled, sid)
		return nil
	})
	if n != 2 {
		t.Fatalf("swept %d, want 2", n)
	}
	if len(cancelled) != 2 || cancelled[0] != "sig-S1" || cancelled[1] != "sig-S3" {
		t.Fatalf("cancel frames wrong: %v", cancelled)
	}
	rows, _ := at.store.ArmedOrders().ListNonTerminal(at.id)
	if len(rows) != 0 {
		t.Fatalf("swept rows must be terminal, %d still non-terminal", len(rows))
	}
	var all []store.ArmedOrderDB
	at.store.ArmedOrders().DB().Find(&all)
	for _, r := range all {
		if r.State != "cancelled" || r.StateReason != BootSweepReason {
			t.Fatalf("row %s: state=%q reason=%q", r.Scenario, r.State, r.StateReason)
		}
	}
	if got, _ := store.BootSweptCount(at.store); got != 2 {
		t.Fatalf("recorded counter = %d, want 2", got)
	}
	// Latched: a second call is a no-op.
	if again := at.sweepPreBootArmsWith(at.store.ArmedOrders(), func(string) error {
		t.Fatal("sweep must run ONCE per process per trader")
		return nil
	}); again != 0 {
		t.Fatalf("second sweep swept %d", again)
	}
}

// E5 — nothing pre-boot: a clean no-op that still says so.
func TestClass33BootSweepNoPreBootRows(t *testing.T) {
	at := class33Trader(t)
	class33Seed(t, at, "S1", "sig-S1", "", "working") // stamped by THIS process
	n := at.sweepPreBootArmsWith(at.store.ArmedOrders(), func(string) error {
		t.Fatal("nothing pre-boot may be cancelled")
		return nil
	})
	if n != 0 {
		t.Fatalf("swept %d, want 0", n)
	}
	if got, _ := store.BootSweptCount(at.store); got != 0 {
		t.Fatalf("counter moved on a no-op: %d", got)
	}
	rows, _ := at.store.ArmedOrders().ListNonTerminal(at.id)
	if len(rows) != 1 {
		t.Fatalf("this process's own arm must survive, got %d", len(rows))
	}
	// F12: leg4's source is a RESOLVED argument now, not a literal in the line.
	if line := BootSweepBootLine(0, 0, "ledger (no snapshot yet)"); !strings.Contains(line, "cancelled 0 pre-boot arm(s)") {
		t.Fatalf("boot line: %s", line)
	}
}

// E6 — a pre-boot row that was AUTHORIZED but never placed has nothing at the
// broker; cancelling it would silently disable the play. Left armed.
func TestClass33BootSweepLeavesUnplacedArms(t *testing.T) {
	at := class33Trader(t)
	class33Seed(t, at, "S2", "", "999-1", "armed")
	n := at.sweepPreBootArmsWith(at.store.ArmedOrders(), func(string) error {
		t.Fatal("no cancel frame may be sent for an order that was never placed")
		return nil
	})
	if n != 0 {
		t.Fatalf("swept %d, want 0", n)
	}
	rows, _ := at.store.ArmedOrders().ListNonTerminal(at.id)
	if len(rows) != 1 || rows[0].State != "armed" {
		t.Fatalf("unplaced arm must stay armed: %+v", rows)
	}
}

// A failed cancel must NOT mark the ledger terminal — that would hide a LIVE
// broker order behind a clean ledger — and must not latch the sweep.
func TestClass33BootSweepCancelFailureKeepsRowLive(t *testing.T) {
	at := class33Trader(t)
	class33Seed(t, at, "S1", "sig-S1", "999-1", "working")
	n := at.sweepPreBootArmsWith(at.store.ArmedOrders(), func(string) error {
		return errors.New("wire down")
	})
	if n != 0 {
		t.Fatalf("swept %d on a failed cancel, want 0", n)
	}
	rows, _ := at.store.ArmedOrders().ListNonTerminal(at.id)
	if len(rows) != 1 || rows[0].State != "working" {
		t.Fatalf("a row whose cancel failed must stay non-terminal: %+v", rows)
	}
	if got, _ := store.BootSweptCount(at.store); got != 0 {
		t.Fatalf("counter moved on a failed cancel: %d", got)
	}
	// Not latched: the retry succeeds.
	if again := at.sweepPreBootArmsWith(at.store.ArmedOrders(), func(string) error { return nil }); again != 1 {
		t.Fatalf("retry swept %d, want 1", again)
	}
}

// E7 — the 0C shadow sweep is untouched: a "shadowed" row is inert by
// construction and is never in the boot sweep's set (nor double-cancelled).
func TestClass33BootSweepIgnoresShadowedRows(t *testing.T) {
	at := class33Trader(t)
	now := time.Now()
	row := &store.ArmedOrderDB{TraderID: at.id, PlanID: "p1", Scenario: "S9", Side: "long",
		EntryPx: 29044, State: "shadowed", StateReason: "condition_shadowed",
		SignalID: "sig-shadow", BootID: "999-1", CreatedAt: now, UpdatedAt: now}
	if err := at.store.ArmedOrders().DB().Create(row).Error; err != nil {
		t.Fatalf("seed shadowed: %v", err)
	}
	n := at.sweepPreBootArmsWith(at.store.ArmedOrders(), func(string) error {
		t.Fatal("a shadowed row is inert — the boot sweep must not touch it")
		return nil
	})
	if n != 0 {
		t.Fatalf("swept %d, want 0", n)
	}
}

// ORDERING PIN — the sweep is reached BEFORE any placement. maybeManageArmedOrders
// is the sole entry to the armed subsystem; runArmedPlacement is called from
// below it. A refactor that moves the sweep under placement fails here.
func TestClass33SweepPrecedesPlacement(t *testing.T) {
	src, err := os.ReadFile("armed_executor.go")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(src)
	sweep := strings.Index(body, "at.sweepPreBootArms(ledger)")
	// The call site is runArmedPlacementAt since 2026-09-10 — the clock is now
	// threaded from maybeManageArmedOrdersAt rather than read inside the callee.
	// Anchored on the shared prefix so this pins the ORDER, which is the class-33
	// invariant, and not the arity of the call.
	place := strings.Index(body, "at.runArmedPlacementAt(bars,")
	if sweep < 0 || place < 0 {
		t.Fatalf("anchors missing: sweep=%d place=%d", sweep, place)
	}
	if sweep > place {
		t.Fatalf("CLASS 33: the boot sweep must precede placement (sweep@%d place@%d)", sweep, place)
	}
	head := strings.Index(body, "func (at *AutoTrader) maybeManageArmedOrders")
	if head < 0 || sweep < head || sweep-head > 1200 {
		t.Fatalf("the sweep must sit at the HEAD of maybeManageArmedOrders (head=%d sweep=%d)", head, sweep)
	}
}
