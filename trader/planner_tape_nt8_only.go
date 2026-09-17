package trader

import (
	"fmt"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// ── THE PLANNER TAPE IS NT8-ONLY ────────────────────────────────────────────
// (CTO ruling under the owner's delegation, 2026-09-16, follow-up to dispatch 101)
//
// Every planner door reads NT8's own rows only (live, or replay verified on
// the live scale): the 12,000-bar 1m candle tape (storeBarReader), the weekly
// reader's weeks and Own1m window (auto_trader_weekly.go), the POC-touch
// historical leg (auto_trader_dayplan.go), the exported planner seam
// (BarsWithStoreDepth). The CHART keeps historical_import rows, labelled
// (owner: "i want fuull data"). Measured 2026-09-16: 426 imported 12-26 1m
// bars sat inside the planner's tape. Removing them moves what RVBaseline and
// the levels are fed (class 82), so the boot line below prints the tape and
// the baseline BOTH ways, measured through the production estimator.

// PlannerTapeBootLine renders the class-82 accounting. nt8Tape is the tape
// the planner will actually use; withImportsTape is the same read through the
// shared reader, measured here only to name what moved. Uncomputed → UNKNOWN.
func PlannerTapeBootLine(symbol, tf, contract string, importRows int64, nt8Tape, withImportsTape, ring5m []market.Kline, maxDays int, now time.Time) string {
	nt8 := ResolveRVBaselineTape(nt8Tape, ring5m, maxDays)
	with := ResolveRVBaselineTape(withImportsTape, ring5m, maxDays)
	bl := func(t RVBaselineTape) string {
		if !t.OK {
			return "UNKNOWN"
		}
		return fmt.Sprintf("%.6f over %d day(s)", t.Baseline, t.Days)
	}
	delta := "UNKNOWN"
	if nt8.OK && with.OK {
		delta = fmt.Sprintf("%+.6f", nt8.Baseline-with.Baseline)
	}
	return fmt.Sprintf(
		"🧮 planner tape [NT8-only, CTO ruling 2026-09-16] @%s: %s %s contract=%s · import rows on contract=%d (excluded from every planner door) · tape NT8-only=%d rows vs with imports=%d rows (Δ%+d) · regime baseline NT8-only=%s vs with imports=%s (Δ%s) · chart keeps imports, labelled",
		kernel.ClockCTSeconds(now), symbol, tf, contract, importRows, len(nt8Tape), len(withImportsTape), len(nt8Tape)-len(withImportsTape), bl(nt8), bl(with), delta)
}

// logPlannerTapeAccounting measures both tapes at boot and prints the line.
// The shared reader is used HERE for the census only — the rows it returns
// never reach a planner; TestNoPlannerDoorReadsTheSharedBarReader names this
// file as the single measurement site.
func (at *AutoTrader) logPlannerTapeAccounting(symbol string, ring5m []market.Kline, now time.Time) {
	if at == nil || at.store == nil || at.store.BarHistory() == nil {
		return
	}
	contract, _ := at.currentContract(symbol)
	if contract == "" {
		at.logWarnf("🧮 planner tape [NT8-only]: %s 1m — no contract named yet; accounting skipped (A24: unknown is not zero)", symbol)
		return
	}
	bh := at.store.BarHistory()
	imports, err := bh.ImportRowsOn(symbol, "1m", contract)
	if err != nil {
		at.logWarnf("🧮 planner tape [NT8-only]: import census failed for %s 1m %s: %v", symbol, contract, err)
		return
	}
	nt8 := at.barsWithStoreDepth(symbol, "1m", plannerCandleTapeBars, now)
	var with []market.Kline
	if rows, err := bh.LastNBarsOn(symbol, "1m", contract, plannerCandleTapeBars); err == nil { // census only
		var ring []market.Kline
		if market.FuturesBarsProvider != nil {
			ring = market.FuturesBarsProvider(symbol, "1m", plannerCandleTapeBars)
		}
		with = barsWithStoreDepthFrom(ring, func(int) ([]market.Kline, error) { return storeRowsToKlines(rows, "1m"), nil }, symbol, "1m", plannerCandleTapeBars, now)
	}
	at.logInfof("%s", PlannerTapeBootLine(symbol, "1m", contract, imports, nt8, with, ring5m, rvBaselineMaxDays, now))
}
