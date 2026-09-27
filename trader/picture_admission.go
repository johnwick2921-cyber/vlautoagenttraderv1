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
//	strict    W5: NOT a refusal any more. Picture enters as a Day Plan
//	          scenario (source=picture, P<n>) and the plan's armed executor
//	          places it; under strict EntryGate leg 0 then judges it like any
//	          cited plan scenario. (W0's strict leg lived here until W5.)
//	legs D, 5, 6, 7 of EntryGate: the daily force-flat trip, R:R at the WORSE
//	          of the 5m reference and the newest 1m close (Q19), the 0B stop
//	          floor (min-SL × ATR5m; leg 6 refuses, composeArmStop's widening
//	          is not used — Q18), one open position
//
// The R:R floor is max(Picture's own min_rr knob, the strategy floor) (Q7).

// PictureRouteDayPlan is Picture's route since W5 — READ by the 📷 boot line
// (plan_gate=) and the plan card (route, rendered neutral).
const PictureRouteDayPlan = "Day Plan scenario (market_in_zone limit)"

// The two refusals Picture's source faces before anything else (admitChain,
// W0 D26 + W5 D21) — the card's refusal carries the same words.
const (
	pictureRefusalStopped    = "picture: the trader is not running"
	pictureRefusalDayPlanOff = "picture: the Day Plan master is off"
)

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
	// W5 — no strict leg here any more: under strict Picture enters as a Day
	// Plan scenario (the plan's armed executor places it, and EntryGate leg 0
	// judges the cited P-scenario there). This pre-claim check is Picture's
	// own evidence plus legs D, 5, 6, 7.
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

// PicturePlanGateView is the plan card's Picture line (CTO Q6; W5), READ from
// what the code enforces: whether the mode is on for this trader, its ROUTE
// (rendered neutral) and a REAL refusal only (rendered red) — the running /
// Day Plan checks admitChain applies to Picture's source, in its own words.
// Route is empty when the mode is off; Refusal is "" when nothing refuses.
type PicturePlanGateView struct {
	Enabled bool   `json:"enabled"`
	Route   string `json:"route,omitempty"`
	Refusal string `json:"refusal"`
}

// PicturePlanGateAt is the plan card's read. It builds nothing: the mode is
// the resolved knob on the NT8 path (what the evaluator is built from), the
// route is PictureRouteDayPlan, the refusal is pictureSourceRefusal — the same
// two questions admitChain asks a Picture intent and a Picture arm row.
func (at *AutoTrader) PicturePlanGateAt(now time.Time) PicturePlanGateView {
	if at == nil {
		return PicturePlanGateView{}
	}
	v := PicturePlanGateView{Enabled: at.exchange == "ninjatrader" && at.pictureHtfResolvedConfig().Enabled}
	if v.Enabled {
		v.Route = PictureRouteDayPlan
		v.Refusal = at.pictureSourceRefusal()
	}
	return v
}

// pictureSourceRefusal is admitChain's first two Picture refusals, READ: ""
// when the trader runs and the Day Plan master is on.
func (at *AutoTrader) pictureSourceRefusal() string {
	switch {
	case !at.runningNow():
		return pictureRefusalStopped
	case !at.dayPlanEnabled():
		return pictureRefusalDayPlanOff
	}
	return ""
}

// pictureStrictVisible is what the 📷 boot line prints after "plan_gate="
// (picture_htf_live.go prints any non-empty value verbatim). Since W5 there is
// no strict refusal to report: Picture is a Day Plan scenario source under
// every plan mode, so this returns the ROUTE and the plan mode it runs under,
// both READ — e.g. "Day Plan scenario (market_in_zone limit) · plan_mode=strict".
func (at *AutoTrader) pictureStrictVisible(now time.Time) string {
	session := ""
	if s, ok := at.sessionRegistry(now).ActiveSession(now); ok {
		session = s.Name
	}
	return PictureRouteDayPlan + " · plan_mode=" + at.planModeFor(session)
}
