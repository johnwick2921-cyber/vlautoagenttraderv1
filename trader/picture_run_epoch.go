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
//
// WAVE 1b E6 — THE EPOCH BELONGS TO AN INSTANCE. A reload builds a NEW
// AutoTrader under the SAME id in the same process; when the new instance
// Runs before the old one Stops, a bare Delete(id) in the old Stop erased the
// new run's epoch (every hand-off then read "trader not running", every
// placement retired its row) — W4's pictureHtfTraders defect a second time
// (picture_htf_live.go's CompareAndDelete). So each entry names its owner:
// the READ stays id-level (the last Run wins — the value placement and
// authoring compare against), but only the owner's Stop may clear it, and the
// clear is a CompareAndDelete on the exact entry it loaded.
var pictureRunEpochs sync.Map // trader id → *pictureRunEpochEntry

type pictureRunEpochEntry struct {
	owner *AutoTrader // the instance whose Run marked it
	epoch int64       // UnixNano at run start (persisted as source_run_epoch)
}

// markPictureRunEpoch starts a new run epoch for this trader and returns it.
func (at *AutoTrader) markPictureRunEpoch(now time.Time) int64 {
	e := now.UnixNano()
	pictureRunEpochs.Store(at.id, &pictureRunEpochEntry{owner: at, epoch: e})
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
	ent, ok := v.(*pictureRunEpochEntry)
	if !ok || ent == nil || ent.epoch == 0 {
		return 0, false
	}
	return ent.epoch, true
}

// clearPictureRunEpoch ends THIS instance's run: nothing recorded under it may
// be placed. A live epoch owned by another instance under the same id (a
// reload's successor that Ran first) is left alone (E6).
func (at *AutoTrader) clearPictureRunEpoch() {
	if at == nil {
		return
	}
	v, ok := pictureRunEpochs.Load(at.id)
	if !ok {
		return
	}
	if ent, _ := v.(*pictureRunEpochEntry); ent != nil && ent.owner != at {
		return // the successor's run — not ours to end
	}
	pictureRunEpochs.CompareAndDelete(at.id, v)
}

// pictureOtherInstanceEpoch is the live run epoch under this trader id when
// it is owned by ANOTHER AutoTrader instance; ok=false when no run is live or
// the live run is this instance's own. It says nothing about which instance
// is newer. Its one caller, Stop's invalidatePictureRows("stopped"), in
// practice meets only a SUCCESSOR's run: AutoTrader.Stop returns early unless
// the instance isRunning, and Run marks this instance's epoch right after
// setting isRunning, so another owner found at Stop time Ran after it (the
// last Run wins). The one exception is a Stop landing between Run's
// isRunning=true and its epoch mark: it would find a still-live predecessor's
// run and leave that run's rows to the predecessor's own Stop.
func (at *AutoTrader) pictureOtherInstanceEpoch() (int64, bool) {
	if at == nil {
		return 0, false
	}
	v, ok := pictureRunEpochs.Load(at.id)
	if !ok {
		return 0, false
	}
	ent, _ := v.(*pictureRunEpochEntry)
	if ent == nil || ent.owner == at || ent.epoch == 0 {
		return 0, false
	}
	return ent.epoch, true
}

// startPictureRun starts the armed event loop — which marks the run epoch —
// BEFORE registering for Picture's live bars (W5 R2, CTO review round 1). A
// frame queued in the live sink can reach the hand-off the instant the trader
// is registered; registered first, it met no live epoch and was durably
// refused ("trader not running") while the trader was in fact starting.
func (at *AutoTrader) startPictureRun() { at.startPictureRunWith(at.registerPictureHtf) }

// startPictureRunWith is startPictureRun with the registration injected, so a
// test can deliver a frame at the exact moment of registration.
func (at *AutoTrader) startPictureRunWith(register func()) {
	at.startArmedEventLoop() // marks the run epoch
	register()
}
