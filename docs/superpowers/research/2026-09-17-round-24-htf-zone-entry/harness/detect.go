//go:build r24harness

package main

// detect.go — the per-read snapshot. Replicates trader/auto_trader_planner.go
// assemblePlannerInputWithCtx's detection feed at the resolved defaults, from
// the bars that were closed at the read instant, ON THE READ'S CONTRACT ONLY
// (BarsBetweenOn semantics; a window never spans a roll — C10).
//
// Live call chain reproduced (all kernel functions called directly, none
// reimplemented):
//   kernel.DetectHTFLevels(fetch, ["D","4h","1h","15m"], "MNQ", readTime)
//   kernel.AssembleResearchLevels("backtest-zone-fade", bars1m,
//       kernel.DefaultSessionRegistry(), "MNQ", 8, readTime, 1.5, "", extras...)
//   kernel.StaleConfirmATR5m(bars1m)
//   kernel.LevelZoneInputs(raw, zoneSeries, readTime)
//   kernel.BuildLevelZones(raw, price, atr5m, inputs, kernel.ResolveZoneOptions(8), readTime)
//
// Stated divergences from the live process (each in the report's methods):
//   - owner levels (👤) not fed (they did not exist across the replay era);
//   - LevelStateProvider not installed → all-fresh (byte-identical to the
//     pre-W11b goldens, kernel/level_state_provider.go:22-27);
//   - nPOC extras fed from the COPY's session_profiles only where rows exist
//     (the store era), mirroring trader/auto_trader_dayplan.go installNakedPOCProvider.

import (
	"database/sql"
	"time"

	"nofx/kernel"
	"nofx/market"
)

type readSnapshot struct {
	IDX         int
	Day         string
	Session     string
	ReadTime    time.Time
	WinStartMs  int64
	FlatMs      int64
	Contract    string
	Bars1m      []market.Kline
	SessionBars []market.Kline
	PrevBar     *market.Kline
	Price       float64
	ATR5m       float64
	Raw         []kernel.DetectedLevel
	Inputs      map[string]kernel.ZoneWidthInput
	Delta       float64
	NPocCount   int
}

var plannerTFs = []string{"D", "4h", "1h", "15m"}

const (
	maxLevelsReplay  = 8
	proximityKReplay = 1.5
	fetch1mCount     = 2000
	fetchHTFCount    = 500
)

// buildRead assembles one read snapshot. ok=false → the read has no tape and is
// skipped (the live planner would render nothing from an empty ring).
func buildRead(d *barDB, db *sql.DB, idx int, day time.Time, session string) (*readSnapshot, bool) {
	readTime, winStart, winEnd := sessionWindow(day, session)
	if len(d.merged1m) == 0 || readTime.UnixMilli() > d.merged1m[len(d.merged1m)-1].openMs {
		return nil, false
	}
	contract, ok := d.contractAt(readTime.UnixMilli())
	if !ok {
		return nil, false
	}
	bars1m := d.lastClosed(contract, "1m", fetch1mCount, readTime)
	if len(bars1m) == 0 {
		return nil, false
	}
	prev, sessionBars := d.sessionBars1m(contract, winStart.UnixMilli(), winEnd.UnixMilli())
	if len(sessionBars) == 0 {
		return nil, false
	}

	// zoneSeries mirrors the live assembly: 1m as-is, 5m/15m AGGREGATED from the
	// 1m slice, then the HTF fetch OVERWRITES each configured tf with the real
	// series (auto_trader_planner.go:2168-2176).
	zoneSeries := map[string][]market.Kline{
		"1m":  bars1m,
		"5m":  kernel.AggregateBars(bars1m, 5*60_000),
		"15m": kernel.AggregateBars(bars1m, 15*60_000),
	}
	fetch := func(tf string, count int) []market.Kline {
		series := d.lastClosed(contract, tf, count, readTime)
		zoneSeries[tf] = series
		return series
	}
	htfLevels := kernel.DetectHTFLevels(fetch, plannerTFs, "MNQ", readTime)

	extra := append([]kernel.DetectedLevel(nil), htfLevels...)
	npoc := 0
	if pocs := npocFor(db, readTime); len(pocs) > 0 {
		extra = append(extra, kernel.NakedPOCs(pocs, bars1m, readTime)...)
		npoc = len(pocs)
	}

	reg := kernel.DefaultSessionRegistry()
	_, _, price, _, raw := kernel.AssembleResearchLevels(
		"backtest-zone-fade", bars1m, reg, "MNQ", maxLevelsReplay, readTime, proximityKReplay, "", extra...)

	atr5m := kernel.StaleConfirmATR5m(bars1m)
	inputs := kernel.LevelZoneInputs(raw, zoneSeries, readTime)

	return &readSnapshot{
		IDX: idx, Day: kernel.CMESessionDayKey(readTime), Session: session,
		ReadTime: readTime, WinStartMs: winStart.UnixMilli(), FlatMs: flatMsOf(winEnd),
		Contract: contract, Bars1m: bars1m, SessionBars: sessionBars, PrevBar: prev,
		Price: price, ATR5m: atr5m, Raw: raw, Inputs: inputs,
		Delta: kernel.MeanAbsIncrement(bars1m), NPocCount: npoc,
	}, true
}

type nPocRow struct {
	sessionDate string
	poc         float64
}

// npocFor mirrors installNakedPOCProvider's store read at a past instant: the
// newest 30 stored session profiles whose session_date <= the read's CME day.
func npocFor(db *sql.DB, readTime time.Time) []kernel.PriorPOC {
	readKey := kernel.CMESessionDayKey(readTime)
	weeklyBefore := kernel.CMESessionDayKey(readTime.AddDate(0, 0, -5))
	rows, err := db.Query(`SELECT session_date, poc FROM session_profiles
		WHERE symbol='MNQ' AND session_date<=? ORDER BY session_date DESC LIMIT 30`, readKey)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []kernel.PriorPOC
	for rows.Next() {
		var r nPocRow
		if err := rows.Scan(&r.sessionDate, &r.poc); err != nil {
			return nil
		}
		if r.poc <= 0 {
			continue
		}
		out = append(out, kernel.PriorPOC{SessionDate: r.sessionDate, POC: r.poc, Weekly: r.sessionDate < weeklyBefore})
	}
	return out
}
