package updaterjob

import (
	"errors"
	"fmt"
)

// ErrForbiddenEdge wraps every move the state table does not allow.
var ErrForbiddenEdge = errors.New("updaterjob: forbidden transition")

// State is one row of the updater job's state machine (brief §3.3 as ruled:
// CTO 1790259689740 accepted C10 — the second "verified" is boot_verified,
// and rolling_back, cancelled and refused are added).
type State string

const (
	StateRequested       State = "requested"
	StateDownloaded      State = "downloaded"
	StateVerified        State = "verified"
	StatePreflightOK     State = "preflight_ok"
	StateMaintenanceHeld State = "maintenance_held"
	StateDrainedAcked    State = "drained_acked"
	StateGateOK          State = "gate_ok"
	StateBackupDone      State = "backup_done"
	StateNT8Skipped      State = "nt8_skipped"
	StateNT8Updated      State = "nt8_updated"
	StateActivated       State = "activated"
	StateBooted          State = "booted"
	StateBootVerified    State = "boot_verified"
	StateComplete        State = "complete"
	StateRollingBack     State = "rolling_back"
	StateRolledBack      State = "rolled_back"
	StateRecoveryNeeded  State = "recovery_needed"
	StateCancelled       State = "cancelled"
	StateRefused         State = "refused"
)

// Phase says whether a state's side effect is under way or finished. Every
// state with a side effect is persisted "started" BEFORE the effect runs and
// "done" (with its receipt) after it; a state with no side effect is entered
// "done".
type Phase string

const (
	PhaseStarted Phase = "started"
	PhaseDone    Phase = "done"
)

// Effect names the side effect a state performs. The worker (U4) owns the
// code behind each name; this package owns WHICH state performs WHICH effect,
// so a second hold write or an early hold clear is a table edit a test sees.
type Effect string

const (
	EffectNone        Effect = "none"         // the state is its own effect (the file) or terminal
	EffectDownload    Effect = "download"     // locate the verified release + re-hash every artifact
	EffectVerify      Effect = "verify"       // re-verify the signature + resolve the release dir
	EffectPreflight   Effect = "preflight"    // stage + host/app/flat/lock checks, before any hold
	EffectHold        Effect = "hold"         // the ONLY hold write (internal/updaterworker/hold.go)
	EffectDrain       Effect = "drain"        // READ the app's view until held+drained+acked+flat
	EffectGate        Effect = "gate"         // ready:true on two reads with distinct ack.received
	EffectBackup      Effect = "backup"       // DB backup + snapshot of the install halves
	EffectNT8         Effect = "nt8"          // record the AddOn decision (a read, C12)
	EffectActivate    Effect = "activate"     // install the release halves + kill by identity
	EffectWatch       Effect = "watch"        // the new process proves itself (log + health)
	EffectBootVerify  Effect = "boot_verify"  // OK boot line, health prefix, AddOn ack build
	EffectRollback    Effect = "rollback"     // restore the snapshot + watch the OLD sha
	EffectReleaseHold Effect = "release_hold" // the ONLY hold clears
)

// Row is one state's contract.
type Row struct {
	State  State
	Effect Effect
	// Success lists the states a DONE step may move to (two only after
	// backup_done: the AddOn decision). Empty ⇔ the state is terminal.
	Success []State
	// Failure is where a STARTED step that failed goes. "" ⇔ Effect is
	// EffectNone (nothing runs, so nothing can fail).
	Failure State
	// Cancellable: cancel-before-boundary may move this state to cancelled.
	// True exactly for the states before maintenance_held — nothing has been
	// held, drained or changed yet.
	Cancellable bool
	// Parks: once DONE the runner stops and waits for an attended resume.
	Parks bool
}

// table is the state machine. Order is the dispatch's order (§2), then the
// C10 additions.
var table = []Row{
	{State: StateRequested, Effect: EffectNone, Success: []State{StateDownloaded}, Cancellable: true},
	{State: StateDownloaded, Effect: EffectDownload, Success: []State{StateVerified}, Failure: StateRefused, Cancellable: true},
	{State: StateVerified, Effect: EffectVerify, Success: []State{StatePreflightOK}, Failure: StateRefused, Cancellable: true},
	{State: StatePreflightOK, Effect: EffectPreflight, Success: []State{StateMaintenanceHeld}, Failure: StateRefused, Cancellable: true},
	// The boundary. A failed hold write leaves no hold of ours, so it is still
	// "refused"; from here on a hold may exist, and no path reaches refused or
	// cancelled (both mean "nothing was held or changed").
	{State: StateMaintenanceHeld, Effect: EffectHold, Success: []State{StateDrainedAcked}, Failure: StateRefused},
	{State: StateDrainedAcked, Effect: EffectDrain, Success: []State{StateGateOK}, Failure: StateRecoveryNeeded},
	{State: StateGateOK, Effect: EffectGate, Success: []State{StateBackupDone}, Failure: StateRecoveryNeeded},
	{State: StateBackupDone, Effect: EffectBackup, Success: []State{StateNT8Skipped, StateNT8Updated}, Failure: StateRecoveryNeeded},
	{State: StateNT8Skipped, Effect: EffectNT8, Success: []State{StateActivated}, Failure: StateRecoveryNeeded},
	{State: StateNT8Updated, Effect: EffectNT8, Success: []State{StateActivated}, Failure: StateRecoveryNeeded, Parks: true},
	// The point of no return: any failure from here restores the snapshot.
	{State: StateActivated, Effect: EffectActivate, Success: []State{StateBooted}, Failure: StateRollingBack},
	{State: StateBooted, Effect: EffectWatch, Success: []State{StateBootVerified}, Failure: StateRollingBack},
	{State: StateBootVerified, Effect: EffectBootVerify, Success: []State{StateComplete}, Failure: StateRollingBack},
	{State: StateComplete, Effect: EffectReleaseHold, Failure: StateRecoveryNeeded},
	{State: StateRollingBack, Effect: EffectRollback, Success: []State{StateRolledBack}, Failure: StateRecoveryNeeded},
	{State: StateRolledBack, Effect: EffectReleaseHold, Failure: StateRecoveryNeeded},
	// recovery_needed keeps whatever hold exists: only an attended operator
	// clears it (brief §3.4.6).
	{State: StateRecoveryNeeded, Effect: EffectNone},
	{State: StateCancelled, Effect: EffectNone},
	{State: StateRefused, Effect: EffectNone},
}

var rowByState = func() map[State]Row {
	m := make(map[State]Row, len(table))
	for _, r := range table {
		m[r.State] = r
	}
	return m
}()

// AllStates lists every state, in table order.
func AllStates() []State {
	out := make([]State, 0, len(table))
	for _, r := range table {
		out = append(out, r.State)
	}
	return out
}

// Lookup returns s's row.
func Lookup(s State) (Row, bool) {
	r, ok := rowByState[s]
	if ok {
		r.Success = append([]State(nil), r.Success...)
	}
	return r, ok
}

// IsTerminal reports whether s has no success successor.
func IsTerminal(s State) bool {
	r, ok := rowByState[s]
	return ok && len(r.Success) == 0
}

// Finished reports whether a job at (s, p) is over: a terminal state whose
// effect, if any, is done. complete/started and rolled_back/started are NOT
// finished — their hold clear has not run yet, and a resume must run it.
func Finished(s State, p Phase) bool {
	return IsTerminal(s) && p == PhaseDone
}

// EntryPhase is the phase a state is entered in: started when it has a side
// effect (persisted BEFORE the effect runs), done when it has none.
func EntryPhase(s State) Phase {
	if r, ok := rowByState[s]; ok && r.Effect == EffectNone {
		return PhaseDone
	}
	return PhaseStarted
}

// CheckMove is the transition validator: nil only for an edge the table
// allows from (from, phase).
//
//   - from DONE: to one of the row's Success states;
//   - from STARTED: to the row's Failure state;
//   - from any unfinished state, either phase: to recovery_needed (the
//     30-minute rule, the attempts cap, a step budget);
//   - from a Cancellable state, either phase: to cancelled.
//
// Everything else — a skipped state, a success edge from STARTED (the step
// has not persisted its receipt), a move out of a finished job, a self-edge
// — is an error.
func CheckMove(from State, phase Phase, to State) error {
	r, ok := rowByState[from]
	if !ok {
		return fmt.Errorf("%w: unknown state %q", ErrForbiddenEdge, from)
	}
	if _, ok := rowByState[to]; !ok {
		return fmt.Errorf("%w: unknown state %q", ErrForbiddenEdge, to)
	}
	if phase != PhaseStarted && phase != PhaseDone {
		return fmt.Errorf("%w: unknown phase %q", ErrForbiddenEdge, phase)
	}
	if Finished(from, phase) {
		return fmt.Errorf("%w: %s is finished", ErrForbiddenEdge, from)
	}
	if to == StateRecoveryNeeded {
		return nil
	}
	if to == StateCancelled && r.Cancellable {
		return nil
	}
	if phase == PhaseDone {
		for _, s := range r.Success {
			if s == to {
				return nil
			}
		}
	} else if to == r.Failure {
		return nil
	}
	return fmt.Errorf("%w: %s/%s → %s", ErrForbiddenEdge, from, phase, to)
}

// CheckFinish is nil only when (s, p) is a STARTED state with a side effect —
// the only thing that may become DONE in place.
func CheckFinish(s State, p Phase) error {
	r, ok := rowByState[s]
	if !ok {
		return fmt.Errorf("%w: unknown state %q", ErrForbiddenEdge, s)
	}
	if r.Effect == EffectNone {
		return fmt.Errorf("%w: %s has no side effect to finish", ErrForbiddenEdge, s)
	}
	if p != PhaseStarted {
		return fmt.Errorf("%w: %s/%q is not started", ErrForbiddenEdge, s, p)
	}
	return nil
}
