package trader

import (
	"encoding/json"
	"math"
	"sync"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// P1.3 — day-plan durable session-profile snapshot + nPOC provider wiring.
//
// snapshotSessionProfiles runs each cycle but is GATED (futures + day_plan
// enabled) so it is DORMANT by default — crypto and plan-off traders are a
// no-op. When active it persists any newly-completed (frozen) session's volume
// profile, idempotently (SaveIfAbsent), so the map warms FORWARD and a restart
// replays with no loss and no dupes. It also installs the nPOC provider (once)
// so BuildKeyLevelsBlock can surface untested prior-session POCs.

var nakedPOCProviderOnce sync.Once

func (at *AutoTrader) snapshotSessionProfiles() {
	if at.exchange != "ninjatrader" {
		return // futures only
	}
	dp := at.config.StrategyConfig.DayPlan
	if dp == nil || !dp.PlanEnabled {
		return // dormant until the owner enables day_plan
	}
	if market.FuturesBarsProvider == nil {
		return
	}
	symbol := at.config.NinjaTraderSymbol
	if symbol == "" {
		symbol = "MNQ"
	}

	// Install the nPOC + level-state providers once (store-backed, symbol-scoped
	// market facts — shared across traders by design). The ACTIVE-PLAN +
	// SESSION-REGISTRY providers are PER-TRADER (P0-A): re-registered
	// idempotently every cycle under THIS trader's id, never a process-global
	// singleton closing over whoever arrived first.
	st := at.store
	nakedPOCProviderOnce.Do(func() {
		installNakedPOCProvider(st)
		installLevelStateProvider(st) // W11b — surface persisted freshness/consumed
	})
	installActivePlanProvider(at, st)

	bars := market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount)
	if len(bars) == 0 {
		return
	}
	prof := kernel.BuildSVPProfile(bars, time.Now())
	for _, s := range prof.Sessions {
		if !s.Frozen || len(s.Bins) == 0 {
			continue // only completed sessions with data
		}
		exists, err := at.store.SessionProfile().Exists(symbol, s.Date)
		if err != nil || exists {
			continue
		}
		hi, lo := sessionHiLoFromBins(s.Bins)
		pj, _ := json.Marshal(s)
		wrote, err := at.store.SessionProfile().SaveIfAbsent(&store.SessionProfileDB{
			Symbol:      symbol,
			SessionDate: s.Date,
			POC:         s.POC,
			VAH:         s.VAH,
			VAL:         s.VAL,
			SessHigh:    hi,
			SessLow:     lo,
			ProfileJSON: string(pj),
			CreatedAt:   time.Now().UnixMilli(),
		})
		if err != nil {
			at.logWarnf("🗺️ day-plan: session-profile snapshot failed for %s %s: %v", symbol, s.Date, err)
			continue
		}
		if wrote {
			at.logInfof("🗺️ day-plan: stored session profile %s %s (POC %.2f) — warming forward.", symbol, s.Date, s.POC)
		}
	}
}

func sessionHiLoFromBins(bins []kernel.SVPBin) (hi, lo float64) {
	hi, lo = math.Inf(-1), math.Inf(1)
	for _, b := range bins {
		if b.Price > hi {
			hi = b.Price
		}
		if b.Price < lo {
			lo = b.Price
		}
	}
	if math.IsInf(hi, 1) {
		hi = 0
	}
	if math.IsInf(lo, 1) {
		lo = 0
	}
	return hi, lo
}

// installLevelStateProvider wires kernel.LevelStateProvider to read this store's
// cross-session level_state (freshness A→B→C, consumed) for a level identity — the
// SAME identity (type-from-label + price-bin) W7's writer uses. This is the W11b
// surfacing: the executor's KEY LEVELS drop consumed levels (freshMult 0) and show
// tested/B; PLAN STATUS annotates burned levels. Unknown level → "" (fresh).
func installLevelStateProvider(st *store.Store) {
	kernel.LevelStateProvider = func(symbol string, l kernel.DetectedLevel) string {
		key := store.MakeLevelKey(symbol, kernel.LevelTypeFromLabel(l.Label), "", kernel.LevelBinIndex(l.Price))
		cur, err := st.LevelState().Get(key)
		if err != nil || cur == nil {
			return "" // no persisted state → fresh (pre-W11b behavior)
		}
		if cur.Consumed || cur.Freshness == store.FreshnessDone {
			return "done"
		}
		return cur.Freshness // "A" | "B" | "C"
	}
}

// installNakedPOCProvider wires kernel.NakedPOCProvider to read this store's
// session_profiles and compute the untested (naked) POCs for a symbol.
func installNakedPOCProvider(st *store.Store) {
	kernel.NakedPOCProvider = func(symbol string) []kernel.DetectedLevel {
		rows, err := st.SessionProfile().List(symbol, 30)
		if err != nil || len(rows) == 0 {
			return nil
		}
		now := time.Now()
		// Sessions older than ~5 CME days rank as weekly (HTF grade bump).
		weeklyBefore := kernel.CMESessionDayKey(now.AddDate(0, 0, -5))
		pocs := make([]kernel.PriorPOC, 0, len(rows))
		for _, r := range rows {
			pocs = append(pocs, kernel.PriorPOC{
				SessionDate: r.SessionDate,
				POC:         r.POC,
				Weekly:      r.SessionDate < weeklyBefore,
			})
		}
		var bars []market.Kline
		if market.FuturesBarsProvider != nil {
			bars = market.FuturesBarsProvider(symbol, kernel.AISVPBarInterval, kernel.AISVPBarCount)
		}
		return kernel.NakedPOCs(pocs, bars, now)
	}
}
