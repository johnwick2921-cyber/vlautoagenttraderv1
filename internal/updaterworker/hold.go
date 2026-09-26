package updaterworker

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"nofx/internal/updaterwire"
	"nofx/store"
)

// ── THE census-admitted worker hold writer ──────────────────────────────────
//
// store/maintenance_hold_writers_test.go admits EXACTLY this file (by name) to
// call store.WriteMaintenanceHold / store.ClearMaintenanceHold and to reference
// store.MaintenanceHoldPath; no other file of the worker, and not
// cmd/nofx-updater, may. CTO ruling 1790258770876: "a hold that must cross a
// process boundary is a file + a reader, not a Go call" — the worker writes
// the hold file, the app engages its in-process barrier from it
// (trader.maintenanceState → Engage), the AddOn refuses entries and acks.
//
// The worker hold is always
//
//	{held:true, job_id:<ours>, since:<RFC3339 UTC>, reason:"updater job <id> → <release>", owner:"updater"}
//
// and NEVER carries withdraw_entries (owner rule: the updater never cancels
// orders — TestTheWorkerHoldNeverCarriesWithdrawEntries reads the raw bytes).
// It is written only at maintenance_held and cleared only at complete and
// rolled_back (internal/updaterjob's state table owns which state does which).

// HoldOwner is the hold's owner field for a worker hold.
const HoldOwner = "updater"

var (
	// ErrForeignHold: a hold that is not this job's is on disk (another job's,
	// the operator CLI's, or one whose owner is not the updater).
	ErrForeignHold = errors.New("updaterworker: a hold that is not this job's is present")
	// ErrCorruptHold: a present hold that cannot be read (it HOLDS; only the
	// operator's force-clear removes it).
	ErrCorruptHold = errors.New("updaterworker: the hold file is unreadable")
)

// Seams (tests only): the two store writers, so a test can observe the job
// file at the instant of the write and play a write that fails AFTER the
// rename (U1 verifier item 9). Production: exactly the store functions.
var (
	writeMaintenanceHold = store.WriteMaintenanceHold
	clearMaintenanceHold = store.ClearMaintenanceHold
)

// holdFor is the one hold value the worker ever writes (withdraw_entries is
// never set — the field is not even named here).
func holdFor(jobID, releaseID string, now time.Time) store.MaintenanceHold {
	return store.MaintenanceHold{
		Held:   true,
		JobID:  jobID,
		Since:  now.UTC().Format(time.RFC3339),
		Reason: fmt.Sprintf("updater job %s → %s", jobID, releaseID),
		Owner:  HoldOwner,
	}
}

// HoldState classifies what is on disk for jobID.
type HoldState int

const (
	HoldAbsent  HoldState = iota // no hold file (or a released held:false)
	HoldOurs                     // held, well-formed, our job id, owner updater, no withdraw
	HoldForeign                  // held by another job / owner, or asking to withdraw
	HoldCorrupt                  // present and unreadable: it HOLDS
)

func (s HoldState) String() string {
	switch s {
	case HoldAbsent:
		return "absent"
	case HoldOurs:
		return "ours"
	case HoldForeign:
		return "foreign"
	case HoldCorrupt:
		return "corrupt"
	}
	return "unknown"
}

// ReadHoldFor reads the hold and classifies it for jobID (read-only).
func ReadHoldFor(dataDir, jobID string) (HoldState, store.MaintenanceHoldState) {
	st := store.ReadMaintenanceHold(dataDir)
	switch {
	case st.Corrupt:
		return HoldCorrupt, st
	case !st.Present || !st.Held:
		return HoldAbsent, st
	case st.Hold.JobID == jobID && st.Hold.Owner == HoldOwner && !st.Hold.WithdrawEntries:
		return HoldOurs, st
	}
	return HoldForeign, st
}

// HoldForJob writes this job's hold. It reads first: our own hold already on
// disk is success with already=true (the step is idempotent for a resume); a
// corrupt or foreign hold refuses and writes nothing. The small window between
// the read and the write against the OPERATOR CLI is stated, not hidden [B]:
// the store serializes writers with a flock, but the read-decide here is not
// under it.
func HoldForJob(dataDir, jobID, releaseID string, now time.Time) (already bool, err error) {
	if !updaterwire.ValidJobID(jobID) {
		return false, fmt.Errorf("updaterworker: hold: invalid job id")
	}
	switch s, st := ReadHoldFor(dataDir, jobID); s {
	case HoldOurs:
		return true, nil
	case HoldCorrupt:
		return false, fmt.Errorf("%w: %s", ErrCorruptHold, st.Err)
	case HoldForeign:
		return false, fmt.Errorf("%w: job %q owner %q", ErrForeignHold, st.Hold.JobID, st.Hold.Owner)
	}
	return false, writeMaintenanceHold(dataDir, holdFor(jobID, releaseID, now))
}

// ReleaseJob clears this job's hold (store.ClearMaintenanceHold is job-scoped
// and idempotent when absent). Called at complete and rolled_back ONLY. It
// clears ONLY the worker's own hold (#206 review fold): the store's clear
// matches on job_id alone, so a hold the operator wrote over ours (owner cli
// and/or withdraw_entries) must not be removed by this job — refusing keeps
// the operator's hold on disk and the runner ends recovery_needed.
func ReleaseJob(dataDir, jobID string) error {
	if !updaterwire.ValidJobID(jobID) {
		return fmt.Errorf("updaterworker: release: invalid job id")
	}
	switch s, st := ReadHoldFor(dataDir, jobID); s {
	case HoldOurs, HoldAbsent:
		return clearMaintenanceHold(dataDir, jobID)
	case HoldCorrupt:
		return fmt.Errorf("%w: %s", ErrCorruptHold, st.Err)
	default:
		return fmt.Errorf("%w: job %q owner %q", ErrForeignHold, st.Hold.JobID, st.Hold.Owner)
	}
}

// HoldFileForDisplay is the hold file's path for operator text (recovery
// steps, the status line). The literal file name stays in the store.
func HoldFileForDisplay(dataDir string) string {
	return store.MaintenanceHoldPath(dataDir)
}

// holdSummary is a one-line, name-free description of a hold state.
func holdSummary(s HoldState, st store.MaintenanceHoldState) string {
	switch s {
	case HoldOurs, HoldForeign:
		return fmt.Sprintf("%s job=%s owner=%s withdraw=%t", s, st.Hold.JobID, orNA(st.Hold.Owner), st.Hold.WithdrawEntries)
	case HoldCorrupt:
		return "corrupt: " + strings.TrimSpace(st.Err)
	}
	return s.String()
}

// orNA prints an unknown value as n/a (L7), never as empty.
func orNA(s string) string {
	if s == "" {
		return "n/a"
	}
	return s
}
