// W1 item 3 — the attainable entry. E5.
//
// The research distinction this exists for: A TOUCH IS NOT A FILL. The level
// price, the arm's entry and the fill price are three different numbers, and
// per-opportunity economics needs the one that was actually available — not the
// level the planner drew.
//
// Every value carries the ASSUMPTION that produced it, because "29600" tells a
// reader nothing about whether it was a resting limit's price, the first print
// after a confirmation, or a guess.

package store

import "testing"

// A confirmed scenario records the first tradeable price AFTER confirmation —
// never the level price, which is what the planner drew rather than what the
// tape offered.
func TestConfirmedRecordsFirstTradeableAfterConfirmation(t *testing.T) {
	e := ResolveAttainableEntry(AttainableInputs{
		Confirmed: true, FirstTradeableAfterConfirm: 29604.25, LevelPrice: 29600,
	})
	if e.Price == nil || *e.Price != 29604.25 {
		t.Fatalf("want the first tradeable price 29604.25, got %v", e.Price)
	}
	if e.Basis != AttainableFirstTradeable {
		t.Errorf("basis = %q, want %q", e.Basis, AttainableFirstTradeable)
	}
	if *e.Price == 29600 {
		t.Error("the LEVEL price is what was drawn, not what was available")
	}
}

// A resting limit records its own entry, and names the fill assumption — a
// resting order's fill is an assumption, not an observation.
func TestRestingLimitRecordsItsEntryWithTheAssumptionNamed(t *testing.T) {
	e := ResolveAttainableEntry(AttainableInputs{
		Confirmed: true, Armed: true, ArmKind: "limit", ArmEntry: 29598,
		FirstTradeableAfterConfirm: 29604.25,
	})
	if e.Price == nil || *e.Price != 29598 {
		t.Fatalf("a resting limit is attainable at ITS price, got %v", e.Price)
	}
	if e.Basis != AttainableRestingLimit {
		t.Errorf("basis = %q, want %q — the assumption must be named", e.Basis, AttainableRestingLimit)
	}
}

// A stop entry is not a limit: it triggers THROUGH its price, so the assumption
// is different and must not be silently reused.
func TestStopEntryNamesItsOwnAssumption(t *testing.T) {
	e := ResolveAttainableEntry(AttainableInputs{
		Confirmed: true, Armed: true, ArmKind: "stop_entry", ArmEntry: 29610,
	})
	if e.Basis != AttainableStopTrigger {
		t.Errorf("basis = %q, want %q", e.Basis, AttainableStopTrigger)
	}
}

// An actual fill beats every assumption — it is the only OBSERVED number here.
func TestAnActualFillOutranksEveryAssumption(t *testing.T) {
	e := ResolveAttainableEntry(AttainableInputs{
		Confirmed: true, Armed: true, ArmKind: "limit", ArmEntry: 29598,
		Filled: true, FillPrice: 29597.5,
	})
	if e.Price == nil || *e.Price != 29597.5 {
		t.Fatalf("an observed fill must win, got %v", e.Price)
	}
	if e.Basis != AttainableObservedFill {
		t.Errorf("basis = %q, want %q", e.Basis, AttainableObservedFill)
	}
}

// E5's NULL half: a scenario that never armed and never confirmed has NO
// attainable entry. Not the level price, not zero — NULL, with a reason.
func TestUnarmedUnconfirmedIsNull(t *testing.T) {
	e := ResolveAttainableEntry(AttainableInputs{LevelPrice: 29600})
	if e.Price != nil {
		t.Fatalf("nothing was attainable; got %v — the level price is not an entry", *e.Price)
	}
	if e.Basis != AttainableNone {
		t.Errorf("basis = %q, want %q", e.Basis, AttainableNone)
	}
}

// A confirmation with no tradeable price captured is NULL too — the confirm
// fired but the price was not observed, and inventing one would be the lie.
func TestConfirmedWithoutACapturedPriceIsNull(t *testing.T) {
	e := ResolveAttainableEntry(AttainableInputs{Confirmed: true, LevelPrice: 29600})
	if e.Price != nil {
		t.Fatalf("no tradeable price was captured; got %v", *e.Price)
	}
	if e.Basis != AttainableNotCaptured {
		t.Errorf("basis = %q, want %q — 'not captured' must differ from 'nothing attainable'", e.Basis, AttainableNotCaptured)
	}
}
