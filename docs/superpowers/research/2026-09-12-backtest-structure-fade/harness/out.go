package main

// out.go — report emission: marginal tables (C1), per-cell trade statistics
// (C4), split comparisons (C7), and the surface outputs (C9).

import (
	"encoding/json"
	"fmt"
	"os"
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

// rateOf splits events into hold/break/ambiguous by pred.
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

// fillStat is one fill assumption's trade statistics for one geometry cell.
type fillStat struct {
	N           int     `json:"n"`
	WinNet      float64 `json:"win_net"`
	WinGross    float64 `json:"win_gross"`
	ExpPts      float64 `json:"exp_pts_net"`
	ExpUSD      float64 `json:"exp_usd_net"`
	PF          float64 `json:"pf_net"`
	MaxDD       float64 `json:"max_dd_pts"`
	Streak      int     `json:"streak"`
	MAEmed      float64 `json:"mae_med"`
	TargetShare float64 `json:"target_share"`
	StopShare   float64 `json:"stop_share"`
	FlatShare   float64 `json:"flat_share"`
}

func fillStatFor(trades []tradeOut) fillStat {
	g := fillStat{}
	var nets, mae []float64
	profitSum, lossSum := 0.0, 0.0
	for _, t := range trades {
		if !t.Filled {
			continue
		}
		g.N++
		nets = append(nets, t.NetPts)
		mae = append(mae, t.MAE)
		switch t.ExitKind {
		case "target":
			g.TargetShare++
		case "stop":
			g.StopShare++
		default:
			g.FlatShare++
		}
		if t.GrossPts > 0 {
			g.WinGross++
		}
		if t.NetPts > 0 {
			g.WinNet++
			profitSum += t.NetPts
		} else {
			lossSum += -t.NetPts
		}
	}
	if g.N > 0 {
		g.WinGross /= float64(g.N)
		g.WinNet /= float64(g.N)
		g.TargetShare /= float64(g.N)
		g.StopShare /= float64(g.N)
		g.FlatShare /= float64(g.N)
		g.ExpPts = meanOf(nets)
		g.ExpUSD = g.ExpPts * 2.0 // MNQ $2/pt
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
		_, _, g.MAEmed, _, _ = quartiles(mae)
	}
	return g
}

// geomRow is one geometry cell (buffer × minR), three fills side by side.
type geomRow struct {
	Buffer    string   `json:"buffer"`
	MinR      float64  `json:"min_r"`
	NTouch    int      `json:"n_touches"`
	FillA     fillStat `json:"fill_a"`
	FillB     fillStat `json:"fill_b"`
	FillBRate float64  `json:"fill_b_rate_vs_a"`
	FillC     fillStat `json:"fill_c"`
	R2Share   float64  `json:"r2_share"`   // fill-a trades with planned R ≥ 2.0
	R2ExpA    float64  `json:"r2_gated_exp_a"` // fill-a expectancy of that subset
}

func geomRowFor(events []touchRecord, bi, mj int) geomRow {
	r := geomRow{Buffer: bufferNames[bi], MinR: minRGrid[mj], NTouch: len(events)}
	a := make([]tradeOut, 0, len(events))
	b := make([]tradeOut, 0, len(events))
	c := make([]tradeOut, 0, len(events))
	var r2nets []float64
	for _, e := range events {
		ta := e.Trades[bi][mj][0]
		tb := e.Trades[bi][mj][1]
		tc := e.Trades[bi][mj][2]
		a = append(a, ta)
		if tb.Filled {
			b = append(b, tb)
		}
		c = append(c, tc)
		if ta.RPlanning >= 2.0 {
			r2nets = append(r2nets, ta.NetPts)
		}
	}
	r.FillA = fillStatFor(a)
	r.FillB = fillStatFor(b)
	r.FillC = fillStatFor(c)
	if r.FillA.N > 0 {
		r.FillBRate = float64(r.FillB.N) / float64(r.FillA.N)
		r.R2Share = float64(len(r2nets)) / float64(r.FillA.N)
		r.R2ExpA = meanOf(r2nets)
	}
	return r
}

func geometryCellStats(events []touchRecord) []geomRow {
	rows := make([]geomRow, 0, 16)
	for bi := 0; bi < 4; bi++ {
		for mj := 0; mj < 4; mj++ {
			rows = append(rows, geomRowFor(events, bi, mj))
		}
	}
	return rows
}

// refRow is the backtest-1 geometry (composeArmStop 1.5×ATR5m, nearest target)
// recomputed on this run's tape.
func refRow(events []touchRecord) map[string]any {
	a := make([]tradeOut, 0, len(events))
	b := make([]tradeOut, 0, len(events))
	c := make([]tradeOut, 0, len(events))
	for _, e := range events {
		a = append(a, e.RefTrades[0])
		if e.RefTrades[1].Filled {
			b = append(b, e.RefTrades[1])
		}
		c = append(c, e.RefTrades[2])
	}
	p, n, amb := holdRate(events, func(e touchRecord) string { return e.ZoneH12 })
	lo, hi := wilsonCI(p, n)
	return map[string]any{
		"n_touches": len(events), "hold_p": p, "hold_n": n, "hold_amb": amb,
		"hold_wilson": []float64{lo, hi},
		"fill_a": fillStatFor(a), "fill_b": fillStatFor(b), "fill_c": fillStatFor(c),
		"fill_b_rate_vs_a": ratioOf(len(b), len(a)),
	}
}

// holdRow: the outcome context (identical across geometry cells).
func holdRow(events []touchRecord) map[string]any {
	p, n, amb := holdRate(events, func(e touchRecord) string { return e.ZoneH12 })
	lo, hi := wilsonCI(p, n)
	p5, n5, a5 := holdRate(events, func(e touchRecord) string { return e.Zone5mH10 })
	return map[string]any{
		"n_touches": len(events),
		"h12_hold_p": p, "h12_n": n, "h12_amb": amb, "h12_wilson": []float64{lo, hi},
		"5m_h10_hold_p": p5, "5m_h10_n": n5, "5m_h10_amb": a5,
	}
}

func ratioOf(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

// writeGeometryCSV: one row per (era, cell).
func writeGeometryCSV(path string, all, in, out []geomRow) {
	var b strings.Builder
	b.WriteString("era,buffer,min_r,n_touches,n_a,exp_a,pf_a,win_a,stop_a,target_a,flat_a,n_b,rate_b,exp_b,n_c,exp_c,r2_share,r2_exp_a\n")
	for _, set := range []struct {
		era  string
		rows []geomRow
	}{{"all", all}, {"in_sample", in}, {"held_out", out}} {
		for _, r := range set.rows {
			b.WriteString(fmt.Sprintf("%s,%s,%g,%d,%d,%.4f,%.3f,%.4f,%.3f,%.3f,%.3f,%d,%.4f,%.4f,%d,%.4f,%.4f,%.4f\n",
				set.era, r.Buffer, r.MinR, r.NTouch, r.FillA.N, r.FillA.ExpPts, r.FillA.PF, r.FillA.WinNet,
				r.FillA.StopShare, r.FillA.TargetShare, r.FillA.FlatShare,
				r.FillB.N, r.FillBRate, r.FillB.ExpPts, r.FillC.N, r.FillC.ExpPts, r.R2Share, r.R2ExpA))
		}
	}
	mustWrite(path, []byte(b.String()))
}
