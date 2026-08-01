package kernel

// Session Volume Profile (SVP) engine — Part B.
//
// Builds a volume-at-price histogram for the CME equity-index RTH session, then
// derives the Point of Control (POC) and the 70% Value Area (VAH/VAL) using the
// CLASSIC two-row expansion method (locked by a mandatory test vector in
// svp_test.go). It is STATELESS: each call recomputes from the 1-minute bars
// currently in the cache, which makes the "developing" profile grow bar-by-bar,
// reset automatically at the RTH open, and stay correct across a mid-session
// restart (a thin re-seed simply raises the `partial` flag).
//
// Session clock: the RTH window is anchored to the ONE session authority in this
// package (cme_calendar.go — isCMEHoliday + weekday), so SVP, the AI gate, and
// the guardrails never disagree about what day/session it is.

import (
	"math"
	"sort"
	"time"

	"nofx/market"
)

// ---- Constants (single source of truth) ------------------------------------

const (
	// SVPTickSize is the MNQ/ES index-futures tick (points per tick).
	SVPTickSize = 0.25
	// SVPRowHeightTicks is the profile row granularity in ticks. 5 ticks → each
	// price row spans 1.25 points.
	SVPRowHeightTicks = 5
	// SVPRowHeight is the price height of one histogram row (1.25 points).
	SVPRowHeight = SVPTickSize * SVPRowHeightTicks
	// SVPValueAreaPercent is the fraction of total session volume enclosed by the
	// Value Area (classic 70%).
	SVPValueAreaPercent = 0.70

	// RTH (Regular Trading Hours) for CME equity-index futures, Chicago time.
	// Open 08:30 CT, close 15:00 CT (documented constants; Part D may reconcile a
	// 15:15/16:00 variant by overriding these).
	svpRTHOpenMinCT  = 8*60 + 30 // 08:30
	svpRTHCloseMinCT = 15 * 60   // 15:00

	// svpPartialToleranceMs: if the earliest cached bar starts more than this
	// after the session open, we could not see the whole session → partial=true.
	svpPartialToleranceMs = 2 * 60 * 1000 // 2 minutes
)

// ---- Output types (also the JSON contract for the chart, Part C) -----------

// SVPBin is one price row: Price is the row MIDPOINT, Vol its accumulated
// volume, InVA whether the row is inside the Value Area.
type SVPBin struct {
	Price float64 `json:"price"`
	Vol   float64 `json:"vol"`
	InVA  bool    `json:"inVA"`
}

// SVPSession is a single session's profile. VAH is the TOP edge of the top VA
// row, VAL the BOTTOM edge of the bottom VA row, POC the mid of the max-volume
// row. Frozen is true once the session has closed (prior is always frozen).
type SVPSession struct {
	Date    string   `json:"date"` // RTH date in Chicago, YYYY-MM-DD
	Partial bool     `json:"partial"`
	POC     float64  `json:"poc"`
	VAH     float64  `json:"vah"`
	VAL     float64  `json:"val"`
	Bins    []SVPBin `json:"bins"`
	Frozen  bool     `json:"frozen"`
}

// SVPProfile bundles the developing (current) session and the frozen prior
// session, with the row height so the renderer can size bars.
type SVPProfile struct {
	RowHeight float64     `json:"rowHeight"`
	Dev       *SVPSession `json:"dev"`
	Prior     *SVPSession `json:"prior"`
}

// ---- Histogram (B1) --------------------------------------------------------

// svpBinIndex maps a price to its ABSOLUTE-grid row index: floor(price/row).
// The grid is anchored at 0, not at the session — so a given price always lands
// in the same row regardless of the session, which lets levels line up across
// days.
func svpBinIndex(price float64) int {
	return int(math.Floor(price / SVPRowHeight))
}

// svpAddBar distributes one bar's volume across the rows its [low,high] range
// touches, weighted by the fraction of the range that overlaps each row
// (uniform-within-bar assumption). A doji (low==high) drops all volume in one
// row.
func svpAddBar(hist map[int]float64, low, high, vol float64) {
	if high < low {
		low, high = high, low
	}
	if high == low {
		hist[svpBinIndex(low)] += vol
		return
	}
	span := high - low
	loBin := svpBinIndex(low)
	hiBin := svpBinIndex(high)
	for b := loBin; b <= hiBin; b++ {
		binLow := float64(b) * SVPRowHeight
		binHigh := binLow + SVPRowHeight
		overlap := math.Min(high, binHigh) - math.Max(low, binLow)
		if overlap <= 0 {
			continue
		}
		hist[b] += vol * (overlap / span)
	}
}

// ---- POC + Value Area (B2, classic two-row) --------------------------------

// svpValueArea computes the POC row and the Value-Area row span [vaLow,vaHigh]
// (inclusive, absolute row indices) using the CLASSIC two-row method:
//
//	POC   = the max-volume row (tie → nearest to the middle of the occupied
//	        range, then the LOWER row).
//	VA    = seed with the POC's volume; each step compare the SUM of the TWO
//	        rows ABOVE the current VA against the SUM of the TWO rows BELOW, and
//	        annex the larger PAIR as a unit (equal → expand UP). If a side is
//	        exhausted, take the other. Stop once VA volume ≥ 70% of the total.
//
// Annexing the pair as a UNIT (checking the target only AFTER both rows are
// added) is what separates this from the greedy one-row method — the mandatory
// test vector fails greedy and passes only this.
func svpValueArea(hist map[int]float64, vaPercent float64) (poc, vaLow, vaHigh int, vaVol, total float64) {
	if len(hist) == 0 {
		return 0, 0, 0, 0, 0
	}
	minB, maxB := math.MaxInt, math.MinInt
	for b, v := range hist {
		total += v
		if b < minB {
			minB = b
		}
		if b > maxB {
			maxB = b
		}
	}

	// POC: max-volume row; tie broken toward the middle of the occupied range,
	// then toward the LOWER row. Ascending iteration + strict "<" makes the lower
	// row win any equidistant tie.
	maxVol := math.Inf(-1)
	for _, v := range hist {
		if v > maxVol {
			maxVol = v
		}
	}
	mid := float64(minB+maxB) / 2.0
	poc = minB
	bestDist := math.Inf(1)
	for b := minB; b <= maxB; b++ {
		if hist[b] != maxVol {
			continue
		}
		if d := math.Abs(float64(b) - mid); d < bestDist {
			bestDist = d
			poc = b
		}
	}

	// Value area expansion.
	vaLow, vaHigh = poc, poc
	vaVol = hist[poc]
	target := total * vaPercent
	for vaVol < target {
		aboveAvail := vaHigh+1 <= maxB
		belowAvail := vaLow-1 >= minB
		if !aboveAvail && !belowAvail {
			break
		}
		above := hist[vaHigh+1] + hist[vaHigh+2] // absent rows read as 0
		below := hist[vaLow-1] + hist[vaLow-2]

		takeAbove := false
		switch {
		case !belowAvail:
			takeAbove = true
		case !aboveAvail:
			takeAbove = false
		case above > below:
			takeAbove = true
		case below > above:
			takeAbove = false
		default:
			takeAbove = true // equal → expand UP (documented)
		}

		// Annex the whole pair (up to two rows) on the chosen side, THEN check
		// the target — never between the two rows of the pair.
		if takeAbove {
			vaHigh++
			vaVol += hist[vaHigh]
			if vaHigh+1 <= maxB {
				vaHigh++
				vaVol += hist[vaHigh]
			}
		} else {
			vaLow--
			vaVol += hist[vaLow]
			if vaLow-1 >= minB {
				vaLow--
				vaVol += hist[vaLow]
			}
		}
	}
	return poc, vaLow, vaHigh, vaVol, total
}

// ---- RTH session windowing (anchored to cme_calendar.go) -------------------

func svpChicago() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return time.UTC
	}
	return loc
}

// svpIsRTHDay reports whether the given Chicago date has a Regular Trading Hours
// session: a weekday that is not a CME holiday. (Sunday has a Globex evening
// session but no RTH; Friday RTH exists.)
func svpIsRTHDay(ct time.Time) bool {
	switch ct.Weekday() {
	case time.Saturday, time.Sunday:
		return false
	}
	return !isCMEHoliday(ct)
}

// svpPrevRTHDay returns 08:30 CT of the most recent RTH day STRICTLY before the
// date of `from`.
func svpPrevRTHDay(from time.Time) time.Time {
	loc := from.Location()
	d := time.Date(from.Year(), from.Month(), from.Day(), 8, 30, 0, 0, loc)
	for i := 0; i < 14; i++ {
		d = d.AddDate(0, 0, -1)
		if svpIsRTHDay(d) {
			return d
		}
	}
	return d
}

// svpSessions resolves the current (developing) RTH day and the prior RTH day
// for `now`, plus whether the current session is still live (building). If now
// is before today's open or on a non-RTH day, the "current" session is the most
// recent completed RTH day.
func svpSessions(now time.Time) (curOpen, priorOpen time.Time, curLive bool) {
	loc := svpChicago()
	ct := now.In(loc)
	todayOpen := time.Date(ct.Year(), ct.Month(), ct.Day(), 8, 30, 0, 0, loc)
	todayClose := time.Date(ct.Year(), ct.Month(), ct.Day(), 15, 0, 0, 0, loc)

	if svpIsRTHDay(ct) && !ct.Before(todayOpen) {
		curOpen = todayOpen
		curLive = ct.Before(todayClose)
	} else {
		curOpen = svpPrevRTHDay(todayOpen)
		curLive = false
	}
	priorOpen = svpPrevRTHDay(curOpen)
	return curOpen, priorOpen, curLive
}

// ---- Assembly --------------------------------------------------------------

// BuildSVPProfile computes the developing + prior RTH volume profiles from the
// supplied 1-minute bars (closed bars only, relative to `now`). Bars outside the
// RTH windows are ignored. Pass the widest 1m history available; the engine
// windows it itself.
func BuildSVPProfile(bars []market.Kline, now time.Time) SVPProfile {
	loc := svpChicago()
	curOpen, priorOpen, live := svpSessions(now)
	nowMs := now.UnixMilli()
	dev := svpBuildSession(bars, curOpen, loc, nowMs, live)
	prior := svpBuildSession(bars, priorOpen, loc, nowMs, false)
	return SVPProfile{RowHeight: SVPRowHeight, Dev: dev, Prior: prior}
}

// svpBuildSession windows the bars to one RTH session and builds its profile.
// `open` is 08:30 CT of the session day; `live` marks the developing session
// (its window is capped at `now`).
func svpBuildSession(bars []market.Kline, open time.Time, loc *time.Location, nowMs int64, live bool) *SVPSession {
	startMs := open.UnixMilli()
	closeT := time.Date(open.Year(), open.Month(), open.Day(), 15, 0, 0, 0, loc)
	endMs := closeT.UnixMilli()

	sess := &SVPSession{
		Date:   open.Format("2006-01-02"),
		Frozen: !live,
	}

	hist := make(map[int]float64)
	var firstOpen int64 = math.MaxInt64
	var seen int
	for i := range bars {
		b := bars[i]
		// RTH window, CLOSED bars only (exclude the in-progress bar).
		if b.OpenTime < startMs || b.OpenTime >= endMs {
			continue
		}
		if b.CloseTime >= nowMs {
			continue // in-progress / future bar
		}
		if b.OpenTime < firstOpen {
			firstOpen = b.OpenTime
		}
		seen++
		svpAddBar(hist, b.Low, b.High, b.Volume)
	}

	if seen == 0 {
		sess.Partial = live // a live session with no bars yet is "partial"; a
		// closed prior session with no bars is just empty.
		return sess
	}

	// partial: we could not see the session from its open.
	sess.Partial = firstOpen > startMs+svpPartialToleranceMs

	poc, vaLow, vaHigh, _, _ := svpValueArea(hist, SVPValueAreaPercent)
	sess.POC = (float64(poc) + 0.5) * SVPRowHeight     // row midpoint
	sess.VAH = float64(vaHigh+1) * SVPRowHeight        // top edge of top VA row
	sess.VAL = float64(vaLow) * SVPRowHeight           // bottom edge of bottom VA row

	// Emit occupied rows ascending by price, tagged inVA.
	idx := make([]int, 0, len(hist))
	for b := range hist {
		idx = append(idx, b)
	}
	sort.Ints(idx)
	sess.Bins = make([]SVPBin, 0, len(idx))
	for _, b := range idx {
		sess.Bins = append(sess.Bins, SVPBin{
			Price: (float64(b) + 0.5) * SVPRowHeight,
			Vol:   hist[b],
			InVA:  b >= vaLow && b <= vaHigh,
		})
	}
	return sess
}
