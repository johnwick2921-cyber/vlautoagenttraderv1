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

// SVPBin is one price row: Price is the row MIDPOINT; Vol its accumulated total
// volume; UpVol/DownVol the split by candle direction (close>=open → up, else
// down); InVA whether the row is inside the Value Area. Invariant: Vol ==
// UpVol + DownVol.
type SVPBin struct {
	Price   float64 `json:"price"`
	Vol     float64 `json:"vol"`     // total volume in the row (= UpVol + DownVol)
	UpVol   float64 `json:"upVol"`   // volume from up candles (close>=open)
	DownVol float64 `json:"downVol"` // volume from down candles (close<open)
	InVA    bool    `json:"inVA"`
}

// SVPSession is a single CME futures trading day's profile. VAH is the TOP edge
// of the top VA row, VAL the BOTTOM edge of the bottom VA row, POC the mid of
// the max-volume row. Frozen is true once the session has closed (every session
// except the current live one). SessionStart is the 17:00 CT session open as a
// UTC-seconds timestamp — the chart x-anchor for this session's histogram.
type SVPSession struct {
	SessionStart int64    `json:"sessionStart"` // UTC SECONDS — chart x-anchor
	Date         string   `json:"date"`         // YYYY-MM-DD (session-day start, Chicago)
	Partial      bool     `json:"partial"`
	POC          float64  `json:"poc"`
	VAH          float64  `json:"vah"`
	VAL          float64  `json:"val"`
	Bins         []SVPBin `json:"bins"`
	Frozen       bool     `json:"frozen"`
}

// SVPProfile is the full multi-session profile: one SVPSession per CME futures
// trading day present in the bar range, ascending by time (like TradingView's
// Session Volume Profile). RowHeight sizes the bars for the renderer. Sessions
// is NEVER nil (empty → []SVPSession{}).
type SVPProfile struct {
	RowHeight float64      `json:"rowHeight"`
	Sessions  []SVPSession `json:"sessions"`
}

// FormatSVPLine renders the MOST RECENT session as the single AI-prompt context
// line:
//
//	SVP (today's session, since 17:00 CT open): POC X VAH Y VAL Z (partial)
//
// The "(partial)" tag appears only when that session could not be seen from its
// open. Returns "" when there are no sessions, or the last session has no bars —
// the prompt gate then injects nothing (keeping the OFF/insufficient case clean).
func FormatSVPLine(p SVPProfile) string {
	if len(p.Sessions) == 0 {
		return ""
	}
	last := p.Sessions[len(p.Sessions)-1]
	if len(last.Bins) == 0 {
		return ""
	}
	s := fmt.Sprintf("SVP (today's session, since 17:00 CT open): POC %.2f VAH %.2f VAL %.2f",
		last.POC, last.VAH, last.VAL)
	if last.Partial {
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

// svpDirBins holds the per-row up/down volume split for one session.
type svpDirBins struct {
	up   map[int]float64
	down map[int]float64
}

func newSvpDirBins() *svpDirBins {
	return &svpDirBins{up: map[int]float64{}, down: map[int]float64{}}
}

// addBar distributes one bar's volume into the UP or DOWN histogram (by candle
// direction: close>=open → up, else down) using the same uniform row-overlap
// logic as svpAddBar.
func (s *svpDirBins) addBar(b market.Kline) {
	hist := s.up
	if b.Close < b.Open {
		hist = s.down
	}
	svpAddBar(hist, b.Low, b.High, b.Volume)
}

// totals returns the combined UP+DOWN volume per occupied row — the input to
// svpValueArea (POC/VA are computed on TOTAL volume).
func (s *svpDirBins) totals() map[int]float64 {
	tot := make(map[int]float64, len(s.up)+len(s.down))
	for b, v := range s.up {
		tot[b] += v
	}
	for b, v := range s.down {
		tot[b] += v
	}
	return tot
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

// BuildSVPProfile computes a MULTI-SESSION Session Volume Profile — one
// SVPSession per CME futures trading day present in the supplied 1-minute bars
// (CLOSED bars only). Bars are grouped by their CME session-day (the most-recent
// 17:00 CT boundary, via CMESessionDayStart): two bars share a session iff their
// CMESessionDayStart is equal, so each futures day (17:00 CT → 16:00 CT next
// day, overnight included) is its own profile — matching TradingView's SVP.
//
// The current live session (the one containing `now` while IsCMEOpen(now)) is
// Frozen=false; every other session is Frozen=true. Sessions are returned
// ascending by SessionStart and the slice is NEVER nil.
func BuildSVPProfile(bars []market.Kline, now time.Time) SVPProfile {
	nowMs := now.UnixMilli()

	// The live session's anchor (only meaningful while the market is open): the
	// session containing `now` is the developing one.
	var liveStartMs int64 = math.MinInt64
	if IsCMEOpen(now) {
		liveStartMs = CMESessionDayStart(now).UnixMilli()
	}

	// Group CLOSED bars by session-day-start (ms).
	type sessAgg struct {
		start     time.Time
		startMs   int64
		firstOpen int64
		dir       *svpDirBins
	}
	groups := make(map[int64]*sessAgg)
	for i := range bars {
		b := bars[i]
		if b.CloseTime >= nowMs {
			continue // in-progress / future bar (only CLOSED bars count)
		}
		start := CMESessionDayStart(time.UnixMilli(b.OpenTime))
		startMs := start.UnixMilli()
		g := groups[startMs]
		if g == nil {
			g = &sessAgg{start: start, startMs: startMs, firstOpen: b.OpenTime, dir: newSvpDirBins()}
			groups[startMs] = g
		} else if b.OpenTime < g.firstOpen {
			g.firstOpen = b.OpenTime
		}
		g.dir.addBar(b)
	}

	sessions := make([]SVPSession, 0, len(groups))
	for _, g := range groups {
		sessions = append(sessions, svpFinalizeSession(g.start, g.startMs, g.firstOpen, g.dir, g.startMs == liveStartMs))
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].SessionStart < sessions[j].SessionStart
	})

	return SVPProfile{
		RowHeight: SVPRowHeight,
		Sessions:  sessions,
	}
}

// svpFinalizeSession turns one session's accumulated up/down histogram into an
// SVPSession: POC/VA from svpValueArea on the TOTAL volume, then per-row bins
// carrying the up/down split. `live` marks the current developing session
// (Frozen=false); every other session is frozen.
func svpFinalizeSession(sessStart time.Time, startMs, firstOpen int64, dir *svpDirBins, live bool) SVPSession {
	sess := SVPSession{
		SessionStart: sessStart.Unix(),
		Date:         sessStart.Format("2006-01-02"),
		Frozen:       !live,
		// partial: we could not see the session from its 17:00 CT open.
		Partial: firstOpen > startMs+svpPartialToleranceMs,
		Bins:    []SVPBin{},
	}

	totals := dir.totals()
	poc, vaLow, vaHigh, _, _ := svpValueArea(totals, SVPValueAreaPercent)
	sess.POC = (float64(poc) + 0.5) * SVPRowHeight // row midpoint
	sess.VAH = float64(vaHigh+1) * SVPRowHeight    // top edge of top VA row
	sess.VAL = float64(vaLow) * SVPRowHeight       // bottom edge of bottom VA row

	// Emit occupied rows ascending by price, tagged inVA, with the up/down split.
	idx := make([]int, 0, len(totals))
	for b := range totals {
		idx = append(idx, b)
	}
	sort.Ints(idx)
	sess.Bins = make([]SVPBin, 0, len(idx))
	for _, b := range idx {
		up := dir.up[b]
		down := dir.down[b]
		sess.Bins = append(sess.Bins, SVPBin{
			Price:   (float64(b) + 0.5) * SVPRowHeight,
			Vol:     up + down,
			UpVol:   up,
			DownVol: down,
			InVA:    b >= vaLow && b <= vaHigh,
		})
	}
	return sess
}
