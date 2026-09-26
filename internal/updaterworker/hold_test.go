package updaterworker

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"nofx/internal/updaterjob"
	"nofx/store"
)

// PIN (dispatch §4(6), behavioural): the worker's hold — read as RAW BYTES the
// moment it is written — is {held, job_id ours, since, reason, owner
// "updater"} and never carries withdraw_entries (owner rule: the updater
// never cancels orders). A hold that is not this job's (the operator CLI's,
// another job's, a corrupt file) refuses the job and is left byte-identical.
func TestTheWorkerHoldNeverCarriesWithdrawEntries(t *testing.T) {
	r := newRig(t)
	var raw []byte
	r.w.crash = func(q string) {
		if q == "maintenance_held/effect" {
			raw, _ = os.ReadFile(store.MaintenanceHoldPath(r.data))
		}
	}
	if j := r.runToEnd(t); j.State != updaterjob.StateComplete {
		t.Fatalf("job %s", j.State)
	}
	if len(raw) == 0 {
		t.Fatal("no hold file at maintenance_held")
	}
	if bytes.Contains(raw, []byte("withdraw")) {
		t.Fatalf("the worker hold carries withdraw: %s", raw)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	if len(m) != 5 || m["held"] != true || m["job_id"] != boxJobID || m["owner"] != "updater" || !strings.Contains(m["reason"].(string), boxReleaseID) {
		t.Fatalf("worker hold = %s (keys %v)", raw, keys)
	}
	if _, err := time.Parse(time.RFC3339, m["since"].(string)); err != nil {
		t.Fatalf("since %v", m["since"])
	}

	for name, foreign := range map[string]func(d string) error{
		"the operator CLI's hold": func(d string) error {
			return store.WriteMaintenanceHold(d, store.MaintenanceHold{Held: true, JobID: "ops-drill", Since: "2026-09-24T14:00:00Z", Owner: "cli", WithdrawEntries: true})
		},
		"another job's updater hold": func(d string) error {
			return store.WriteMaintenanceHold(d, store.MaintenanceHold{Held: true, JobID: "job-u4-9999abcd", Since: "2026-09-24T14:00:00Z", Owner: "updater"})
		},
		// U4 re-verify note 3 (mutants M4/M4b): OUR job id is not enough — a
		// hold carrying it is ours only with owner "updater" and no
		// withdraw_entries; taking any of these as "already held" would later
		// clear an operator's (withdrawing) hold as if it were the worker's
		"our job id, owner cli": func(d string) error {
			return store.WriteMaintenanceHold(d, store.MaintenanceHold{Held: true, JobID: boxJobID, Since: "2026-09-24T14:00:00Z", Owner: "cli"})
		},
		"our job id, owner updater, withdraw_entries": func(d string) error {
			return store.WriteMaintenanceHold(d, store.MaintenanceHold{Held: true, JobID: boxJobID, Since: "2026-09-24T14:00:00Z", Owner: "updater", WithdrawEntries: true})
		},
		"our job id, owner cli, withdraw_entries": func(d string) error {
			return store.WriteMaintenanceHold(d, store.MaintenanceHold{Held: true, JobID: boxJobID, Since: "2026-09-24T14:00:00Z", Owner: "cli", WithdrawEntries: true})
		},
		"a corrupt hold": func(d string) error {
			if err := os.MkdirAll(d+"/updater", 0o700); err != nil {
				return err
			}
			return os.WriteFile(store.MaintenanceHoldPath(d), []byte("{not json"), 0o600)
		},
	} {
		for _, when := range []string{"before preflight", "between preflight and the hold write"} {
			t.Run(name+"/"+when, func(t *testing.T) {
				r := newRig(t)
				plant := func() {
					if err := foreign(r.data); err != nil {
						t.Fatal(err)
					}
				}
				if when == "before preflight" {
					plant()
				} else {
					r.w.crash = func(q string) {
						if q == "maintenance_held/started" {
							plant()
						}
					}
				}
				r.install()
				if err := r.drive(); err != nil {
					t.Fatal(err)
				}
				j := r.job()
				if j.State != updaterjob.StateRefused {
					t.Fatalf("job %s with %s, want refused\n%v", j.State, name, states(j))
				}
				after, _ := os.ReadFile(store.MaintenanceHoldPath(r.data))
				if s, _ := ReadHoldFor(r.data, boxJobID); s == HoldOurs || s == HoldAbsent || len(after) == 0 {
					t.Fatalf("%s was replaced or removed: %s", name, after)
				}
				if r.callCount("activate")+r.callCount("backup") != 0 {
					t.Fatal("a refused job changed something")
				}
			})
		}
	}
}

// PIN (#206 review fold, P1-pending hold.go:129): a hold carrying THIS job's
// id but owner cli and/or withdraw_entries, written AFTER the hold step (the
// operator's `maintenance-hold set --job <this job> --withdraw-entries` during
// drain, gate or backup), must stop the job: the worker must not backup,
// activate (SIGKILL) or clear under a hold it does not own, and the operator's
// hold must stay on disk byte-identical. The old code never re-read the hold
// after the hold step, ran the whole machine and cleared the foreign hold by
// job id alone.
func TestAForeignHoldWrittenAfterTheHoldStepStopsTheJob(t *testing.T) {
	plants := map[string]store.MaintenanceHold{
		"our job id, owner cli": {
			Held: true, JobID: boxJobID, Since: "2026-09-24T14:00:00Z", Owner: "cli", Reason: "operator drill",
		},
		"our job id, owner updater, withdraw_entries": {
			Held: true, JobID: boxJobID, Since: "2026-09-24T14:00:00Z", Owner: "updater", Reason: "withdraw", WithdrawEntries: true,
		},
		"our job id, owner cli, withdraw_entries": {
			Held: true, JobID: boxJobID, Since: "2026-09-24T14:00:00Z", Owner: "cli", Reason: "withdraw drill", WithdrawEntries: true,
		},
	}
	for name, plant := range plants {
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			r.hookAt("gate_ok/started", func() {
				if err := store.WriteMaintenanceHold(r.data, plant); err != nil {
					t.Fatal(err)
				}
			})
			r.install()
			if err := r.drive(); err != nil {
				t.Fatal(err)
			}
			r.noViolations(t)
			j := r.job()
			if j.State != updaterjob.StateRecoveryNeeded {
				t.Fatalf("job %s after the operator overwrote the hold, want recovery_needed\n%v", j.State, states(j))
			}
			if r.callCount("activate") != 0 || r.callCount("backup") != 0 || r.callCount("hold_clear") != 0 {
				t.Fatalf("the job changed something under a foreign hold: calls %v", r.calls)
			}
			s, _ := ReadHoldFor(r.data, boxJobID)
			if s != HoldForeign {
				t.Fatalf("the operator's hold is %s — it must survive untouched", s)
			}
		})
	}
}

// PIN (U1 verifier item 9): a hold write that errs AFTER the hold landed on
// disk is recovery_needed with the hold kept — "refused" would claim nothing
// was held. A write that errs with nothing on disk is refused.
func TestFailedHoldWriteWithTheHoldOnDiskIsRecoveryNeeded(t *testing.T) {
	r := newRig(t)
	r.holdWriteLies = true
	r.install()
	if err := r.drive(); err != nil {
		t.Fatal(err)
	}
	j := r.job()
	if j.State != updaterjob.StateRecoveryNeeded || !strings.Contains(j.RecoveryReason, "this job's hold is on disk") {
		t.Fatalf("job %s (reason %q), want recovery_needed\n%v", j.State, j.RecoveryReason, states(j))
	}
	if s, _ := ReadHoldFor(r.data, boxJobID); s != HoldOurs {
		t.Fatalf("hold is %s, want this job's, kept", s)
	}

	c := newRig(t)
	c.holdWriteFails = true
	c.install()
	if err := c.drive(); err != nil {
		t.Fatal(err)
	}
	if j := c.job(); j.State != updaterjob.StateRefused {
		t.Fatalf("a hold write with nothing on disk: job %s, want refused", j.State)
	}
	if c.hold().Present {
		t.Fatal("a refused job left a hold")
	}
}
