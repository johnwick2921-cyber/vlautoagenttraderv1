// W1 item 3 — the attainable entry: what was ACTUALLY available.
//
// The research distinction: A TOUCH IS NOT A FILL. The level price, the arm's
// entry and the fill price are three different numbers, and per-opportunity
// economics needs the one the tape offered — not the one the planner drew.
//
// Every answer carries the ASSUMPTION that produced it. "29600" tells a reader
// nothing about whether it was a resting limit's price, the first print after a
// confirmation, or an observed fill — and those are not equally trustworthy. An
// observed fill is a measurement; a resting limit's fill is an assumption; a
// first-tradeable price is an approximation of one.
//
// NULL is a legitimate and common answer, and it comes in two kinds that must
// not be collapsed: NOTHING WAS ATTAINABLE (never confirmed, never armed) and
// NOT CAPTURED (it confirmed, but no price was recorded). The first is a fact
// about the opportunity; the second is a gap in our own recording.

package store

const (
	AttainableObservedFill   = "observed_fill" // measured
	AttainableRestingLimit   = "resting_limit:assumed_fill_at_entry"
	AttainableStopTrigger    = "stop_entry:assumed_fill_at_trigger"
	AttainableFirstTradeable = "first_tradeable_after_confirm"
	AttainableNone           = "none:never_confirmed_never_armed"
	AttainableNotCaptured    = "not_captured:confirmed_but_no_price_recorded"
)

// AttainableInputs are the observations available at close. Zero values mean
// "not observed" — never "observed as zero".
type AttainableInputs struct {
	LevelPrice float64

	Confirmed                  bool
	FirstTradeableAfterConfirm float64

	Armed    bool
	ArmKind  string // "limit" | "stop_entry"
	ArmEntry float64

	Filled    bool
	FillPrice float64
}

// AttainableEntry is a price or NULL, always with its basis.
type AttainableEntry struct {
	Price *float64
	Basis string
}

// ResolveAttainableEntry picks the most trustworthy available answer.
//
// The order is deliberate and runs from MEASURED to ASSUMED: an observed fill
// beats an arm's assumed fill, which beats a first-tradeable approximation. The
// LEVEL PRICE never appears — it is what was drawn, not what was available, and
// using it is the exact "a touch is not a fill" error.
func ResolveAttainableEntry(in AttainableInputs) AttainableEntry {
	price := func(v float64, basis string) AttainableEntry {
		p := v
		return AttainableEntry{Price: &p, Basis: basis}
	}

	if in.Filled && in.FillPrice > 0 {
		return price(in.FillPrice, AttainableObservedFill)
	}
	if in.Armed && in.ArmEntry > 0 {
		switch in.ArmKind {
		case "stop_entry":
			return price(in.ArmEntry, AttainableStopTrigger)
		default:
			return price(in.ArmEntry, AttainableRestingLimit)
		}
	}
	if in.Confirmed && in.FirstTradeableAfterConfirm > 0 {
		return price(in.FirstTradeableAfterConfirm, AttainableFirstTradeable)
	}
	if in.Confirmed {
		// It confirmed and we failed to record what was available. That is a
		// gap in OUR recording, not a fact about the opportunity, and the two
		// must stay distinguishable.
		return AttainableEntry{Basis: AttainableNotCaptured}
	}
	return AttainableEntry{Basis: AttainableNone}
}
