// ── W2 D4 — THE SURFACE: the chip's data and the counter, READ live ─────────
package trader

import (
	"fmt"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// FadeLabelView is what a scenario chip renders. Label is one of
// "permitted" | "excluded" | "not evaluated"; an excluded label carries the
// exclusion names and, for each, what was measured against what — so the chip
// can say `fade: excluded — or_wide 1.9× (k=1.28)` rather than a bare word.
type FadeLabelView struct {
	Label      string            `json:"label"`
	Exclusions []string          `json:"exclusions,omitempty"`
	Unknown    []string          `json:"unknown,omitempty"`
	Detail     map[string]string `json:"detail,omitempty"`
	AtMs       int64             `json:"at_ms"`
}

// FadeCounterView is the strip's `fade permitted N / excluded M / not-evaluated
// K today`, READ from the episode table for the CME session-day.
type FadeCounterView struct {
	Permitted    int64 `json:"permitted"`
	Excluded     int64 `json:"excluded"`
	NotEvaluated int64 `json:"not_evaluated"`
	Readable     bool  `json:"readable"`
}

// FadeLabelsFor evaluates the label for every scenario in doc at `now`. This
// is the LIVE reading for the card and the strip; the durable one is the
// per-episode stamp fixed at each episode's open. The two answer different
// questions and are labelled as such on the card.
func (at *AutoTrader) FadeLabelsFor(now time.Time, doc *kernel.PlanDoc, price float64, seated []kernel.ScoredLevel) map[string]FadeLabelView {
	out := map[string]FadeLabelView{}
	if at == nil || doc == nil {
		return out
	}
	for _, sc := range doc.Scenarios {
		facts := at.fadeFactsAt(now, at.futuresSymbol(), price, seated, strings.ToLower(sc.Direction))
		v := kernel.FadePermissionAt(now, facts)
		out[sc.ID] = fadeLabelView(v, now)
	}
	return out
}

func fadeLabelView(v kernel.FadeVerdict, now time.Time) FadeLabelView {
	lv := FadeLabelView{AtMs: now.UnixMilli(), Unknown: v.Unknown}
	switch {
	case !v.Evaluated:
		lv.Label = "not evaluated"
	case v.Permitted:
		lv.Label = "permitted"
	default:
		lv.Label = "excluded"
		lv.Detail = map[string]string{}
		for _, ex := range v.Exclusions {
			lv.Exclusions = append(lv.Exclusions, ex.Name)
			switch ex.Name {
			case kernel.FadeExOpeningRangeWide:
				lv.Detail[ex.Name] = fmt.Sprintf("OR %.2f vs %.2fx median (n=%d)", ex.Measured, ex.Threshold, ex.SampleN)
			case kernel.FadeExIBBrokenHeld:
				lv.Detail[ex.Name] = fmt.Sprintf("price %.2f held beyond IB %.2f", ex.Measured, ex.Threshold)
			case kernel.FadeExBeyondMap:
				lv.Detail[ex.Name] = fmt.Sprintf("price %.2f past last seated %.2f", ex.Measured, ex.Threshold)
			case kernel.FadeExFirstNMinutes:
				lv.Detail[ex.Name] = fmt.Sprintf("%.0f min into session (< %.0f)", ex.Measured, ex.Threshold)
			default:
				lv.Detail[ex.Name] = ex.Basis
			}
		}
	}
	return lv
}

// FadeCounterToday READS the session-day's label distribution.
func (at *AutoTrader) FadeCounterToday(now time.Time) FadeCounterView {
	if at == nil || at.store == nil {
		return FadeCounterView{}
	}
	p, e, n, err := at.store.TouchOutcomes().CountFadeLabels(at.id, kernel.CMESessionDayStart(now).UnixMilli())
	if err != nil {
		return FadeCounterView{}
	}
	return FadeCounterView{Permitted: p, Excluded: e, NotEvaluated: n, Readable: true}
}

// FadeCounterText is the strip's one line for the counter, with an explicit
// "table unreadable" instead of three zeros (A24).
func (c FadeCounterView) Text() string {
	if !c.Readable {
		return "fade permitted=n/a excluded=n/a not-evaluated=n/a (table unreadable)"
	}
	return fmt.Sprintf("fade permitted %d / excluded %d / not-evaluated %d today", c.Permitted, c.Excluded, c.NotEvaluated)
}

// fadeChipText renders one scenario's label for the strip.
func fadeChipText(v FadeLabelView) string {
	switch v.Label {
	case "permitted":
		return "fade: permitted"
	case "excluded":
		parts := make([]string, 0, len(v.Exclusions))
		for _, n := range v.Exclusions {
			if d, ok := v.Detail[n]; ok {
				parts = append(parts, n+" "+d)
			} else {
				parts = append(parts, n)
			}
		}
		return "fade: excluded — " + strings.Join(parts, "; ")
	default:
		return "fade: not evaluated"
	}
}

// lastPriceForDesk is the last 1m close, the same read feedwatch and the
// wake-levels path use. Zero when no bar exists — the predicate treats zero
// as "price unknown" and (b)/(c) stay unevaluated rather than firing on 0.
func (at *AutoTrader) LastPriceForDesk() float64 {
	if at == nil || market.FuturesBarsProvider == nil {
		return 0
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), "1m", 1)
	if len(bars) == 0 {
		return 0
	}
	return bars[len(bars)-1].Close
}
