package ninjatrader

import (
	"errors"
	"testing"
	"time"

	ntwire "vl/provider/ninjatrader"
)

// M10 pin (REVIEW-309 r2, PR B 2026-10-03): the stop-limit build floor. An
// AddOn that proves the stop-SLOT floor but not the stop-LIMIT floor would
// build StopMarket when handed stop_limit=true — fail closed. Remove the floor
// check in placeStopEntry and this test turns RED twice over: the error is no
// longer ErrAddonBuildTooOld AND a signal frame reaches the wire.
func TestStopLimitFloorRefusesAnAddOnBelowIt(t *testing.T) {
	s, frames := allFramesServer(t) // proves 2026-09-20-p1: ≥ stop-slot, < stop-limit
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	tr.traderID = "floor-pin"
	_, err := tr.PlaceStopEntryWithLimit("MNQ", "long", 1, 100, 99, 102)
	if !errors.Is(err, ntwire.ErrAddonBuildTooOld) {
		t.Fatalf("a stop-limit on an AddOn below the floor must refuse with ErrAddonBuildTooOld, got %v", err)
	}
	// The bind emits account_register frames; the ONLY frame that proves the
	// floor leaked is a signal frame.
	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case f := <-frames:
			if f == ntwire.FrameSignal {
				t.Fatalf("the floor refusal must not reach the wire, got a signal frame")
			}
		case <-deadline:
			return
		}
	}
}
