package main

// out.go — report emission: marginal tables (C1), per-cell trade statistics
// (C4), split comparisons (C7), and the surface outputs (C9).

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

func mustWrite(path string, b []byte) {
	if err := os.WriteFile(path, b, 0o644); err != nil {
		panic(fmt.Sprintf("write %s: %v", path, err))
	}
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		panic(err)
	}
	mustWrite(path, append(b, '\n'))
}

// groupStat is one C4 cell's trade statistics for one fill assumption.
type groupStat struct {
	Group  string  `json:"group"`
	NAll   int     `json:"n_touches"`
	NAmb   int     `json:"n_ambiguous_h12"`
	NDet   int     `json:"n_decided_h12"` // hold+break
	HoldP  float64 `json:"hold_p"`
	HoldLo float64 `json:"hold_lo"`
	HoldHi float64 `json:"hold_hi"`
	NTrade int     `json:"n_trades"`
	FillRate float64 `json:"fill_rate_vs_a"`
	WinRGross float64 `json:"win_rate_gross"` // GrossPts > 0 (exit before friction)
	WinRNet  float64 `json:"win_rate_net"`   // NetPts > 0 (after friction)
	AvgR    float64 `json:"avg_r"`
	ExpPts float64 `json:"expectancy_pts_net"`
	ExpUSD float64 `json:"expectancy_usd_net"`
	PF     float64 `json:"profit_factor_net"`
	MaxDD  float64 `json:"max_dd_pts_net"`
	Streak int     `json:"longest_losing_streak"`
	MAEmin float64 `json:"mae_min"`
	MAEq1  float64 `json:"mae_q1"`
	MAEmed float64 `json:"mae_med"`
	MAEq3  float64 `json:"mae_q3"`
	MAEmax float64 `json:"mae_max"`
}

func rateOf(events []touchRecord, pred func(touchRecord) string) (hold int, brk int, amb int) {
	for _, e := range events {
		switch pred(e) {
		case "hold":
			hold++
		case "break":
			brk++
		default:
			amb++
		}
	}
	return
}

// holdRate returns p, n, ambiguous for events classified by pred.
func holdRate(events []touchRecord, pred func(touchRecord) string) (p float64, n, amb int) {
	h, b, a := rateOf(events, pred)
	n = h + b
	if n == 0 {
		return 0, 0, a
	}
	return float64(h) / float64(n), n, a
}

func tradeStatFor(events []touchRecord, stopIdx, fillIdx int) groupStat {
	g := groupStat{}
	var nets, gross, risk, mae []float64
	profitSum, lossSum := 0.0, 0.0
	for _, e := range events {
		t := e.Trades[stopIdx][fillIdx]
		if !t.Filled {
			continue
		}
		g.NTrade++
		nets = append(nets, t.NetPts)
		gross = append(gross, t.GrossPts)
		risk = append(risk, t.RMultiple)
		mae = append(mae, t.MAE)
		if t.GrossPts > 0 {
			g.WinRGross++
		}
		if t.NetPts > 0 {
			g.WinRNet++
			profitSum += t.NetPts
		} else {
			lossSum += -t.NetPts
		}
	}
	if g.NTrade > 0 {
		g.WinRGross /= float64(g.NTrade)
		g.WinRNet /= float64(g.NTrade)
		g.ExpPts = meanOf(nets)
		g.ExpUSD = g.ExpPts * 2.0 // MNQ $2/pt
		g.AvgR = meanOf(risk)
		if lossSum > 0 {
			g.PF = profitSum / lossSum
		}
		cum := make([]float64, len(nets))
		s := 0.0
		for i, v := range nets {
			s += v
			cum[i] = s
		}
		g.MaxDD = maxDD(cum)
		g.Streak = longestLosingStreak(nets)
		g.MAEmin, g.MAEq1, g.MAEmed, g.MAEq3, g.MAEmax = quartiles(mae)
	}
	return g
}

// c4Cell computes the full C4 stat row (fill variants side by side) for a group
// of events, plus the C2 hold rate on the zone H12 outcome.
type c4Row struct {
	Group string
	NAll, NAmb, NDet int
	HoldP, HoldLo, HoldHi float64
	FillA, FillB, FillC groupStat
}

func c4RowFor(group string, events []touchRecord) c4Row {
	r := c4Row{Group: group, NAll: len(events)}
	p, n, amb := holdRate(events, func(e touchRecord) string { return e.ZoneH12 })
	r.NDet, r.NAmb = n, amb
	r.HoldP = p
	if n > 0 {
		r.HoldLo, r.HoldHi = wilsonCI(p, n)
	}
	r.FillA = tradeStatFor(events, 1, 0) // default stop mult 1.5
	r.FillB = tradeStatFor(events, 1, 1)
	r.FillC = tradeStatFor(events, 1, 2)
	if r.FillA.NTrade > 0 {
		r.FillB.FillRate = float64(r.FillB.NTrade) / float64(r.FillA.NTrade)
		r.FillC.FillRate = 1
		r.FillA.FillRate = 1
	}
	return r
}

// dims enumerates the C1 grouping dimensions over the primary events.
func dimGroups(events []touchRecord) map[string][]string {
	groups := map[string][]string{}
	byYear := map[string][]touchRecord{}
	bySession := map[string][]touchRecord{}
	byApproach := map[string][]touchRecord{}
	byTF := map[string][]touchRecord{}
	byFam := map[string][]touchRecord{}
	byWidth := map[string][]touchRecord{}
	byEra := map[string][]touchRecord{}
	for _, e := range events {
		y := fmt.Sprintf("year=%d", e.Year)
		byYear[y] = append(byYear[y], e)
		s := "session=" + e.Session
		bySession[s] = append(bySession[s], e)
		a := "approach=" + e.Approach
		byApproach[a] = append(byApproach[a], e)
		tf := "tf=" + e.SrcTF
		byTF[tf] = append(byTF[tf], e)
		fam := "families=1"
		switch {
		case e.Families == 2:
			fam = "families=2"
		case e.Families >= 3:
			fam = "families=3+"
		}
		byFam[fam] = append(byFam[fam], e)
		w := "width=≥1.0×ATR (broad)"
		switch {
		case e.WidthATR < 0.25:
			w = "width=<0.25×ATR"
		case e.WidthATR < 0.5:
			w = "width=0.25–0.5×ATR"
		case e.WidthATR < 0.75:
			w = "width=0.5–0.75×ATR"
		case e.WidthATR < 1.0:
			w = "width=0.75–1.0×ATR"
		}
		byWidth[w] = append(byWidth[w], e)
		byEra[e.Era] = append(byEra[e.Era], e)
	}
	for k := range byYear {
		groups["year"] = append(groups["year"], k)
	}
	for k := range bySession {
		groups["session"] = append(groups["session"], k)
	}
	for k := range byApproach {
		groups["approach"] = append(groups["approach"], k)
	}
	for k := range byTF {
		groups["tf"] = append(groups["tf"], k)
	}
	for k := range byFam {
		groups["families"] = append(groups["families"], k)
	}
	for k := range byWidth {
		groups["width"] = append(groups["width"], k)
	}
	for k := range byEra {
		groups["era"] = append(groups["era"], k)
	}
	sort.Strings(groups["year"])
	sort.Strings(groups["session"])
	sort.Strings(groups["approach"])
	sort.Strings(groups["tf"])
	sort.Strings(groups["families"])
	sort.Strings(groups["width"])
	sort.Strings(groups["era"])
	return groups
}

func groupOf(key string, e touchRecord) bool {
	switch {
	case strings.HasPrefix(key, "year="):
		return fmt.Sprintf("year=%d", e.Year) == key
	case strings.HasPrefix(key, "session="):
		return "session="+e.Session == key
	case strings.HasPrefix(key, "approach="):
		return "approach="+e.Approach == key
	case strings.HasPrefix(key, "tf="):
		return "tf="+e.SrcTF == key
	case strings.HasPrefix(key, "families="):
		switch {
		case key == "families=1":
			return e.Families == 1
		case key == "families=2":
			return e.Families == 2
		default:
			return e.Families >= 3
		}
	case strings.HasPrefix(key, "width="):
		switch {
		case key == "width=<0.25×ATR":
			return e.WidthATR < 0.25
		case key == "width=0.25–0.5×ATR":
			return e.WidthATR >= 0.25 && e.WidthATR < 0.5
		case key == "width=0.5–0.75×ATR":
			return e.WidthATR >= 0.5 && e.WidthATR < 0.75
		case key == "width=0.75–1.0×ATR":
			return e.WidthATR >= 0.75 && e.WidthATR < 1.0
		default:
			return e.WidthATR >= 1.0
		}
	case key == "in_sample" || key == "held_out":
		return e.Era == key
	}
	return false
}

// c1c4Table renders the spine table rows for one dimension.
func c1c4Table(dim string, keys []string, events []touchRecord) []c4Row {
	rows := []c4Row{}
	for _, k := range keys {
		sub := []touchRecord{}
		for _, e := range events {
			if groupOf(k, e) {
				sub = append(sub, e)
			}
		}
		r := c4RowFor(k, sub)
		rows = append(rows, r)
	}
	_ = dim
	return rows
}
