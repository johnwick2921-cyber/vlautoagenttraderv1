package main

// eval.go — episode generation + Q1/Q4 aggregation + Q5 entry-timing variants.
//
// Population (documented, this is THE round-23 sampling decision):
//   every level in the raw detected universe of a read (AssembleResearchLevels'
//   raw return, which already ran dedupeSameKind per read), scanned in the
//   referencing session's window [max(session start, FormedAtMs), session flat).
//   A level referenced by several sessions contributes one scan per session —
//   each scan is a distinct (plan session × level) use of the map, which is the
//   question Q1 asks. All D1′ episodes (touch, then barrier resolution) in the
//   window are recorded with their ordinal.
//
// Instrument: kernel.DetectTouchOutcomes (the ONE live detector, not a port),
//   anchored at the level's Price, k = kernel.DetectorK() = 3,
//   Δ = trailing-5-day mean |1m close increment| at the read (the calibrated
//   ±16pt band), H = kernel.DetectorHorizonBars() = 12, exit_on = close.
//
// Q5 variants are computed ONLY on ordinal-1 episodes, each starting from a
//   later entry instant on the SAME tape and the SAME anchored barriers, via
//   localOutcomeFrom — a line-for-line copy of the D1′ inner loop with the
//   open condition replaced by "start at index, entry side given". A parity
//   assertion (class-97 guard) requires localOutcomeFrom at the touch index to
//   reproduce the kernel episode's outcome/MFE/MAE exactly, so the variant
//   loop is proven identical machinery.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"nofx/kernel"
	"nofx/market"
)

type episodeRec struct {
	Kind        string  `json:"kind"`
	TF          string  `json:"tf"`
	Family      string  `json:"family"`
	HTF         bool    `json:"htf"`
	Entry       string  `json:"entry,omitempty"` // D1′ entry side: below=long-side touch, above=short-side touch
	Ordinal     int     `json:"ordinal"`
	Outcome     string  `json:"outcome"`
	MFE         float64 `json:"mfe"`
	MAE         float64 `json:"mae"`
	OpenedAtMs  int64   `json:"opened_at_ms"`
	ClosedAtMs  int64   `json:"closed_at_ms"`
	DistAtRead  float64 `json:"dist_at_read"`
	AgeAtReadMs int64   `json:"age_at_read_ms"` // -1 = unknown (no FormedAtMs)
	Day         string  `json:"day"`
	Session     string  `json:"session"`
	// Q5 variants (ordinal-1 only; empty outcome = not computed / no fire)
	Confirm5mOutcome string  `json:"confirm_5m_outcome,omitempty"`
	Confirm5mMFE     float64 `json:"confirm_5m_mfe,omitempty"`
	Confirm5mBars    int     `json:"confirm_5m_bars,omitempty"` // touch→confirm, 1m bars
	MSSOutcome       string  `json:"mss_outcome,omitempty"`
	MSSMFE           float64 `json:"mss_mfe,omitempty"`
	MSSBars          int     `json:"mss_bars,omitempty"`
}

type cellAgg struct {
	Hold, Break, Ambig int
	SumMFE             float64
}

type cellOut struct {
	Key     string  `json:"key"`
	TF      string  `json:"tf,omitempty"`
	Kind    string  `json:"kind,omitempty"`
	Family  string  `json:"family,omitempty"`
	Ordinal string  `json:"ordinal,omitempty"`
	N       int     `json:"n"`
	Hold    int     `json:"hold"`
	Break   int     `json:"break"`
	Ambig   int     `json:"ambig"`
	Rate    float64 `json:"rate"`
	Lo      float64 `json:"wilson_lo"`
	Hi      float64 `json:"wilson_hi"`
	PvsNull float64 `json:"p_vs_null_0_5067"`
	MeanMFE float64 `json:"mean_mfe"`
}

var refKindsQ4 = map[string]bool{
	"PDH": true, "PDL": true, "ONH": true, "ONL": true,
	"OR-H": true, "OR-L": true, "VWAP": true,
}

func runEval(bd *barDB, reads []*readSnapshot, outDir string) error {
	return runEvalWith(bd, reads, outDir, nil)
}

// runEvalWith is runEval plus the optional S4 pass (-s4): episodes gain their
// entry side, every read emits its S1 trend row, and HTF level-scans emit Q-A
// freshness grades under BOTH the 1m-touch and the own-TF (S2) grading.
func runEvalWith(bd *barDB, reads []*readSnapshot, outDir string, s4 *s4State) error {
	epFile, err := os.Create(filepath.Join(outDir, "episodes.jsonl"))
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(epFile, 1<<20)
	defer func() {
		w.Flush()
		epFile.Close()
	}()

	q1 := map[string]*cellAgg{} // "K|tf|kind" and "F|tf|family"
	q4 := map[string]*cellAgg{} // "kind|ordinal"
	var nEp, nLevelScans, nSkipped int

	for _, r := range reads {
		if s4 != nil {
			if err := s4.emitTrends(bd, r); err != nil {
				return err
			}
		}
		np, ns, nsk, err := processRead(bd, r, w, q1, q4, s4)
		if err != nil {
			return err
		}
		nEp += np
		nLevelScans += ns
		nSkipped += nsk
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if err := epFile.Close(); err != nil {
		return err
	}
	fmt.Printf("episodes=%d level_scans=%d skipped=%d\n", nEp, nLevelScans, nSkipped)

	if err := writeCells(filepath.Join(outDir, "q1_cells.json"), q1); err != nil {
		return err
	}
	if err := writeCells(filepath.Join(outDir, "q4_ordinals.json"), q4); err != nil {
		return err
	}
	return nil
}

// processRead evaluates one read snapshot (streaming — snapshots are not kept).
func processRead(bd *barDB, r *readSnapshot, w *bufio.Writer, q1, q4 map[string]*cellAgg, s4 *s4State) (nEp, nLevelScans, nSkipped int, err error) {
	delta := delta5d(bd, r.Contract, r.ReadTime.UnixMilli())
	if delta <= 0 {
		delta = r.Delta // tape too thin for 5 days — fall back to the read's own Δ
	}
	for _, l := range r.Raw {
		if l.Price <= 0 {
			continue
		}
		tf := l.TF
		if tf == "" {
			tf = "1m"
		}
		fam := kernel.ZoneFamily(l.Kind)
		scanStart := r.WinStartMs
		if l.FormedAtMs > scanStart && l.FormedAtMs < r.FlatMs {
			scanStart = l.FormedAtMs
		}
		bars := bd.window1mWithPrev(r.Contract, scanStart, r.FlatMs)
		if len(bars) < 2 {
			nSkipped++
			continue
		}
		eps := kernel.DetectTouchOutcomes(bars, l.Price, kernel.DetectorK(), delta, kernel.DetectorHorizonBars(), "close")
		nLevelScans++
		dist := math.Abs(r.Price - l.Price)
		age := int64(-1)
		if l.FormedAtMs > 0 {
			age = r.ReadTime.UnixMilli() - l.FormedAtMs
		}
		for _, e := range eps {
			rec := episodeRec{
				Kind: string(l.Kind), TF: tf, Family: fam, HTF: l.HTF,
				Entry:   e.Entry,
				Ordinal: e.Ordinal, Outcome: e.Outcome, MFE: e.MFE, MAE: e.MAE,
				OpenedAtMs: e.OpenedAtMs, ClosedAtMs: e.ClosedAtMs,
				DistAtRead: dist, AgeAtReadMs: age, Day: r.Day, Session: r.Session,
			}
			if s4 != nil && e.Ordinal == 1 && l.HTF && s1HTFFreshTF(tf) {
				if err := s4.emitQA(bd, r, l, tf, e); err != nil {
					return nEp, nLevelScans, nSkipped, err
				}
			}
			if e.Ordinal == 1 {
				ti := idxByOpen(bars, e.OpenedAtMs)
				if ti > 0 {
					// parity guard: the variant loop must reproduce the kernel episode exactly
					oc, _, mfe, mae, _ := localOutcomeFrom(bars, ti, e.Entry, l.Price, kernel.DetectorK(), delta, kernel.DetectorHorizonBars())
					if oc != e.Outcome || mfe != e.MFE || mae != e.MAE {
						return nEp, nLevelScans, nSkipped, fmt.Errorf("parity mismatch kind=%s price=%.2f entry=%s: kernel %s/%.6f/%.6f vs local %s/%.6f/%.6f",
							l.Kind, l.Price, e.Entry, e.Outcome, e.MFE, e.MAE, oc, mfe, mae)
					}
					rec.Confirm5mOutcome, rec.Confirm5mMFE, rec.Confirm5mBars =
						confirm5mVariant(bars, ti, l.Price, kernel.DetectorK(), delta, kernel.DetectorHorizonBars())
					rec.MSSOutcome, rec.MSSMFE, rec.MSSBars =
						mssVariant(bars, ti, l.Price, kernel.DetectorK(), delta, kernel.DetectorHorizonBars())
				}
			}
			line, err := json.Marshal(rec)
			if err != nil {
				return nEp, nLevelScans, nSkipped, err
			}
			w.Write(line)
			w.WriteByte('\n')
			nEp++

			addCell(q1, "K|"+tf+"|"+string(l.Kind), rec)
			addCell(q1, "F|"+tf+"|"+fam, rec)
			if refKindsQ4[string(l.Kind)] {
				ord := "3+"
				if e.Ordinal == 1 {
					ord = "1"
				} else if e.Ordinal == 2 {
					ord = "2"
				}
				addCell(q4, string(l.Kind)+"|"+ord, rec)
			}
		}
	}
	return nEp, nLevelScans, nSkipped, nil
}

func addCell(m map[string]*cellAgg, key string, rec episodeRec) {
	c := m[key]
	if c == nil {
		c = &cellAgg{}
		m[key] = c
	}
	switch {
	case rec.Outcome == "hold":
		c.Hold++
		c.SumMFE += rec.MFE
	case rec.Outcome == "break":
		c.Break++
		c.SumMFE += rec.MFE
	default:
		c.Ambig++
	}
}

func writeCells(path string, m map[string]*cellAgg) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []cellOut
	for _, k := range keys {
		c := m[k]
		n := c.Hold + c.Break
		if n == 0 {
			continue
		}
		p := float64(c.Hold) / float64(n)
		lo, hi := kernel.WilsonInterval(p, n)
		co := cellOut{Key: k, N: n, Hold: c.Hold, Break: c.Break, Ambig: c.Ambig,
			Rate: p, Lo: lo, Hi: hi, PvsNull: binomTwoSidedP(c.Hold, n, 0.5067),
			MeanMFE: c.SumMFE / float64(n)}
		parts := split3(k) // [K|tf|kind], [F|tf|family], or [kind|ordinal]
		if len(parts) == 3 {
			co.TF = parts[1]
			if parts[0] == "F" {
				co.Family = parts[2]
			} else {
				co.Kind = parts[2]
			}
		} else if len(parts) == 2 {
			co.Kind = parts[0]
			co.Ordinal = parts[1]
		}
		out = append(out, co)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func split3(s string) []string {
	var parts []string
	cur := ""
	for _, ch := range s {
		if ch == '|' {
			parts = append(parts, cur)
			cur = ""
			continue
		}
		cur += string(ch)
	}
	return append(parts, cur)
}

// ── the variant loop: D1′ inner loop, line-for-line, open condition replaced ──

func localOutcomeFrom(bars []market.Kline, idx int, entry string, level, k, delta float64, horizon int) (outcome, exit string, mfe, mae float64, barsTo int) {
	up, lo := level+k*delta, level-k*delta
	n := len(bars)
	j := idx + 1
	for ; j < n && (j-idx) <= horizon; j++ {
		c := bars[j]
		if entry == "below" {
			mfe = math.Max(mfe, c.High-level)
			mae = math.Min(mae, c.Low-level)
		} else {
			mfe = math.Max(mfe, level-c.Low)
			mae = math.Min(mae, level-c.High)
		}
		cu, cd := c.Close > up, c.Close < lo
		if cu && cd {
			return "ambiguous_span", "", mfe, mae, j - idx
		}
		if cu {
			if entry == "above" {
				return "hold", "above", mfe, mae, j - idx
			}
			return "break", "above", mfe, mae, j - idx
		}
		if cd {
			if entry == "below" {
				return "hold", "below", mfe, mae, j - idx
			}
			return "break", "below", mfe, mae, j - idx
		}
	}
	return "ambiguous_horizon", "", mfe, mae, j - idx
}

// confirm5mVariant: entry at the first 5m CLOSE strictly beyond the level
// (either side) at/after the touch, within the episode's own H-bar window.
// Entry side = the side the previous 1m bar closed on. Same anchored barriers.
func confirm5mVariant(bars []market.Kline, ti int, level, k, delta float64, horizon int) (string, float64, int) {
	horizonMs := int64(horizon) * 60_000
	touchMs := bars[ti].OpenTime
	limit := touchMs + horizonMs
	m5 := kernel.AggregateBars(bars, 300_000)
	for _, b := range m5 {
		// AggregateBars leaves CloseTime unset — the bucket's last 1m close
		// instant is openMs + bucketMs − 1m (the D1′ open instant of that bar).
		closeMs := b.OpenTime + 300_000 - 60_000
		if closeMs <= touchMs {
			continue
		}
		if closeMs > limit {
			break
		}
		if b.Close > level || b.Close < level {
			ci := idxByOpen(bars, closeMs)
			if ci <= 0 || ci <= ti {
				continue
			}
			entry := "above"
			if bars[ci-1].Close < level {
				entry = "below"
			}
			oc, _, mfe, _, bt := localOutcomeFrom(bars, ci, entry, level, k, delta, horizon)
			return oc, mfe, bt
		}
	}
	return "no_confirm", 0, 0
}

// mssVariant: entry at the first 1m MSS (either side, kernel.EvaluateMSS, the
// production evaluator) whose BreakTimeMs falls after the touch, within the
// episode's H-bar window. Entry side = the side the previous 1m bar closed on.
func mssVariant(bars []market.Kline, ti int, level, k, delta float64, horizon int) (string, float64, int) {
	horizonMs := int64(horizon) * 60_000
	touchMs := bars[ti].OpenTime
	limit := touchMs + horizonMs
	best := struct {
		ms  int64
		idx int
	}{ms: math.MaxInt64}
	for i := ti + 1; i < len(bars) && bars[i].OpenTime <= limit; i++ {
		nowMs := bars[i].OpenTime + 59_999 // 1m close instant (repo convention)
		for _, side := range []string{"above", "below"} {
			v := kernel.EvaluateMSS(bars, side, nowMs)
			if v.Met && v.BreakTimeMs > touchMs && v.BreakTimeMs < best.ms {
				bi := idxByOpen(bars, v.BreakTimeMs)
				if bi > 0 {
					best.ms = v.BreakTimeMs
					best.idx = bi
				}
			}
		}
	}
	if best.idx == 0 {
		return "no_mss", 0, 0
	}
	entry := "above"
	if bars[best.idx-1].Close < level {
		entry = "below"
	}
	oc, _, mfe, _, bt := localOutcomeFrom(bars, best.idx, entry, level, k, delta, horizon)
	return oc, mfe, bt
}

// ── helpers ──

// window1mWithPrev returns 1m bars of the contract with openTime in
// [startMs, endMs), plus the single bar immediately before startMs as the
// D1′ "previous bar" context. Ascending.
func (b *barDB) window1mWithPrev(contract string, startMs, endMs int64) []market.Kline {
	rows := b.byKey[seriesKey{contract, "1m"}]
	start := sort.Search(len(rows), func(i int) bool { return rows[i].openMs >= startMs })
	if start > 0 {
		start--
	}
	end := sort.Search(len(rows), func(i int) bool { return rows[i].openMs >= endMs })
	if end <= start {
		return nil
	}
	return toKline(rows[start:end], "1m")
}

func idxByOpen(bars []market.Kline, openMs int64) int {
	return sort.Search(len(bars), func(i int) bool { return bars[i].OpenTime >= openMs })
}

// delta5d is the calibrated Δ: the mean |1m close-to-close increment| over the
// 5 calendar days before beforeMs (vet-02: ≈5.34 → ±16pt at k=3).
func delta5d(b *barDB, contract string, beforeMs int64) float64 {
	rows := b.byKey[seriesKey{contract, "1m"}]
	lo := beforeMs - 5*86_400_000
	start := sort.Search(len(rows), func(i int) bool { return rows[i].openMs >= lo })
	sum, n := 0.0, 0
	prev := math.NaN()
	for i := start; i < len(rows) && rows[i].openMs < beforeMs; i++ {
		if !math.IsNaN(prev) {
			sum += math.Abs(rows[i].c - prev)
			n++
		}
		prev = rows[i].c
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
