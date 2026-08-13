package discipline

import "testing"

const minMs = int64(60_000)

// TestReentry_TimeUnlock: a stop-loss arms the cooldown; a same-direction re-entry
// is blocked until the cooldown elapses, then unlocks (and stays unlocked).
func TestReentry_TimeUnlock(t *testing.T) {
	ResetReentryForTest()
	defer ResetReentryForTest()

	t0 := int64(1_000_000)
	NoteStopLossExit("T1", "MNQ", "long", 20000, t0)

	// 5 min later, price barely moved (0.5 < ATR15=10) → still blocked.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "long", 20, 10, 20000.5, t0+5*minMs); !blocked {
		t.Fatal("within cooldown + price unmoved must block")
	}
	// 19 min → still blocked (cooldown 20).
	if rem, _, blocked := ReentryBlocked("T1", "MNQ", "long", 20, 10, 20000.5, t0+19*minMs); !blocked || rem <= 0 {
		t.Fatalf("19<20 min must still block with positive remaining; rem=%d", rem)
	}
	// 20 min → cooldown elapsed → unlock.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "long", 20, 10, 20000.5, t0+20*minMs); blocked {
		t.Fatal("cooldown elapsed must unlock")
	}
	// Record cleared on unlock → still unlocked afterwards even at t=0 again.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "long", 20, 10, 20000.5, t0+1*minMs); blocked {
		t.Fatal("record must be cleared after a time unlock")
	}
}

// TestReentry_PriceMoveUnlock: price moving ≥1×ATR15 from the stop unlocks BEFORE
// the cooldown elapses (the "whichever first" branch).
func TestReentry_PriceMoveUnlock(t *testing.T) {
	ResetReentryForTest()
	defer ResetReentryForTest()

	t0 := int64(2_000_000)
	NoteStopLossExit("T1", "MNQ", "short", 20000, t0)

	// 2 min in, price 5 away (<10=ATR15) → blocked.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "short", 20, 10, 19995, t0+2*minMs); !blocked {
		t.Fatal("price move < ATR15 within cooldown must block")
	}
	// Price now a full ATR15 (10) away → unlock even though only 2 min passed.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "short", 20, 10, 19990, t0+2*minMs); blocked {
		t.Fatal("price moved ≥1×ATR15 must unlock before cooldown")
	}
	// Record cleared → subsequent check unblocked.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "short", 20, 10, 20000, t0+3*minMs); blocked {
		t.Fatal("record must be cleared after a price-move unlock")
	}
}

// TestReentry_Isolation: cooldown is scoped to (trader, symbol, side). The opposite
// direction, a different symbol, and a different trader are never blocked. 0 = OFF.
func TestReentry_Isolation(t *testing.T) {
	ResetReentryForTest()
	defer ResetReentryForTest()

	t0 := int64(3_000_000)
	NoteStopLossExit("T1", "MNQ", "long", 20000, t0)
	now := t0 + 1*minMs

	// Same key → blocked.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "long", 20, 10, 20000, now); !blocked {
		t.Fatal("same key must block")
	}
	// Opposite direction → not blocked.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "short", 20, 10, 20000, now); blocked {
		t.Fatal("opposite direction must not be blocked")
	}
	// Different symbol → not blocked.
	if _, _, blocked := ReentryBlocked("T1", "ES", "long", 20, 10, 20000, now); blocked {
		t.Fatal("different symbol must not be blocked")
	}
	// Different trader → not blocked (per-trader isolation).
	if _, _, blocked := ReentryBlocked("T2", "MNQ", "long", 20, 10, 20000, now); blocked {
		t.Fatal("different trader must not be blocked")
	}
	// cooldownMinutes = 0 → OFF (never blocks) even on the armed key.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "long", 0, 10, 20000, now); blocked {
		t.Fatal("cooldownMinutes=0 must disable the gate")
	}
}

// TestReentry_ZeroATRFallsBackToTimer: with ATR15 unavailable (≤0) the price-move
// unlock can't be evaluated, so only the timer unlocks — it must still block within
// the window and never crash.
func TestReentry_ZeroATRFallsBackToTimer(t *testing.T) {
	ResetReentryForTest()
	defer ResetReentryForTest()

	t0 := int64(4_000_000)
	NoteStopLossExit("T1", "MNQ", "long", 20000, t0)

	// atr15=0, huge price move → still blocked (can't confirm the move unlock).
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "long", 20, 0, 99999, t0+1*minMs); !blocked {
		t.Fatal("atr15<=0 must fall back to the timer and stay blocked within the window")
	}
	// Timer still unlocks.
	if _, _, blocked := ReentryBlocked("T1", "MNQ", "long", 20, 0, 99999, t0+20*minMs); blocked {
		t.Fatal("timer must unlock even with atr15<=0")
	}
}
