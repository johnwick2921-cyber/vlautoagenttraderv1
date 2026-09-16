package kernel

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"nofx/market"
)

// ── BAR HORIZON (wave BARS HORIZON, 2026-09-09) ──────────────────────────────
//
// A COUNT IS NOT A HORIZON. N bars served is not N intervals of history, and
// until this file nothing in the process could tell the difference.
//
// The live instance: on 2026-09-09 the machine was powered off 01:28 -> 13:01
// CT. The store genuinely lacks 696 consecutive MNQ 1m buckets across that
// span. The planner read at 13:18:13 CT was served 2000 bars — exactly what it
// asked for — spanning 3055 minutes with 696 open-market minutes missing inside
// them. planner_read_facts id 66 recorded `scope_bars=2000` and was RIGHT: the
// count was never short. The SPAN was wrong, and no field expressed it.
//
// So a served-count check alone would have been SILENT on the read that mattered.
// That is why this type carries three arms — SHORT (served < requested), HOLED
// (open-market intervals missing inside the served span) and EMPTY — and why
// HOLED is the one that speaks for the live instance.
//
// EVERY UNKNOWN IS -1 WITH A REASON IN Note, NEVER 0 (A24). A gap count of 0
// means "computed, and there are none"; -1 means "could not be computed", and
// the two must never be confused by a reader or by a query.

// BarHorizon is the horizon of ONE served tape.
type BarHorizon struct {
	Requested    int    `json:"asked"`
	Served       int    `json:"served"`
	Interval     string `json:"intv"`
	OldestOpenMs int64  `json:"oldest_open_ms"` // -1 = no bars served
	NewestOpenMs int64  `json:"newest_open_ms"` // -1 = no bars served
	SpanMs       int64  `json:"span_ms"`        // -1 = UNKNOWN
	OldestAgeMs  int64  `json:"oldest_age_ms"`  // -1 = UNKNOWN
	GapCount     int    `json:"gaps"`           // -1 = UNKNOWN (unmapped TF, or scan capped)
	Note         string `json:"note,omitempty"` // why a field is UNKNOWN
	Who          string `json:"who,omitempty"`  // which read this describes: "void" | "atr5m"
}

// Short reports that fewer bars came back than were asked for.
func (h BarHorizon) Short() bool { return h.Requested > 0 && h.Served < h.Requested }

// Holed reports open-market intervals missing INSIDE the served span. A holed
// window can be Short()==false — that is the 2026-09-09 13:18 CT case exactly.
func (h BarHorizon) Holed() bool { return h.GapCount > 0 }

// Empty reports that nothing at all came back.
func (h BarHorizon) Empty() bool { return h.Served == 0 }

// Why names the arms that fired, joined, or "" when the read was healthy. It is
// part of the warn dedupe key, so a read that changes CHARACTER speaks again.
func (h BarHorizon) Why() string {
	out := ""
	add := func(s string) {
		if out == "" {
			out = s
			return
		}
		out += "+" + s
	}
	if h.Empty() {
		add("EMPTY")
	}
	if h.Short() {
		add("SHORT")
	}
	if h.Holed() {
		add("HOLED")
	}
	return out
}

// Line renders the ONE horizon form the warn and the report both read.
func (h BarHorizon) Line() string {
	who := ""
	if h.Who != "" {
		who = h.Who + " "
	}
	s := fmt.Sprintf("%s%s asked=%d served=%d span=%s oldest=%s gaps=%s",
		who, h.Interval, h.Requested, h.Served,
		msDurTxt(h.SpanMs), msAgeTxt(h.OldestAgeMs), intTxt(h.GapCount))
	if h.Note != "" {
		s += " · " + h.Note
	}
	return s
}

// msDurTxt / msAgeTxt / intTxt render UNKNOWN as the word, never as a number.
func msDurTxt(ms int64) string {
	if ms < 0 {
		return "UNKNOWN"
	}
	return time.Duration(ms * int64(time.Millisecond)).String()
}

func msAgeTxt(ms int64) string {
	if ms < 0 {
		return "UNKNOWN"
	}
	return time.Duration(ms*int64(time.Millisecond)).String() + " ago"
}

func intTxt(n int) string {
	if n < 0 {
		return "UNKNOWN"
	}
	return fmt.Sprintf("%d", n)
}

// barHorizonScanCap bounds the open-interval walk. A scan that would exceed it
// reports UNKNOWN with the reason rather than a truncated number as fact. It is
// a var only so a pin can lower it; production never assigns it.
var barHorizonScanCap = 200_000

// barHorizonMemoMax bounds the memo. (symbol,tf) pairs are few and the window
// signature moves once per bar, so this holds several minutes of live keys.
const barHorizonMemoMax = 256

type horizonKey struct {
	tf             string
	oldest, newest int64
	served         int
}

type horizonVal struct {
	gaps int
	note string
}

var (
	barHorizonMemoMu sync.Mutex
	barHorizonMemo   = map[horizonKey]horizonVal{}
	// barHorizonScans counts CALENDAR WALKS actually performed (the memo's
	// denominator). Read by the memo pin; never reset in production.
	barHorizonScans atomic.Int64
)

// resetBarHorizonMemo clears the memo. Test seam only — a memo that survives a
// cap change would answer from the wrong regime.
func resetBarHorizonMemo() {
	barHorizonMemoMu.Lock()
	barHorizonMemo = map[horizonKey]horizonVal{}
	barHorizonMemoMu.Unlock()
}

// HorizonOf measures what a served slice ACTUALLY covers. `now` is the caller's
// clock (A28) — this function never reads one. Bars are assumed ascending by
// OpenTime (the BarCache protocol contract).
//
// It never consults the calendar when span/step+1 == served: that tape is
// contiguous by construction and its 0 is a REAL computed answer, not an
// assumption. The memo keys on the WINDOW (tf, oldest, newest, served), never
// on `now`, so ~40 identical reads per planner cycle cost one walk and the age
// still moves with the clock.
func HorizonOf(bars []market.Kline, tf string, requested int, now time.Time) BarHorizon {
	h := BarHorizon{
		Requested: requested, Served: len(bars), Interval: tf,
		OldestOpenMs: -1, NewestOpenMs: -1, SpanMs: -1, OldestAgeMs: -1, GapCount: -1,
	}
	if len(bars) == 0 {
		h.Note = "no bars served — span, age and gaps UNKNOWN"
		return h
	}
	h.OldestOpenMs = bars[0].OpenTime
	h.NewestOpenMs = bars[len(bars)-1].OpenTime
	h.SpanMs = h.NewestOpenMs - h.OldestOpenMs
	h.OldestAgeMs = now.UnixMilli() - h.OldestOpenMs

	step, ok := TFDurationMs(tf)
	if !ok || step <= 0 {
		h.Note = fmt.Sprintf("gaps UNKNOWN: timeframe %q is not in kernel/timeframes.go", tf)
		return h
	}
	if h.SpanMs < 0 {
		h.Note = "gaps UNKNOWN: the served slice is not ascending by open time"
		return h
	}
	if int(h.SpanMs/step)+1 == h.Served {
		h.GapCount = 0 // contiguous — computed, not assumed
		return h
	}
	h.GapCount, h.Note = horizonGaps(tf, h.OldestOpenMs, h.NewestOpenMs, h.Served, step)
	return h
}

// horizonGaps is the memoised calendar walk behind HorizonOf.
func horizonGaps(tf string, oldest, newest int64, served int, step int64) (int, string) {
	k := horizonKey{tf: tf, oldest: oldest, newest: newest, served: served}
	barHorizonMemoMu.Lock()
	if v, ok := barHorizonMemo[k]; ok {
		barHorizonMemoMu.Unlock()
		return v.gaps, v.note
	}
	barHorizonMemoMu.Unlock()

	barHorizonScans.Add(1)
	open, ok := OpenIntervalsBetween(oldest, newest+step, step, barHorizonScanCap)
	var v horizonVal
	switch {
	case !ok:
		v = horizonVal{gaps: -1, note: fmt.Sprintf("gaps UNKNOWN: the open-interval scan exceeded its cap of %d steps", barHorizonScanCap)}
	case open-served < 0:
		v = horizonVal{gaps: 0, note: fmt.Sprintf("served %d exceeds the %d OPEN intervals in this span — bars are stamped inside a closed session", served, open)}
	default:
		v = horizonVal{gaps: open - served}
	}

	barHorizonMemoMu.Lock()
	if len(barHorizonMemo) >= barHorizonMemoMax {
		barHorizonMemo = map[horizonKey]horizonVal{}
	}
	barHorizonMemo[k] = v
	barHorizonMemoMu.Unlock()
	return v.gaps, v.note
}

// OpenIntervalsBetween counts grid points in [fromMs, toMs) at stepMs whose
// instant falls in an OPEN CME session, per the SAME IsCMEOpen the trading gate
// reads. Closed runs are skipped in O(1) via NextCMEOpen, so a weekend costs one
// iteration rather than 2,940 minute probes.
//
// Returns (count, true), or (0, false) when the walk exceeded cap — the caller
// then reports UNKNOWN, never a truncated number as fact (A24).
func OpenIntervalsBetween(fromMs, toMs, stepMs int64, cap int) (int, bool) {
	if stepMs <= 0 || toMs <= fromMs {
		return 0, true
	}
	n, steps := 0, 0
	for t := fromMs; t < toMs; {
		steps++
		if steps > cap {
			return 0, false
		}
		if IsCMEOpen(time.UnixMilli(t)) {
			n++
			t += stepMs
			continue
		}
		nxt := NextCMEOpen(time.UnixMilli(t)).UnixMilli()
		if nxt <= t {
			nxt = t + stepMs
		}
		// Snap UP to this window's own grid so the count stays aligned with the
		// bar stamps it is compared against.
		if d := (nxt - fromMs) % stepMs; d != 0 {
			nxt += stepMs - d
		}
		t = nxt
	}
	return n, true
}
