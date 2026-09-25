package main

// surface.go — the C9 parameter surface, reported WHOLE, with a named multiple-
// comparison correction (Westfall–Young max-T via permutation, plus Bonferroni
// bounds). Two surfaces:
//   S1 (expectancy): widthK {0.25,0.5,1.0} × mergeATR {0.25,0.5,1.0} ×
//        familyCap {2,3,5} × stopMult {1.0,1.5,2.0}  → 81 cells, metric = mean
//        NET points per trade (fill a), statistic = |mean| under sign flips.
//   S2 (hold rate): widthK × mergeATR × familyCap × horizon {12@1m,10@5m,20@5m}
//        → 81 cells, metric = zone hold rate, statistic = |cellRate − pooled|,
//        pooled per horizon column, under label permutations.
// Both computed on IN-SAMPLE events only (the held-out year is a readout, never
// tuned on — D4).

import (
	"math"
	"sort"

	"nofx/kernel"
)

// cellEvent is one first-touch event for ONE map cell (compact form).
type cellEvent struct {
	era   byte // 0=in_sample, 1=held_out
	net   [3]float32 // net points per stop multiple, fill (a), H=12
	hold  [3]int8    // zone outcome per horizon column: 0=break 1=hold 2=ambiguous
	valid [3]bool    // hold column non-ambiguous
}

var sweepK = []float64{0.25, 0.5, 1.0}
var sweepM = []float64{0.25, 0.5, 1.0}
var sweepCap = []int{2, 3, 5}

// sweepOpts builds the options for map cell i in 0..26 (k outer, m middle, cap inner).
func sweepOpts(i int, base kernel.ZoneOptions) kernel.ZoneOptions {
	o := base
	o.WidthK = sweepK[i/9]
	o.MergeATR = sweepM[(i/3)%3]
	o.FamilyCap = sweepCap[i%3]
	return o
}

const sweepMaps = 27

type surfaceCell struct {
	K, M      float64
	Cap       int
	StopMult  float64 // S1 only
	Horizon   string  // S2 only
	N         int
	Mean      float64 // S1: mean net pts; S2: hold rate
	SE        float64
	Stat      float64 // S1: |mean|; S2: |rate−pooled|
	PMaxT     float64
	PBonf     float64
	NetPos    bool // S1: mean > 0
}

// surfaceEvents holds compact per-event data per map cell.
type surfaceEvents struct {
	byMap [sweepMaps][]cellEvent
}

func (s *surfaceEvents) add(mi int, e cellEvent) { s.byMap[mi] = append(s.byMap[mi], e) }

// surface1From computes S1 from the per-map event slices.
func surface1From(se *surfaceEvents, perms int, seed uint64) []surfaceCell {
	cells := make([]surfaceCell, 81)
	var flat []cellEvent
	mapOf := []int{}
	for mi := 0; mi < sweepMaps; mi++ {
		for _, e := range se.byMap[mi] {
			flat = append(flat, e)
			mapOf = append(mapOf, mi)
		}
	}
	for ei, e := range flat {
		if e.era != 0 {
			continue
		}
		mi := mapOf[ei]
		for s := 0; s < 3; s++ {
			c := &cells[mi*3+s]
			c.K, c.M = sweepK[mi/9], sweepM[(mi/3)%3]
			c.Cap = sweepCap[mi%3]
			c.StopMult = stopMultiples[s]
			c.N++
			c.Mean += float64(e.net[s])
		}
	}
	for i := range cells {
		c := &cells[i]
		if c.N == 0 {
			continue
		}
		c.Mean /= float64(c.N)
		var ss float64
		for ei, e := range flat {
			if e.era != 0 {
				continue
			}
			mi := mapOf[ei]
			if mi*3+(i%3) == i {
				d := float64(e.net[i%3]) - c.Mean
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
		sum := make([]float64, 81)
		cnt := make([]int, 81)
		for ei, e := range flat {
			if e.era != 0 {
				continue
			}
			mi := mapOf[ei]
			for s := 0; s < 3; s++ {
				ci := mi*3 + s
				sum[ci] += flips[ei] * float64(e.net[s])
				cnt[ci]++
			}
		}
		var best float64
		for ci := range cells {
			if cnt[ci] == 0 || cells[ci].SE <= 0 {
				continue
			}
			t := math.Abs((sum[ci] / float64(cnt[ci])) / cells[ci].SE)
			if t > best {
				best = t
			}
		}
		maxT[p] = best
	}
	sort.Float64s(maxT)
	for i := range cells {
		cells[i].PMaxT = maxTPValue(maxT, cells[i].Stat)
		cells[i].PBonf = math.Min(1, cells[i].PMaxT*81)
	}
	return cells
}

// surface2From computes the hold-rate surface: 27 maps × 3 horizons.
func surface2From(se *surfaceEvents, perms int, seed uint64) []surfaceCell {
	cells := make([]surfaceCell, 81)
	var flat []cellEvent
	mapOf := []int{}
	for mi := 0; mi < sweepMaps; mi++ {
		for _, e := range se.byMap[mi] {
			flat = append(flat, e)
			mapOf = append(mapOf, mi)
		}
	}
	// pooled rates per horizon column over in-sample non-ambiguous events.
	pooled := [3]float64{}
	pooledN := [3]int{}
	for _, e := range flat {
		if e.era != 0 {
			continue
		}
		for h := 0; h < 3; h++ {
			if e.valid[h] {
				pooled[h] += float64(e.hold[h])
				pooledN[h]++
			}
		}
	}
	for h := 0; h < 3; h++ {
		if pooledN[h] > 0 {
			pooled[h] /= float64(pooledN[h])
		}
	}
	for ei, e := range flat {
		if e.era != 0 {
			continue
		}
		mi := mapOf[ei]
		for h := 0; h < 3; h++ {
			c := &cells[mi*3+h]
			c.K, c.M = sweepK[mi/9], sweepM[(mi/3)%3]
			c.Cap = sweepCap[mi%3]
			c.Horizon = horizonLabel(h)
			if e.valid[h] {
				c.N++
				c.Mean += float64(e.hold[h])
			}
		}
	}
	for i := range cells {
		c := &cells[i]
		if c.N == 0 {
			continue
		}
		c.Mean /= float64(c.N)
		pool := pooled[i%3]
		c.SE = math.Sqrt(c.Mean*(1-c.Mean)/float64(c.N))
		c.Stat = math.Abs(c.Mean - pool)
	}
	maxT := make([]float64, perms)
	for p := 0; p < perms; p++ {
		// one shuffled label vector per horizon column.
		labels := [3][]float64{}
		for h := 0; h < 3; h++ {
			labels[h] = make([]float64, len(flat))
			n := 0
			for i, e := range flat {
				if e.era != 0 || !e.valid[h] {
					labels[h][i] = -1
					continue
				}
				labels[h][i] = float64(e.hold[h])
				n++
			}
			vals := make([]float64, 0, n)
			for _, v := range labels[h] {
				if v >= 0 {
					vals = append(vals, v)
				}
			}
			r := newRNG(seed + uint64(p)*7 + uint64(h))
			for i := len(vals) - 1; i > 0; i-- {
				j := r.intn(i + 1)
				vals[i], vals[j] = vals[j], vals[i]
			}
			idx := 0
			for i := range labels[h] {
				if labels[h][i] >= 0 {
					labels[h][i] = vals[idx]
					idx++
				}
			}
		}
		sum := make([]float64, 81)
		cnt := make([]int, 81)
		for ei, e := range flat {
			if e.era != 0 {
				continue
			}
			mi := mapOf[ei]
			for h := 0; h < 3; h++ {
				if e.valid[h] {
					sum[mi*3+h] += labels[h][ei]
					cnt[mi*3+h]++
				}
			}
		}
		var best float64
		for ci := range cells {
			if cnt[ci] == 0 {
				continue
			}
			rate := sum[ci] / float64(cnt[ci])
			t := math.Abs(rate - pooled[ci%3])
			if t > best {
				best = t
			}
		}
		maxT[p] = best
	}
	sort.Float64s(maxT)
	for i := range cells {
		cells[i].PMaxT = maxTPValue(maxT, cells[i].Stat)
		cells[i].PBonf = math.Min(1, cells[i].PMaxT*81)
	}
	return cells
}

func horizonLabel(h int) string {
	switch h {
	case 0:
		return "12@1m"
	case 1:
		return "10@5m"
	default:
		return "20@5m"
	}
}

// maxTPValue: fraction of permuted maxT at least as large as stat.
func maxTPValue(sortedMaxT []float64, stat float64) float64 {
	if len(sortedMaxT) == 0 || stat <= 0 {
		return 1
	}
	i := sort.SearchFloat64s(sortedMaxT, stat)
	return float64(len(sortedMaxT)-i) / float64(len(sortedMaxT))
}
