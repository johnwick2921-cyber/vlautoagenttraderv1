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
	// ISBArms are the live inside-bar orders (stacking counter, §2.1).
	ISBArms map[string]ISBArm `json:"isb_arms,omitempty"`
	// Trigger is the 5m trigger-line state (§5.1).
	Trigger TriggerLine `json:"trigger"`
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
		Touches: map[string]Touch{},
		ISBOnly: map[string]bool{},
		ISBArms: map[string]ISBArm{},
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
	var out []Intent

	// 5m trigger line advances every 1m close (aggregated 5m bars).
	e.State.Trigger = TriggerTick(e.State.Trigger, barsTF(bars, 5), e.Cfg)

	// the location set grows by the trigger-line retest and the EMA34-HTF
	// line (fold item 1).
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
			in, ok, _ := PHLPLH(tr, ex.level, ex.idx, len(bars)-1, e.Cfg)
			if ok {
				out = append(out, in)
				break
			}
		}
	}
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
