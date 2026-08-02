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
	"fmt"
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

	// The SVP session is the CME futures trading day: it opens at 17:00 CT and
	// RESETS every 17:00 CT (the daily-break roll). This is anchored to
	// CMESessionDayStart in cme_calendar.go — NOT the 08:30 RTH open — so the
	// profile matches the full futures session the bot actually trades.

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
	// SessionStartTime is the developing session's RTH open as a UTC-seconds
	// timestamp — the x-anchor the chart primitive uses for the developing
	// histogram (matches the lightweight-charts UTCTimestamp unit).
	SessionStartTime int64 `json:"sessionStartTime"`
}

// FormatSVPLine renders the profile as the single AI-prompt context line:
//
//	SVP: dev POC X VAH Y VAL Z (partial) | prior POC A VAH B VAL C
//
// The "(partial)" tag appears only when the developing session could not be
// seen from its open. Returns "" when the developing session has no bars — the
// prompt gate then injects nothing (keeping the OFF/insufficient case clean).
func FormatSVPLine(p SVPProfile) string {
	if p.Dev == nil || len(p.Dev.Bins) == 0 {
		return ""
	}
	s := fmt.Sprintf("SVP (today's session, since the 17:00 CT open): POC %.2f VAH %.2f VAL %.2f",
		p.Dev.POC, p.Dev.VAH, p.Dev.VAL)
	if p.Dev.Partial {
		s += " (partial)"
	}
	return s
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

// ---- Assembly (anchored to the CME session-day, cme_calendar.go) -----------

// BuildSVPProfile computes the developing Session Volume Profile for the CURRENT
// CME futures session-day from the supplied 1-minute bars (closed bars only).
// The session anchors to the 17:00 CT open (CMESessionDayStart) and RESETS at
// every 17:00 CT — one futures trading day. Bars before this session's open are
// ignored, and ONLY the current day is returned (Prior is always nil — no
// prior-session overlay).
func BuildSVPProfile(bars []market.Kline, now time.Time) SVPProfile {
	sessStart := CMESessionDayStart(now)
	dev := svpBuildSession(bars, sessStart, now.UnixMilli(), IsCMEOpen(now))
	return SVPProfile{
		RowHeight:        SVPRowHeight,
		Dev:              dev,
		Prior:            nil, // one futures day only
		SessionStartTime: sessStart.Unix(),
	}
}

// svpBuildSession windows the bars to the current session-day [sessStart, now]
// and builds its profile from CLOSED 1-minute bars. `sessStart` is the 17:00 CT
// session open; `live` marks a still-open session (frozen once the market closes).
func svpBuildSession(bars []market.Kline, sessStart time.Time, nowMs int64, live bool) *SVPSession {
	startMs := sessStart.UnixMilli()

	sess := &SVPSession{
		Date:   sessStart.Format("2006-01-02"),
		Frozen: !live,
	}

	hist := make(map[int]float64)
	var firstOpen int64 = math.MaxInt64
	var seen int
	for i := range bars {
		b := bars[i]
		if b.OpenTime < startMs {
			continue // before this session's 17:00 CT open
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
		// No bars in this session window → no profile at all. Return nil (JSON
		// dev:null) rather than a bins-less object, so the chart's `if (dev)`
		// guard cleanly skips it instead of dereferencing a null bins array.
		return nil
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
