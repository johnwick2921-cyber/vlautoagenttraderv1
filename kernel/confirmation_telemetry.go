package kernel

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"nofx/logger"
	"sync/atomic"
	"time"
)

//go:embed confirmation_replay_receipt.json
var confirmationReplayReceipt []byte

var formingBucketRefusals atomic.Uint64
var outOfOrderRefusals atomic.Uint64

// W2 (d) — the canonical confirmation tape's RECORDED counters (per
// evaluation that met the bar, never inferred): a strictly-older bar dropped,
// and a same-minute copy that replaced the kept bar. The live cache keeps bars
// ascending and unique, so a non-zero drop count names an input defect; the
// first drop in a process WARNs once with the two instants.
var confirmationTapeDropped atomic.Uint64
var confirmationTapeReplaced atomic.Uint64
var confirmationTapeDropWarned atomic.Bool

func recordConfirmationTapeDrop(openMs, lastMs int64) {
	n := confirmationTapeDropped.Add(1)
	if confirmationTapeDropWarned.CompareAndSwap(false, true) {
		logger.Warnf("⚠️ confirmation tape: dropped an out-of-order 1m bar (open %s arrived after %s) — it cannot count a close (drops=%d, counted in the 🔎 line)",
			FormatCT(time.UnixMilli(openMs)), FormatCT(time.UnixMilli(lastMs)), n)
	}
}

// ConfirmationTapeCounters returns the recorded tape counters (dropped
// out-of-order bars, same-minute replacements) since process start.
func ConfirmationTapeCounters() (dropped, replaced uint64) {
	return confirmationTapeDropped.Load(), confirmationTapeReplaced.Load()
}

func recordConfirmationVerdict(v ConfirmVerdict) {
	forming, outOfOrder := false, false
	inspect := func(l ConfirmVerdict) {
		forming = forming || (!l.Met && l.Bucket != nil && !l.Bucket.Closed)
		outOfOrder = outOfOrder || l.Refusal == "out_of_order"
	}
	inspect(v)
	for _, l := range v.Legs {
		inspect(l)
	}
	if forming {
		formingBucketRefusals.Add(1)
	}
	if outOfOrder {
		outOfOrderRefusals.Add(1)
	}
	logger.Infof("🔎 confirmation verdict: rule=%s outcome=%s evaluated_ms=%d · %s", v.Rule, v.Outcome, v.EvaluatedMs, v.Detail)
}

// ConfirmationBootLine separates frozen audit observations from live process
// counters. A malformed receipt says UNKNOWN; telemetry never panics.
func ConfirmationBootLine() string {
	var r struct {
		Final        int `json:"final_validation_reject_to_pass"`
		Additional   int `json:"immediate_displacement_additional"`
		RejectToPass int `json:"validation_reject_to_pass"`
		Scenarios    int `json:"validation_scenarios"`
		Evaluated    int `json:"evaluated_scenarios"`
		Unevaluated  int `json:"unevaluated_scenarios"`
	}
	audit := "UNKNOWN (audit receipt unavailable)"
	if err := json.Unmarshal(confirmationReplayReceipt, &r); err == nil && r.Evaluated > 0 {
		audit = fmt.Sprintf("%d observations/%d scenarios (closure-only audit; evaluated=%d unevaluated=%d) · with-1m_displacement=%d (+%d audit observation)", r.RejectToPass, r.Scenarios, r.Evaluated, r.Unevaluated, r.Final, r.Additional)
	}
	closed, ordered := "off", "off"
	if confirmationClosedBuckets {
		closed = "on"
	}
	if confirmationOrderedSequence {
		ordered = "enforced"
	}
	dropped, replaced := ConfirmationTapeCounters()
	return fmt.Sprintf("🔎 confirmation: close-requires-closed-bucket=%s · sequence-order=%s · missing-reference=%s(not met) · immediate-displacement=%s · forming-bucket refusals=%d · out-of-order refusals=%d · tape out-of-order drops=%d · tape same-minute replacements=%d · validation-REJECT→PASS=%s",
		closed, ordered, confirmationUnknown, immediateDisplacementRule, formingBucketRefusals.Load(), outOfOrderRefusals.Load(), dropped, replaced, audit)
}
