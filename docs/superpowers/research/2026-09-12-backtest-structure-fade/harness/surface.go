package main

// surface.go — BACKTEST 1B: the GEOMETRY surface, reported whole, with a named
// multiple-comparison correction (Westfall–Young max-T via permutation, plus
// Bonferroni). One surface:
//   G (expectancy): stop buffer {1 tick, 2 ticks, 0.25×width, 0.5×width} ×
//        target minR {0.5, 1.0, 1.5, 2.0} → 16 cells, metric = mean NET points
//        per trade (fill a), statistic = |mean| under sign flips.
// The map is fixed at the live defaults (k=0.5, merge=0.5×ATR5m, cap=1.0×ATR5m,
// family cap 3) — the geometry is the question here, not the map.
// In-sample for the correction; the held-out year is a readout, never tuned on.

import (
	"math"
	"sort"
)

// cellEvent is one first-touch event (compact form).
type cellEvent struct {
	era byte    // 0=in_sample, 1=held_out
	net float32 // net points (fill a) for its one geometry cell
}

const geomCells = 16 // 4 buffers × 4 minR

// surfaceEvents holds compact per-event data per geometry cell.
type surfaceEvents struct {
	byCell [geomCells][]cellEvent
}

func (s *surfaceEvents) add(cell int, e cellEvent) { s.byCell[cell] = append(s.byCell[cell], e) }

// cellLabel names a geometry cell (buffer × minR).
func cellLabel(cell int) (string, float64) {
	return bufferNames[cell/4], minRGrid[cell%4]
}

type surfaceCell struct {
	Buffer string
	MinR   float64
	N      int
	Mean   float64
	SE     float64
	Stat   float64
	PMaxT  float64
	PBonf  float64
	NetPos bool
}

// surfaceFrom computes the 16-cell expectancy surface.
func surfaceFrom(se *surfaceEvents, perms int, seed uint64) []surfaceCell {
	cells := make([]surfaceCell, geomCells)
	var flat []cellEvent
	cellOf := []int{}
	for c := 0; c < geomCells; c++ {
		for _, e := range se.byCell[c] {
			flat = append(flat, e)
			cellOf = append(cellOf, c)
		}
	}
	for ei, e := range flat {
		if e.era != 0 {
			continue
		}
		c := &cells[cellOf[ei]]
		c.Buffer, c.MinR = cellLabel(cellOf[ei])
		c.N++
		c.Mean += float64(e.net)
	}
	for i := range cells {
		c := &cells[i]
		if c.N == 0 {
			c.Buffer, c.MinR = cellLabel(i)
			continue
		}
		c.Mean /= float64(c.N)
		var ss float64
		for ei, e := range flat {
			if e.era != 0 {
				continue
			}
			if cellOf[ei] == i {
				d := float64(e.net) - c.Mean
				ss += d * d
			}
		}
		if c.N > 1 {
			c.SE = math.Sqrt(ss/float64(c.N-1)) / math.Sqrt(float64(c.N))
		}
		if c.SE > 0 {
			c.Stat = math.Abs(c.Mean / c.SE)
		}
		c.NetPos = c.Mean > 0
	}
	maxT := make([]float64, perms)
	for p := 0; p < perms; p++ {
		flips := shuffledSigns(len(flat), seed+uint64(p))
		sum := make([]float64, geomCells)
		cnt := make([]int, geomCells)
		for ei, e := range flat {
			if e.era != 0 {
				continue
			}
			c := cellOf[ei]
			sum[c] += flips[ei] * float64(e.net)
			cnt[c]++
		}
		var best float64
		for c := range cells {
			if cnt[c] == 0 || cells[c].SE <= 0 {
				continue
			}
			t := math.Abs((sum[c] / float64(cnt[c])) / cells[c].SE)
			if t > best {
				best = t
			}
		}
		maxT[p] = best
	}
	sort.Float64s(maxT)
	for i := range cells {
		cells[i].PMaxT = maxTPValue(maxT, cells[i].Stat)
		cells[i].PBonf = math.Min(1, cells[i].PMaxT*geomCells)
	}
	return cells
}

// maxTPValue: fraction of permuted maxT at least as large as stat.
func maxTPValue(sortedMaxT []float64, stat float64) float64 {
	if len(sortedMaxT) == 0 || stat <= 0 {
		return 1
	}
	i := sort.SearchFloat64s(sortedMaxT, stat)
	return float64(len(sortedMaxT)-i) / float64(len(sortedMaxT))
}
