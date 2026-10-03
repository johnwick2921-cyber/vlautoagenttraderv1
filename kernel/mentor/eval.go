package mentor

import (
	"encoding/json"
	"fmt"
	"time"

	"vl/kernel"
	"vl/market"
)

// State is everything the evaluator carries between ticks. It is rebuildable
// from bars plus the ledger (Replay): the evaluator never stores more than
// this, and a restart recomputes it deterministically from the bar history.
type State struct {
	// Touches holds the first-touch classification per level key (§3).
	Touches map[string]Touch `json:"touches,omitempty"`
	// ISBOnly marks levels invalidated by a wrong-way close — only ISBs may
	// ever trade there after [D5.2 p2 @ 20:48].
	ISBOnly map[string]bool `json:"isb_only,omitempty"`
	// DeletedLevels holds key levels deleted by a CLOSED 1H candle whose BODY
	// closed through (KEY-LEVEL ruling 5b, slide 31–32): they stay deleted for
	// the session. A 1H wick through does NOT delete.
	DeletedLevels map[string]bool `json:"deleted_levels,omitempty"`
	// ISBArms are the live inside-bar orders (stacking counter, §2.1).
	ISBArms map[string]ISBArm `json:"isb_arms,omitempty"`
	// Trigger is the 5m trigger-line state (§5.1).
	Trigger TriggerLine `json:"trigger"`
	// HTF is the §5.4 4h/1h direction state (DS-106, fold item 3).
	HTF HTF `json:"htf,omitempty"`
	// Day is the §7 pre-session verdict latch (DS-106, fold item 4).
	Day DayLatch `json:"day_latch,omitempty"`
	// Swing is the §8 4h-EMA34 swing state (DS-106 slice; rebuildable by
	// replaying bars through SwingTick).
	Swing SwingState `json:"swing,omitempty"`
	// ArmSeq names the next arm.
	ArmSeq int `json:"arm_seq"`
}

// MarshalState renders the state for the ledger (rebuild source).
func MarshalState(s State) ([]byte, error) { return json.Marshal(s) }

// UnmarshalState restores a state from the ledger.
func UnmarshalState(b []byte) (State, error) {
	var s State
	err := json.Unmarshal(b, &s)
	if s.Touches == nil {
		s.Touches = map[string]Touch{}
	}
	if s.ISBOnly == nil {
		s.ISBOnly = map[string]bool{}
	}
	if s.DeletedLevels == nil {
		s.DeletedLevels = map[string]bool{}
	}
	if s.ISBArms == nil {
		s.ISBArms = map[string]ISBArm{}
	}
	return s, err
}

// Evaluator runs the mentor rules once per closed 1m candle. It emits intents
// only; nothing here places an order (P3's injector does). Ticks are pure
// functions of (bars, now, state): the same history always rebuilds the same
// state.
type Evaluator struct {
	Cfg   Config
	State State
}

func New(cfg Config) *Evaluator {
	return &Evaluator{Cfg: cfg, State: State{
		Touches:       map[string]Touch{},
		ISBOnly:       map[string]bool{},
		DeletedLevels: map[string]bool{},
		ISBArms:       map[string]ISBArm{},
	}}
}

// Levels computes the full mentor level set from the 1m history: colour-change
// key levels (§4.3), EMA 34/9 (§8/§11), and the old highs/lows — the bot's own
// swing levels reused [DS-106 §1: kernel/levels_swing.go SwingPointLevels].
func Levels(bars []market.Kline, cfg Config, now int64) []Level {
	var out []Level
	out = append(out, KeyLevels(bars, cfg)...)
	out = append(out, EMALevels(bars, cfg)...)
	for _, d := range kernel.SwingPointLevels(bars, time.UnixMilli(now)) {
		switch d.Kind {
		case kernel.KindSWGH, kernel.KindSWGL:
			out = append(out, Level{
				Key:   fmt.Sprintf("%s:%.2f", KindOldExtreme, d.Price),
				Kind:  KindOldExtreme,
				Price: d.Price,
			})
		}
	}
	return out
}

// withoutDeleted drops the deleted key levels from the level set (deletion is
// by key; recomputed levels re-derive the same keys).
func withoutDeleted(levels []Level, deleted map[string]bool) []Level {
	if len(deleted) == 0 {
		return levels
	}
	out := make([]Level, 0, len(levels))
	for _, l := range levels {
		if !deleted[l.Key] {
			out = append(out, l)
		}
	}
	return out
}

// closedBuckets returns the 5m buckets with the still-forming one dropped
// (B4): a bucket whose close time has not been reached by `now` is forming.
func closedBuckets(bars []market.Kline, now int64, cfg Config) []market.Kline {
	b5 := barsTF(bars, 5)
	if len(b5) > 0 && b5[len(b5)-1].CloseTime >= now {
		return b5[:len(b5)-1]
	}
	return b5
}

// nextLevelBeyond is the §6 target ladder [D3.3 p1 @ 05:07]: the NEAREST level
// beyond price in the trade direction ("the first thing standing in your way").
// 0 = no level beyond.
func nextLevelBeyond(levels []Level, price float64, side Side) float64 {
	best := 0.0
	for _, l := range levels {
		switch side {
		case SideLong:
			if l.Price > price && (best == 0 || l.Price < best) {
				best = l.Price
			}
		case SideShort:
			if l.Price < price && (best == 0 || l.Price > best) {
				best = l.Price
			}
		}
	}
	return best
}

// freshTouch resets a classified touch when the level's price has moved away
// from where it was touched (the EMA case): a touch against the line at one
// price is not a touch of the same line after it drifted.
func freshTouch(tr Touch, lvl Level) Touch {
	if tr.Outcome != TouchNone && abs(lvl.Price-tr.PriceAtTouch) > 0.01 {
		return Touch{LevelKey: lvl.Key}
	}
	return tr
}

// Tick evaluates the newest closed 1m candle. bars is the closed history up to
// now (the bot's BarCache slice); now is the current time for the swing
// detector's closed-bar filter. Returns the intents for this candle.
func (e *Evaluator) Tick(bars []market.Kline, now int64) []Intent {
	if !e.Cfg.Enabled || len(bars) < 2 {
		return nil
	}
	levels := Levels(bars, e.Cfg, now)

	// KEY-LEVEL RULING (item 5b, slide 31–32): a CLOSED 1H candle whose BODY
	// closed through a key level DELETES it; a 1H wick through does not. The
	// deletion is permanent for the session and the level leaves the set at
	// once — it can no longer be touched, located, or laddered to.
	for _, l := range levels {
		if l.Kind != KindKeyLevel || e.State.DeletedLevels[l.Key] {
			continue
		}
		if levelDeletedBy1HBody(l, bars, now) {
			e.State.DeletedLevels[l.Key] = true
		}
	}
	levels = withoutDeleted(levels, e.State.DeletedLevels)

	var out []Intent

	// 5m trigger line advances every 1m close (aggregated 5m bars).
	e.State.Trigger = TriggerTick(e.State.Trigger, barsTF(bars, 5), 5, e.Cfg)

	// §5.4 HTF direction (DS-106, fold item 3): the 4h/1h lines advance with
	// the same bar history.
	e.State.HTF = HTFAdvance(e.State.HTF, barsTF(bars, 240), barsTF(bars, 60), e.Cfg)

	// §7 day gate (fold item 4): Globex run → per-trading-day latch (L1/L2).
	dg := DayGate{SpentPts: e.Cfg.DayGateSpentPts, TargetCapPts: e.Cfg.DayGateTargetCapPts}
	run, haveRun := GlobexRun(bars, now, ctime())
	e.State.Day = LatchDay(e.State.Day, now, ctime(), run, haveRun, HTFConflict(e.State.HTF), dg)

	// the location set grows by the trigger-line retest and the EMA 34
	// location line (fold item 1; R3: 1m default).
	if e.State.Trigger.Dir != "" && e.State.Trigger.Price != 0 {
		levels = append(levels, Level{Key: string(KindTriggerRetest), Kind: KindTriggerRetest, Price: e.State.Trigger.Price})
	}
	if el, ok := EMALocationLevel(bars, e.Cfg); ok {
		levels = append(levels, el)
	}

	// §3: first-touch classification per level.
	for _, lvl := range levels {
		tr := e.State.Touches[lvl.Key]
		if e.State.ISBOnly[lvl.Key] {
			continue // invalid level: no PHL/PLH, and touches need no re-read
		}
		// a MOVING line (EMA) that drifted away from where it was touched is
		// a fresh line for touch purposes — reset the classification.
		tr = freshTouch(tr, lvl)
		intents := TouchTick(&tr, lvl, bars[len(bars)-2].Close, bars[len(bars)-1], e.Cfg)
		e.State.Touches[lvl.Key] = tr
		for _, in := range intents {
			if in.Action == LevelInvalid {
				e.State.ISBOnly[lvl.Key] = true
			}
			out = append(out, in)
		}
	}

	// §2.1: ISB on the last closed pair, direction from the trigger line.
	prev, cur := bars[len(bars)-2], bars[len(bars)-1]
	if IsISB(prev, cur) {
		if dirOK, side, _ := TriggerVerdict(e.State.Trigger, cur.Close); dirOK {
			if _, stopOK, _ := ISBStopVerdict(cur, e.Cfg); stopOK {
				// B4: the 15m/5m conflict reads CLOSED buckets only — the
				// still-forming 5m bucket is dropped [D4.2 p1 @ 05:10: "a
				// 15-minute candle is only confirmed once CLOSED; trade from
				// the next one"].
				if conflict := ISBConflictVerdict(closedBuckets(bars, now, e.Cfg)); !conflict {
					// LOCATION GATE (fold item 1, owner ruling): the ISB's
					// reference candle must touch a real location — mid-air
					// inside bars are not setups.
					if loc, where := LocationVerdict(cur, levels, e.State.Trigger, bars, e.Cfg); loc {
						_ = where
						// fold item 3: entries only with the 4h trigger direction
						// (1h agreeing or silent); fold item 4: a DayOff shuts the
						// machine off for the day.
						if htfOK, htfSide, htfReason := HTFVerdict(e.State.HTF); !htfOK {
							_ = htfReason
						} else if side != "" && side != htfSide {
							// 5m trigger side against the 4h — no entry
						} else if e.State.Day.Verdict == DayOff {
							// day off — no mentor entries today
						} else {
							long, short := ISBOrders(cur, e.Cfg)
							chosen := long
							if side == SideShort {
								chosen = short
							}
							// §6 [D3.3 p1 @ 05:07]: "TARGET LÀ VỀ NHỮNG LEVEL KẾ
							// TIẾP" — a setup gives entry, stop AND target
							// [D4.1 p1 @ 01:39]; no level beyond → no trade.
							if target := nextLevelBeyond(levels, chosen.Price, side); target != 0 {
								chosen.Target = target
							} else {
								return out // missing target — not a setup [D4.1 p1 @ 01:45]
							}
							e.State.ArmSeq++
							id := fmt.Sprintf("isb-%d", e.State.ArmSeq)
							e.State.ISBArms[id] = ISBArm{FirstBar: cur}
							chosen.ArmID = id
							out = append(out, chosen)
						}
					}
				} else {
					out = append(out, Intent{Action: CancelArm, Reason: "15m/5m ISB conflict — no trade [D4.2 p1 @ 14:35]"})
				}
			}
		}
	}

	// Stacking ticks for live arms (§2.1): bodies staying inside the first ISB.
	for id, arm := range e.State.ISBArms {
		if arm.Inside < 0 {
			delete(e.State.ISBArms, id)
			continue
		}
		for _, in := range ISBStackTick(&arm, cur, e.Cfg) {
			in.ArmID = id
			out = append(out, in)
		}
		if arm.Inside < 0 {
			delete(e.State.ISBArms, id)
		} else {
			e.State.ISBArms[id] = arm
		}
	}

	// §2.2: PHL/PLH from a fresh reject touch against an old extreme,
	// gated by trigger side, mid-range and the setup's own gates.
	oldExtremes := oldExtremeIndexes(levels, bars)
	for _, lvl := range levels {
		tr := e.State.Touches[lvl.Key]
		if tr.Outcome != TouchReject || e.State.ISBOnly[lvl.Key] {
			continue
		}
		// LOCATION GATE (fold item 1): a PHL/PLH entry level must be a real
		// location — a bare old high/low is not one.
		if !levelIsLocation(lvl, levels) {
			continue
		}
		side, price, ok := RejectEntry(tr, e.Cfg)
		if !ok {
			continue
		}
		if dirOK, trigSide, _ := TriggerVerdict(e.State.Trigger, price); !dirOK || trigSide != "" && trigSide != side {
			continue
		}
		if allowed, _ := SetupPermittedVerdict("PHL", levels, price, e.Cfg); !allowed {
			continue
		}
		for _, ex := range oldExtremes {
			in, ok, _ := PHLPLHGated(tr, ex.level, ex.idx, len(bars)-1, e.Cfg, e.State.HTF, e.State.Day.Verdict, dg)
			if ok {
				out = append(out, in)
				break
			}
		}
	}

	// §8 SWING4H (DS-106 slice) on FINAL 5m bars only — the still-forming
	// bucket is excluded (CTO wiring ruling 2026-10-03).
	out = append(out, runSwing(e, bars, now)...)

	return out
}

// oldExtreme holds an old high/low and its index in the 1m history.
type oldExtreme struct {
	level Level
	idx   int
}

// oldExtremeIndexes maps old-extreme levels to the index of their defining bar.
// The level was drawn at a swing point; the index is the last bar whose high
// (low) equals the level price — the ≥3-candle distance [D4.1 p1 @ 09:40
// written] is measured from there.
func oldExtremeIndexes(levels []Level, bars []market.Kline) []oldExtreme {
	var out []oldExtreme
	for _, l := range levels {
		if l.Kind != KindOldExtreme {
			continue
		}
		idx := -1
		for i := len(bars) - 1; i >= 0; i-- {
			if abs(bars[i].High-l.Price) < 0.26 || abs(bars[i].Low-l.Price) < 0.26 {
				idx = i
				break
			}
		}
		if idx >= 0 {
			out = append(out, oldExtreme{level: l, idx: idx})
		}
	}
	return out
}

// runSwing evaluates the §8 4h-EMA34 swing on FINAL 5m bars (the forming
// bucket excluded — CTO wiring ruling 2026-10-03) and returns its intents.
func runSwing(e *Evaluator, bars []market.Kline, now int64) []Intent {
	return SwingTick(&e.State.Swing, closedBuckets(bars, now, e.Cfg), e.Cfg.Swing, now)
}
