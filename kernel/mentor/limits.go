package mentor

import (
	"math"
	"strconv"
	"strings"
	"time"

	"vl/market"
)

// Limits is the G1 leg budget + G2 loss box state machine (DS-107, CTO build
// job 2026-10-03 13:00:13Z). It is simulated on CLOSED candles exactly like
// DS-103's E2: a stop entry is FILLED when price reaches the entry level
// (high >= entry for longs, low <= entry for shorts); a filled trade is a
// LOSS when its stop trades before its target; an order that never fills
// before its expiry is NO loss. That pins the CTO's three E2 cases
// (12:50:13Z) for every place: (a) entry touched (filled), then the stop
// touched -> blocked; (b) entry never touched before the expiry -> NOT
// blocked; (c) entry filled, then the target reached first -> NOT blocked.
//
// G1 LEG BUDGET (PLAN Filters, 05-EXTRA 2.7, X15 @00:36-02:46): inside ONE
// leg — from the PHL/PLH up to the old extreme — there are at most 2
// entries: the PHL/PLH and, if it appears, a same-direction ISB. A stop-out
// inside the leg closes it (no more entries); a NEW leg exists only on a 1m
// CLOSE beyond the prior high (long) / low (short). Knobs: LegBudgetEnabled
// (default ON), LegResetOn ("close" default | "touch").
//
// G2 LOSS BOX AT A PLACE (PLAN Filters, D4.2 p2 @01:56, @03:48): after a
// LOSS at a place (entry filled, then the stop hit), no re-entry there until
// price departs — the same departure rule as DS-103's E2: a closed candle
// whose range does NOT touch the place. Two losses at one place -> that
// place is OFF FOR THE DAY (cleared at the trading-day rollover).
//
// The leg is created at FILL (not placement), the budget is consulted at
// placement, and a stop-out closes the leg — all three exactly as the replay
// (engine.py LegBudget). G2 places (CTO R-b 2026-10-03): the setup's PLACE —
// the level price, the box edge, or the EMA — carried on the intent
// (AnchorKey/Anchor); a plain ISB carries none and is not loss-boxed. The
// G1 leg extreme is separate: the nearest old extreme beyond the entry.
type Limits struct {
	Long  *Leg
	Short *Leg
	// Places is the per-place loss registry, keyed by the quarter-tick
	// anchor price (round(anchor*4)) — the replay's key. It lives for one
	// trading day.
	Places map[string]*Place

	dayKey string
	pend   []*pendOrder
	open   []*openTrade
}

// Leg is one G1 leg per side.
type Leg struct {
	Side    Side
	Extreme float64 // the old extreme the leg runs to (the prior high/low)
	Entries int     // fills registered in this leg
	Stopped bool    // a stop-out inside the leg closed it
}

// Place is the G2 loss box at one place. Lo/Hi bound the place for the
// departure test: a level / EMA is a single price (Lo == Hi), a box spans
// its edges (CTO 13:20:08Z: a loss at either edge blocks the WHOLE box
// until price leaves it).
type Place struct {
	Lo      float64
	Hi      float64
	Losses  int
	Blocked bool
	OffDay  bool // two losses — the place is off for the day
	DayKey  string
	just    bool // blocked on the CURRENT candle: no departure check yet
}

type pendOrder struct {
	side   Side
	entry  float64
	stop   float64
	target float64
	expiry int64
	isISB  bool
	legExt float64 // G1 extreme carried from placement; 0 = none
	lo     float64 // G2 place bounds (a level/EMA: lo == hi); 0/0 = none
	hi     float64
	place  string // G2 place key (AnchorKey); "" = use the quarter-tick
}

type openTrade struct {
	side   Side
	stop   float64
	target float64
	lo     float64
	hi     float64
	place  string
}

// Apply is the single Tick hook (DS-103 merges the call): it advances the
// closed-candle simulation with cur (the just-closed 1m candle), applies the
// G1 leg budget and G2 loss box to the emitted entries, and registers the
// surviving entries for the next tick's simulation. prev is the candle
// before cur (its close is the leg-reset input for the "close" knob).
func (l *Limits) Apply(out []Intent, prev, cur market.Kline, now int64, levels []Level, cfg Config) []Intent {
	day := tradingDayKey(time.UnixMilli(now).In(ctime()))
	if l.dayKey != "" && l.dayKey != day {
		// New trading day: G2 places (including off-for-day) and the
		// unfinished simulations die with the day. Legs persist — the
		// replay keeps one LegBudget for the whole run.
		l.dayKey = day
		l.Places = nil
		l.pend = nil
		l.open = nil
	} else if l.dayKey == "" {
		l.dayKey = day
	}

	// G1: the reset is known at the close of the PREVIOUS candle and gates
	// this candle's entries (the replay's pc_of(m)).
	l.resetBreaks(prev, cur, cfg)

	// Simulate the orders placed on earlier ticks and the open trades
	// against the just-closed candle.
	l.simulate(cur, now, levels)

	// G2 departure: a blocked place clears when a closed candle does NOT
	// touch it — the same rule as DS-103's E2. A box clears only when price
	// LEAVES it (outside [Lo, Hi]). The loss candle itself never counts as
	// the departure; off-for-day places never clear by departure. Loss
	// counts survive the day — two losses at one place turn it off for the
	// day.
	for _, p := range l.Places {
		if p.just {
			p.just = false
			continue
		}
		if p.Blocked && !p.OffDay && p.Lo != 0 && (cur.High < p.Lo || cur.Low > p.Hi) {
			p.Blocked = false
		}
	}

	kept := out[:0]
	for _, in := range out {
		switch in.Action {
		case PlaceStopEntry:
			ext := nearestOldExtreme(levels, in.Price, in.Side)
			if cfg.LegBudgetEnabled {
				if ban := l.legVerdict(in.Side, false); ban != "" {
					continue // G1: second PHL in the leg / leg closed by a stop-out
				}
			}
			pkey, plo, phi, orphan := normalizePlace(in, levels)
			if orphan {
				continue // K2: an old extreme with no coincident key level is not a location
			}
			if pid := placeID(pkey, plo); pid != "" {
				if p := l.Places[pid]; p != nil && (p.Blocked || p.OffDay) {
					continue // G2: the place is boxed after a loss
				}
			}
			kept = append(kept, in)
			l.pend = append(l.pend, &pendOrder{
				side:   in.Side,
				entry:  in.Price,
				stop:   in.Stop,
				target: in.Target,
				expiry: in.ExpiryMs,
				legExt: ext,
				lo:     plo,
				hi:     phi,
				place:  pkey,
			})
		case PlaceStopLimitEntry:
			if cfg.LegBudgetEnabled {
				if ban := l.legVerdict(in.Side, true); ban != "" {
					continue // G1: the leg is full (the PHL + one ISB)
				}
			}
			kept = append(kept, in)
			l.pend = append(l.pend, &pendOrder{
				side:   in.Side,
				entry:  in.Price,
				stop:   in.Stop,
				target: in.Target,
				expiry: in.ExpiryMs,
				isISB:  true,
			})
		default:
			kept = append(kept, in) // cancels, extends, moves — never filtered
		}
	}
	return kept
}

// resetBreaks — G1: a new leg starts only on a break beyond the prior
// extreme. "close" (default): the previous candle's CLOSE strictly beyond
// the extreme, known at its close, gates the next candle. "touch": this
// candle's wick reaching the extreme.
func (l *Limits) resetBreaks(prev, cur market.Kline, cfg Config) {
	if cfg.LegResetOn == "touch" {
		if l.Long != nil && cur.High >= l.Long.Extreme {
			l.Long = nil
		}
		if l.Short != nil && cur.Low <= l.Short.Extreme {
			l.Short = nil
		}
		return
	}
	if l.Long != nil && prev.Close > l.Long.Extreme {
		l.Long = nil
	}
	if l.Short != nil && prev.Close < l.Short.Extreme {
		l.Short = nil
	}
}

// simulate advances the closed-candle fill/loss simulation one candle.
func (l *Limits) simulate(cur market.Kline, now int64, levels []Level) {
	keepP := l.pend[:0]
	for _, p := range l.pend {
		filled := p.side == SideLong && cur.High >= p.entry ||
			p.side == SideShort && cur.Low <= p.entry
		if filled {
			// Entry touched = filled. A stop-out on the SAME candle is a
			// loss (the trade existed for the span of the candle) — the
			// open-trade loop below catches it.
			l.registerLeg(p.side, p.legExt, levels, p.entry)
			l.open = append(l.open, &openTrade{
				side:   p.side,
				stop:   p.stop,
				target: p.target,
				lo:     p.lo,
				hi:     p.hi,
				place:  p.place,
			})
			continue
		}
		if p.expiry != 0 && now >= p.expiry {
			continue // never touched before the expiry — NOT a loss (case b)
		}
		keepP = append(keepP, p)
	}
	l.pend = keepP

	keepO := l.open[:0]
	for _, t := range l.open {
		stopped := t.side == SideLong && cur.Low <= t.stop ||
			t.side == SideShort && cur.High >= t.stop
		if stopped {
			l.loss(t.place, t.lo, t.hi, t.side) // filled, then the stop hit — case (a)
			continue
		}
		targetHit := t.side == SideLong && cur.High >= t.target ||
			t.side == SideShort && cur.Low <= t.target
		if targetHit {
			continue // the target reached first — NOT a loss (case c)
		}
		keepO = append(keepO, t)
	}
	l.open = keepO
}

// registerLeg records a FILL in the side's leg, creating the leg at the
// extreme it runs to. ext 0 falls back to the nearest old extreme beyond
// the entry (the replay's most-recent swing fallback); no extreme -> no leg
// (the replay skips registration too).
func (l *Limits) registerLeg(side Side, ext float64, levels []Level, entry float64) {
	if ext == 0 {
		ext = nearestOldExtreme(levels, entry, side)
	}
	if ext == 0 {
		return
	}
	leg := l.leg(side)
	if leg == nil {
		leg = &Leg{Side: side, Extreme: ext}
		l.setLeg(side, leg)
	}
	leg.Entries++
}

// loss registers a stop-out: one loss at the place (G2) and the leg is
// closed (G1). lo/hi bound the place for the departure test.
func (l *Limits) loss(place string, lo, hi float64, side Side) {
	pid := placeID(place, lo)
	if pid != "" {
		if l.Places == nil {
			l.Places = map[string]*Place{}
		}
		p := l.Places[pid]
		if p == nil {
			p = &Place{Lo: lo, Hi: hi, DayKey: l.dayKey}
			l.Places[pid] = p
		}
		p.Losses++
		p.Blocked = true
		p.just = true
		if p.Losses >= 2 {
			p.OffDay = true // two losses at one place — off for the day
		}
	}
	if leg := l.leg(side); leg != nil {
		leg.Stopped = true // a stop-out inside the leg closes it
	}
}

// legVerdict is the G1 budget check at PLACEMENT time ("" = allowed). The
// names mirror the replay's counters.
func (l *Limits) legVerdict(side Side, isISB bool) string {
	leg := l.leg(side)
	if leg == nil {
		return ""
	}
	if leg.Stopped {
		return "leg_budget_stopped"
	}
	if isISB {
		if leg.Entries >= 2 {
			return "leg_budget_full"
		}
		return ""
	}
	if leg.Entries >= 1 {
		return "leg_budget_second_phl" // the PHL is only ever the FIRST entry
	}
	return ""
}

func (l *Limits) leg(side Side) *Leg {
	if side == SideLong {
		return l.Long
	}
	return l.Short
}

func (l *Limits) setLeg(side Side, leg *Leg) {
	if side == SideLong {
		l.Long = leg
		return
	}
	l.Short = leg
}

// nearestOldExtreme returns the nearest old-extreme (swing) level beyond
// price in the trade direction — the G1 leg extreme and the G2 anchor.
// 0 = none.
func nearestOldExtreme(levels []Level, price float64, side Side) float64 {
	best := 0.0
	for _, lvl := range levels {
		if lvl.Kind != KindOldExtreme {
			continue
		}
		switch side {
		case SideLong:
			if lvl.Price > price && (best == 0 || lvl.Price < best) {
				best = lvl.Price
			}
		case SideShort:
			if lvl.Price < price && (best == 0 || lvl.Price > best) {
				best = lvl.Price
			}
		}
	}
	return best
}

// placeKey is the quarter-tick anchor key — the replay's round(anchor*4).
func placeKey(price float64) string {
	return strconv.FormatInt(int64(math.Round(price*4)), 10)
}

// placeID is the G2 registry key: the setup's place key when the emit site
// named one (the level key, the box edge key, the EMA key), else the
// quarter-tick anchor price. "" = no place (a plain ISB).
func placeID(anchorKey string, anchor float64) string {
	if anchorKey != "" {
		return anchorKey
	}
	if anchor != 0 {
		return placeKey(anchor)
	}
	return ""
}

// normalizePlace maps the raw emit-site place to the G2 key (CTO K1/K2,
// 13:15:27Z; box ruling 13:20:08Z):
//
//	K1 — every EMA keys on its CONSTANT line key ("ema34"/"ema9"/
//	"ema34_htf", which is what the EMA levels already carry); a moving line
//	cannot be keyed by price. The loss price is kept on the Place for the
//	departure test.
//	K2 — an old-extreme entry keys on the COINCIDENT KEY LEVEL within
//	±LocationCoincidePts (the level that makes it a location). An old
//	extreme with no coincident key level is NOT a location: orphan=true —
//	the trade should not exist and the caller drops it.
//	BOX — a box is ONE place: the edge suffix (":top"/":bottom") is
//	stripped, and the place spans the box edges — a loss at either edge
//	blocks the WHOLE box until price leaves it.
//	Everything else (key levels, trigger retest, plain prices) passes
//	through unchanged with lo == hi.
func normalizePlace(in Intent, levels []Level) (key string, lo, hi float64, orphan bool) {
	key = in.AnchorKey
	lo, hi = in.Anchor, in.Anchor
	if key == "" {
		return "", 0, 0, false // a plain ISB — no place at all
	}
	switch {
	case key == string(KindEMA34) || key == string(KindEMA9) || key == string(KindEMA34HTF):
		return key, lo, hi, false // constant line key, loss price for departure
	case strings.HasPrefix(key, string(KindOldExtreme)):
		if kl := nearestKeyLevelWithin(levels, lo, LocationCoincidePts); kl != nil {
			return kl.Key, kl.Price, kl.Price, false
		}
		return "", 0, 0, true // K2: not a location — the trade should not exist
	case strings.HasSuffix(key, ":top") || strings.HasSuffix(key, ":bottom"):
		base := strings.TrimSuffix(strings.TrimSuffix(key, ":top"), ":bottom")
		blo, bhi := boxRange(levels, base, lo)
		return base, blo, bhi, false // the box is ONE place
	}
	return key, lo, hi, false
}

// boxRange returns the live box edges for base (the box key without the
// edge suffix). Fallback: the touched edge itself (lo == hi).
func boxRange(levels []Level, base string, fallback float64) (lo, hi float64) {
	lo, hi = fallback, fallback
	for _, l := range levels {
		switch l.Key {
		case base + ":bottom":
			lo = l.Price
		case base + ":top":
			hi = l.Price
		}
	}
	return lo, hi
}

// nearestKeyLevelWithin is the coincident key level within tol of price
// (K2): the level that makes an old extreme a location.
func nearestKeyLevelWithin(levels []Level, price, tol float64) *Level {
	var best *Level
	for i := range levels {
		l := &levels[i]
		if l.Kind != KindKeyLevel || abs(l.Price-price) > tol {
			continue
		}
		if best == nil || abs(l.Price-price) < abs(best.Price-price) {
			best = l
		}
	}
	return best
}
