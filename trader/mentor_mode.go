package trader

import (
	"fmt"
	"strings"
	"sync"

	"vl/kernel"
	"vl/kernel/mentor"
	"vl/telemetry"
	ntTrader "vl/trader/ninjatrader"
)

// ── MENTOR MODE P3 (owner order 2026-10-02 22:2x, option 2) ────────────────
//
// Mentor mode sizes like the mentor (5 contracts normally, up to 20 on strong
// setups, SIM ONLY). The AI mode keeps the 0B size clamp and its suspended
// exits, byte-identical. mentor_mode is per strategy and OFF by default (L4).
//
// Every refusal and every size choice is LOGGED and COUNTED (tier + why).

// Mentor size-table defaults (CTO defaults; the owner changes them per
// strategy via the knobs above).
const (
	mentorBaseContractsDefault       = 5
	mentorConfluenceContractsDefault = 10
	mentorBigContractsDefault        = 20 // the hard cap
	mentorReducedContractsDefault    = 3
	mentorSwing4HContractsDefault    = 3
	mentorSpentDayContractsDefault   = 2
	mentorMaxContractsDefault        = 20

	mentorTargetBigPts       = 30.0 // target ≥ this for the big tier
	mentorRoomBigMultiple    = 2.0  // room ≥ this × risk for the big tier
	mentorStopTwentiesMinPts = 20.0 // stop 20–25 pts → reduced [D3.3 p1 @ 01:09]
	mentorStopTwentiesMaxPts = 25.0
	// R8 (RULES-FIX v3): a SWING4H stop of 30–60 pts is allowed, ~100 is
	// refused, and SWING4H is EXEMPT from the 25-pt ceiling.
	mentorSwingStopMaxPts = 60.0
	// R9 (RULES-FIX v3): the target is never smaller than the stop; on a spent
	// day (cap 15) any setup whose stop is over 15 is skipped.
	mentorSpentDayStopCapPts = 15.0
)

// mentorTierInputs is everything the size table reads. The trader computes
// these from the intent + evaluator filters; tests pin the table itself.
type mentorTierInputs struct {
	Setup        string  // "ISB", "PHL", "PLH", "SWING4H"
	StopPts      float64 // |entry - stop|
	TargetPts    float64 // |target - entry|
	RoomMultiple float64 // reward/risk actually available
	Confluence   bool    // box/zone + key level + 5m trigger agreeing (§6)
	HTFAgree     bool    // 4h AND 1h agree
	SpentDay     bool    // §7 spent day
}

// mentorSizeChoice is the tier decision: contracts, the tier name and why.
type mentorSizeChoice struct {
	Contracts int
	Tier      string
	Why       string
}

// mentorContractsFor is THE size table (pure — the call site the tests pin).
// Order matters: big → confluence → reduced (twenties/spent) → swing4h → base,
// every branch capped at maxContracts. A non-positive max refuses (fail-closed).
func mentorContractsFor(in mentorTierInputs, base, conf, big, reduced, swing4h, spentCap, maxContracts int) (mentorSizeChoice, error) {
	if maxContracts <= 0 {
		return mentorSizeChoice{}, fmt.Errorf("mentor size: mentor_max_contracts not set (max=%d)", maxContracts)
	}
	clamp := func(n int) int {
		if n > maxContracts {
			return maxContracts
		}
		return n
	}
	if in.Confluence && in.HTFAgree && in.RoomMultiple >= mentorRoomBigMultiple && in.TargetPts >= mentorTargetBigPts {
		return mentorSizeChoice{Contracts: clamp(big), Tier: "big", Why: fmt.Sprintf(
			"confluence + 4h&1h agree + room %.1fx ≥ %.0fx + target %.1f pts ≥ %.0f (hard cap %d)",
			in.RoomMultiple, mentorRoomBigMultiple, in.TargetPts, mentorTargetBigPts, maxContracts)}, nil
	}
	if in.Confluence {
		return mentorSizeChoice{Contracts: clamp(conf), Tier: "confluence", Why: "box/zone + key level + 5m trigger agreeing (§6)"}, nil
	}
	if in.SpentDay {
		return mentorSizeChoice{Contracts: clamp(spentCap), Tier: "spent_day", Why: "§7 spent day — hold 1–2 only [D5.1 p1 @ 16:13]"}, nil
	}
	if in.StopPts >= mentorStopTwentiesMinPts && in.StopPts <= mentorStopTwentiesMaxPts {
		return mentorSizeChoice{Contracts: clamp(reduced), Tier: "reduced", Why: fmt.Sprintf(
			"stop %.1f pts in the twenties → reduce size or don't trade [D3.3 p1 @ 01:09]", in.StopPts)}, nil
	}
	if strings.EqualFold(in.Setup, "SWING4H") {
		return mentorSizeChoice{Contracts: clamp(swing4h), Tier: "swing4h", Why: "SWING4H setup — 3 [D5.2 p1]"}, nil
	}
	return mentorSizeChoice{Contracts: clamp(base), Tier: "base", Why: fmt.Sprintf(
		"base setup at a location (setup %s, stop %.1f pts)", in.Setup, in.StopPts)}, nil
}

// mentorCounters records every size tier and every refusal for the session
// ("which tier and why" — the spec's audit trail).
var (
	mentorCounterMu sync.Mutex
	mentorCounters  = map[string]int{}
)

func mentorCount(kind string) {
	mentorCounterMu.Lock()
	mentorCounters[kind]++
	mentorCounterMu.Unlock()
}

// MentorCountSnapshot returns a copy of the mentor counters (tests + reports).
func MentorCountSnapshot() map[string]int {
	mentorCounterMu.Lock()
	defer mentorCounterMu.Unlock()
	out := make(map[string]int, len(mentorCounters))
	for k, v := range mentorCounters {
		out[k] = v
	}
	return out
}

// ResetMentorCountersForTest clears the per-process mentor counters.
func ResetMentorCountersForTest() {
	mentorCounterMu.Lock()
	mentorCounters = map[string]int{}
	mentorCounterMu.Unlock()
}

// mentorKnobs resolves the per-strategy size knobs with the CTO defaults.
func (at *AutoTrader) mentorKnobs() (base, conf, big, reduced, swing4h, spentCap, maxContracts int) {
	base, conf, big = mentorBaseContractsDefault, mentorConfluenceContractsDefault, mentorBigContractsDefault
	reduced, swing4h, spentCap = mentorReducedContractsDefault, mentorSwing4HContractsDefault, mentorSpentDayContractsDefault
	maxContracts = mentorMaxContractsDefault
	if at.config.StrategyConfig == nil {
		return
	}
	rc := at.config.StrategyConfig.RiskControl
	if rc.MentorBaseContracts > 0 {
		base = rc.MentorBaseContracts
	}
	if rc.MentorConfluenceContracts > 0 {
		conf = rc.MentorConfluenceContracts
	}
	if rc.MentorBigContracts > 0 {
		big = rc.MentorBigContracts
	}
	if rc.MentorReducedContracts > 0 {
		reduced = rc.MentorReducedContracts
	}
	if rc.MentorSwing4HContracts > 0 {
		swing4h = rc.MentorSwing4HContracts
	}
	if rc.MentorSpentDayContracts > 0 {
		spentCap = rc.MentorSpentDayContracts
	}
	if rc.MentorMaxContracts > 0 {
		maxContracts = rc.MentorMaxContracts
	}
	return
}

// mentorEnabled reports the per-strategy switch (OFF by default).
func (at *AutoTrader) mentorEnabled() bool {
	return at.config.StrategyConfig != nil && at.config.StrategyConfig.RiskControl.MentorMode
}

// mentorMaxContracts is the mentor-mode ceiling for resolveMaxContracts: the
// 0B size clamp (maxFuturesContracts=2) does NOT apply to a mentor-mode
// trader. AI mode is untouched (mentorEnabled false → the existing path).
func (at *AutoTrader) mentorMaxContracts() (int, bool) {
	if !at.mentorEnabled() {
		return 0, false
	}
	_, _, _, _, _, _, mx := at.mentorKnobs()
	return mx, true
}

// mentorSuppressAIEntry is the AI-entries-off half of the mode switch: a
// mentor-mode trader takes NO AI open decisions (closes, flattens and safety
// paths are untouched). Returns the refusal, "" when allowed.
func (at *AutoTrader) mentorSuppressAIEntry(d *kernel.Decision) string {
	if !at.mentorEnabled() {
		return ""
	}
	if d.MentorSourced {
		return ""
	}
	switch d.Action {
	case "open_long", "open_short":
		mentorCount("ai_entry_suppressed")
		telemetry.IncGateBlock(at.id, "mentor_ai_entries_off")
		return "mentor_mode: AI entries are OFF — every entry comes from the mentor evaluator"
	}
	return ""
}

// mentorRuleGate is the injector-side R8/R9 gate: it refuses intents the
// evaluator should never have let through, fail-closed, before any sizing.
// (R8) SWING4H: stop 30–60 allowed, ≥100 refused, exempt from the 25-pt
// ceiling. (R9) the target is never smaller than the stop; on a spent day
// (cap 15) a stop over 15 skips. Returns the skip reason, "" when the intent
// passes.
func mentorRuleGate(in mentor.Intent, extra mentorTierInputs) string {
	swing := strings.EqualFold(in.Setup, "SWING4H")
	if swing {
		if in.StopPts > mentorSwingStopMaxPts {
			return fmt.Sprintf("R8: SWING4H stop %.1f pts — ~100 is refused (allowed 30–60, no 25-pt ceiling for the swing)", in.StopPts)
		}
	} else {
		if in.StopPts > mentorStopTwentiesMaxPts {
			return fmt.Sprintf("R8: stop %.1f pts over the 25-pt ceiling — skip (the swing is the only exemption)", in.StopPts)
		}
	}
	if in.TargetPts > 0 && in.TargetPts < in.StopPts {
		return fmt.Sprintf("R9: target %.1f pts smaller than the stop %.1f pts — never trade it [D1.2 p1 @ 07:48–09:00]", in.TargetPts, in.StopPts)
	}
	if extra.SpentDay && in.StopPts > mentorSpentDayStopCapPts {
		return fmt.Sprintf("R9: spent day cap 15 — stop %.1f pts skips", in.StopPts)
	}
	return ""
}

// mentorSizeFor chooses the tier for one intent, logs it and counts it. The
// tier inputs beyond the intent's own fields (confluence, 4h/1h agreement,
// spent day) come from the caller's evaluator context.
func (at *AutoTrader) mentorSizeFor(in mentor.Intent, extra mentorTierInputs) (mentorSizeChoice, error) {
	extra.Setup = in.Setup
	extra.StopPts = in.StopPts
	extra.TargetPts = in.TargetPts
	if extra.RoomMultiple == 0 && in.TargetPts > 0 && in.StopPts > 0 {
		extra.RoomMultiple = in.TargetPts / in.StopPts
	}
	base, conf, big, reduced, swing4h, spentCap, mx := at.mentorKnobs()
	choice, err := mentorContractsFor(extra, base, conf, big, reduced, swing4h, spentCap, mx)
	if err != nil {
		mentorCount("refused_no_max")
		at.logErrorf("🧑‍🏫 mentor size refused: %v", err)
		return mentorSizeChoice{}, err
	}
	mentorCount("tier_" + choice.Tier)
	at.logInfof("🧑‍🏫 mentor size: %d contracts (tier %s) — %s [setup %s, stop %.1f pts]",
		choice.Contracts, choice.Tier, choice.Why, in.Setup, in.StopPts)
	return choice, nil
}

// ── EXITS (§6) ─────────────────────────────────────────────────────────────

// mentorPosition is one open mentor trade's exit-driver state.
type mentorPosition struct {
	Symbol    string
	Side      string // "long" | "short"
	Entry     float64
	Stop      float64
	R         float64
	Contracts int
	Mode      string // "B", "A-resonance", "C"
	ArmedBE   bool   // +1R seen → stop is at entry
	Scaled    bool   // the ceil(n/2) scale-out already happened
}

// mentorStopEntryPlacer is the broker's stop-entry surface (E7 frame on the
// NT8 TCP trader). A broker without it refuses the mentor entry fail-closed.
type mentorStopEntryPlacer interface {
	PlaceStopEntry(symbol, side string, quantity float64, stopPx, sl, tp float64) (map[string]interface{}, error)
}

// reducePositionWire is the LAST hop before the socket for the mentor
// scale-out. DS-101's reduce_position frame plugs in here; until the AddOn
// carries it the function is nil and the driver falls back to all-or-nothing
// B, logged + counted (the spec's explicit fallback).
var reducePositionWire func(quantity int) error

// mentorScaleOutOutcome reports what a scale-out attempt did.
type mentorScaleOutOutcome string

const (
	scaleOutSent     mentorScaleOutOutcome = "sent"
	scaleOutFallback mentorScaleOutOutcome = "fallback_all_or_nothing"
)

// mentorScaleOut attempts the ceil(n/2) reduction. With no reduce_position
// frame it returns the logged fallback.
func (at *AutoTrader) mentorScaleOut(pos *mentorPosition) mentorScaleOutOutcome {
	take := (pos.Contracts + 1) / 2 // ceil(n/2)
	if reducePositionWire == nil {
		mentorCount("scaleout_fallback")
		at.logWarnf("🧑‍🏫 mentor scale-out skipped: the AddOn carries no reduce_position frame — all-or-nothing B for %s (%d contracts ride)",
			pos.Symbol, pos.Contracts)
		return scaleOutFallback
	}
	if err := reducePositionWire(take); err != nil {
		mentorCount("scaleout_failed")
		at.logErrorf("🧑‍🏫 mentor scale-out FAILED: %v — all-or-nothing B for %s", err, pos.Symbol)
		return scaleOutFallback
	}
	mentorCount("scaleout_sent")
	pos.Contracts -= take
	pos.Scaled = true
	at.logInfof("🧑‍🏫 mentor scale-out: %d contracts off at +1R, %d riding for %s", take, pos.Contracts, pos.Symbol)
	return scaleOutSent
}

// mentorExitB applies the B rules to ONE closed 1m candle (pure). It returns
// the new stop and whether the position exits (with the exit price + reason).
// Rules: before arming → stop or +1R (arm BE); after arming → BE/trail check
// against the NEW stop behind the candle just closed (the trail moves AFTER a
// candle closes — no look-ahead).
func mentorExitB(pos mentorPosition, h, l float64) (newStop float64, exitPrice float64, exitReason string, exited bool) {
	newStop = pos.Stop
	long := pos.Side == "long"
	var hitStop, hit1R bool
	if long {
		hitStop = l <= pos.Stop
		hit1R = h >= pos.Entry+pos.R
	} else {
		hitStop = h >= pos.Stop
		hit1R = l <= pos.Entry-pos.R
	}
	if !pos.ArmedBE {
		if hitStop && hit1R {
			return pos.Stop, pos.Stop, "stop(worse-same-bar)", true
		}
		if hitStop {
			return pos.Stop, pos.Stop, "stop", true
		}
		if hit1R {
			return pos.Entry, 0, "", false // arm BE (trail from the NEXT candle)
		}
		return pos.Stop, 0, "", false
	}
	// armed: the stop is BE or a trailed level; a candle touching it exits at
	// the stop (no intra-bar order knowledge).
	if hitStop {
		return pos.Stop, pos.Stop, "trail", true
	}
	if long {
		if l > pos.Stop {
			newStop = l // trail behind the closed candle
		}
	} else {
		if h < pos.Stop {
			newStop = h
		}
	}
	return newStop, 0, "", false
}

// mentorExitC applies the confluence hold: the stop NEVER moves (pure).
func mentorExitC(pos mentorPosition, l, h float64) (exitPrice float64, exitReason string, exited bool) {
	var hitStop bool
	if pos.Side == "long" {
		hitStop = l <= pos.Stop
	} else {
		hitStop = h >= pos.Stop
	}
	if hitStop {
		return pos.Stop, "stop", true
	}
	return 0, "", false
}

// mentorSpentDayClamp is the §7 rule: at most 2 contracts running on a spent
// day (pure — the caller sizes the entry with it; here for the exit side the
// caller re-sizes the ride before placing).
func mentorSpentDayClamp(contracts, spentCap int) int {
	if spentCap <= 0 {
		return contracts
	}
	if contracts > spentCap {
		return spentCap
	}
	return contracts
}

// mentorMoveStop sends a mentor stop move through the SAME last hop the AI
// mechanisms use, but WITHOUT the 0B suspension gate: EXIT_MECHS_SUSPENDED does
// NOT apply to mentor mode (the AI mechanisms keep it — both sides are pinned
// by TestMentorExitMechSuspensionAppliesToAIOnly).
func (at *AutoTrader) mentorMoveStop(nt *ntTrader.TCPTrader, side string, newStop float64) error {
	return moveStopWire(nt, side, newStop)
}

// mentorLogPositionState dumps the driver state for the daily log.
func (at *AutoTrader) mentorLogPositionState(pos *mentorPosition, event string) {
	at.logInfof("🧑‍🏫 mentor exit driver [%s]: %s side=%s entry=%.2f stop=%.2f R=%.2f contracts=%d mode=%s armedBE=%v scaled=%v",
		event, pos.Symbol, pos.Side, pos.Entry, pos.Stop, pos.R, pos.Contracts, pos.Mode, pos.ArmedBE, pos.Scaled)
}
