package trader

import (
	"fmt"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/store"
)

func (at *AutoTrader) deskScenarioEconomics(now time.Time) DeskLine {
	const source = "versioned plan document; authored economics, not broker prices"
	p := kernel.ActivePlanFor(at.id, at.futuresSymbol())
	if p == nil || p.BirthMs <= 0 {
		return deskUnknown(14, "scenarios", "SCENARIOS", source, "no dated active plan")
	}
	lines := make([]string, 0, len(p.Doc.Scenarios))
	unknown := false
	// W2 — every scenario carries its fade label on the strip (D4). The
	// live reading at `now`; the durable per-episode stamp is fixed at open.
	fade := at.FadeLabelsFor(now, &p.Doc, at.LastPriceForDesk(), nil)
	// ONE SETUP (dispatch 102) — the seam's RECORDED verdict per scenario, read
	// from the scenario record it wrote; never re-evaluated here (A15: the strip
	// shows what the seam decided).
	var osRec *store.OneSetupRecord
	if at.store != nil {
		osRec = at.store.GetOneSetupRecord(at.id, p.PlanID, p.Version)
	}
	for _, s := range p.Doc.Scenarios {
		lines = append(lines, kernel.EconomicsSummary(s)+" · "+fadeChipText(fade[s.ID])+" · "+oneSetupChipText(osRec, s.ID, at.oneSetupConfig(nil).Enabled))
		v := kernel.EconomicsFor(s)
		unknown = unknown || v.ObstacleR == nil || v.ArmR == nil
	}
	lines = append(lines, at.FadeCounterToday(now).Text())
	if len(lines) == 0 {
		return deskUnknown(14, "scenarios", "SCENARIOS", source, "no authored scenarios")
	}
	line := DeskLine{N: 14, Key: "scenarios", Label: "SCENARIOS", Source: source, AsOfMs: p.BirthMs, State: "ok", Verified: !unknown, Text: fmt.Sprintf("%s v%d · %s", p.Session, p.Version, strings.Join(lines, " | "))}
	if unknown {
		line.State = "unknown"
		line.Reason = "legacy or missing economics remains UNKNOWN"
	}
	if now.UnixMilli() < p.BirthMs {
		line.State = "unknown"
		line.Verified = false
		line.Reason = "plan timestamp is in the future"
	}
	return line
}

// oneSetupChipText renders one scenario's recorded one-setup verdict for the
// strip. Absent record → "not evaluated" (or "off"), never "allowed" (A24).
func oneSetupChipText(rec *store.OneSetupRecord, scID string, enabled bool) string {
	if !enabled {
		return "one-setup: off"
	}
	if rec == nil {
		return "one-setup: not evaluated"
	}
	v, ok := rec.Scenarios[scID]
	if !ok {
		return "one-setup: not evaluated"
	}
	switch {
	case v.Allowed && v.Waiting:
		return "one-setup: allowed, second_setup_waiting"
	case v.Allowed:
		t := v.Target
		if t == "" {
			t = "authored"
		}
		return "one-setup: ALLOWED · target=" + t
	default:
		return "one-setup: declined — level=" + v.Level + " play=" + v.Play + " permission=" + v.Permission
	}
}
