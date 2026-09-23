package trader

import (
	"fmt"
	"math"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// ── W-EXEC-TRUTH W0 (a/e) — PICTURE HTF THROUGH THE ONE ENTRY GATE ─────────
//
// Picture reached NT8 calling only the maintenance hold, its own positions
// read and its own pending rows (D8). It now runs admitEntry like every other
// producer, and its EntryGate step is a PICTURE intent:
//
//	pre-check (fail-closed, CTO Q2): entry/stop/target present and each on
//	          the correct side, ATR5m > 0, positions readable, a strategy
//	          config and a resolvable R:R floor — missing evidence refuses
//	strict    (CTO Q6): refused, visibly — Picture is not a Day Plan scenario
//	          until W5, and strict executes plan scenarios only
//	legs D, 5, 6, 7 of EntryGate: the daily force-flat trip, R:R at the WORSE
//	          of the 5m reference and the newest 1m close (Q19, the send is a
//	          market order), the 0B stop floor (min-SL × ATR5m; leg 6 refuses,
//	          composeArmStop's widening is not used — Q18), one open position
//
// The R:R floor is max(Picture's own min_rr knob, the strategy floor) (Q7).

// PictureStrictRefusal is the exact refusal Picture gets under plan_mode=strict
// until W5 — READ by the 📷 boot line and the plan card (CTO Q6).
const PictureStrictRefusal = "refused under strict until W5 (source not yet a Day Plan scenario)"

// pictureAdmission is Picture's evidence for the one entry gate.
type pictureAdmission struct {
	EntryRef    float64 // the newest completed 5m close the evaluator measured at
	LatestClose float64 // the newest 1m close (0 = unknown)
	Stop        float64
	Target      float64
	ATR5m       float64
	KnobMinRR   float64 // day_plan.picture_htf.min_rr (0 = unset)
}

// pictureRREntry is the entry price R:R is judged at: the WORSE of the 5m
// reference and the newest 1m close for this side (Q19).
func pictureRREntry(side string, p *pictureAdmission) float64 {
	e := p.EntryRef
	if p.LatestClose > 0 {
		if side == "long" {
			e = math.Max(e, p.LatestClose)
		} else {
			e = math.Min(e, p.LatestClose)
		}
	}
	return e
}

// pictureMinRR is max(knob, strategy floor); ok=false when no floor resolves.
func (at *AutoTrader) pictureMinRR(knob float64) (float64, bool) {
	if at.config.StrategyConfig == nil {
		return 0, false
	}
	floor := at.armMinRRFor(nil)
	if floor <= 0 {
		return 0, false
	}
	if knob > floor {
		return knob, true
	}
	return floor, true
}

// pictureEntryGate is the picture path's EntryGate step.
func (at *AutoTrader) pictureEntryGate(in admitIntent) (string, bool) {
	p := in.Picture
	side := in.side()
	if p == nil {
		return "entry_gate: picture evidence missing (fail-closed)", true
	}
	// Q2 — Picture's own evidence, fail-closed.
	entry := pictureRREntry(side, p)
	if entry <= 0 || p.Stop <= 0 || p.Target <= 0 {
		return fmt.Sprintf("entry_gate: picture evidence incomplete (entry=%.2f stop=%.2f target=%.2f) — fail-closed", entry, p.Stop, p.Target), true
	}
	if side == "long" && !(p.Stop < entry && entry < p.Target) || side == "short" && !(p.Target < entry && entry < p.Stop) {
		return fmt.Sprintf("entry_gate: picture geometry on the wrong side for a %s (stop=%.2f entry=%.2f target=%.2f) — fail-closed", side, p.Stop, entry, p.Target), true
	}
	if p.ATR5m <= 0 {
		return "entry_gate: picture ATR5m unavailable — the stop floor cannot be judged (fail-closed)", true
	}
	minRR, ok := at.pictureMinRR(p.KnobMinRR)
	if !ok {
		return "entry_gate: picture R:R floor unresolvable (no strategy config) — fail-closed", true
	}
	openSide := ""
	if at.store == nil {
		return "entry_gate: picture positions unreadable (no store) — fail-closed", true
	}
	opens, err := at.store.Position().GetOpenPositions(at.id)
	if err != nil {
		return "entry_gate: picture positions unreadable (" + err.Error() + ") — fail-closed", true
	}
	sym := market.Normalize(in.Symbol)
	for _, op := range opens {
		if s := positionSide(op.Side); strings.EqualFold(op.Symbol, sym) && s != "" {
			openSide = s
			break
		}
	}
	// Q6 — strict executes plan scenarios; Picture becomes one in W5.
	session := ""
	if s, ok := at.sessionRegistry(in.Now).ActiveSession(in.Now); ok {
		session = s.Name
	}
	if at.planModeFor(session) == "strict" {
		return "entry_gate: picture " + PictureStrictRefusal, true
	}
	// Legs D, 5, 6, 7 — the SAME EntryGate, with no plan context (PlanMode,
	// citation and scenario left empty so legs 0, 2, 3, 4 abstain).
	return EntryGate(EntryIntent{
		Path:             "picture",
		Action:           in.Action,
		Symbol:           in.Symbol,
		Entry:            entry,
		Stop:             p.Stop,
		Target:           p.Target,
		ATR5m:            p.ATR5m,
		MinRR:            minRR,
		MinSLMult:        kernel.MinSLATRMult(),
		OpenPositionSide: openSide,
		DailyForceFlat:   func() string { return kernel.DailyForceFlatReason(at.id) },
	})
}

// PicturePlanGateView is the plan card's Picture line (CTO Q6), READ from the
// verdicts the gate enforces: whether the mode is on for this trader, and the
// strict refusal — "" when plan mode admits Picture.
type PicturePlanGateView struct {
	Enabled bool   `json:"enabled"`
	Refusal string `json:"refusal"`
}

// PicturePlanGateAt is the plan card's read. It builds nothing: the mode is
// the resolved knob on the NT8 path (what the evaluator is built from), the
// refusal is pictureStrictVisible — the same plan-mode read pictureEntryGate
// refuses on and the 📷 boot line prints.
func (at *AutoTrader) PicturePlanGateAt(now time.Time) PicturePlanGateView {
	if at == nil {
		return PicturePlanGateView{}
	}
	return PicturePlanGateView{
		Enabled: at.exchange == "ninjatrader" && at.pictureHtfResolvedConfig().Enabled,
		Refusal: at.pictureStrictVisible(now),
	}
}

// pictureStrictVisible reports the strict refusal for the 📷 boot line and the
// plan card: "" when Picture is not refused by plan mode right now.
func (at *AutoTrader) pictureStrictVisible(now time.Time) string {
	session := ""
	if s, ok := at.sessionRegistry(now).ActiveSession(now); ok {
		session = s.Name
	}
	if at.planModeFor(session) == "strict" {
		return PictureStrictRefusal
	}
	return ""
}
