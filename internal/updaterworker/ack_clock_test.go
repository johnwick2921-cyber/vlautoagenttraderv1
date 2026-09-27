package updaterworker

// P2 #206 review fold (steps.go:286, steps.go:489-491): the gate's "two
// distinct acks" proof and boot_verify's "ack after the kill" compared the
// app's WALL-CLOCK rendering of a monotonic ack age. This box steps its clock
// (deploy/fix-wsl2-clock.sh: chrony `makestep 1 -1` plus a 10-minute `chronyc
// makestep` cron, after a measured −41 s drift), so:
//   - a ≥1 s FORWARD step between two polls that see the SAME ack moved the
//     rendered received time enough to pass the old "≥1 s apart and both ≥
//     start" test — C11 proven by a single ack;
//   - a BACKWARD step larger than the time since the kill made the new
//     process's fresh ack read as older than WatchSince — a good release
//     timed out and rolled back.
// The wallStep knob shifts ONLY the rendered received string (AgeMs stays the
// monotonic age of the same ack), which is exactly what a wall step does to
// the app's rendering.

import (
	"testing"
	"time"

	"nofx/internal/updaterjob"
)

// A forward wall step must NOT let one ack count twice: the gate passes only
// after two DISTINCT acks are observed. A new ack is visible as an age RESET
// (a strict drop in the observed AgeMs sequence); the old wall-clock compare
// passed the gate on the SAME ack — its age sequence only grows — so this
// pins the mechanism, not just the terminal state.
func TestGateNeedsTwoDistinctAcksOnOneClock(t *testing.T) {
	r := newRig(t)
	r.set(func() { r.wallStep = time.Hour }) // every rendered received +1h: the old code sees two "distinct" acks
	r.install()
	if err := r.drive(); err != nil {
		t.Fatal(err)
	}
	r.noViolations(t)
	j := r.job()
	if j.State != updaterjob.StateComplete {
		t.Fatalf("job %s: the gate must still pass on two distinct acks\n%v", j.State, states(j))
	}
	r.mu.Lock()
	ages := append([]int64(nil), r.ackAges...)
	r.mu.Unlock()
	reset := false
	for i := 1; i < len(ages); i++ {
		if ages[i] < ages[i-1] {
			reset = true // a NEW ack was observed between two reads
		}
	}
	if !reset {
		t.Fatalf("the gate passed without an age reset — one ack counted twice (ages %v)", ages)
	}
}

// A backward wall step after the kill must NOT roll back a good release: the
// boot_verify ack wait compares the monotonic age against now−since, so the
// fresh ack passes even though its rendered time reads before the kill.
func TestBootVerifyAckAfterTheKillOnOneClock(t *testing.T) {
	r := newRig(t)
	r.hookAt("booted/started", func() {
		r.set(func() { r.wallStep = -time.Hour }) // the new process's rendered received reads BEFORE the kill
	})
	j := r.runToEnd(t)
	r.noViolations(t)
	if j.State != updaterjob.StateComplete {
		t.Fatalf("job %s: the backward wall step rolled back a good release\n%v", j.State, states(j))
	}
}
