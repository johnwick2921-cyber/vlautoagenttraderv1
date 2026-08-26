package store

import (
	"testing"
)

func TestRecordPlayDecaysFreshness(t *testing.T) {
	ls := newLevelStateStore(t).LevelState()
	l := poc("MNQ", "2026-08-14", 1, 100)
	if err := ls.EnsureLevel(l); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	// A → B → C → done, each play stamps last_play + increments times_tested.
	for i, want := range []string{FreshnessB, FreshnessC, FreshnessDone} {
		nowMs := int64((i + 1) * 1000)
		got, err := ls.RecordPlay(l.LevelKey, nowMs)
		if err != nil {
			t.Fatalf("play %d: %v", i, err)
		}
		if got != want {
			t.Fatalf("play %d freshness → %q want %q", i, got, want)
		}
		row, _ := ls.Get(l.LevelKey)
		if row.TimesTested != i+1 || row.LastPlayMs != nowMs {
			t.Fatalf("play %d state: tested=%d last=%d", i, row.TimesTested, row.LastPlayMs)
		}
	}
	final, _ := ls.Get(l.LevelKey)
	if !final.Consumed { // reaching done marks consumed
		t.Fatalf("done must mark consumed")
	}
}

func TestReArmEligible(t *testing.T) {
	base := &LevelStateDB{Freshness: FreshnessB, Consumed: false, LastPlayMs: 0}

	// Fresh, reformed, no recent play → eligible.
	if ok, why := ReArmEligible(base, 100*60_000, ReArmCooldownMin, true); !ok {
		t.Fatalf("should be eligible: %s", why)
	}
	// Bare re-touch (not reformed) → not.
	if ok, _ := ReArmEligible(base, 100*60_000, ReArmCooldownMin, false); ok {
		t.Fatalf("bare re-touch must not re-arm")
	}
	// Consumed → not.
	if ok, _ := ReArmEligible(&LevelStateDB{Freshness: FreshnessB, Consumed: true}, 100*60_000, ReArmCooldownMin, true); ok {
		t.Fatalf("consumed must not re-arm")
	}
	// Below-floor (done) → not.
	if ok, _ := ReArmEligible(&LevelStateDB{Freshness: FreshnessDone}, 100*60_000, ReArmCooldownMin, true); ok {
		t.Fatalf("done must not re-arm")
	}
	// Within cooldown → not.
	cooling := &LevelStateDB{Freshness: FreshnessB, LastPlayMs: 100 * 60_000}
	if ok, _ := ReArmEligible(cooling, 100*60_000+5*60_000, ReArmCooldownMin, true); ok {
		t.Fatalf("within 20m cooldown must not re-arm")
	}
	// After the cooldown → eligible again.
	if ok, why := ReArmEligible(cooling, 100*60_000+21*60_000, ReArmCooldownMin, true); !ok {
		t.Fatalf("after cooldown should re-arm: %s", why)
	}
}
