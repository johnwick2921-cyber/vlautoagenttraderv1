package api

import (
	"os"
	"testing"
	"time"
)

// prodCurrentPasswordFailSleep is the production value of the F9 delay seam
// (handler_user.go currentPasswordFailSleep), captured at package init —
// before TestMain swaps it.
var prodCurrentPasswordFailSleep = currentPasswordFailSleep

// TestMain: PR #200 fold F9 — no test in this binary sleeps the real 1 s
// failed-compare delay (several pins drive a wrong current_password through
// the production router). A pin that needs to SEE the delay installs its own
// recorder over this no-op.
func TestMain(m *testing.M) {
	currentPasswordFailSleep = func(time.Duration) {}
	os.Exit(m.Run())
}
