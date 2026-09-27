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
	"math"
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
		// PRIMARY — default options (the live map), zone arm only.
		m := buildZoneMap(r.Raw, r.Price, r.ATR5m, r.Inputs, baseOpts, r.ReadTime.UnixMilli())
		for _, rec := range evalMap(r, m, ctx1m, ctx5m, "zone") {
			rec.EventID = fmt.Sprintf("ev-%07d", eventSeq)
			eventSeq++
			events = append(events, rec)
			// SWEEP — 16 geometry cells (buffer × minR), SEATED zones only.
			if rec.Shortlist {
				for bi := 0; bi < 4; bi++ {
					for mj := 0; mj < 4; mj++ {
						surf.add(bi*4+mj, cellEvent{era: eraOf(r.ReadTime), net: float32(rec.Trades[bi][mj][0].NetPts)})
					}
				}
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
			Width: hi - lo, WidthATR: 0, Broad: z.Broad, Shortlist: z.Shortlisted,
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

		// 1B geometry: structure stop (far edge + buffer) × target minR, plus
		// the backtest-1 reference geometry recomputed on the same tape.
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
		for bi := 0; bi < 4; bi++ {
			stop := structureStop(side, lo, hi, bufferFor(hi-lo, bi))
			risk := math.Abs(stop - z.Anchor)
			for mj := 0; mj < 4; mj++ {
				target := nearestOpposingQualified(z.Anchor, side, m.Zones, minRGrid[mj]*risk)
				rec.Trades[bi][mj][0] = simulateTrade(ctx1m, j, side, z.Anchor, stop, target, flatClose)
				if rec.FillB {
					rec.Trades[bi][mj][1] = simulateTrade(ctx1m, j, side, z.Anchor, stop, target, flatClose)
				} else {
					rec.Trades[bi][mj][1] = tradeOut{Filled: false}
				}
				rec.Trades[bi][mj][2] = simulateTrade(ctx1m, j, side, rec.EntryC, stop, target, flatClose)
			}
		}
		comp := composeTradeStop(side, z.Anchor, lo, hi, r.ATR5m, tickMNQ, others, 1.5)
		rt := nearestOpposing(z.Anchor, side, m.Zones)
		rec.RefTrades[0] = simulateTrade(ctx1m, j, side, z.Anchor, comp.Stop, rt, flatClose)
		if rec.FillB {
			rec.RefTrades[1] = simulateTrade(ctx1m, j, side, z.Anchor, comp.Stop, rt, flatClose)
		} else {
			rec.RefTrades[1] = tradeOut{Filled: false}
		}
		rec.RefTrades[2] = simulateTrade(ctx1m, j, side, rec.EntryC, comp.Stop, rt, flatClose)
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

	// PRIMARY POPULATION: seated = shortlisted zones (the map's own seat flag).
	seated := filterSeated(events)
	in := filterEra(seated, 0)
	out := filterEra(seated, 1)

	// 16-cell geometry surface, all three fills, per era.
	cells := geometryCellStats(seated)
	cellsIn := geometryCellStats(in)
	cellsOut := geometryCellStats(out)
	writeJSON(*flagOut+"/geometry_cells.json", map[string]any{"all": cells, "in_sample": cellsIn, "held_out": cellsOut})
	writeGeometryCSV(*flagOut+"/geometry_cells.csv", cells, cellsIn, cellsOut)

	// Reference cell: backtest-1 geometry on THIS tape.
	ref := map[string]any{
		"all": refRow(seated), "in_sample": refRow(in), "held_out": refRow(out),
	}
	writeJSON(*flagOut+"/reference_cell.json", ref)

	// Hold-rate context (outcomes unchanged by geometry; population sanity).
	hold := map[string]any{
		"seated": holdRow(seated), "in_sample": holdRow(in), "held_out": holdRow(out),
		"live_detector_anchor": c3Stats(seated),
	}
	writeJSON(*flagOut+"/hold_context.json", hold)

	// Surface + correction (in-sample; the held-out year is a readout).
	cells16 := surfaceFrom(surf, *flagPerms, 0xC0FFEE)
	writeSurfaceCSV(*flagOut+"/surface_geometry.csv", cells16)
	writeJSON(*flagOut+"/surface_summary.json", map[string]any{
		"summary": surfaceSummary(cells16), "perms": *flagPerms,
		"correction": "Westfall-Young max-T (permutation, sign flips), Bonferroni bound per cell",
	})

	writeJSON(*flagOut+"/summary.json", summary(events, stats, cells, ref))
	fmt.Printf("done: events=%d seated=%d\n", len(events), len(seated))
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

func writeSurfaceCSV(path string, cells []surfaceCell) {
	var b strings.Builder
	b.WriteString("buffer,min_r,n,mean,se,stat,p_maxT,p_bonf,net_positive\n")
	for _, c := range cells {
		b.WriteString(fmt.Sprintf("%s,%g,%d,%.4f,%.4f,%.4f,%.4f,%.4f,%v\n",
			c.Buffer, c.MinR, c.N, c.Mean, c.SE, c.Stat, c.PMaxT, c.PBonf, c.NetPos))
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

func summary(events []touchRecord, stats *runStats, cells []geomRow, ref map[string]any) map[string]any {
	seated := filterSeated(events)
	return map[string]any{
		"reads_built": stats.built, "reads_skipped": stats.skipped,
		"reads_in_sample": stats.inSample, "reads_held_out": stats.heldOut,
		"events_total": len(events), "seated_events": len(seated),
		"hold_context": holdRow(seated),
		"reference_cell": ref,
		"geometry_cells": cells,
	}
}
