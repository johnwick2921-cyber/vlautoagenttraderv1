// A4 (G4) — the freeze latch: a mismatch freezes a trader (new entries blocked),
// and the owner clear resumes it.
package discipline

import "testing"

func TestFreeze_FreezeClearLifecycle(t *testing.T) {
	ResetFreezeForTest()
	defer ResetFreezeForTest()

	if _, frozen := IsFrozen("T1"); frozen {
		t.Fatal("a fresh trader must not be frozen")
	}

	// Forged mismatch → FREEZE.
	if !FreezeTrader("T1", "echo mismatch on fill", 1000) {
		t.Fatal("first freeze must report NEW")
	}
	reason, frozen := IsFrozen("T1")
	if !frozen || reason == "" {
		t.Fatalf("T1 must be frozen with a reason; frozen=%v reason=%q", frozen, reason)
	}
	// Re-freeze is idempotent (already frozen → not new; reason preserved).
	if FreezeTrader("T1", "another reason", 2000) {
		t.Fatal("re-freezing an already-frozen trader must report NOT new")
	}
	if r, _ := IsFrozen("T1"); r != "echo mismatch on fill" {
		t.Fatalf("the ORIGINAL freeze reason must be preserved, got %q", r)
	}

	// A different trader is unaffected (isolation).
	if _, frozen := IsFrozen("T2"); frozen {
		t.Fatal("T2 must not be frozen by T1's freeze")
	}

	// Snapshot shows the frozen trader (for the API).
	if snap := FrozenSnapshot(); snap["T1"] == "" {
		t.Fatalf("snapshot must include T1; got %v", snap)
	}

	// Owner CLEAR → resume.
	if !ClearFreeze("T1") {
		t.Fatal("clearing a frozen trader must report it WAS frozen")
	}
	if _, frozen := IsFrozen("T1"); frozen {
		t.Fatal("after clear, T1 must resume (not frozen)")
	}
	// Clearing an unfrozen trader is a no-op false.
	if ClearFreeze("T1") {
		t.Fatal("clearing an unfrozen trader must report false")
	}
}

func TestFreeze_BlankTraderIsNoop(t *testing.T) {
	ResetFreezeForTest()
	defer ResetFreezeForTest()
	if FreezeTrader("", "x", 1) {
		t.Fatal("a blank trader id must never freeze")
	}
	if _, frozen := IsFrozen(""); frozen {
		t.Fatal("blank trader must never be frozen")
	}
}
