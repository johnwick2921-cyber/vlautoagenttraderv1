package trader

import (
	"testing"
	"time"

	"nofx/discipline"
)

// ── W-EXEC-TRUTH W0 (CTO M3) — an arm/picture refusal is logged and counted
// once per change of (path, key, CLASS), never once per change of its reason ─
//
// The reasons carry moving values: the re-entry cooldown embeds the live price
// and its distance from the stop, EntryGate's R:R leg the execution price and
// the ratio, the breaker its loss count. Deduped on the reason, the SAME
// refusal re-logged and re-counted on every tick the price moved.

func TestArmRefusalWithAMovingReasonIsCountedOnce(t *testing.T) {
	withMaintenanceDir(t)
	at, _, _, now := liveArmFixture(t)
	at.config.StrategyConfig.RiskControl.ReentryCooldownMinutes = 30
	discipline.ResetReentryForTest()
	t.Cleanup(discipline.ResetReentryForTest)
	sym := at.futuresSymbol()
	discipline.NoteStopLossExit(at.id, sym, "long", 99, now.Add(-time.Minute).UnixMilli())

	in := admitIntent{Path: admitArm, Symbol: sym, Action: "open_long", Key: "p:S1:leg1"}
	before := gateBlocks(at.id, "reentry_cooldown")
	for i, px := range []float64{100, 100.25, 100.5, 101} {
		in.Now, in.Price = now.Add(time.Duration(i)*time.Second), px
		if reason, refused := at.admitEntry(in); !refused || reason == "" {
			t.Fatalf("tick %d: the cooldown must refuse the arm: %q", i, reason)
		}
	}
	if got := gateBlocks(at.id, "reentry_cooldown"); got != before+1 {
		t.Fatalf("one refusal whose reason's PRICE moved must count once, got %d", got-before)
	}

	// A change of CLASS is a new refusal: counted again.
	at.pauseUntilMs.Store(now.Add(time.Hour).UnixMilli())
	t.Cleanup(func() { at.pauseUntilMs.Store(0) })
	stopBefore := gateBlocks(at.id, "stop_until")
	in.Now = now.Add(10 * time.Second)
	if _, refused := at.admitEntry(in); !refused {
		t.Fatal("the owner pause must refuse")
	}
	if gateBlocks(at.id, "stop_until") != stopBefore+1 {
		t.Fatal("a refusal of a NEW class must be counted")
	}
}

// Within entry_gate, the LEG is the class: a new leg re-logs, a moving number
// inside one leg does not. The leg is armRefusalClass's — the same
// "entry_gate:<leg>" string the arm-refusal counter family is keyed on.
func TestEntryGateDedupeClassIsTheLeg(t *testing.T) {
	cases := []struct{ reason, want string }{
		{"entry_gate: R:R 1.20 below floor 2.00 at execution price 29600.2500 (SL 29590 TP 29612)", "entry_gate:rr"},
		{"entry_gate: R:R 1.35 below floor 2.00 at execution price 29601.0000 (SL 29590 TP 29616)", "entry_gate:rr"},
		{"entry_gate: stop 29590.00 too close (4.00 < 6.00 = 1.5×ATR5m)", "entry_gate:min_sl"},
		{"entry_gate: refused: daily_force_flat — limit hit (new entries blocked on the picture path …)", "entry_gate:daily_force_flat"},
		{"entry_gate: picture " + PictureStrictRefusal, "entry_gate:other"},
	}
	for _, c := range cases {
		if got := admitDedupeClass("entry_gate", c.reason); got != c.want {
			t.Errorf("%q → %q, want %q", c.reason, got, c.want)
		}
	}
	if got := admitDedupeClass("reentry_cooldown", "reentry_cooldown: … price 100.2500 …"); got != "reentry_cooldown" {
		t.Errorf("a non-entry_gate class is its own dedupe value, got %q", got)
	}
}
