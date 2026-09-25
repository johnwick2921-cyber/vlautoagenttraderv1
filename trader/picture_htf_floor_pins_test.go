package trader

import (
	"testing"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// DEFAULTS-SANE (CTO #212 fold, DS-105): the old pins compared the resolver to
// its own constant — a tautology that stays green at ANY value. These are
// FLOOR pins derived from PRODUCTION constants at the production call site.
//
// The trace (all [A], read at fix/defaults-sane + merge of origin/dev):
//
//   - The evaluator's entry window is anchored at the 5m interval that follows
//     the confirming H1 close (picture_htf_evaluator.go: nextFiveMBoundary),
//     and a boundary candle is actionable only if its receipt lands inside the
//     window (the freshest5mAt <= intervalStart+windowMs check).
//   - The live sink refuses frames older than LiveFrameMaxAgeMs as live entry
//     events (bar_live_sink.go) — a frame may arrive up to 30s after its close.
//   - A dropped boundary frame (measured at EVERY hour-boundary storm, PR #212
//     STEP 1) is recoverable only by the SUCCESSOR completed 5m frame, whose
//     close sits one full 5m interval (fiveMMs) after the window anchor; its
//     receipt can be up to the sink's own admission late. So the latest
//     actionable receipt is fiveMMs + LiveFrameMaxAgeMs into the window.
//   - Placement is event-driven (armedEventMinGap = 1s, poked by every live
//     bar; measured arm CREATE→PLACED 215–810 ms, PR #212 STEP 1), so
//     detection→placement adds ~1s — already inside the frame-receipt bound.
//
// Floor 1: the window default must EXCEED the worst actionable receipt
// (one 5m interval + the sink's admission bound). 10s and 90s both FAIL here.
func TestPictureHtfDefaultWindowExceedsWorstDetectionToPlacement(t *testing.T) {
	worstReceiptSec := fiveMMs/1000 + ntwire.LiveFrameMaxAgeMs/1000 // 300 + 30
	if got := int64(store.PictureHtfDefaultEntryWindowSec); got <= worstReceiptSec {
		t.Fatalf("entry window default %ds does not exceed the worst actionable receipt %ds — a dropped boundary frame (measured at every hour storm) is unrecoverable and Picture can miss every setup", got, worstReceiptSec)
	}
}

// Floor 2: the freshness default must ADMIT every frame the live sink admitted
// as a live entry event — a stricter default would refuse, inside the
// evaluator, frames the pipeline itself accepted. 2s and 15s both FAIL here.
func TestPictureHtfDefaultFreshnessAdmitsEverySinkAdmittedFrame(t *testing.T) {
	sinkBoundSec := int64(ntwire.LiveFrameMaxAgeMs) / 1000
	if got := int64(store.PictureHtfDefaultFreshnessSec); got < sinkBoundSec {
		t.Fatalf("freshness default %ds is stricter than the live sink's own admission bound %ds — the sink admits a frame the evaluator then refuses as stale", got, sinkBoundSec)
	}
}
