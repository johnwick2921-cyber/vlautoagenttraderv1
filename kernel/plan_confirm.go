package kernel

import (
	"fmt"
	"strings"

	"nofx/market"
)

// C1 (F3) — machine evaluation of a scenario's structured confirmation, using
// the SAME acceptance machinery as plan death (A2-consistent rule semantics).

// confirmAcceptanceRule maps the confirm vocabulary onto acceptance rules.
func confirmAcceptanceRule(rule string) string {
	switch rule {
	case "15m_close":
		return "15m-close"
	case "1x5m_close":
		return "5m-close" // the A2-fixed one-close rule
	default:
		return "2x5m" // "2x5m_close"
	}
}

// ConfirmVerdict is one scenario's machine-computed confirmation state.
type ConfirmVerdict struct {
	Rule     string  `json:"rule"`
	RefPrice float64 `json:"ref_price"`
	Side     string  `json:"side"`
	Met      bool    `json:"met"`
	Detail   string  `json:"detail"` // e.g. "last 15m close 29641.00"
}

// EvaluateConfirm computes MET/NOT-MET for one confirm object over the bars
// since the plan's birth (touch-gated like plan death; windowed identically).
func EvaluateConfirm(c PlanConfirm, bars []market.Kline, sinceMs, nowMs int64) ConfirmVerdict {
	v := ConfirmVerdict{Rule: c.Rule, RefPrice: c.RefPrice, Side: c.Side}
	w := BarsSince(bars, sinceMs)
	if len(w) == 0 {
		v.Detail = "no bars yet"
		return v
	}
	if c.Rule == "touch" {
		for i := range w {
			if w[i].Low <= c.RefPrice && w[i].High >= c.RefPrice {
				v.Met = true
				v.Detail = "level touched"
				return v
			}
		}
		v.Detail = "not touched since plan birth"
		return v
	}
	rule := confirmAcceptanceRule(c.Rule)
	above := strings.EqualFold(c.Side, "above")
	// EVER-fired semantics via the sanctioned facts API (the acceptance-interval
	// guard forbids raw counting outside scenario_facts.go).
	best, need, lastClose := AcceptanceRunEver(w, rule, c.RefPrice, above)
	if lastClose == 0 && best == 0 {
		v.Detail = "no closed bars at the rule timeframe yet"
		return v
	}
	v.Met = best >= need
	v.Detail = fmt.Sprintf("last %s close %.2f (best run %d/%d closes %s %.2f since plan birth)",
		strings.TrimSuffix(c.Rule, "_close"), lastClose, best, need, c.Side, c.RefPrice)
	return v
}

// RenderConfirmLines renders the per-scenario advisory lines for the executor
// prompt: machine truth the AI reasons FROM — never a gate.
func RenderConfirmLines(doc PlanDoc, bars []market.Kline, sinceMs, nowMs int64) string {
	var b strings.Builder
	for _, s := range doc.Scenarios {
		if s.Confirm == nil {
			continue
		}
		v := EvaluateConfirm(*s.Confirm, bars, sinceMs, nowMs)
		met := "NOT MET"
		if v.Met {
			met = "MET"
		}
		b.WriteString(fmt.Sprintf("  %s confirm: %s %s %.2f — %s (%s)\n",
			s.ID, strings.ReplaceAll(v.Rule, "_", " "), v.Side, v.RefPrice, met, v.Detail))
	}
	if b.Len() == 0 {
		return ""
	}
	return "Machine-computed confirmations (advisory — you remain the judge):\n" + b.String()
}
