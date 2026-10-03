package mentor

import (
	"testing"

	"vl/market"
)

// TestTriggerVerdictZoneIncludesTheLines — between two opposing trigger lines
// there is NO trade at all, ISB included: "KHỎI ĐÁNH… đợi nó thoát ra khỏi 2
// cái" [D3.4 p1 @ 16:56–17:17]. The zone covers the lines themselves — price
// must escape BOTH before anything may trade.
func TestTriggerVerdictZoneIncludesTheLines(t *testing.T) {
	// long line at 100, reversal moved it to short at 97 (old long line 100).
	tl := TriggerLine{Dir: SideShort, Price: 97, Moved: true, OldPrice: 100, OldDir: SideLong}
	for _, p := range []float64{97, 100, 98.5} {
		if ok, _, _ := TriggerVerdict(tl, p); ok {
			t.Fatalf("price %.2f inside/at the zone must refuse everything, ISB included [D3.4 p1 @ 16:56]", p)
		}
	}
	// escaped BOTH on the trigger side → allowed short
	if ok, side, _ := TriggerVerdict(tl, 96.5); !ok || side != SideShort {
		t.Fatalf("escaped below both lines must allow shorts: ok=%v side=%q", ok, side)
	}
	// above the old line the zone is escaped, but the trigger side is short —
	// still no trade [D3.4 p1 @ 10:21 entries only on the trigger side]
	if ok, _, _ := TriggerVerdict(tl, 100.5); ok {
		t.Fatal("above both lines on the wrong trigger side must still refuse")
	}
}

// TestTriggerVerdictISBInTheZoneIsRefused — the ISB call site passes the
// inside-bar candle's close into TriggerVerdict; a close inside the zone
// refuses the arm (the evaluator's ISB branch then never arms).
func TestTriggerVerdictISBInTheZoneIsRefused(t *testing.T) {
	tl := TriggerLine{Dir: SideShort, Price: 97, Moved: true, OldPrice: 100, OldDir: SideLong}
	// a perfect ISB pair whose close sits between the two lines
	prev := market.Kline{Open: 99, High: 101, Low: 96, Close: 98}
	cur := market.Kline{Open: 98.5, High: 99, Low: 97.5, Close: 98.5}
	if !IsISB(prev, cur) {
		t.Fatal("test fixture: the pair must be an inside bar")
	}
	if ok, _, _ := TriggerVerdict(tl, cur.Close); ok {
		t.Fatal("an ISB closing between two trigger lines must not arm")
	}
}
