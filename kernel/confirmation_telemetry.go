package kernel

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"nofx/logger"
	"sync/atomic"
)

//go:embed confirmation_replay_receipt.json
var confirmationReplayReceipt []byte

var formingBucketRefusals atomic.Uint64
var outOfOrderRefusals atomic.Uint64

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
	return fmt.Sprintf("🔎 confirmation: close-requires-closed-bucket=%s · sequence-order=%s · missing-reference=%s(not met) · immediate-displacement=%s · forming-bucket refusals=%d · out-of-order refusals=%d · validation-REJECT→PASS=%s",
		closed, ordered, confirmationUnknown, immediateDisplacementRule, formingBucketRefusals.Load(), outOfOrderRefusals.Load(), audit)
}
