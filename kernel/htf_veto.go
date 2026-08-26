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

// HTFVetoVerdict is the pure gate: returns (blocked, refusal message).
// action is open_long/open_short; snap is the cycle's structure snapshot;
// tf is the veto timeframe ("1h" default). Empty refusal = pass.
func HTFVetoVerdict(snap map[string]StructureState, action, tf string) (bool, string) {
	if tf == "" {
		tf = DefaultHTFVetoTF
	}
	st, ok := snap[tf]
	if !ok {
		// FAIL-OPEN (dispatch 1.4): detector-unavailable passes with WARN.
		logger.Warnf("🛡️ HTF veto SKIPPED — no %s structure snapshot this cycle (detector unavailable); entry proceeds (fail-open).", tf)
		return false, ""
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
	opposed := (side == "short" && st.Trend == "TRENDING_UP") || (side == "long" && st.Trend == "TRENDING_DOWN")
	if !opposed {
		return false, ""
	}
	ref := ""
	if st.Swing != nil {
		ref = fmt.Sprintf("%s %.2f @%s", st.Swing.Kind, st.Swing.Price, ClockCT(time.UnixMilli(st.Swing.TimeMs)))
	}
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
