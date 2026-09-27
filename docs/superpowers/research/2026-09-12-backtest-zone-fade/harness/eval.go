package main

// eval.go — touch detection, outcome classification, and the fade trade
// simulation. Outcome mechanics are the D1′ detector's exact loop
// (kernel/detector_d1prime.go, DetectTouchOutcomes), with barriers at the zone
// band edges instead of L±k·Δ: an episode opens when a bar's range enters the
// band and the previous bar's range did not; it closes when a CLOSE passes a
// band edge — the entry side is a HOLD, the far side a BREAK, horizon
// exhaustion is ambiguous_horizon. Fill assumptions and stop composition are
// per-dispatch C4/C6.

import (
	"math"

	"nofx/kernel"
	"nofx/market"
)

// touchRecord is one first-touch event of one zone in one session.
type touchRecord struct {
	ReadIdx   int     `json:"read_idx"`
	ZoneIdx   int     `json:"zone_idx"`
	EventID   string  `json:"event_id"`
	Era       string  `json:"era"` // in_sample | held_out
	Day       string  `json:"day"`
	Session   string  `json:"session"`
	Contract  string  `json:"contract"`
	Year      int     `json:"year"`
	BandType  string  `json:"band_type"` // zone | line (the 3.00-pt pre-map band)
	Anchor    float64 `json:"anchor"`
	Lo        float64 `json:"lo"`
	Hi        float64 `json:"hi"`
	WidthATR  float64 `json:"width_atr"`
	Broad     bool    `json:"broad"`
	Shortlist bool    `json:"shortlisted"`
	Families  int     `json:"families"`
	FamilyCap int     `json:"family_cap"`
	SrcTF     string  `json:"src_tf"`
	SrcKind   string  `json:"src_kind"`
	SrcLabel  string  `json:"src_label"`
	Sources   int     `json:"sources"`
	CrossTF   bool    `json:"cross_tf"`
	ATR5m     float64 `json:"atr5m"`
	Approach  string  `json:"approach"` // below | above
	TouchBarMs int64  `json:"touch_bar_ms"`

	// C2 outcomes — zone band (the dispatch's hold/break definition).
	ZoneH12    string `json:"zone_h12"` // hold|break|ambiguous_*
	ZoneH12Bars int   `json:"zone_h12_bars"`
	Zone5mH10  string `json:"zone_5m_h10"`
	Zone5mH20  string `json:"zone_5m_h20"`
	// C7 LINE: anchor ± 1.5 pt band (the pre-map 3.00-pt cluster tolerance),
	// same mechanics, same touch bar.
	LineH12 string `json:"line_h12"`
	// C3 LINE-A: the live detector VERBATIM on the anchor (k=3, Δ=read mean
	// |inc|, H=12, close exits).
	LiveDet  string `json:"live_det"`
	LiveDetBars int `json:"live_det_bars"`

	// C6 fill variants.
	FillA bool    `json:"fill_a"`
	FillB bool    `json:"fill_b"` // requires 1 tick THROUGH the anchor
	FillC bool    `json:"fill_c"`
	EntryC float64 `json:"entry_c"` // anchor ± 1 tick

	// C4 trades, per stop floor multiple (3), fill (3). index [stopIdx][fillIdx].
	Trades [3][3]tradeOut `json:"trades"`
}

type tradeOut struct {
	Filled    bool    `json:"filled"`
	Side      string  `json:"side"`
	Entry     float64 `json:"entry"`
	Stop      float64 `json:"stop"`
	Target    float64 `json:"target"` // 0 = none
	Exit      float64 `json:"exit"`
	ExitKind  string  `json:"exit_kind"` // stop | target | flat
	ExitBarMs int64   `json:"exit_bar_ms"`
	GrossPts  float64 `json:"gross_pts"`
	NetPts    float64 `json:"net_pts"`
	Risk      float64 `json:"risk"`
	RMultiple float64 `json:"r_multiple"`
	MFE       float64 `json:"mfe"`
	MAE       float64 `json:"mae"`
	BarsHeld  int     `json:"bars_held"`
}

// zoneZonesForOptions is the per-read zone build under one options cell.
func buildZoneMap(raw []kernel.DetectedLevel, price, atr5m float64, inputs map[string]kernel.ZoneWidthInput, o kernel.ZoneOptions, now int64) kernel.LevelZoneMap {
	return kernel.BuildLevelZones(raw, price, atr5m, inputs, o, unixMsTime(now))
}

// bandIntersect reports whether a bar's range intersects [lo,hi].
func bandIntersect(b market.Kline, lo, hi float64) bool { return b.Low <= hi && lo <= b.High }

// firstTouchIdx returns the index in ctx (>=1) of the first bar whose range
// intersects the band while the previous bar's does not. -1 = never touched.
func firstTouchIdx(ctx []market.Kline, lo, hi float64) int {
	for i := 1; i < len(ctx); i++ {
		if bandIntersect(ctx[i], lo, hi) && !bandIntersect(ctx[i-1], lo, hi) {
			return i
		}
	}
	return -1
}

// bandOutcome runs the detector's gambler's-ruin loop with barriers at the band
// edges, close-based exits, horizon bars, starting at bar touchIdx (the touch
// bar itself; exits are evaluated from the NEXT bar, matching DetectTouchOutcomes).
// mfe/mae are relative to the reference (anchor), signed like the detector's:
// mfe ≥ 0 in the entry direction... NOTE: detector convention — entry "below":
// mfe=max(high−ref), mae=min(low−ref).
func bandOutcome(ctx []market.Kline, touchIdx int, lo, hi float64, ref float64, horizon int, exitOn string) (outcome string, barsToExit int, mfe, mae float64) {
	if touchIdx < 1 || touchIdx >= len(ctx)-1 || horizon <= 0 || lo <= 0 || hi <= lo {
		return "ambiguous_horizon", 0, 0, 0
	}
	entry := "above"
	if ctx[touchIdx-1].Close < lo {
		entry = "below"
	}
	j := touchIdx + 1
	n := len(ctx)
	for ; j < n && (j-touchIdx) <= horizon; j++ {
		c := ctx[j]
		if entry == "below" {
			mfe = math.Max(mfe, c.High-ref)
			mae = math.Min(mae, c.Low-ref)
		} else {
			mfe = math.Max(mfe, ref-c.Low)
			mae = math.Min(mae, ref-c.High)
		}
		var cu, cd bool
		if exitOn == "close" {
			cu, cd = c.Close > hi, c.Close < lo
		} else {
			cu, cd = c.High > hi, c.Low < lo
		}
		if cu && cd {
			return "ambiguous_span", j - touchIdx, mfe, mae
		}
		if cu {
			if entry == "above" {
				return "hold", j - touchIdx, mfe, mae
			}
			return "break", j - touchIdx, mfe, mae
		}
		if cd {
			if entry == "below" {
				return "hold", j - touchIdx, mfe, mae
			}
			return "break", j - touchIdx, mfe, mae
		}
	}
	return "ambiguous_horizon", j - touchIdx, mfe, mae
}

// first5mTouch finds the first touch on the 5m series (with its own prev bar).
func first5mTouch(ctx []market.Kline, lo, hi float64) int { return firstTouchIdx(ctx, lo, hi) }

// composeTradeStop mirrors the live call at trader/armed_executor.go:488:
// composeArmStop(side, entry, authored, atr5m, tick, levels, mult, clearTicks,
// maxAnchorATR) with authored = zone far edge ± clearance (the fade's authored
// stop is beyond the zone), levels = the OTHER zones' anchors, mult = the stop
// floor multiple (1.5 default), clearTicks = kernel.MinSLTickClearance (2),
// maxAnchorATR = 3.0 (ArmStopAnchorMaxATRDefault).
func composeTradeStop(side string, entry, lo, hi, atr5m, tick float64, otherAnchors []kernel.PlanLevel, mult float64) StopComposition {
	clearance := float64(kernel.MinSLTickClearance) * tick
	authored := lo - clearance
	if side == "short" {
		authored = hi + clearance
	}
	return composeArmStop(side, entry, authored, atr5m, tick, otherAnchors, mult, kernel.MinSLTickClearance, 3.0)
}

// nearestOpposing returns the nearest complete-width zone anchor strictly on
// the opposite side of entry in the trade direction. short → highest anchor
// below entry; long → lowest anchor above entry. 0 = none.
func nearestOpposing(entry float64, side string, zones []kernel.LevelZone) float64 {
	best := 0.0
	for _, z := range zones {
		if z.Lo == nil {
			continue
		}
		a := z.Anchor
		if side == "short" {
			if a < entry && (best == 0 || a > best) {
				best = a
			}
		} else {
			if a > entry && (best == 0 || a < best) {
				best = a
			}
		}
	}
	return best
}

// simulateTrade walks bars after the touch bar. Fill is assumed AT the touch
// bar (all fill variants keep entry fixed by the caller). Stop-first on a bar
// that hits both stop and target (the conservative choice, stated in the
// report). Flat at the last session bar's close.
func simulateTrade(ctx []market.Kline, touchIdx int, side string, entry, stop, target, flatClose float64) tradeOut {
	out := tradeOut{Filled: true, Side: side, Entry: entry, Stop: stop, Target: target, Exit: flatClose, ExitKind: "flat", ExitBarMs: ctx[len(ctx)-1].OpenTime}
	long := side == "long"
	dir := 1.0
	if !long {
		dir = -1.0
	}
	for j := touchIdx + 1; j < len(ctx); j++ {
		b := ctx[j]
		if long {
			out.MFE = math.Max(out.MFE, b.High-entry)
			out.MAE = math.Min(out.MAE, b.Low-entry)
		} else {
			out.MFE = math.Max(out.MFE, entry-b.Low)
			out.MAE = math.Min(out.MAE, entry-b.High)
		}
		stopHit := false
		targetHit := false
		if long {
			stopHit = stop > 0 && b.Low <= stop
			targetHit = target > 0 && b.High >= target
		} else {
			stopHit = stop > 0 && b.High >= stop
			targetHit = target > 0 && b.Low <= target
		}
		if stopHit {
			out.Exit, out.ExitKind, out.ExitBarMs = stop, "stop", b.OpenTime
			out.BarsHeld = j - touchIdx
			break
		}
		if targetHit {
			out.Exit, out.ExitKind, out.ExitBarMs = target, "target", b.OpenTime
			out.BarsHeld = j - touchIdx
			break
		}
		if j == len(ctx)-1 {
			out.BarsHeld = j - touchIdx
		}
	}
	out.GrossPts = (out.Exit - entry) * dir
	out.NetPts = out.GrossPts - frictionPts
	out.Risk = math.Abs(stop - entry)
	if out.Risk > 0 {
		out.RMultiple = out.GrossPts / out.Risk
	}
	return out
}

// frictionPts is the round-turn friction in points (Mesfin's 2.0 on MNQ).
const frictionPts = 2.0

const tickMNQ = 0.25

// aggregate5m aggregates 1m bars into 5m buckets (kernel.AggregateBars).
func aggregate5m(bars []market.Kline) []market.Kline { return kernel.AggregateBars(bars, 5*60_000) }

// stopMultiples are the C9 sweep values of the stop floor multiple.
var stopMultiples = []float64{1.0, 1.5, 2.0}

// horizons1m5m: C2's stated horizons — 12 on 1m (the live one), 10 and 20 on 5m.
const horizonLive = 12

var horizon5m = []int{10, 20}
