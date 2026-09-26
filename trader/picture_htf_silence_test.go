package trader

import (
	"testing"

	"nofx/store"
)

// DEFAULTS-SANE step 4: every silent decision point is counted, so the log
// always says why Picture did nothing. This test drives the REAL evaluator
// through a no-break evaluation — the production call site, not a rebuilt
// input. RED: drop the noteSilent call at the break verdict → counts stay 0.
func TestPictureHtfSilentWatchesAreCounted(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	res := env.eval.Evaluate("MNQ", env.now)
	if res.Stage != "watching" {
		t.Fatalf("expected a watching evaluation, got %q", res.Stage)
	}
	if env.eval.WatchReasonTotal() == 0 {
		t.Fatalf("a silent watching decision must be counted — total = 0")
	}
	if len(env.eval.WatchReasonCounts()) == 0 {
		t.Fatalf("watch reason counts must be non-empty")
	}
}
