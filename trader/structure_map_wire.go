package trader

import (
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── S1 — THE STRUCTURE TABLE AT THE PLANNER READ (2026-09-16) ──────────────
//
// One call per read, knob-gated (day_plan.structure_map, default OFF): the
// D/4h/1h bars the planner's own provider serves, the uncapped HTF zone pool
// the read already scored, the read's price and clock → kernel.StructureMap.
// Nil when the knob is off or no TF had bars — the doc field stays ABSENT and
// the prompt stays byte-identical.

// structureMapTFProvider maps the map's TF keys onto the bar provider's
// ("D" → "1d"); the provider's own key otherwise.
func structureMapTFProvider(tf string) string {
	if tf == "D" {
		return "1d"
	}
	return tf
}

// structureMapBarsAsk is how many bars each TF is asked for — the ring holds
// up to 2,500; the daily/4h series are far shorter than that in practice.
const structureMapBarsAsk = 2500

// structureMapEnabled resolves the knob from THE STRATEGY BOUND TO THIS TRADER.
func (at *AutoTrader) structureMapEnabled() (enabled, known bool) {
	if at == nil || at.config.StrategyConfig == nil {
		return false, false
	}
	en, _ := store.ResolveStructureMap(at.config.StrategyConfig)
	return en, true
}

// structureMapForRead computes the STRUCTURE table for one planner read, or
// nil (knob off / no provider / no bars). `now` comes from the read (A28).
func (at *AutoTrader) structureMapForRead(symbol, contract string, pool []kernel.ScoredLevel, price float64, now time.Time) *kernel.StructureMap {
	if en, _ := at.structureMapEnabled(); !en {
		return nil
	}
	if market.FuturesBarsProvider == nil {
		return nil
	}
	reader := func(tf string) []market.Kline {
		return market.FuturesBarsProvider(symbol, structureMapTFProvider(tf), structureMapBarsAsk)
	}
	return kernel.ComputeStructureMap(reader, strings.TrimSpace(contract), pool, price, now.UnixMilli(), kernel.DefaultStructureTrendSwings)
}
