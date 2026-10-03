package mentor

import (
	"testing"

	"vl/market"
)

// TestTriggerFormingBucketCommittedOnce — B2: the forming bucket may fire a
// break once and is NEVER re-applied. Re-ticking with the SAME bucket grown
// (its high now breaks the previous bucket) must not fire a line.
// (Mutant: drop the LastBucket skip → RED.)
func TestTriggerFormingBucketCommittedOnce(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	// bucket 17:00 recorded; bucket 17:05 partial (no break yet)
	first := TriggerTick(TriggerLine{}, []market.Kline{
		b5(0, 100, 96, 97),
		b5(5, 101, 98, 99), // high 101 does NOT break 100
	}, cfg)
	if first.Dir != "" {
		t.Fatalf("no break should have fired: %+v", first)
	}
	// the SAME forming bucket now grows a high that WOULD break → committed
	// once means it is never re-applied.
	second := TriggerTick(first, []market.Kline{
		b5(0, 100, 96, 97),
		b5(5, 104, 98, 99), // grown: high 104 > 100 — but already processed
	}, cfg)
	if second.Dir != "" {
		t.Fatalf("the forming bucket was re-applied after growing: %+v", second)
	}
	// a NEW bucket may then break (the stream continues normally)
	third := TriggerTick(second, []market.Kline{
		b5(0, 100, 96, 97),
		b5(5, 104, 98, 99),
		b5(10, 99, 95, 95), // breaks the LOW of the (now past) bucket → sell line at 98
	}, cfg)
	if third.Dir != SideShort || third.Price != 98 {
		t.Fatalf("next bucket break = %+v, want short @ 98", third)
	}
}
