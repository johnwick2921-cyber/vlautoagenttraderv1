package ninjatrader

import "testing"

// W-EXEC-TRUTH W4 / D21 prerequisite — a LIVE bar's close must reach the cache.
//
// The AddOn's boundary frame carries the newly-forming bar FIRST and the
// just-closed previous bar LAST, with final=true:
// VLBarsSubscriptionManager.cs builds the forming bars in the MinIndex..MaxIndex
// loop (:518-533, `false` = forming) and only AFTER that loop appends the
// previous bar once with `true` (:539-551, "the boundary finalization is ours").
// That final bar therefore has a STRICTLY SMALLER timestamp than the bar just
// appended, so Upsert's ascending-order rule sent it to the out-of-order
// `default:` branch and dropped it. The consequence is that NO live-closed bar
// was ever Final in the cache — only historical-seed bars were — and every
// Picture consumer filters on Final.

func TestUpsert_BoundaryFinalisationReachesTheCache(t *testing.T) {
	c := NewBarCache(500)
	const tf = "4h"
	dur := timeframeMs(tf)
	t1 := int64(1789000000000) + dur
	t2 := t1 + dur

	// The first bar while it is still forming.
	c.Upsert("MNQ", tf, []Bar{{T: t1, O: 1, H: 2, L: 0.5, C: 1.5, Final: false}})
	// The boundary frame, in the AddOn's own order.
	c.Upsert("MNQ", tf, []Bar{
		{T: t2, O: 1.5, H: 2.5, L: 1, C: 2, Final: false}, // the NEW forming bar
		{T: t1, O: 1, H: 2, L: 0.5, C: 1.6, Final: true},  // the bar that just CLOSED
	})

	got := c.Get("MNQ", tf)
	if len(got) != 2 {
		t.Fatalf("want 2 cached bars, got %d: %+v", len(got), got)
	}
	if !got[0].Final {
		t.Fatalf("the just-closed bar never became Final in the cache: %+v", got[0])
	}
	if got[0].C != 1.6 {
		t.Fatalf("the finalisation must carry the closing values, got C=%v", got[0].C)
	}
	if got[1].Final {
		t.Fatalf("the still-forming bar must not be marked Final: %+v", got[1])
	}
	if n := c.Finalisations(); n != 1 {
		t.Fatalf("Finalisations() = %d, want 1", n)
	}
}

// A genuine out-of-order INSERT — a bar for a timestamp the ring has never
// held — stays dropped exactly as before. The finalisation path must not
// become a back-door for rewriting history.
func TestUpsert_GenuineOutOfOrderInsertIsStillDropped(t *testing.T) {
	c := NewBarCache(500)
	const tf = "4h"
	dur := timeframeMs(tf)
	t2 := int64(1789000000000) + 2*dur
	older := t2 - 5*dur // never seen (real OHLC: an O==H==L==C bar is an NT8
	// empty-minute placeholder and is refused before the ordering rules run)

	c.Upsert("MNQ", tf, []Bar{{T: t2, O: 1, H: 2, L: 0.5, C: 1.5, Final: false}})
	c.Upsert("MNQ", tf, []Bar{{T: older, O: 9, H: 10, L: 8.5, C: 9.5, V: 12, Final: true}})

	got := c.Get("MNQ", tf)
	if len(got) != 1 {
		t.Fatalf("an unknown older bar must never be inserted, got %d: %+v", len(got), got)
	}
	if n := c.OutOfOrderDrops(); n != 1 {
		t.Fatalf("OutOfOrderDrops() = %d, want 1", n)
	}
}
