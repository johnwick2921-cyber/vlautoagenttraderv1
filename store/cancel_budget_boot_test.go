package store

import (
	"testing"
)

// B2 — THE CANCEL ATTEMPT BUDGET IS PER PROCESS, NOT PER ROW FOREVER.
//
// RequestCancel bumps cancel_attempts monotonically. A row that reached the cap
// before a restart arrived at the new process still capped, so
// confirmPendingCancels (trader/cancel_confirm.go:400) neither re-requested it
// nor promoted it — it logged "attempt cap reached" forever. The restart is
// precisely the event that changes the facts (a new wire, a re-seeded book), so
// it must also reset the budget.
//
// The reset is keyed on a DEDICATED column. The existing boot_id means "the
// process that AUTHORED this row" (class 33) and reusing it would conflate two
// different questions on one field — the failure mode this repo keeps hitting.
func TestCancelBudgetResetsAcrossBoot(t *testing.T) {
	st, id := cancelStore(t)

	// Three requests inside ONE process: attempts must accumulate.
	for i := 1; i <= 3; i++ {
		if err := st.RequestCancel(id, "attempt", int64(1000+i)); err != nil {
			t.Fatalf("RequestCancel %d: %v", i, err)
		}
		row := readArmForCancelBudget(t, st, id)
		if row.CancelAttempts != i {
			t.Fatalf("same process, request %d: cancel_attempts = %d, want %d — attempts must accumulate WITHIN a boot", i, row.CancelAttempts, i)
		}
	}

	// The row now carries the boot that counted those attempts.
	row := readArmForCancelBudget(t, st, id)
	if row.CancelAttemptsBoot == "" {
		t.Fatal("cancel_attempts_boot is empty after a request — the row cannot say which process counted its attempts, so a restart can never reset the budget")
	}
	if row.CancelAttemptsBoot != ProcessBootID() {
		t.Fatalf("cancel_attempts_boot = %q, want the current process boot %q", row.CancelAttemptsBoot, ProcessBootID())
	}

	// Simulate the restart: the row was counted under a DIFFERENT process.
	if err := st.DB().Exec(
		"UPDATE armed_orders SET cancel_attempts_boot = ? WHERE id = ?",
		"1-1700000000000-a-previous-process", id).Error; err != nil {
		t.Fatalf("stamp a foreign boot: %v", err)
	}

	// The next request is attempt ONE of the new process, not four.
	if err := st.RequestCancel(id, "first attempt after the restart", 2000); err != nil {
		t.Fatalf("RequestCancel after boot: %v", err)
	}
	row = readArmForCancelBudget(t, st, id)
	if row.CancelAttempts != 1 {
		t.Fatalf("across a boot: cancel_attempts = %d, want 1 — a row that arrived at the cap must get a fresh budget from the process that can actually act on it", row.CancelAttempts)
	}
	if row.CancelAttemptsBoot != ProcessBootID() {
		t.Fatalf("across a boot: cancel_attempts_boot = %q, want %q — the reset must re-stamp the counting process", row.CancelAttemptsBoot, ProcessBootID())
	}
}

// A row already AT the cap is the case the defect actually stranded: it must be
// re-requestable after a restart, which is only true if the budget resets.
func TestCancelBudgetAtCapIsReRequestableAfterBoot(t *testing.T) {
	st, id := cancelStore(t)

	const cap = 5
	if err := st.DB().Exec(
		"UPDATE armed_orders SET cancel_attempts = ?, cancel_attempts_boot = ? WHERE id = ?",
		cap, "1-1700000000000-a-previous-process", id).Error; err != nil {
		t.Fatalf("seed a capped row from a previous process: %v", err)
	}

	before := readArmForCancelBudget(t, st, id)
	if before.CancelAttempts < cap {
		t.Fatalf("fixture is not at the cap: %d", before.CancelAttempts)
	}

	if err := st.RequestCancel(id, "re-requested under the new process", 3000); err != nil {
		t.Fatalf("RequestCancel: %v", err)
	}
	after := readArmForCancelBudget(t, st, id)
	if after.CancelAttempts != 1 {
		t.Fatalf("a capped row across a boot: cancel_attempts = %d, want 1 — it stayed capped, so confirmPendingCancels will still refuse to re-request it and the row is stranded", after.CancelAttempts)
	}
}

func readArmForCancelBudget(t *testing.T, st *ArmedOrderStore, id int64) ArmedOrderDB {
	t.Helper()
	var row ArmedOrderDB
	if err := st.DB().First(&row, id).Error; err != nil {
		t.Fatalf("read arm %d: %v", id, err)
	}
	return row
}
