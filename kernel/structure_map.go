package kernel

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"nofx/market"
)

// ── S1 — THE STRUCTURE LAYER (CTO dispatch under the owner's delegation, 2026-09-16) ──
//
// WHY. The plan is ONE 12-seat table. On 2026-09-16 the 16:31 CT read detected
// 244 HTF levels (≈70 4h, ≈12 daily) and seated 11: 8 today-references, 3 HTF,
// zero daily — isTodayPriority sorts first by rule, freshMult decays a 4h zone
// on 1m touches, collapseLevelClusters renames a 4h level under the reference,
// seatHTF caps HTF at 2. The owner's target (approved 17:55 CT): TWO tables —
// STRUCTURE (D/4h/1h, direction only, never an entry) and ENTRY (the existing
// 12-seat logic). This file is the STRUCTURE table. Every knob defaults OFF;
// nothing here changes the live plan until DS-R's S4 measurement.
//
// FIELD NAMES ARE THE CONTRACT for S3 (DS-102, validator) and S5 (DS-103):
// published on the bridge before the rest of this wave was written.

// StructureMapTFs are the structure table's timeframes, coarsest first.
var StructureMapTFs = []string{"D", "4h", "1h"}

// DefaultStructureTrendSwings is N: the trend is read from the last N
// labelled swings (HH/HL → up, LH/LL → down, mixed → range).
const DefaultStructureTrendSwings = 3

// SwingPoint is one confirmed fractal swing.
type SwingPoint struct {
	Price  float64 `json:"price"`
	TimeMs int64   `json:"time_ms"`
}

// StructureZone is one HTF zone of the structure table — label INTACT (never
// collapsed into a reference), ranked by the existing scorer.
type StructureZone struct {
	Kind  string  `json:"kind"` // the level kind, e.g. "OB", "Supply", "FVG"
	Lo    float64 `json:"lo"`
	Hi    float64 `json:"hi"`
	TF    string  `json:"tf"`    // D | 4h | 1h
	Fresh string  `json:"fresh"` // the scorer's freshness label
	Score float64 `json:"score"`
}

// StructureTF is one timeframe's structure read.
type StructureTF struct {
	Trend           string          `json:"trend"` // up | down | range
	LastSwingHigh   *SwingPoint     `json:"last_swing_high,omitempty"`
	LastSwingLow    *SwingPoint     `json:"last_swing_low,omitempty"`
	ImpulseLo       float64         `json:"impulse_lo,omitempty"`
	ImpulseHi       float64         `json:"impulse_hi,omitempty"`
	PremiumDiscount float64         `json:"premium_discount"` // 0..1, price's position in the last impulse range (0 = at the low)
	Zones           []StructureZone `json:"zones,omitempty"`
	Bars            int             `json:"bars"` // closed bars the read rested on (A21)
}

// StructureMap is the STRUCTURE table — bias only, never an entry.
type StructureMap struct {
	AsOf     int64                  `json:"as_of_ms"`
	Contract string                 `json:"contract,omitempty"`
	TFs      map[string]StructureTF `json:"tfs"`
}

// StructureBarsReader hands the map its bars per timeframe ("D" | "4h" | "1h"),
// oldest first — the planner's own bar resolver, so the map reads what the
// planner reads.
type StructureBarsReader func(tf string) []market.Kline

// structureMapTFMinutes is the bar length per structure timeframe.
func structureMapTFMinutes(tf string) int {
	switch tf {
	case "D":
		return 24 * 60
	case "4h":
		return 240
	default:
		return 60
	}
}

// structureMapTFKey folds the level pool's TF strings onto the map's keys
// ("1d"/"d"/"daily" → "D"); anything else is returned lower-cased as-is.
func structureMapTFKey(tf string) string {
	switch strings.ToLower(strings.TrimSpace(tf)) {
	case "1d", "d", "daily":
		return "D"
	case "4h":
		return "4h"
	case "1h":
		return "1h"
	}
	return strings.ToLower(strings.TrimSpace(tf))
}

// StructureZoneCap is the most zones one TF of the structure table shows.
const StructureZoneCap = 6

// ComputeStructureMap builds the STRUCTURE table. Per TF: closed bars →
// fractalSwings (the SAME detector the 5m/15m/1h structure engine runs) →
// trend from the last `trendSwings` labelled swings (all HH/HL → up, all
// LH/LL → down, otherwise range; fewer swings than N → range, never a
// guess) → the last swing high/low → the last impulse (the range between
// them) → price's position in it (0..1). Zones are the pool's own zones for
// that TF, best score first, labels INTACT, capped at StructureZoneCap; a
// line (Lo == Hi) is not a zone. A TF with no closed bars is ABSENT from the
// map; no TF at all → nil (canon: no fabricated values).
func ComputeStructureMap(bars StructureBarsReader, contract string, pool []ScoredLevel, price float64, nowMs int64, trendSwings int) *StructureMap {
	if bars == nil {
		return nil
	}
	if trendSwings <= 0 {
		trendSwings = DefaultStructureTrendSwings
	}
	out := &StructureMap{AsOf: nowMs, Contract: contract, TFs: map[string]StructureTF{}}
	for _, tf := range StructureMapTFs {
		iv := int64(structureMapTFMinutes(tf)) * 60_000
		var closed []market.KlineBar
		for _, k := range bars(tf) {
			if k.OpenTime+iv <= nowMs {
				closed = append(closed, market.KlineBar{Time: k.OpenTime, Open: k.Open, High: k.High, Low: k.Low, Close: k.Close, Volume: k.Volume})
			}
		}
		if len(closed) == 0 {
			continue
		}
		highs := make([]float64, len(closed))
		lows := make([]float64, len(closed))
		closes := make([]float64, len(closed))
		for i, b := range closed {
			highs[i], lows[i], closes[i] = b.High, b.Low, b.Close
		}
		atr := simpleATR14(highs, lows, closes)
		st := StructureTF{Trend: "range", Bars: len(closed)}
		swings := fractalSwings(closed, iv, atr)
		if len(swings) >= trendSwings {
			last := swings[len(swings)-trendSwings:]
			up, down := true, true
			for _, sw := range last {
				if sw.kind != "HH" && sw.kind != "HL" {
					up = false
				}
				if sw.kind != "LH" && sw.kind != "LL" {
					down = false
				}
			}
			switch {
			case up:
				st.Trend = "up"
			case down:
				st.Trend = "down"
			}
		}
		for i := len(swings) - 1; i >= 0 && (st.LastSwingHigh == nil || st.LastSwingLow == nil); i-- {
			sw := swings[i]
			if sw.high && st.LastSwingHigh == nil {
				st.LastSwingHigh = &SwingPoint{Price: sw.price, TimeMs: sw.timeMs}
			}
			if !sw.high && st.LastSwingLow == nil {
				st.LastSwingLow = &SwingPoint{Price: sw.price, TimeMs: sw.timeMs}
			}
		}
		if st.LastSwingHigh != nil && st.LastSwingLow != nil {
			st.ImpulseLo = math.Min(st.LastSwingHigh.Price, st.LastSwingLow.Price)
			st.ImpulseHi = math.Max(st.LastSwingHigh.Price, st.LastSwingLow.Price)
			if rng := st.ImpulseHi - st.ImpulseLo; rng > 0 && price > 0 {
				pd := (price - st.ImpulseLo) / rng
				st.PremiumDiscount = math.Max(0, math.Min(1, pd))
			}
		}
		st.Zones = structureZonesFor(pool, tf)
		out.TFs[tf] = st
	}
	if len(out.TFs) == 0 {
		return nil
	}
	return out
}

// structureZonesFor picks the pool's zones for one structure TF — best score
// first, labels intact, at most StructureZoneCap. Lines are not zones.
func structureZonesFor(pool []ScoredLevel, tf string) []StructureZone {
	var zs []StructureZone
	for _, l := range pool {
		if structureMapTFKey(l.TF) != tf || l.Hi <= l.Lo {
			continue
		}
		zs = append(zs, StructureZone{Kind: string(l.Kind), Lo: l.Lo, Hi: l.Hi, TF: tf, Fresh: l.Fresh, Score: l.Score})
	}
	sort.SliceStable(zs, func(i, j int) bool { return zs[i].Score > zs[j].Score })
	if len(zs) > StructureZoneCap {
		zs = zs[:StructureZoneCap]
	}
	return zs
}

// RenderStructureSection renders the STRUCTURE table for the planner prompt —
// bias only, never an entry. nil → "" (byte-identical prompt, knob off).
func RenderStructureSection(m *StructureMap) string {
	if m == nil || len(m.TFs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## STRUCTURE — bias only, not entries\n")
	b.WriteString("  Direction per timeframe from the last swings; pd = where price sits in the last impulse (0 = low, 1 = high). A structure zone is context — never an entry; entries come only from the ranked table below.\n")
	for _, tf := range StructureMapTFs {
		st, ok := m.TFs[tf]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "  %s: %s", tf, st.Trend)
		if st.ImpulseHi > st.ImpulseLo {
			fmt.Fprintf(&b, " · impulse %.2f–%.2f · pd=%.2f", st.ImpulseLo, st.ImpulseHi, st.PremiumDiscount)
		}
		if st.LastSwingHigh != nil && st.LastSwingLow != nil {
			fmt.Fprintf(&b, " · last swing H %.2f / L %.2f", st.LastSwingHigh.Price, st.LastSwingLow.Price)
		}
		fmt.Fprintf(&b, " · %d bars", st.Bars)
		if len(st.Zones) > 0 {
			parts := make([]string, 0, len(st.Zones))
			for _, z := range st.Zones {
				parts = append(parts, fmt.Sprintf("%s %.2f–%.2f (%s)", z.Kind, z.Lo, z.Hi, z.Fresh))
			}
			b.WriteString(" · zones: " + strings.Join(parts, "; "))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// StructureLogLine is the per-read observability line:
// "🗺 structure @NY: D=up 4h=range 1h=down zones=3 pd4h=0.33". nil → n/a.
func StructureLogLine(m *StructureMap, session string) string {
	if m == nil || len(m.TFs) == 0 {
		return "🗺 structure @" + session + ": n/a (no map computed)"
	}
	parts := make([]string, 0, 5)
	zones := 0
	for _, tf := range StructureMapTFs {
		st, ok := m.TFs[tf]
		if !ok {
			parts = append(parts, tf+"=n/a")
			continue
		}
		parts = append(parts, tf+"="+st.Trend)
		zones += len(st.Zones)
	}
	line := "🗺 structure @" + session + ": " + strings.Join(parts, " ") + fmt.Sprintf(" zones=%d", zones)
	if st, ok := m.TFs["4h"]; ok {
		line += fmt.Sprintf(" pd4h=%.2f", st.PremiumDiscount)
	} else {
		line += " pd4h=n/a"
	}
	return line
}

// StructureBootLine — "🗺 structure: off|on(D/4h/1h)|n/a", READ from the knob;
// known=false means no strategy config was there to read.
func StructureBootLine(enabled, known bool) string {
	switch {
	case !known:
		return "🗺 structure: n/a"
	case enabled:
		return "🗺 structure: on(" + strings.Join(StructureMapTFs, "/") + ")"
	default:
		return "🗺 structure: off"
	}
}
