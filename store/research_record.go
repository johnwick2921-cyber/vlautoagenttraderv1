package store

import (
	"nofx/researchsnapshot"
	"time"
)

func recordResearchPlacementTimeout(id int64, waited time.Duration, reason string, rows int64, err error) {
	recordResearchPlacementTimeoutAt(id, waited, reason, rows, err, time.Now())
}
func recordResearchPlacementTimeoutAt(id int64, waited time.Duration, reason string, rows int64, err error, now time.Time) {
	defer researchsnapshot.Contain("placement timeout recording")
	var failure *string
	if err != nil {
		failure = researchsnapshot.Value(err.Error())
	}
	researchsnapshot.Record("exec:placement_timeout", func() []researchsnapshot.Fact {
		f := researchsnapshot.NewFact("exec", "placement_timeout", nil, researchsnapshot.Clocks{ObservationMS: researchsnapshot.Value(now.UnixMilli()), ReceiptMS: researchsnapshot.Value(now.UnixMilli())})
		f.Set("timeout", map[string]any{"arm_id": id, "waited_ms": float64(waited) / float64(time.Millisecond), "rows_affected": rows, "write_error": failure, "slot_state": StatePlacePending})
		f.Set("reason", reason)
		return []researchsnapshot.Fact{f}
	})
}
