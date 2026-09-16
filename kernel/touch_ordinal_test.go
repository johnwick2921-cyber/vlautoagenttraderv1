package kernel

import "testing"

// ORDINAL SEED (data-integrity wave, D2) — E2.
//
// touchRegistry is package-level memory and touchLevelState.opened starts at 0,
// so TouchEpisode.Number restarts at 1 on every process boot — while CLOSED
// episodes ARE persisted. The live table shows the skew that produces:
//
//	touch_number  1 → 513 rows · 2 → 229 · 3 → 131 · 4 → 95 · 5 → 62 · 6 → 34
//
// A level touched for the fourth time today is recorded as its first if the
// process restarted in between, and nothing downstream can tell.
//
// The seed is INJECTED because kernel cannot import store. Ordinals are scoped
// by session-day: a new day legitimately restarts at 1.

func TestTouchOrdinalSeedsFromTheStore(t *testing.T) {
	prev := TouchOrdinalSeed
	t.Cleanup(func() { TouchOrdinalSeed = prev })

	var asked []string
	TouchOrdinalSeed = func(traderID, symbol, label string, price float64, sessionDay string) int {
		asked = append(asked, sessionDay)
		if sessionDay == "2026-09-03" {
			return 3 // three episodes already stored today
		}
		return 0 // a different day has none
	}

	if got := seedOrdinalFor("t1", "MNQ", "PDL", 29285, "2026-09-03"); got != 3 {
		t.Errorf("seed = %d, want 3 — the next touch must be the 4th, not the 1st", got)
	}
	if got := seedOrdinalFor("t1", "MNQ", "PDL", 29285, "2026-09-04"); got != 0 {
		t.Errorf("a NEW session-day seeds 0 so the next touch is the 1st, got %d", got)
	}
	if len(asked) != 2 {
		t.Errorf("the seeder must be asked once per miss, got %d calls", len(asked))
	}
}

// No seeder installed (tests, crypto, any caller that never wired it) → 0, and
// behaviour is exactly what it was before this wave.
func TestTouchOrdinalWithoutASeederIsUnchanged(t *testing.T) {
	prev := TouchOrdinalSeed
	t.Cleanup(func() { TouchOrdinalSeed = prev })
	TouchOrdinalSeed = nil
	if got := seedOrdinalFor("t1", "MNQ", "PDL", 29285, "2026-09-03"); got != 0 {
		t.Errorf("no seeder must mean no seed, got %d", got)
	}
}

// A seeder that fails must not invent an ordinal — 0 is the honest answer and
// the episode numbering degrades to today's behaviour rather than lying.
func TestTouchOrdinalSeederFailureIsZero(t *testing.T) {
	prev := TouchOrdinalSeed
	t.Cleanup(func() { TouchOrdinalSeed = prev })
	TouchOrdinalSeed = func(string, string, string, float64, string) int { return -7 }
	if got := seedOrdinalFor("t1", "MNQ", "PDL", 29285, "2026-09-03"); got != 0 {
		t.Errorf("a negative seed is not a count; want 0, got %d", got)
	}
}

// The registry key carries the session-day, so yesterday's state cannot serve
// today's ordinals even in one long-running process.
func TestTouchKeyIsScopedBySessionDay(t *testing.T) {
	a := touchKey("t1", "MNQ", "PDL", 29285, "2026-09-03")
	b := touchKey("t1", "MNQ", "PDL", 29285, "2026-09-04")
	if a == b {
		t.Fatalf("the key must be session-day scoped, both are %q", a)
	}
	if a == touchKey("t1", "MNQ", "PDL", 29286, "2026-09-03") {
		t.Fatal("the key must still separate levels")
	}
}
