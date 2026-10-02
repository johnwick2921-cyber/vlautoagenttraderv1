package updaterworker

import (
	"strings"
	"testing"
	"time"

	"vl/internal/updaterjob"
)

// F2 — the bounded planner wait (knob Budgets.PlannerWait, default 15 m).
// A live AI-plan read is a KNOWN BENIGN blocker: preflight/drain/gate wait for
// it (the job shows "waiting for the AI plan (started hh:mm:ss)" as the live
// blocker, never "not passed within"). On planner-wait EXPIRY: preflight
// refuses with NO hold (as today); drain/gate RELEASE the hold and refuse —
// a benign blocker must not leave the desk held.

func shrinkPlanner(r *rig) {
	r.w.cfg.Budgets.Poll = 10 * time.Millisecond
	r.w.cfg.Budgets.PlannerWait = 100 * time.Millisecond
}

// watchBlocker samples the REAL job file (the production persistence the poll
// writes the live blocker to) and reports whether the waiting text ever
// appeared. Enter() clears Blocker on the terminal transition, so the live
// blocker is only observable mid-wait.
func watchBlocker(t *testing.T, r *rig) (saw <-chan bool) {
	t.Helper()
	out := make(chan bool, 1)
	go func() {
		sawIt := false
		for i := 0; i < 5000; i++ {
			// r.job() Fatals on a mid-write read; read the raw file and skip
			// the moment of a writer (the watcher must never kill the test).
			if j, err := updaterjob.Read(r.data, boxJobID); err == nil &&
				strings.Contains(j.Blocker, "waiting for the AI plan (started") {
				sawIt = true
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
		out <- sawIt
	}()
	return out
}

// Preflight: the planner read outlives the wait → refused, NO hold, and the
// live blocker was persisted to the job file mid-wait.
func TestPreflightPlannerWaitExpiryRefusesNoHold(t *testing.T) {
	r := newRig(t, func(b *box) { b.plannerInFlight = true })
	shrinkPlanner(r)
	watch := watchBlocker(t, r)
	if r.runCrashing(t) {
		t.Fatal("preflight passed although the AI plan read outlived the planner wait")
	}
	j := r.job()
	if j.State != updaterjob.StateRefused {
		t.Fatalf("state=%s, want refused", j.State)
	}
	if r.hold().Present {
		t.Fatal("a planner-wait expiry at preflight must refuse with NO hold")
	}
	if !strings.Contains(j.Error, "AI plan still running after") {
		t.Fatalf("error %q must name the planner wait", j.Error)
	}
	if !<-watch {
		t.Fatal("the live blocker must read the waiting text while the job waits")
	}
}

// Drain: the planner read appears AFTER the hold — expiry must RELEASE the
// hold and refuse (nothing installed; the desk must not stay held on a benign
// blocker).
func TestDrainPlannerWaitExpiryReleasesHoldAndRefuses(t *testing.T) {
	r := newRig(t)
	shrinkPlanner(r)
	r.w.crash = func(q string) {
		if q == "drained_acked/started" {
			panic(crashPanic{q})
		}
	}
	if !r.runCrashing(t) {
		t.Fatal("drain never started")
	}
	r.w.crash = nil
	r.set(func() { r.plannerInFlight = true })
	watch := watchBlocker(t, r)
	if r.runCrashing(t) {
		t.Fatal("drain passed although the AI plan read outlived the planner wait")
	}
	j := r.job()
	if j.State != updaterjob.StateRefused {
		t.Fatalf("state=%s, want refused (planner-wait expiry releases the hold)", j.State)
	}
	if r.hold().Present {
		t.Fatal("the hold must be RELEASED on planner-wait expiry — a benign blocker never keeps the desk held")
	}
	if !strings.Contains(j.Error, "AI plan still running after") {
		t.Fatalf("error %q must name the planner wait", j.Error)
	}
	rc := mustReceipt(t, j, "release_hold")
	if rc.Evidence["reason"] != "planner wait expired" {
		t.Fatalf("release_hold receipt must name the reason: %+v", rc.Evidence)
	}
	if !<-watch {
		t.Fatal("the live blocker must read the waiting text while the drain waits")
	}
}

// Gate: same release-and-refuse semantics at the gate step.
func TestGatePlannerWaitExpiryReleasesHoldAndRefuses(t *testing.T) {
	r, _ := atDrained(t)
	shrinkPlanner(r)
	r.set(func() { r.plannerInFlight = true })
	watch := watchBlocker(t, r)
	if r.runCrashing(t) {
		t.Fatal("gate passed although the AI plan read outlived the planner wait")
	}
	j := r.job()
	if j.State != updaterjob.StateRefused {
		t.Fatalf("state=%s, want refused (planner-wait expiry releases the hold)", j.State)
	}
	if r.hold().Present {
		t.Fatal("the hold must be RELEASED on planner-wait expiry at the gate")
	}
	if !strings.Contains(j.Error, "AI plan still running after") {
		t.Fatalf("error %q must name the planner wait", j.Error)
	}
	if !<-watch {
		t.Fatal("the live blocker must read the waiting text while the gate waits")
	}
}

// The planner wait is a WAIT, not a refusal: when the read ends inside the
// budget the step passes.
func TestPlannerWaitPassesWhenTheReadEnds(t *testing.T) {
	r := newRig(t)
	shrinkPlanner(r)
	r.w.crash = func(q string) {
		if q == "drained_acked/started" {
			panic(crashPanic{q})
		}
	}
	if !r.runCrashing(t) {
		t.Fatal("drain never started")
	}
	r.w.crash = nil
	r.set(func() { r.plannerInFlight = true })
	// The read ends well inside the planner wait (100 ms).
	go func() {
		time.Sleep(40 * time.Millisecond)
		r.set(func() { r.plannerInFlight = false })
	}()
	r.w.crash = func(q string) {
		if q == "drained_acked/done" {
			panic(crashPanic{q})
		}
	}
	if !r.runCrashing(t) {
		j := r.job()
		t.Fatalf("the drain must pass once the planner read ends; state=%s error=%q", j.State, j.Error)
	}
}
