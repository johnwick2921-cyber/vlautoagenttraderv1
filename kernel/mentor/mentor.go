// Package mentor is the mentor-mode evaluator (PLAN.md v1, P2): a pure,
// deterministic rule engine that emits trade intents from recorded bars.
//
// Every rule cites /home/hoang/mm-course/notes-local/00-METHOD.md as
// [D<lesson> p<part> @ mm:ss]. The package NEVER places orders — DS-102's
// injector (P3) converts intents into synthetic kernel.Decisions fed through
// the existing executeDecisionWithRecord pipeline. All behaviour sits behind
// Config knobs; Config.Enabled (mentor_mode) defaults to false, and with it
// off nothing in this package is consulted (L4: byte-identical bot).
package mentor

import "vl/market"

// Action is what the evaluator asks the outside world to do. It is an intent,
// not an execution: no order, wire frame or ledger write happens here.
type Action string

const (
	// PlaceStopEntry asks the injector to rest a stop-entry order at Price
	// (side, stop and target as computed by the rules).
	PlaceStopEntry Action = "place_stop_entry"
	// CancelArm asks the injector to cancel a previously emitted arm
	// (ArmID names it). Reason says which rule demands the cancel.
	CancelArm Action = "cancel_arm"
	// LevelInvalid marks a level invalid per §3 [D5.2 p1 @ 19:51]:
	// only ISBs may ever trade there after.
	LevelInvalid Action = "level_invalid"
)

// Side is the trade direction.
type Side string

const (
	SideLong  Side = "long"
	SideShort Side = "short"
)

// LevelKind is the mentor-level taxonomy (local enum; the bot's kernel.LevelKind
// is a different, wider set — see DS-106 §1 for the mapping).
type LevelKind string

const (
	// KindKeyLevel is a colour-change OPEN line on RTH bars (§4.3).
	KindKeyLevel LevelKind = "key_level"
	// KindOldExtreme is an old high/low — the bot's swing levels reused
	// (kernel/levels_swing.go SwingPointLevels).
	KindOldExtreme LevelKind = "old_extreme"
	// KindEMA34 / KindEMA9 — EMA values promoted to levels (§8, §10, §11).
	KindEMA34 LevelKind = "ema34"
	KindEMA9  LevelKind = "ema9"
	// KindEMA34HTF is the location-gate EMA 34 on a HIGHER timeframe
	// (knob ema34_tf, default 4h; never 1m — location fold item 1).
	KindEMA34HTF LevelKind = "ema34_htf"
	// KindTriggerRetest is the 5m trigger-line retest location (§5.1
	// [D3.4 p2 @ 20:51]: the retest is a level confirm).
	KindTriggerRetest LevelKind = "trigger_retest"
)

// Level is one mentor level line. Lo/Hi are equal (a line); AtTime is the bar
// open time that produced it (recency and the ≥3-candle PHL/PLH distance use it).
type Level struct {
	Key    string // stable id "<kind>:<price>@<session>" — state joins on this
	Kind   LevelKind
	Price  float64
	AtTime int64
}

// Intent is one evaluator output.
type Intent struct {
	Action Action
	Reason string // cites the rule, never empty for emits
	Side   Side   // for PlaceStopEntry
	Symbol string // "MNQ" — the mentor instrument (00-METHOD header)

	// PlaceStopEntry fields.
	Price  float64 // stop-entry trigger price
	Stop   float64 // stop-loss
	Target float64 // take-profit level

	// CancelArm / LevelInvalid fields.
	ArmID    string
	LevelKey string
}

// Config is every knob. Enabled is mentor_mode and defaults to false (L4):
// while false the evaluator is never consulted and the bot is byte-identical.
type Config struct {
	Enabled bool

	// Levels (PLAN v1 §1).
	KeyLevelTFMinutes int     // TF of the colour-change walk; default 1 (the RTH chart the mentor draws on — DS-108 §2.1 measured ~200 colour changes per RTH day on 1m)
	KeyLevelRTHOnly   bool    // walk RTH bars only; default true
	KeyLevelPrunePts  float64 // prune pairs closer than this, keep the more recent; default 20 (§4.3 step 5)
	EMAPeriod34       int     // default 34
	EMAPeriod9        int     // default 9
	EMATFMinutes      int     // TF the EMA lines are computed on; default 1

	// EMALocationTFMinutes is the LOCATION-GATE EMA 34 timeframe (fold item
	// 1): default 240 (4h), allowed 60/15/5 — NEVER 1 (the owner's ruling:
	// setups happen only at important levels).
	EMALocationTFMinutes int

	// Touch / close-side (PLAN v1 §2).
	TouchBandPts float64 // literal-touch band around a level; default 4.0 (the bot's 16-tick touch band)

	// ISB (PLAN v1 §3).
	ISBBufferPts   float64 // order buffer beyond the wick extremes; default 1.5 [D2.1 p1 @ 05:36]
	ISBStopMinPts  float64 // stop below this is too small — skip; default 5 [D3.2 p1 @ 14:18]
	ISBTwentiesPts float64 // a stop at/above this ("in the twenties") must not be taken; default 20 [D4.1 p1 @ 05:41]

	// PHL/PLH (PLAN v1 §3).
	PHLMinCandlesFromExtreme int     // entry at least N candles from the old extreme; default 3 [D4.1 p1 @ 09:40 written]
	PHLTargetShyPts          float64 // target this far short of the old extreme; default 5 (Q2 knob; worked example 5–15 [D2.2 p1 @ 06:50])

	// Filters (PLAN v1 §4).
	StopCeilingPts float64 // hard stop ceiling; default 25 [D3.3 p1 @ 02:04]
	RoomMultiple   float64 // room rule: reward >= RoomMultiple x risk; default 2 [D5.3 p1 @ 09:16]
	RangeGapPts    float64 // mid-range: levels bracketing price within this gap both sides; default 0 = disabled

	// §7 day gate knobs (fold item 4, DS-106): the method defaults until the
	// routed Config integration lands.
	DayGateSpentPts     float64 // run >= this before the open = spent; default 300 [D5.1 p1 @ 15:57]
	DayGateTargetCapPts float64 // spent-day target cap; default 15 ("15 điểm bán, 10 điểm bán")
}

// DefaultConfig returns the mentor defaults per PLAN v1 (knob values start from
// DS-108's replay per the dispatch; Enabled stays false).
func DefaultConfig() Config {
	return Config{
		Enabled: false,

		KeyLevelTFMinutes: 1,
		KeyLevelRTHOnly:   true,
		KeyLevelPrunePts:  20,
		EMAPeriod34:       34,
		EMAPeriod9:        9,
		EMATFMinutes:      1,

		EMALocationTFMinutes: 240,

		TouchBandPts: 0, // §3: wait for the LITERAL touch [D3.3 p1 @ 00:13]; §8's "KHÔNG ĐƯỢC GẦN ĐỤNG" is the same strictness

		ISBBufferPts:   1.5,
		ISBStopMinPts:  5,
		ISBTwentiesPts: 20,

		PHLMinCandlesFromExtreme: 3,
		PHLTargetShyPts:          5,

		StopCeilingPts: 25,
		RoomMultiple:   2,
		RangeGapPts:    0,

		DayGateSpentPts:     300,
		DayGateTargetCapPts: 15,
	}
}

// barsTF aggregates 1m bars into the given timeframe on CLOCK-ALIGNED buckets
// in CT (B1, CTO review): 5m/15m/1h floor the open time to the TF; 4h anchors
// to the CME session open 17:00 CT (17–21, 21–01, 01–05, 05–09, 09–13, 13–16)
// the way NT8 draws them. Buckets never re-anchor when the window slides or a
// gap appears.
func barsTF(bars []market.Kline, tfMin int) []market.Kline {
	if tfMin <= 1 || len(bars) == 0 {
		return bars
	}
	ms := int64(tfMin) * 60_000
	out := make([]market.Kline, 0, len(bars)/tfMin)
	var cur *market.Kline
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}
	for _, b := range bars {
		bucket := bucketOpen(b.OpenTime, tfMin)
		if cur == nil || bucket != cur.OpenTime {
			flush()
			c := b
			c.OpenTime = bucket
			c.CloseTime = bucket + ms - 1
			cur = &c
			continue
		}
		if b.High > cur.High {
			cur.High = b.High
		}
		if b.Low < cur.Low {
			cur.Low = b.Low
		}
		cur.Close = b.Close
		cur.Volume += b.Volume
	}
	flush()
	return out
}

// bucketOpen floors a CT-based epoch-millis open time to its TF bucket. For
// 240 minutes the anchor is the CME session open at 17:00 CT (B1).
func bucketOpen(openMs int64, tfMin int) int64 {
	t := openMs / 60_000 // minutes since epoch, CT basis (DS-108 §1.2)
	if tfMin == 240 {
		const sess = 17 * 60 // 17:00 CT
		d := t - sess
		day := d / (24 * 60)
		rem := d % (24 * 60)
		if rem < 0 {
			day--
			rem += 24 * 60
		}
		return (day*(24*60) + sess + (rem/240)*240) * 60_000
	}
	return (t / int64(tfMin)) * int64(tfMin) * 60_000
}
