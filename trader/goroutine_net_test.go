package trader

import (
	"testing"
	"time"

	"nofx/discipline"
	"nofx/safe"
)

// TestGoNettedPanicFreezesTheOwner is the trader-side RED pin for the
// panic-net-complete wave: a panic inside any at.goNetted goroutine recovers,
// FREEZES this trader (the #246 contract — entries refuse, exits keep
// working), and the goroutine exits cleanly. A mutant that unwraps a wrapped
// site dies in this test (the injected panic is unrecovered).
func TestGoNettedPanicFreezesTheOwner(t *testing.T) {
	discipline.ResetFreezeForTest()
	t.Cleanup(discipline.ResetFreezeForTest)

	at := plannerTestTrader(t)
	at.id = "gonet-a"

	safe.TestPanicNextGoNet.Store(true)
	t.Cleanup(func() { safe.TestPanicNextGoNet.Store(false) })

	at.goNetted("test-gonet", func() { t.Fatal("fn must not run after the injected panic") })

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, frozen := discipline.IsFrozen(at.id); frozen {
			return // recovered, frozen — the net held
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the goNetted panic did not freeze the owner within 10s")
}
