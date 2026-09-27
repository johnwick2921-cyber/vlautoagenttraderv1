package trader

import (
	"strings"
	"testing"

	"nofx/store"
)

// ── W-ONE-BUTTON M2 site 6 — Resume does not lift the hold ─────────────────
//
// POST /traders/:id/resume clears the per-trader stop_until pause
// (ResumeEntries). The maintenance hold is installation-wide and is not a
// pause: resuming a trader must leave the hold file in place, new entries must
// stay refused, and the resume log must not claim "entries resume".
func TestResumeDoesNotLiftTheMaintenanceHold(t *testing.T) {
	dir := withMaintenanceDir(t)
	at, _ := pauseTrader(t)
	setHold(t, dir, "job-resume")
	logs := captureTraderLog(t)

	at.ResumeEntries("owner")
	out := logs.String() // the resume's own lines only

	if st := store.ReadMaintenanceHold(dir); !st.Held || st.Hold.JobID != "job-resume" {
		t.Fatalf("resume must not touch the hold file: %+v", st)
	}
	rec, _, past := runDecision(at, "open_long")
	if past {
		t.Fatalf("after a resume the hold must still refuse new entries: %+v", rec)
	}
	if !strings.Contains(out, "maintenance hold") {
		t.Fatalf("a resume under a hold must say entries stay refused, log was:\n%s", out)
	}
	if strings.Contains(out, "entries resume.") {
		t.Fatalf("a resume under a hold must not claim entries resume, log was:\n%s", out)
	}
}
