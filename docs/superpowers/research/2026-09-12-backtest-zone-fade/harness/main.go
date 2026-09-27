package main

// main.go — BACKTEST 1 orchestration: does the fade at a zone pay?
// Read-only. No trading-path code, no boot, no orders, no writes to the live
// store (a COPY in the worktree is read; the live DB is never opened).

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
)

var (
	flagDB    = flag.String("db", "data/backtest.db", "path to the DB COPY (worktree)")
	flagOut   = flag.String("out", "data", "output dir")
	flagLimit = flag.Int("limit", 0, "limit reads (0 = all) — smoke runs")
	flagPerms = flag.Int("perms", 2000, "permutation count for Westfall–Young max-T")
)

const heldOutStart = "2025-09-12"

func unixMsTime(ms int64) time.Time { return time.UnixMilli(ms) }

func main() {
	flag.Parse()
	if err := verifyPort(); err != nil {
		panic(err)
	}
	os.MkdirAll(*flagOut, 0o755)

	db, err := openCopy(*flagDB)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// C10(a) — every series is single-contract by construction; the queries
	// below return contract-stamped rows only and the harness never concatenates
	// two contracts. Assert the import census matches the wave-101 D5 table.
	d, err := loadBars(db, []string{"1m", "15m", "1h", "4h", "1d"})
	if err != nil {
		panic(err)
	}
	seam := seamChecks(db, d)
	writeJSON(*flagOut+"/seam.json", seam)
	fmt.Printf("seam: %s\n", seam.Summary)

	// Stream reads: each snapshot is built, evaluated, and DROPPED — the full
	// corpus of snapshots would be ~1GB in memory and no statistic needs it.
	stats := &runStats{}
	var events []touchRecord
	surf := &surfaceEvents{}
	eventSeq := 0
	runStream(db, d, stats, func(r *readSnapshot) {
		ctx1m, ctx5m := touchContexts(r)
		baseOpts := kernel.ResolveZoneOptions(maxLevelsReplay)
		// PRIMARY — default options (the live map).
		m := buildZoneMap(r.Raw, r.Price, r.ATR5m, r.Inputs, baseOpts, r.ReadTime.UnixMilli())
		for _, rec := range evalMap(r, m, ctx1m, ctx5m, "zone") {
			rec.EventID = fmt.Sprintf("ev-%07d", eventSeq)
			eventSeq++
			events = append(events, rec)
		}
		for _, rec := range evalLine(r, m, ctx1m, ctx5m) {
			rec.EventID = fmt.Sprintf("ev-%07d", eventSeq)
			eventSeq++
			events = append(events, rec)
		}
		// SWEEP — 27 map cells, compact, SEATED zones only (shortlisted — the
		// map's own seat flag; rank < ShortlistCap).
		for mi := 0; mi < sweepMaps; mi++ {
			om := sweepOpts(mi, baseOpts)
			mm := buildZoneMap(r.Raw, r.Price, r.ATR5m, r.Inputs, om, r.ReadTime.UnixMilli())
			for _, rec := range evalMap(r, mm, ctx1m, ctx5m, "zone") {
				if !rec.Shortlist {
					continue
				}
				ce := cellEvent{era: eraOf(r.ReadTime)}
				for s := 0; s < 3; s++ {
					ce.net[s] = float32(rec.Trades[s][0].NetPts)
				}
				ce.hold[0], ce.valid[0] = holdCode(rec.ZoneH12)
				ce.hold[1], ce.valid[1] = holdCode(rec.Zone5mH10)
				ce.hold[2], ce.valid[2] = holdCode(rec.Zone5mH20)
				surf.add(mi, ce)
			}
		}
	})
	fmt.Printf("reads built=%d skipped=%d (in=%d out=%d)\n", stats.built, stats.skipped, stats.inSample, stats.heldOut)
	saveAll(events, surf, seam, stats)
}

type runStats struct {
	built, skipped, inSample, heldOut int
}

// runStream walks every CME day with tape, builds the three session reads
// (LONDON 01:30, NY 08:00, ASIA 16:30 CT) and hands each to fn.
func runStream(db *sql.DB, d *barDB, stats *runStats, fn func(*readSnapshot)) {
	loc := kernel.CTLocation()
	first := unixMsTime(d.merged1m[0].openMs).In(loc)
	last := unixMsTime(d.merged1m[len(d.merged1m)-1].openMs).In(loc)
	firstDay := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc)
	lastDay := time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, loc)
	idx := 0
outer:
	for day := firstDay; !day.After(lastDay); day = day.AddDate(0, 0, 1) {
		for _, session := range []string{"LONDON", "NY", "ASIA"} {
			readTime, _, _ := sessionWindow(day, session)
			if readTime.UnixMilli() > last.UnixMilli() {
				continue
			}
			r, ok := buildRead(d, db, idx, day, session)
			idx++
			if !ok {
				stats.skipped++
				continue
			}
			stats.built++
			if eraOf(r.ReadTime) == 1 {
				stats.heldOut++
			} else {
				stats.inSample++
			}
			fn(r)
			if *flagLimit > 0 && stats.built >= *flagLimit {
				break outer
			}
		}
	}
}

func heldOutStartMs() int64 {
	loc := kernel.CTLocation()
	t, _ := time.ParseInLocation("2006-01-02", heldOutStart, loc)
	return t.UnixMilli()
}

func eraOf(t time.Time) byte {
	if t.UnixMilli() >= heldOutStartMs() {
		return 1
	}
	return 0
}

func holdCode(outcome string) (int8, bool) {
	switch outcome {
	case "hold":
		return 1, true
	case "break":
		return 0, true
	default:
		return 2, false
	}
}

// touchContexts builds the 1m and 5m context series for one read: the last bar
// before the session window (touch context) + the session bars.
func touchContexts(r *readSnapshot) (ctx1m, ctx5m []market.Kline) {
	ctx1m = append(ctx1m, r.SessionBars...)
	if r.PrevBar != nil {
		ctx1m = append([]market.Kline{*r.PrevBar}, ctx1m...)
	}
	agg := kernel.AggregateBars(r.Bars1m, 5*60_000)
	ctx5m = kernel.AggregateBars(r.SessionBars, 5*60_000)
	for i := len(agg) - 1; i >= 0; i-- {
		if agg[i].OpenTime < r.WinStartMs {
			ctx5m = append([]market.Kline{agg[i]}, ctx5m...)
			break
		}
	}
	return ctx1m, ctx5m
}

// evalMap evaluates every complete-width zone of one map against the session
// tape and simulates the fade trades.
func evalMap(r *readSnapshot, m kernel.LevelZoneMap, ctx1m, ctx5m []market.Kline, band string) []touchRecord {
	flatClose := r.SessionBars[len(r.SessionBars)-1].Close
	var out []touchRecord
	for zi := range m.Zones {
		z := &m.Zones[zi]
		if z.Lo == nil || z.Hi == nil || z.Anchor <= 0 {
			continue
		}
		lo, hi := *z.Lo, *z.Hi
		if hi <= lo {
			continue
		}
		j := firstTouchIdx(ctx1m, lo, hi)
		if j < 1 {
			continue
		}
		rec := touchRecord{
			ReadIdx: r.IDX, ZoneIdx: zi, Era: eraStr(eraOf(r.ReadTime)),
			Day: r.Day, Session: r.Session, Contract: r.Contract,
			Year: r.ReadTime.Year(), Anchor: z.Anchor, Lo: lo, Hi: hi,
			WidthATR: 0, Broad: z.Broad, Shortlist: z.Shortlisted,
			Families: len(z.Families), FamilyCap: m.Options.FamilyCap,
			Sources: len(z.Sources), ATR5m: r.ATR5m, TouchBarMs: ctx1m[j].OpenTime,
			BandType: band,
		}
		if r.ATR5m > 0 {
			rec.WidthATR = (hi - lo) / r.ATR5m
		}
		if len(z.Sources) > 0 {
			rec.SrcTF = z.Sources[0].TF
			rec.SrcKind = string(z.Sources[0].Kind)
			rec.SrcLabel = z.Sources[0].Label
			tfs := map[string]bool{}
			for _, s := range z.Sources {
				tfs[s.TF] = true
			}
			rec.CrossTF = len(tfs) > 1
		}
		if ctx1m[j-1].Close < lo {
			rec.Approach = "below"
		} else {
			rec.Approach = "above"
		}
		side := "long"
		if rec.Approach == "below" {
			side = "short"
		}
		zo, _, _, _ := bandOutcome(ctx1m, j, lo, hi, z.Anchor, horizonLive, "close")
		rec.ZoneH12 = zo
		if j5 := firstTouchIdx(ctx5m, lo, hi); j5 >= 1 {
			zo10, _, _, _ := bandOutcome(ctx5m, j5, lo, hi, z.Anchor, 10, "close")
			zo20, _, _, _ := bandOutcome(ctx5m, j5, lo, hi, z.Anchor, 20, "close")
			rec.Zone5mH10, rec.Zone5mH20 = zo10, zo20
		} else {
			rec.Zone5mH10, rec.Zone5mH20 = "no_touch", "no_touch"
		}
		loL, hiL := z.Anchor-1.5, z.Anchor+1.5
		zl, _, _, _ := bandOutcome(ctx1m, j, loL, hiL, z.Anchor, horizonLive, "close")
		rec.LineH12 = zl
		if eps := kernel.DetectTouchOutcomes(ctx1m, z.Anchor, kernel.DetectorK(), r.Delta, kernel.DetectorHorizonBars(), kernel.DetectorExitOn()); len(eps) > 0 {
			for _, e := range eps {
				if e.OpenedAtMs >= r.WinStartMs {
					rec.LiveDet = e.Outcome
					rec.LiveDetBars = e.BarsToExit
					break
				}
			}
			if rec.LiveDet == "" {
				rec.LiveDet = "none_in_window"
			}
		} else {
			rec.LiveDet = "none"
		}
		// C6 fill variants.
		rec.FillA = true
		if side == "short" {
			rec.FillB = ctx1m[j].High >= z.Anchor+tickMNQ
			rec.EntryC = z.Anchor + tickMNQ
		} else {
			rec.FillB = ctx1m[j].Low <= z.Anchor-tickMNQ
			rec.EntryC = z.Anchor - tickMNQ
		}
		rec.FillC = true

			// C4 trades — three stop multiples × three fill assumptions.
			others := []kernel.PlanLevel{}
			for zj := range m.Zones {
				oz := &m.Zones[zj]
				if zj == zi || oz.Lo == nil || oz.Anchor <= 0 {
					continue
				}
				label := ""
				if len(oz.Sources) > 0 {
					label = oz.Sources[0].Label
				}
				others = append(others, kernel.PlanLevel{Price: oz.Anchor, Label: label})
			}
			for s, mult := range stopMultiples {
				comp := composeTradeStop(side, z.Anchor, lo, hi, r.ATR5m, tickMNQ, others, mult)
				target := nearestOpposing(z.Anchor, side, m.Zones)
				rec.Trades[s][0] = simulateTrade(ctx1m, j, side, z.Anchor, comp.Stop, target, flatClose)
				if rec.FillB {
					rec.Trades[s][1] = simulateTrade(ctx1m, j, side, z.Anchor, comp.Stop, target, flatClose)
				} else {
					rec.Trades[s][1] = tradeOut{Filled: false}
				}
				rec.Trades[s][2] = simulateTrade(ctx1m, j, side, rec.EntryC, comp.Stop, target, flatClose)
			}
		out = append(out, rec)
	}
	return out
}

// evalLine evaluates the pre-map presentation on the SAME underlying anchors:
// a bare 3.00-pt band (±1.5) instead of the volatility-scaled zone band. Touch,
// outcome and authored stop all use the line band; target/ATR rules unchanged.
func evalLine(r *readSnapshot, m kernel.LevelZoneMap, ctx1m, ctx5m []market.Kline) []touchRecord {
	flatClose := r.SessionBars[len(r.SessionBars)-1].Close
	var out []touchRecord
	for zi := range m.Zones {
		z := &m.Zones[zi]
		if z.Lo == nil || z.Hi == nil || z.Anchor <= 0 {
			continue
		}
		lo, hi := z.Anchor-1.5, z.Anchor+1.5
		j := firstTouchIdx(ctx1m, lo, hi)
		if j < 1 {
			continue
		}
		rec := touchRecord{
			ReadIdx: r.IDX, ZoneIdx: zi, Era: eraStr(eraOf(r.ReadTime)),
			Day: r.Day, Session: r.Session, Contract: r.Contract,
			Year: r.ReadTime.Year(), Anchor: z.Anchor, Lo: lo, Hi: hi,
			WidthATR: 3.0, Broad: false, Shortlist: z.Shortlisted,
			Families: len(z.Families), FamilyCap: m.Options.FamilyCap,
			Sources: len(z.Sources), ATR5m: r.ATR5m, TouchBarMs: ctx1m[j].OpenTime,
			BandType: "line",
		}
		if r.ATR5m > 0 {
			rec.WidthATR = 3.0 / r.ATR5m
		}
		if len(z.Sources) > 0 {
			rec.SrcTF = z.Sources[0].TF
			rec.SrcKind = string(z.Sources[0].Kind)
			rec.SrcLabel = z.Sources[0].Label
			tfs := map[string]bool{}
			for _, s := range z.Sources {
				tfs[s.TF] = true
			}
			rec.CrossTF = len(tfs) > 1
		}
		if ctx1m[j-1].Close < lo {
			rec.Approach = "below"
		} else {
			rec.Approach = "above"
		}
		side := "long"
		if rec.Approach == "below" {
			side = "short"
		}
		zo, _, _, _ := bandOutcome(ctx1m, j, lo, hi, z.Anchor, horizonLive, "close")
		rec.ZoneH12 = zo
		rec.LineH12 = zo
		if eps := kernel.DetectTouchOutcomes(ctx1m, z.Anchor, kernel.DetectorK(), r.Delta, kernel.DetectorHorizonBars(), kernel.DetectorExitOn()); len(eps) > 0 {
			rec.LiveDet = "none_in_window"
			for _, e := range eps {
				if e.OpenedAtMs >= r.WinStartMs {
					rec.LiveDet = e.Outcome
					rec.LiveDetBars = e.BarsToExit
					break
				}
			}
		} else {
			rec.LiveDet = "none"
		}
		rec.FillA = true
		if side == "short" {
			rec.FillB = ctx1m[j].High >= z.Anchor+tickMNQ
			rec.EntryC = z.Anchor + tickMNQ
		} else {
			rec.FillB = ctx1m[j].Low <= z.Anchor-tickMNQ
			rec.EntryC = z.Anchor - tickMNQ
		}
		rec.FillC = true
		others := []kernel.PlanLevel{}
		for zj := range m.Zones {
			oz := &m.Zones[zj]
			if zj == zi || oz.Lo == nil || oz.Anchor <= 0 {
				continue
			}
			label := ""
			if len(oz.Sources) > 0 {
				label = oz.Sources[0].Label
			}
			others = append(others, kernel.PlanLevel{Price: oz.Anchor, Label: label})
		}
		for s, mult := range stopMultiples {
			comp := composeTradeStop(side, z.Anchor, lo, hi, r.ATR5m, tickMNQ, others, mult)
			target := nearestOpposing(z.Anchor, side, m.Zones)
			rec.Trades[s][0] = simulateTrade(ctx1m, j, side, z.Anchor, comp.Stop, target, flatClose)
			if rec.FillB {
				rec.Trades[s][1] = simulateTrade(ctx1m, j, side, z.Anchor, comp.Stop, target, flatClose)
			} else {
				rec.Trades[s][1] = tradeOut{Filled: false}
			}
			rec.Trades[s][2] = simulateTrade(ctx1m, j, side, rec.EntryC, comp.Stop, target, flatClose)
		}
		out = append(out, rec)
	}
	return out
}

func eraStr(b byte) string {
	if b == 1 {
		return "held_out"
	}
	return "in_sample"
}

// saveAll writes every artifact.
func saveAll(events []touchRecord, surf *surfaceEvents, seam seamResult, stats *runStats) {
	// events.jsonl
	f, err := os.Create(*flagOut + "/events.jsonl")
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f)
	for _, e := range events {
		b, _ := json.Marshal(e)
		w.Write(append(b, '\n'))
	}
	w.Flush()
	f.Close()

	zoneEvents := filterBand(events, "zone")
	lineEvents := filterBand(events, "line")
	// PRIMARY POPULATION: seated = shortlisted zones (the map's own seat flag).
	seated := filterSeated(zoneEvents)
	seatedLine := filterSeated(lineEvents)

	// C1 spine + C4 per cell (both eras + split).
	spine := map[string]any{}
	for dim, keys := range dimGroups(seated) {
		spine[dim] = c1c4Table(dim, keys, seated)
	}
	writeJSON(*flagOut+"/c1c4_spine.json", spine)
	// sensitivity: every complete-width zone (seated and not).
	spineAll := map[string]any{}
	for dim, keys := range dimGroups(zoneEvents) {
		spineAll[dim] = c1c4Table(dim, keys, zoneEvents)
	}
	writeJSON(*flagOut+"/c1c4_spine_all_zones.json", spineAll)

	// era splits for every headline number.
	eraSplits := map[string]any{
		"in_sample":  c4RowFor("in_sample", filterEra(seated, 0)),
		"held_out":   c4RowFor("held_out", filterEra(seated, 1)),
	}
	writeJSON(*flagOut+"/c8_era_splits.json", eraSplits)
	eraSplitsAll := map[string]any{
		"in_sample":  c4RowFor("in_sample", filterEra(zoneEvents, 0)),
		"held_out":   c4RowFor("held_out", filterEra(zoneEvents, 1)),
	}
	writeJSON(*flagOut+"/c8_era_splits_all_zones.json", eraSplitsAll)

	// C3 baseline — live detector verbatim.
	c3 := map[string]any{
		"seated_zone_anchor_live_detector": c3Stats(seated),
		"seated_line_anchor_live_detector": c3Stats(seatedLine),
		"all_zones_anchor_live_detector":   c3Stats(zoneEvents),
	}
	writeJSON(*flagOut+"/c3_baseline.json", c3)

	// C7 splits (seated population primary; all-zones reported too).
	writeJSON(*flagOut+"/c7_splits.json", c7Splits(seated, seatedLine))
	writeJSON(*flagOut+"/c7_splits_all_zones.json", c7Splits(zoneEvents, lineEvents))

	// C9 surfaces.
	s1 := surface1From(surf, *flagPerms, 0xC0FFEE)
	s2 := surface2From(surf, *flagPerms, 0xBEEF)
	writeSurfaceCSV(*flagOut+"/c9_surface_expectancy.csv", s1)
	writeSurfaceCSV(*flagOut+"/c9_surface_holdrate.csv", s2)
	writeJSON(*flagOut+"/c9_surface_summary.json", map[string]any{
		"s1": surfaceSummary(s1), "s2": surfaceSummary(s2),
		"perms": *flagPerms, "correction": "Westfall-Young max-T (permutation), Bonferroni bound reported per cell",
	})

	// reads census
	writeJSON(*flagOut+"/summary.json", summary(events, stats))
	fmt.Printf("done: events=%d (zone=%d line=%d) seated=%d\n", len(events), len(zoneEvents), len(lineEvents), len(seated))
}

func filterSeated(events []touchRecord) []touchRecord {
	out := []touchRecord{}
	for _, e := range events {
		if e.Shortlist {
			out = append(out, e)
		}
	}
	return out
}

func filterBand(events []touchRecord, band string) []touchRecord {
	out := []touchRecord{}
	for _, e := range events {
		if e.BandType == band {
			out = append(out, e)
		}
	}
	return out
}

func filterEra(events []touchRecord, era byte) []touchRecord {
	want := "in_sample"
	if era == 1 {
		want = "held_out"
	}
	out := []touchRecord{}
	for _, e := range events {
		if e.Era == want {
			out = append(out, e)
		}
	}
	return out
}

func c3Stats(events []touchRecord) map[string]any {
	p, n, amb := holdRate(events, func(e touchRecord) string { return e.LiveDet })
	lo, hi := wilsonCI(p, n)
	return map[string]any{
		"p_hold": p, "n": n, "ambiguous_excluded": amb, "wilson": []float64{lo, hi},
	}
}

// c7Splits answers round 21: zone vs line, and family 1 vs 2 vs 3+.
func c7Splits(zone, line []touchRecord) map[string]any {
	zp, zn, zamb := holdRate(zone, func(e touchRecord) string { return e.ZoneH12 })
	lp, ln, lamb := holdRate(line, func(e touchRecord) string { return e.ZoneH12 })
	zlo, zhi := wilsonCI(zp, zn)
	llo, lhi := wilsonCI(lp, ln)
	dlo, dhi := diffCI(zp, zn, lp, ln)
	fam := map[string]any{}
	for _, fk := range []int{1, 2, 3} {
		sub := []touchRecord{}
		for _, e := range zone {
			switch {
			case fk == 1 && e.Families == 1:
				sub = append(sub, e)
			case fk == 2 && e.Families == 2:
				sub = append(sub, e)
			case fk == 3 && e.Families >= 3:
				sub = append(sub, e)
			}
		}
		p, n, amb := holdRate(sub, func(e touchRecord) string { return e.ZoneH12 })
		lo, hi := wilsonCI(p, n)
		tr := c4RowFor(fmt.Sprintf("families=%d", fk), sub)
		fam[fmt.Sprintf("fam%d", fk)] = map[string]any{
			"p_hold": p, "n": n, "ambiguous": amb, "wilson": []float64{lo, hi},
			"n_touches": len(sub), "trade_fill_a": tr.FillA,
		}
	}
	p1, n1, _ := holdRate(filterFam(zone, 1), func(e touchRecord) string { return e.ZoneH12 })
	p3, n3, _ := holdRate(filterFam(zone, 3), func(e touchRecord) string { return e.ZoneH12 })
	d31lo, d31hi := diffCI(p3, n3, p1, n1)
	return map[string]any{
		"zone_vs_line": map[string]any{
			"zone_p": zp, "zone_n": zn, "zone_amb": zamb, "zone_wilson": []float64{zlo, zhi},
			"line_p": lp, "line_n": ln, "line_amb": lamb, "line_wilson": []float64{llo, lhi},
			"diff": zp - lp, "diff_ci": []float64{dlo, dhi},
			"round21_threshold_zone_vs_line": round21Verdict(dlo, dhi, zp-lp),
		},
		"multi_source": map[string]any{
			"fam3plus_vs_fam1_diff": p3 - p1, "diff_ci": []float64{d31lo, d31hi},
			"fam1_n": n1, "fam3plus_n": n3,
			"round21_threshold_multi_source": round21Verdict(d31lo, d31hi, p3-p1),
		},
		"families": fam,
	}
}

func filterFam(events []touchRecord, fam int) []touchRecord {
	out := []touchRecord{}
	for _, e := range events {
		switch {
		case fam == 1 && e.Families == 1:
			out = append(out, e)
		case fam == 2 && e.Families == 2:
			out = append(out, e)
		case fam == 3 && e.Families >= 3:
			out = append(out, e)
		}
	}
	return out
}

// round21Verdict: kill the feature if the interval includes 0 with an upper
// bound below +4pp (stated, no recommendation).
func round21Verdict(lo, hi, _ float64) string {
	if lo <= 0 && hi < 0.04 {
		return "interval includes 0 and upper bound < +4pp — feature does not clear round-21's threshold (no recommendation)"
	}
	return "interval does not meet round-21's kill condition"
}

func writeSurfaceCSV(path string, cells []surfaceCell) {
	var b strings.Builder
	b.WriteString("k,merge,cap,stop_mult,horizon,n,mean,se,stat,p_maxT,p_bonf,net_positive\n")
	for _, c := range cells {
		b.WriteString(fmt.Sprintf("%g,%g,%d,%g,%s,%d,%.4f,%.4f,%.4f,%.4f,%.4f,%v\n",
			c.K, c.M, c.Cap, c.StopMult, c.Horizon, c.N, c.Mean, c.SE, c.Stat, c.PMaxT, c.PBonf, c.NetPos))
	}
	mustWrite(path, []byte(b.String()))
}

func surfaceSummary(cells []surfaceCell) map[string]any {
	means := []float64{}
	pos := 0
	minMean, maxMean := 1e18, -1e18
	minStat := 1e18
	bestPMaxT := 1.0
	for _, c := range cells {
		if c.N == 0 {
			continue
		}
		means = append(means, c.Mean)
		if c.Mean > 0 {
			pos++
		}
		if c.Mean < minMean {
			minMean = c.Mean
		}
		if c.Mean > maxMean {
			maxMean = c.Mean
		}
		if c.Stat < minStat {
			minStat = c.Stat
		}
		if c.PMaxT < bestPMaxT {
			bestPMaxT = c.PMaxT
		}
	}
	_, _, med, _, _ := quartiles(means)
	return map[string]any{
		"cells_with_data": len(means), "cells_net_positive": pos,
		"median_mean": med, "min_mean": minMean, "max_mean": maxMean,
		"min_stat_abs": minStat, "best_p_maxT": bestPMaxT,
	}
}

func summary(events []touchRecord, stats *runStats) map[string]any {
	zone := filterBand(events, "zone")
	seated := filterSeated(zone)
	seatedLine := filterSeated(filterBand(events, "line"))
	in := filterEra(seated, 0)
	out := filterEra(seated, 1)
	return map[string]any{
		"reads_built": stats.built, "reads_skipped": stats.skipped,
		"reads_in_sample": stats.inSample, "reads_held_out": stats.heldOut,
		"events_total": len(events),
		"zone_events":  len(zone), "line_events": len(events) - len(zone),
		"seated_zone_events": len(seated), "seated_line_events": len(seatedLine),
		"zone_events_in_sample": len(in), "zone_events_held_out": len(out),
		"seated_headline": c4RowFor("all", seated),
		"seated_in_sample": c4RowFor("in_sample", in),
		"seated_held_out":  c4RowFor("held_out", out),
		"all_zones_headline": c4RowFor("all_zones", zone),
	}
}
