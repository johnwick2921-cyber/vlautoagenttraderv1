package store

import (
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func class33Ledger(t *testing.T) (*Store, *ArmedOrderStore) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	st, err := NewFromGorm(db)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	l := st.ArmedOrders()
	if err := l.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return st, l
}

// E6 — a row armed by THIS process is never pre-boot; a row carrying another
// process's stamp (or none at all — the legacy orphans) always is.
func TestClass33PreBootDecidability(t *testing.T) {
	_, l := class33Ledger(t)
	now := time.Now()
	mine := &ArmedOrderDB{TraderID: "t1", PlanID: "p1", Scenario: "S1", Side: "long",
		EntryPx: 29044, State: "working", SignalID: "sig-mine", CreatedAt: now, UpdatedAt: now}
	if err := l.UpsertArm(mine); err != nil {
		t.Fatalf("upsert mine: %v", err)
	}
	dead := &ArmedOrderDB{TraderID: "t1", PlanID: "p1", Scenario: "S3", Side: "long",
		EntryPx: 29068.05, State: "working", SignalID: "sig-dead", BootID: "999-1",
		CreatedAt: now, UpdatedAt: now}
	if err := l.UpsertArm(dead); err != nil {
		t.Fatalf("upsert dead: %v", err)
	}
	legacy := &ArmedOrderDB{TraderID: "t1", PlanID: "p1", Scenario: "S4", Side: "short",
		EntryPx: 29100, State: "working", SignalID: "sig-legacy", BootID: "",
		CreatedAt: now, UpdatedAt: now}
	if err := l.db.Create(legacy).Error; err != nil { // bypass UpsertArm's stamp
		t.Fatalf("create legacy: %v", err)
	}

	pre, err := l.ListPreBoot("t1", ProcessBootID())
	if err != nil {
		t.Fatalf("ListPreBoot: %v", err)
	}
	if len(pre) != 2 {
		t.Fatalf("want the dead process's row + the unstamped legacy row, got %d: %+v", len(pre), pre)
	}
	for _, r := range pre {
		if r.Scenario == "S1" {
			t.Fatalf("a row armed by THIS process must never be swept: %+v", r)
		}
	}

	// Another trader's orphan is never returned (per-trader scoping).
	other, err := l.ListPreBoot("t2", ProcessBootID())
	if err != nil || len(other) != 0 {
		t.Fatalf("cross-trader leak: %d rows, err=%v", len(other), err)
	}
}

// A same-identity re-arm must NOT refresh the stamp — otherwise a cycle that
// upserts before the sweep would erase the evidence the sweep needs.
func TestClass33ReArmKeepsForeignStamp(t *testing.T) {
	_, l := class33Ledger(t)
	now := time.Now()
	row := &ArmedOrderDB{TraderID: "t1", PlanID: "p1", Scenario: "S3", Side: "long",
		EntryPx: 29068.05, State: "working", SignalID: "sig-dead", BootID: "999-1",
		CreatedAt: now, UpdatedAt: now}
	if err := l.UpsertArm(row); err != nil {
		t.Fatalf("seed: %v", err)
	}
	again := &ArmedOrderDB{TraderID: "t1", PlanID: "p1", Scenario: "S3", Side: "long",
		EntryPx: 29070, State: "armed", CreatedAt: now, UpdatedAt: now}
	// D5 (owner ruling 2026-09-04): a re-arm under a WORKING row is refused
	// outright now, which makes this test's subject stronger rather than
	// weaker — the foreign stamp cannot be touched at all, not merely left
	// un-refreshed.
	if err := l.UpsertArm(again); err == nil {
		t.Fatal("a re-arm under a working row must be refused")
	}
	pre, _ := l.ListPreBoot("t1", ProcessBootID())
	if len(pre) != 1 {
		t.Fatalf("the foreign stamp must survive a same-identity re-arm, got %d rows", len(pre))
	}
}

// The counter is RECORDED (survives a restart), not log-only — the class-35
// lesson, proved across a real reopen rather than an in-memory handle.
func TestClass33BootSweptCounter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ctr.db")
	st, err := New(path)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if n, err := BootSweptCount(st); err != nil || n != 0 {
		t.Fatalf("fresh counter: %d err=%v", n, err)
	}
	if n, err := IncBootSwept(st, 2); err != nil || n != 2 {
		t.Fatalf("inc 2: %d err=%v", n, err)
	}
	if n, err := IncBootSwept(st, 1); err != nil || n != 3 {
		t.Fatalf("inc 1: %d err=%v", n, err)
	}
	if n, err := IncBootSwept(st, 0); err != nil || n != 3 {
		t.Fatalf("inc 0 must be a no-op read: %d err=%v", n, err)
	}
	st.Close()
	st2, err := New(path) // the restart
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	if n, _ := BootSweptCount(st2); n != 3 {
		t.Fatalf("counter did not survive the restart: %d, want 3", n)
	}
}
