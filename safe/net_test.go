package safe

import (
	"sync/atomic"
	"testing"
	"time"

	"nofx/discipline"
	"nofx/telemetry"
)

// TestGoNetPanicRecoversFreezesAndCounts is the panic-net-complete RED pin: an
// injected panic in a GoNet goroutine must NOT kill the process — it recovers,
// runs the onPanic callback, FREEZES the owner, and lands on the errors API.
// A mutant that removes the recover from GoNet dies inside this test (the
// injected panic is unrecovered).
func TestGoNetPanicRecoversFreezesAndCounts(t *testing.T) {
	discipline.ResetFreezeForTest()
	t.Cleanup(discipline.ResetFreezeForTest)

	TestPanicNextGoNet.Store(true)
	t.Cleanup(func() { TestPanicNextGoNet.Store(false) })

	var ran atomic.Bool
	var alerted atomic.Bool
	done := make(chan struct{})
	GoNet("test-net", "t-net", func() { ran.Store(true) }, func(r interface{}) {
		alerted.Store(true)
		close(done)
	})

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("GoNet did not recover the injected panic")
	}
	if ran.Load() {
		t.Fatal("fn must not run after the injected pre-fn panic")
	}
	if !alerted.Load() {
		t.Fatal("the onPanic callback must run on a recovered panic")
	}
	if _, frozen := discipline.IsFrozen("t-net"); !frozen {
		t.Fatal("the owner trader must be FROZEN")
	}
	found := false
	for _, row := range telemetry.ErrorSummary("t-net") {
		if row.Type == "goroutine_panic" && row.Count >= 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("the panic must be RECORDED (goroutine_panic) on the errors API")
	}
}
