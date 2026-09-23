package trader

import (
	"sync"
	"time"
)

// W-EXEC-TRUTH W5 D21 — THE RUN EPOCH. "Stop, reload and Day Plan OFF
// invalidate entry permission": a Picture scenario is recorded with the epoch
// of the trader run that recorded it, and the executor places it only while
// that same run is alive. A Stop clears the epoch; the next Run starts a new
// one — so a scenario recorded before a reload can never be placed after it.
//
// Keyed by trader id rather than an AutoTrader field so W5 does not touch the
// struct's Run/Stop region (W4's lifecycle work, lane 103). The executor's
// event loop marks it on start and clears it on stop (armed_event_pass.go).
var pictureRunEpochs sync.Map // trader id → int64 (UnixNano at run start)

// markPictureRunEpoch starts a new run epoch for this trader and returns it.
func (at *AutoTrader) markPictureRunEpoch(now time.Time) int64 {
	e := now.UnixNano()
	pictureRunEpochs.Store(at.id, e)
	return e
}

// pictureRunEpoch is the live run's epoch; ok=false when no run is alive.
func (at *AutoTrader) pictureRunEpoch() (int64, bool) {
	if at == nil {
		return 0, false
	}
	v, ok := pictureRunEpochs.Load(at.id)
	if !ok {
		return 0, false
	}
	e, ok := v.(int64)
	return e, ok && e != 0
}

// clearPictureRunEpoch ends the run: nothing recorded under it may be placed.
func (at *AutoTrader) clearPictureRunEpoch() {
	if at != nil {
		pictureRunEpochs.Delete(at.id)
	}
}
