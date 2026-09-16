package kernel

// Zones are an uncut presentation view. No ScoredLevel, detector, identity,
// permission, or execution anchor is changed by this file.
import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type ZoneOptions struct {
	WidthK         float64 `json:"width_k"`
	MergeATR       float64 `json:"merge_atr"`
	MaxWidthATR    float64 `json:"max_width_atr"`
	BroadATR       float64 `json:"broad_atr"`
	RoundWidth     float64 `json:"round_width"`
	FamilyCap      int     `json:"family_cap"`
	ShortlistCap   int     `json:"shortlist_cap"`
	TouchWeight    float64 `json:"touch_weight"`
	RoundWeight    float64 `json:"round_weight"`
	FamilyWeight   float64 `json:"family_weight"`
	DistanceWeight float64 `json:"distance_weight"`
}

func DefaultZoneOptions(cap int) ZoneOptions {
	return ZoneOptions{WidthK: .5, MergeATR: .5, MaxWidthATR: 1, BroadATR: 1, RoundWidth: 2,
		FamilyCap: 3, ShortlistCap: cap, TouchWeight: 1, RoundWeight: 1, FamilyWeight: 1, DistanceWeight: 1}
}

// Resolvers are display-only; invalid environment values revert to documented
// defaults rather than changing any trading configuration or refusing a read.
func ResolveZoneOptions(cap int) ZoneOptions {
	o := DefaultZoneOptions(cap)
	positive := func(key string, fallback float64) float64 {
		v := envFloatDefault(key, fallback)
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return fallback
		}
		return v
	}
	o.WidthK = positive("LEVEL_ZONE_WIDTH_K", o.WidthK)
	o.MergeATR = positive("LEVEL_ZONE_MERGE_ATR", o.MergeATR)
	o.MaxWidthATR = positive("LEVEL_ZONE_MAX_WIDTH_ATR", o.MaxWidthATR)
	o.BroadATR = positive("LEVEL_ZONE_BROAD_ATR", o.BroadATR)
	o.RoundWidth = positive("LEVEL_ZONE_ROUND_WIDTH", o.RoundWidth)
	o.FamilyCap = envIntDefault("LEVEL_ZONE_FAMILY_CAP", o.FamilyCap)
	if o.FamilyCap < 1 {
		o.FamilyCap = 3
	}
	o.TouchWeight = positive("LEVEL_ZONE_TOUCH_WEIGHT", o.TouchWeight)
	o.RoundWeight = positive("LEVEL_ZONE_ROUND_WEIGHT", o.RoundWeight)
	o.FamilyWeight = positive("LEVEL_ZONE_FAMILY_WEIGHT", o.FamilyWeight)
	o.DistanceWeight = positive("LEVEL_ZONE_DISTANCE_WEIGHT", o.DistanceWeight)
	return o
}

type ZoneWidthInput struct {
	ATR          float64
	Wick         *float64
	PriorTouches *int
}
type ZoneSource struct {
	Kind         LevelKind `json:"kind"`
	TF           string    `json:"tf"`
	Price        float64   `json:"price"`
	Label        string    `json:"label"`
	FormedAt     *int64    `json:"formed_at"`
	Lo           *float64  `json:"lo"`
	Hi           *float64  `json:"hi"`
	WidthRule    string    `json:"width_rule"`
	PriorTouches *int      `json:"prior_touches"`
}
type LevelZone struct {
	Anchor           float64      `json:"anchor"`
	Lo               *float64     `json:"lo"`
	Hi               *float64     `json:"hi"`
	Incomplete       bool         `json:"incomplete_width"`
	Broad            bool         `json:"broad"`
	Sources          []ZoneSource `json:"sources"`
	Families         []string     `json:"families"`
	FamilyCount      int          `json:"family_count"`
	PriorTouches     *int         `json:"prior_touches"`
	RankValue        *float64     `json:"rank_value"`
	Shortlisted      bool         `json:"shortlisted"`
	boundLo, boundHi float64
}
type LevelZoneMap struct {
	At            time.Time   `json:"at"`
	Options       ZoneOptions `json:"options"`
	ATR5m         float64     `json:"atr5m"`
	Zones         []LevelZone `json:"zones"`
	Detected      int         `json:"detected"`
	Broad         int         `json:"broad"`
	Merged        int         `json:"merged"`
	NullWidths    int         `json:"null_widths"`
	WidestMerged  float64     `json:"widest_merged"`
	CrossTF       int         `json:"cross_tf"`
	CrossFamily   int         `json:"cross_family"`
	MergeRefusals int         `json:"merge_refusals"`
}

func ZoneSourceKey(l DetectedLevel) string {
	return fmt.Sprintf("%s|%s|%.8f|%d|%s", l.Kind, l.TF, l.Price, l.FormedAtMs, l.Label)
}
func ZoneFamily(k LevelKind) string {
	switch k {
	case KindSWGH, KindSWGL, KindEQH, KindEQL:
		return "swing-structure"
	case KindPOC, KindNPOC, KindVAH, KindVAL:
		return "volume-node"
	case KindRound:
		return "round-number"
	case KindSupply, KindDemand, KindOB, KindFVG, KindIFVG:
		return "imbalance"
	default:
		return "session/derived"
	}
}

func zoneSource(l DetectedLevel, in ZoneWidthInput, o ZoneOptions) ZoneSource {
	s := ZoneSource{Kind: l.Kind, TF: l.TF, Price: l.Price, Label: l.Label, FormedAt: identityInt64(l.FormedAtMs), PriorTouches: in.PriorTouches}
	lo, hi := l.Lo, l.Hi
	switch {
	case lo > 0 && hi > lo:
		s.WidthRule = "native detector bounds"
	case l.Kind == KindRound:
		lo = l.Price - o.RoundWidth/2
		hi = l.Price + o.RoundWidth/2
		s.WidthRule = "fixed round width [I]"
	case in.Wick != nil && *in.Wick >= 0 && in.ATR > 0:
		width := math.Max(*in.Wick, o.WidthK*in.ATR)
		lo = l.Price - width/2
		hi = l.Price + width/2
		s.WidthRule = "max(defining wick,k×ATR(tf)) [I]"
	default:
		s.WidthRule = "NULL: defining wick or ATR(tf) unavailable"
		return s
	}
	s.Lo = &lo
	s.Hi = &hi
	return s
}

// Stable ascending original anchor order; ties retain detector emission order.
// A cluster's anchor never moves. Candidates choose ONE nearest compatible
// existing cluster. Clusters never merge with each other (no transitive chain).
func BuildLevelZones(raw []DetectedLevel, price, atr5m float64, inputs map[string]ZoneWidthInput, o ZoneOptions, now time.Time) LevelZoneMap {
	v := LevelZoneMap{At: now, Options: o, ATR5m: atr5m, Detected: len(raw), Zones: []LevelZone{}}
	order := append([]DetectedLevel(nil), raw...)
	sort.SliceStable(order, func(i, j int) bool { return order[i].Price < order[j].Price })
	for _, l := range order {
		s := zoneSource(l, inputs[ZoneSourceKey(l)], o)
		lo, hi := l.Price, l.Price
		if s.Lo != nil {
			lo, hi = *s.Lo, *s.Hi
		} else {
			v.NullWidths++
		}
		// Without ATR, native bands remain separate context: breadth cannot be
		// compared to volatility. Missing width is counted, not called zero.
		broad := s.Lo != nil && (atr5m <= 0 || hi-lo > o.BroadATR*atr5m || hi-lo > o.MaxWidthATR*atr5m)
		best := -1
		distance := math.Inf(1)
		if !broad && s.Lo != nil && atr5m > 0 {
			for j, c := range v.Zones {
				if c.Broad || c.Lo == nil {
					continue
				}
				d := math.Abs(c.Anchor - l.Price)
				if d > o.MergeATR*atr5m {
					continue
				}
				if math.Max(c.boundHi, hi)-math.Min(c.boundLo, lo) > o.MaxWidthATR*atr5m {
					v.MergeRefusals++
					continue
				}
				if d < distance {
					best = j
					distance = d
				}
			}
		}
		if best < 0 {
			v.Zones = append(v.Zones, LevelZone{Anchor: l.Price, Broad: broad, Sources: []ZoneSource{}, boundLo: lo, boundHi: hi})
			best = len(v.Zones) - 1
		}
		z := &v.Zones[best]
		z.boundLo = math.Min(z.boundLo, lo)
		z.boundHi = math.Max(z.boundHi, hi)
		z.Sources = append(z.Sources, s)
		z.Incomplete = z.Incomplete || s.Lo == nil
		if s.Lo != nil || z.Lo != nil {
			a, b := z.boundLo, z.boundHi
			z.Lo = &a
			z.Hi = &b
		}
		z.Families = appendDistinct(z.Families, ZoneFamily(l.Kind))
	}
	for j := range v.Zones {
		z := &v.Zones[j]
		z.FamilyCount = min(len(z.Families), o.FamilyCap)
		if z.Broad {
			v.Broad++
		}
		if len(z.Sources) > 1 {
			v.Merged++
			v.WidestMerged = math.Max(v.WidestMerged, z.boundHi-z.boundLo)
			tfs := map[string]bool{}
			for _, s := range z.Sources {
				tfs[s.TF] = true
			}
			if len(tfs) > 1 {
				v.CrossTF++
			}
			if len(z.Families) > 1 {
				v.CrossFamily++
			}
		}
		// One zone's touch term comes from its original anchor source only.
		// Summing member counts would count the same episode repeatedly.
		z.PriorTouches = z.Sources[0].PriorTouches
		if atr5m > 0 && !z.Broad {
			touches := 0.
			if z.PriorTouches != nil {
				touches = math.Log1p(float64(*z.PriorTouches))
			}
			round := math.Inf(1)
			for _, l := range raw {
				if l.Kind == KindRound {
					round = math.Min(round, math.Abs(z.Anchor-l.Price))
				}
			}
			roundTerm := 0.
			if !math.IsInf(round, 1) {
				roundTerm = 1 / (1 + round/atr5m)
			}
			rank := o.TouchWeight*touches + o.RoundWeight*roundTerm + o.FamilyWeight*float64(z.FamilyCount) - o.DistanceWeight*math.Abs(z.Anchor-price)/atr5m
			z.RankValue = &rank
		}
	}
	indices := []int{}
	for j, z := range v.Zones {
		if z.RankValue != nil {
			indices = append(indices, j)
		}
	}
	sort.SliceStable(indices, func(i, j int) bool { return *v.Zones[indices[i]].RankValue > *v.Zones[indices[j]].RankValue })
	for rank, j := range indices {
		v.Zones[j].Shortlisted = rank < o.ShortlistCap
	}
	return v
}

func (v LevelZoneMap) Counts() string {
	return fmt.Sprintf("detected=%d zones=%d merged=%d context=%d cross-tf=%d cross-family=%d NULL-width=%d widest-merged=%.2f merge-refusals=%d", v.Detected, len(v.Zones), v.Merged, v.Broad, v.CrossTF, v.CrossFamily, v.NullWidths, v.WidestMerged, v.MergeRefusals)
}
func ZoneBootLine(o ZoneOptions) string {
	return fmt.Sprintf("🗺 zones: width=max(wick,k×ATR) k=%g[I] · merge=%g×ATR5m[I] max-width=%g×ATR5m[I] broad>%g×ATR5m[I] (also standalone above max-width) round-width=%gpt[I] · families capped=%d[I] · rank=[touches,round,families,distance] weights=[%g,%g,%g,%g][I] htf-mult=removed-from-zone-order (score unchanged) · cap=%d[O] · detected/merged/context/NULL-width=n/a (resolver=BuildLevelZones per read)", o.WidthK, o.MergeATR, o.MaxWidthATR, o.BroadATR, o.RoundWidth, o.FamilyCap, o.TouchWeight, o.RoundWeight, o.FamilyWeight, o.DistanceWeight, o.ShortlistCap)
}
func (v LevelZoneMap) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "ZONE MAP (%s; merge=%gpt cap=%gpt broad=%gpt; presentation only):\n", v.Counts(), v.Options.MergeATR*v.ATR5m, v.Options.MaxWidthATR*v.ATR5m, v.Options.BroadATR*v.ATR5m)
	for _, z := range v.Zones {
		bounds := "NULL"
		if z.Lo != nil {
			bounds = fmt.Sprintf("%.2f–%.2f", *z.Lo, *z.Hi)
		}
		kind := "local"
		if z.Broad {
			kind = "broad context"
		}
		fmt.Fprintf(&b, "  anchor %.2f · band %s · %s · families=%d · incomplete-width=%v\n", z.Anchor, bounds, kind, z.FamilyCount, z.Incomplete)
		for _, s := range z.Sources {
			formed := "NULL"
			if s.FormedAt != nil {
				formed = fmt.Sprint(*s.FormedAt)
			}
			fmt.Fprintf(&b, "    %s · tf=%s · anchor %.2f · formed_at=%s · width=%s\n", s.Label, s.TF, s.Price, formed, s.WidthRule)
		}
	}
	b.WriteString("ZONE ENTRY SHORTLIST [I] (touches,round,families,distance; no permission change):\n")
	short := []LevelZone{}
	for _, z := range v.Zones {
		if z.Shortlisted {
			short = append(short, z)
		}
	}
	sort.SliceStable(short, func(i, j int) bool { return *short[i].RankValue > *short[j].RankValue })
	for _, z := range short {
		touches := "NULL"
		if z.PriorTouches != nil {
			touches = fmt.Sprint(*z.PriorTouches)
		}
		fmt.Fprintf(&b, "  %.2f rank=%.4f touches=%s families=%d\n", z.Anchor, *z.RankValue, touches, z.FamilyCount)
	}
	return b.String()
}
