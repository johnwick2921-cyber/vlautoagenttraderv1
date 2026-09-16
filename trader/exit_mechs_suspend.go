package trader

import (
	"fmt"

	"nofx/kernel"
	"os"
	"strings"
	"sync"

	ntTrader "nofx/trader/ninjatrader"
)

// ── 0B (2026-09-02) — BE+40 AND THE ATR TRAIL ARE SUSPENDED ─────────────────
//
// Both mechanisms are LIVE and FIRING (full audit 09-01: 2 breakeven moves, 8
// trail ratchets that day) with NO measurement of whether they help. Round-7
// research ranks ATR/Chandelier trails in the worst group of 15 exit families
// across 567,000 backtests, and the net effect of breakeven moves is contested.
// Our own tape shows $719.50 of giveback with ZERO trail EXITS ever recorded.
//
// The problem is not the direction of the effect — it is that unmeasured
// mechanisms are moving live stops. Suspended (not deleted) until MFE data
// exists (wave 1A). Exits while suspended: fixed stop · fixed target · EOD flat
// · the existing invalidation/dormant logic.
//
// Env EXIT_MECHS_SUSPENDED=0 re-enables both (the knobs are retained; per-
// strategy breakeven_enabled / trailing_enabled still gate them underneath).

// ExitMechSuspendedLabel is the one-line reason shown in logs and the Guide.
const ExitMechSuspendedLabel = "suspended 2026-09-02 pending MFE data (wave 1A)"

// exitMechsSuspended resolves the suspension (env EXIT_MECHS_SUSPENDED,
// default TRUE = suspended).
func exitMechsSuspended() bool {
	if v := strings.TrimSpace(os.Getenv("EXIT_MECHS_SUSPENDED")); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			return false
		}
	}
	return true
}

// moveStopWire is the LAST hop before the socket: both suspended mechanisms
// reach the broker only through this variable. Fixtures substitute a recorder
// and assert ZERO calls, so "no move_stop frame was sent" is proved at the wire
// boundary rather than from internal state.
var moveStopWire = func(nt *ntTrader.TCPTrader, side string, newStop float64) error {
	return nt.MoveStopToBreakeven(side, newStop)
}

// suspendedOnce keeps the suspension notice to one line per mechanism per
// process — loud enough to see, quiet enough not to flood a 60s monitor.
var suspendedOnce sync.Map

// exitMechSuspendedRefuse reports the suspension for `mech` and returns true
// when the caller must return WITHOUT touching the wire. A9: never a silent
// skip — the first refusal per mechanism logs, with the trigger that would
// have fired.
func (at *AutoTrader) exitMechSuspendedRefuse(mech, detail string) bool {
	if !exitMechsSuspended() {
		return false
	}
	key := at.id + ":" + mech
	if _, seen := suspendedOnce.LoadOrStore(key, struct{}{}); !seen {
		at.logWarnf("⏸ %s SUSPENDED (0B) — %s; trigger condition MET and NOT sent to the broker: %s. Exits are stop/target/EOD-flat/invalidation only. Set EXIT_MECHS_SUSPENDED=0 to restore.",
			mech, ExitMechSuspendedLabel, detail)
	}
	return true
}

// ResetExitMechSuspendNoticeForTest clears the once-per-process notice latch.
func ResetExitMechSuspendNoticeForTest() { suspendedOnce = sync.Map{} }

// ExitPolicyBootLine (D8) states the whole exit posture in one boot line.
func ExitPolicyBootLine(minSLMult, anchorMaxATR float64, contractCap int, reArmAfterSweep bool) string {
	be, trail := "off", "off"
	if !exitMechsSuspended() {
		be, trail = "on", "on"
	}
	return fmt.Sprintf("exits: stop=max(anchor+clr, %.1f×ATR5m) · anchor_max=%.1f×ATR5m · BE=%s · trail=%s · size=%d · re-arm-after-sweep=%s (0B)",
		minSLMult, anchorMaxATR, be, trail, contractCap, onOff(reArmAfterSweep))
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// ExitPolicyBootLineLive resolves every value from its resolver (A11) — never a
// file default — and renders the boot line.
func ExitPolicyBootLineLive(minSLMult float64) string {
	return ExitPolicyBootLine(minSLMult, armStopAnchorMaxATR(), stageAContractCapForBoot(), true)
}

// stageAContractCapForBoot is the resolved Stage-A ceiling for the boot line.
func stageAContractCapForBoot() int { return kernel.StageAContractCap() }
