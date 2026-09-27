package updaterworker

import (
	"testing"
	"time"

	"nofx/internal/updaterjob"
)

// PIN (#206 review fold, runner.go:367): the rollback's boot-watch log was
// predicted BEFORE the kill and never re-predicted. A restart that crosses
// local midnight makes the bot log to nofx_<D+1>.log (its file is named by
// its BOOT date), so scanning only the D file turned a SUCCESSFUL rollback
// into recovery_needed even though the old build was proven running. The
// path is now re-resolved after the kill (like the activate path): a new
// file is scanned whole.
func TestRollbackReResolvesTheLogAcrossMidnight(t *testing.T) {
	r := newRig(t)
	r.watchFail[boxNew] = true
	// The Watch failure happens late in the day: the kill lands after local
	// midnight and the new boot logs to nofx_<D+1>.log.
	r.hookAt("booted/started", func() {
		now := r.clock.Now()
		midnightish := time.Date(now.Year(), now.Month(), now.Day(), 23, 58, 25, 0, now.Location())
		r.clock.Advance(midnightish.Sub(now))
	})
	j := r.runToEnd(t)
	r.noViolations(t)
	if j.State != updaterjob.StateRolledBack {
		t.Fatalf("job %s (error %q): a midnight-crossing rollback must end rolled_back\n%v", j.State, j.Error, states(j))
	}
}
