package ninjatrader

import "testing"

// ── D3 PINS (wave BARS HORIZON, 2026-09-09) — THE RING REHYDRATES BACKWARDS ──
//
// A Go restart drops the ring to the AddOn's 2000-bar seed (defaultAutoBarsBack)
// while the store holds 21 days. SeedHistorical MERGES within a process
// (bar_cache.go, mergeBarsByTime) so the ring climbs 2000 → 2500 across a
// session — but nothing rehydrates it from the store, so every restart shortens
// the horizon again.
//
// RehydrateOlder restores that depth WITHOUT ever touching the live tail.

func d3Bars(fromT int64, n int, stepMs int64, close float64) []Bar {
	out := make([]Bar, 0, n)
	for i := 0; i < n; i++ {
		t := fromT + int64(i)*stepMs
		out = append(out, Bar{T: t, O: 100, H: 102, L: 98, C: close, V: 5})
	}
	return out
}

// EQUIVALENT MUTANTS, recorded so a later reader does not "fix" a test that is
// already correct: the live-bar guard is DOUBLE. RehydrateOlder both filters to
// strictly-older bars AND passes `existing` as mergeBarsByTime's `incoming`
// (which wins every tie). Removing exactly ONE of the two is behaviourally
// equivalent — the other still protects the live bar — and no test can catch
// it. Removing BOTH is caught by this pin (verified: "live bar 0 was replaced
// by a stored one … C:-1 want C:7").
//
// PIN D3-A — IT EXTENDS BACKWARDS AND NEVER REPLACES A LIVE BAR.
func TestRehydrateOlderExtendsBackwardsOnly(t *testing.T) {
	c := NewBarCache(2500)
	c.SeedHistorical("MNQ", "1m", d3Bars(1_000_000, 100, 60_000, 7)) // live tail, C=7
	// SeedHistorical applies the wire's CLOSE-stamp → OPEN-stamp conversion
	// (openStampBars), so the ring's bar times are NOT the fixture's. Read the
	// tape back rather than assuming: store rows are already open-stamped
	// (bar_persist_wire applies the same conversion before writing), so
	// RehydrateOlder must NOT stamp again, and this pin proves the two agree.
	live := c.Get("MNQ", "1m")

	// The store carries the SAME window plus 300 minutes before it, with a
	// different Close — so a merge that let the store win would be visible.
	stored := d3Bars(live[0].T-300*60_000, 400, 60_000, -1)

	added := c.RehydrateOlder("MNQ", "1m", stored)
	if added != 300 {
		t.Fatalf("rehydrated %d bars, want 300 (only those strictly older than the ring's oldest)", added)
	}
	got := c.Get("MNQ", "1m")
	if len(got) != 400 {
		t.Fatalf("ring holds %d bars, want 400", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].T <= got[i-1].T {
			t.Fatalf("ring is not strictly ascending at %d", i)
		}
	}
	tail := got[len(got)-len(live):]
	for i := range live {
		if tail[i] != live[i] {
			t.Fatalf("live bar %d was replaced by a stored one: %+v want %+v", i, tail[i], live[i])
		}
	}
	if got[0].T != stored[0].T {
		t.Fatalf("oldest bar is %d, want the store's oldest %d", got[0].T, stored[0].T)
	}
}

// PIN D3-B — A COLD KEY IS NEVER FILLED FROM THE STORE.
//
// An empty ring means the feed is down or the AddOn seed has not landed. The
// store DEEPENS a live tape; filling a cold cache with stored bars would make a
// dead feed look alive to every reader downstream.
func TestRehydrateOlderIsANoOpOnAColdKey(t *testing.T) {
	c := NewBarCache(2500)
	if n := c.RehydrateOlder("MNQ", "1m", d3Bars(1_000_000, 500, 60_000, 3)); n != 0 {
		t.Fatalf("rehydrated %d bars into an EMPTY key — the store must never substitute for the live feed", n)
	}
	if got := c.Get("MNQ", "1m"); len(got) != 0 {
		t.Fatalf("cold key now holds %d bars", len(got))
	}
}

// PIN D3-C — IT IS BOUNDED BY THE RING CAPACITY.
func TestRehydrateOlderIsCappedAtMaxBars(t *testing.T) {
	c := NewBarCache(500)
	c.SeedHistorical("MNQ", "1m", d3Bars(10_000_000, 100, 60_000, 7))
	live := c.Get("MNQ", "1m")
	c.RehydrateOlder("MNQ", "1m", d3Bars(live[0].T-5000*60_000, 5000, 60_000, -1))
	got := c.Get("MNQ", "1m")
	if len(got) != 500 {
		t.Fatalf("ring holds %d bars, want the 500 cap", len(got))
	}
	// The cap must trim the OLDEST, never the live tail.
	tail := got[len(got)-len(live):]
	for i := range live {
		if tail[i] != live[i] {
			t.Fatalf("the cap trimmed a LIVE bar at %d", i)
		}
	}
}

// PIN D3-D — NOTHING IS INVENTED. Rehydration adds only bars it was handed,
// and a placeholder bar is refused at this door exactly as at every other
// (isPlaceholderBar, the NO SYNTHETIC BARS law).
func TestRehydrateOlderRefusesPlaceholderBars(t *testing.T) {
	c := NewBarCache(2500)
	c.SeedHistorical("MNQ", "1m", d3Bars(1_000_000, 10, 60_000, 7))
	oldest := c.Get("MNQ", "1m")[0].T
	stored := d3Bars(oldest-5*60_000, 5, 60_000, -1)
	stored[2] = Bar{T: stored[2].T, O: 50, H: 50, L: 50, C: 50, V: 0} // NT8 empty-minute placeholder
	added := c.RehydrateOlder("MNQ", "1m", stored)
	if added != 4 {
		t.Fatalf("rehydrated %d bars, want 4 — the placeholder must be dropped, not stored", added)
	}
	for _, b := range c.Get("MNQ", "1m") {
		if isPlaceholderBar(b) {
			t.Fatalf("a placeholder bar reached the ring at t=%d", b.T)
		}
	}
}
