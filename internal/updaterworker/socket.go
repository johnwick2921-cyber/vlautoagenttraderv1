package updaterworker

import (
	"errors"

	"nofx/internal/updaterjob"
	"nofx/internal/updaterwire"
)

// ── the socket handler (C14 as ruled: EXACTLY the four verbs) ──────────────
//
// wireserver.Serve has already refused a peer that is not this uid (per
// connection, before a byte is read), an unknown verb and every forged frame
// (DecodeRequest). Handle then authenticates by JOB: every verb that names a
// job is compared against the worker's own job file — the same-uid trust
// boundary M3 states. It answers in well under the client's 10 s: nothing
// here runs a step; the runner does, woken by signal().
//
// The resume verb is dispatched on its PAYLOAD (req.Resume != nil), which
// Validate makes equivalent to the verb: the resume census
// (internal/updaterwire/resume_census_test.go) forbids naming the resume
// builders outside the wire package and the attended CLI, and this handler
// only READS a request.

// Status answers for the worker as a whole (status with no job id).
const (
	StatusIdle    = "idle"    // no unfinished job
	StatusBusy    = "busy"    // a job is being driven
	StatusParked  = "parked"  // the job waits at nt8_updated for an attended resume
	StatusStopped = "stopped" // recovery_needed (or a persist failure): install refused until restart
)

// Handle answers one decoded, validated request.
func (w *Worker) Handle(req updaterwire.Request) updaterwire.Response {
	switch {
	case req.Status != nil:
		return w.handleStatus(req.Status.JobID)
	case req.Install != nil:
		return w.handleInstall(req.Install.ReleaseID, req.Install.JobID)
	case req.Cancel != nil:
		return w.handleCancel(req.Cancel.JobID)
	case req.Resume != nil:
		return w.handleResume(req.Resume.JobID)
	}
	w.logf("updater: refused a request with no payload")
	return refuse("rejected")
}

func ok(state string) updaterwire.Response { return updaterwire.Response{OK: true, State: state} }

func refuse(why string) updaterwire.Response { return updaterwire.Response{OK: false, Error: why} }

func (w *Worker) handleStatus(jobID string) updaterwire.Response {
	if jobID == "" {
		w.mu.Lock()
		active, stopped, running := w.active, w.stopped, w.running
		w.mu.Unlock()
		switch {
		case stopped != "":
			return ok(StatusStopped)
		case active == "":
			return ok(StatusIdle)
		case !running:
			if j, err := updaterjob.Read(w.dataDir(), active); err == nil && j.State == updaterjob.StateNT8Updated && j.Phase == updaterjob.PhaseDone {
				return ok(StatusParked)
			}
		}
		return ok(StatusBusy)
	}
	j, err := updaterjob.Read(w.dataDir(), jobID)
	switch {
	case errors.Is(err, updaterjob.ErrNotFound), errors.Is(err, updaterjob.ErrBadJobID):
		return refuse("unknown job")
	case err != nil:
		w.logf("updater: status %s: %v", jobID, err)
		return refuse("job unreadable")
	}
	return ok(string(j.State))
}

// handleInstall writes the job file (requested) and wakes the runner. The
// app has already authorized and verified (M3); the worker additionally
// requires its own verdict for the release and refuses while any job is
// unfinished or while stopped.
func (w *Worker) handleInstall(releaseID, jobID string) updaterwire.Response {
	w.mu.Lock()
	defer w.mu.Unlock()
	if existing, err := updaterjob.Read(w.dataDir(), jobID); err == nil {
		// an idempotent re-send of the SAME install answers its state
		if existing.ReleaseID == releaseID && !updaterjob.Finished(existing.State, existing.Phase) && w.active == jobID {
			return ok(string(existing.State))
		}
		return refuse("job exists")
	} else if !errors.Is(err, updaterjob.ErrNotFound) {
		w.logf("updater: install %s: existing job file: %v", jobID, err)
		return refuse("job unreadable")
	}
	if w.stopped != "" {
		return refuse("recovery needed")
	}
	if w.active != "" {
		return refuse("busy")
	}
	if _, err := w.rel.Verdict(releaseID); err != nil {
		w.logf("updater: install %s: release %s has no verdict: %v", jobID, releaseID, err)
		return refuse("release not verified")
	}
	j, err := updaterjob.New(jobID, releaseID, w.host.Now())
	if err != nil {
		return refuse("invalid job")
	}
	if err := updaterjob.Write(w.dataDir(), j); err != nil {
		w.logf("updater: install %s: %v", jobID, err)
		return refuse("job not written")
	}
	w.active = jobID
	w.signal()
	return ok(string(updaterjob.StateRequested))
}

// handleCancel: only the active job, only before maintenance_held, persisted
// under the job mutex (the runner's next transition re-reads and stops).
func (w *Worker) handleCancel(jobID string) updaterwire.Response {
	w.mu.Lock()
	defer w.mu.Unlock()
	j, err := updaterjob.Read(w.dataDir(), jobID)
	if errors.Is(err, updaterjob.ErrNotFound) || errors.Is(err, updaterjob.ErrBadJobID) {
		return refuse("unknown job")
	}
	if err != nil {
		return refuse("job unreadable")
	}
	if w.active != jobID {
		return refuse("job mismatch")
	}
	row, _ := updaterjob.Lookup(j.State)
	if !row.Cancellable {
		return refuse("past boundary")
	}
	if _, err := w.updateLocked(jobID, j.State, j.Phase, func(k *updaterjob.Job) error {
		k.Blocker = ""
		return k.Enter(updaterjob.StateCancelled, w.now(*k))
	}); err != nil {
		w.logf("updater: cancel %s: %v", jobID, err)
		return refuse("cancel not written")
	}
	w.active = ""
	return ok(string(updaterjob.StateCancelled))
}

// handleResume: only the active job, only parked at nt8_updated (done), only
// within the 30-minute rule (C21: a stale park becomes recovery_needed here,
// never resumed). The runner re-proves the AddOn before it acts.
func (w *Worker) handleResume(jobID string) updaterwire.Response {
	w.mu.Lock()
	defer w.mu.Unlock()
	// #206 note socket.go:181: a stopped worker's runner never acts — the
	// old code answered ok("resuming") while run() skips every wake. The
	// install verb refuses this already; the resume verb must too.
	if w.stopped != "" {
		return refuse("recovery needed")
	}
	j, err := updaterjob.Read(w.dataDir(), jobID)
	if errors.Is(err, updaterjob.ErrNotFound) || errors.Is(err, updaterjob.ErrBadJobID) {
		return refuse("unknown job")
	}
	if err != nil {
		return refuse("job unreadable")
	}
	if w.active != jobID {
		return refuse("job mismatch")
	}
	if j.State != updaterjob.StateNT8Updated || j.Phase != updaterjob.PhaseDone {
		return refuse("not resumable")
	}
	if updaterjob.Stale(j, w.host.Now()) {
		if _, err := w.updateLocked(jobID, j.State, j.Phase, func(k *updaterjob.Job) error {
			markRecovery(k, "stale at resume: older than 30 minutes — never resumed (C21)")
			return k.Enter(updaterjob.StateRecoveryNeeded, w.now(*k))
		}); err != nil {
			return refuse("job not written")
		}
		w.active = ""
		w.stopped = "recovery_needed: " + jobID
		return refuse("stale job recovery needed")
	}
	w.resume[jobID] = true
	w.signal()
	return ok("resuming")
}
