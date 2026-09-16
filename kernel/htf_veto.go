package kernel

import (
	"fmt"
	"os"
	"strings"
	"time"

	"nofx/logger"
)

// G1 (regime wave, 2026-08-21) — HTF VETO: a NEW entry whose direction opposes
// the CONFIRMED higher-timeframe trend (G2 structure) is refused. Confirmed =
// the detector's 3-swing standard, so RANGING / unconfirmed NEVER vetoes.
// FAIL-OPEN: a missing snapshot (detector unavailable) logs WARN and passes —
// a veto that cannot resolve its inputs must never silence the bot.
//
// OFF = today's pre-wave behavior (Studio toggle htf_veto, default ON;
// HTF_VETO_TF env picks the veto timeframe, default 1h).

const DefaultHTFVetoTF = "1h"

// HTFVetoTF resolves the veto timeframe (env HTF_VETO_TF, default 1h).
func HTFVetoTF() string {
	if v := strings.TrimSpace(os.Getenv("HTF_VETO_TF")); v != "" {
		return v
	}
	return DefaultHTFVetoTF
}

// HTFVetoMode resolves the veto MODE (grand-audit response F3, 2026-08-28):
//   - "1h"   (default): veto on the configured veto TF alone (today's behavior)
//   - "cross": veto ONLY when 1h AND 4h both confirm the counter-trend
//   - "4h"   : veto on the 4h trend alone
//
// Evidence for cross: the 2026-08-28 autopsy replayed 7 vetoed arms — the 1h
// veto blocked 3 would-have-won entries (+$352) while 4h was RANGING at all 7
// timestamps (it would have vetoed nothing). Unknown values fall back to "1h"
// with a WARN (never fail-open to a NEW behavior silently).
func HTFVetoMode() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("HTF_VETO_MODE")))
	switch v {
	case "1h", "cross", "4h":
		return v
	case "":
		return "1h"
	default:
		logger.Warnf("🛡️ HTF veto: unknown HTF_VETO_MODE %q — falling back to \"1h\"", os.Getenv("HTF_VETO_MODE"))
		return "1h"
	}
}

// vetoOpposes is the pure trend-vs-side predicate.
func vetoOpposes(side, trend string) bool {
	return (side == "short" && trend == "TRENDING_UP") || (side == "long" && trend == "TRENDING_DOWN")
}

// vetoRef renders the swing/BOS evidence suffix for the refusal message.
func vetoRef(st StructureState) string {
	if st.Swing != nil {
		return fmt.Sprintf("%s %.2f @%s", st.Swing.Kind, st.Swing.Price,
			ClockCT(time.UnixMilli(st.Swing.TimeMs)))
	}
	return ""
}

// HTFVetoVerdict is the pure gate: returns (blocked, refusal message).
// action is open_long/open_short; snap is the cycle's structure snapshot;
// tf is the veto timeframe ("1h" default). Empty refusal = pass.
// Mode (HTFVetoMode) applies on top of tf: "cross" requires BOTH 1h and 4h to
// oppose; "4h" vetoes on the 4h trend alone. A missing snapshot for either
// TF FAILS OPEN for that TF (an unconfirmed trend never vetoes).
func HTFVetoVerdict(snap map[string]StructureState, action, tf string) (bool, string) {
	if tf == "" {
		tf = DefaultHTFVetoTF
	}
	side := ""
	switch action {
	case "open_long":
		side = "long"
	case "open_short":
		side = "short"
	default:
		return false, ""
	}

	mode := HTFVetoMode()
	if mode == "4h" {
		tf = "4h"
	}
	if mode == "cross" {
		// BOTH 1h AND 4h must confirm the counter-trend. A missing snapshot
		// fails open for that TF, so cross vetoes strictly LESS than 1h-only.
		st1, ok1 := snap["1h"]
		st4, ok4 := snap["4h"]
		if !ok1 || !ok4 || !vetoOpposes(side, st1.Trend) || !vetoOpposes(side, st4.Trend) {
			return false, ""
		}
		return true, fmt.Sprintf("htf_veto: %s vs 1h %s + 4h %s (cross-check) (%s / %s)",
			side, st1.Trend, st4.Trend, vetoRef(st1), vetoRef(st4))
	}

	st, ok := snap[tf]
	if !ok {
		// FAIL-OPEN (dispatch 1.4): detector-unavailable passes with WARN.
		logger.Warnf("🛡️ HTF veto SKIPPED — no %s structure snapshot this cycle (detector unavailable); entry proceeds (fail-open).", tf)
		return false, ""
	}
	if !vetoOpposes(side, st.Trend) {
		return false, ""
	}
	ref := vetoRef(st)
	// Prefer the latest with-trend BOS as the evidence (the dispatch's shape:
	// "htf_veto: short vs 1h TRENDING_UP (BOS 29470.25 @04:45)").
	for _, e := range st.LastEvents {
		if e.Type == "BOS" && ((st.Trend == "TRENDING_UP" && e.Dir == "up") || (st.Trend == "TRENDING_DOWN" && e.Dir == "down")) {
			ref = fmt.Sprintf("BOS-%s %.2f @%s", e.Dir, e.Price, ClockCT(time.UnixMilli(e.TimeMs)))
			break
		}
	}
	return true, fmt.Sprintf("htf_veto: %s vs %s %s (%s)", side, tf, st.Trend, ref)
}
