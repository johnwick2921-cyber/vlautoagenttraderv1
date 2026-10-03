package trader

import (
	"os"
	"strings"
	"time"

	"vl/kernel"
	"vl/kernel/mentor"
	"vl/market"
	"vl/store"
)

// ── MENTOR P3 — the cycle hook ─────────────────────────────────────────────
//
// One evaluator tick per NEW 1m close. Inert unless the per-strategy
// mentor_mode is ON. Placements stay behind the MENTOR_PLACE env gate until
// P1 (#309) lands — until then the hook sizes and LOGS/COUNTS every intent
// (the size audit the spec demands) and places NOTHING (L4, default OFF).

// mentorPlaceEnv resolves the placement gate (env MENTOR_PLACE, default OFF).
func mentorPlaceEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MENTOR_PLACE"))) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// mentorTick runs the evaluator on the latest 1m bars. It never touches the
// wire unless mentor mode AND MENTOR_PLACE are both on.
func (at *AutoTrader) mentorTick(ctx *kernel.Context) {
	if !at.mentorEnabled() {
		return
	}
	if market.FuturesBarsProvider == nil {
		return
	}
	bars := market.FuturesBarsProvider("MNQ", "1m", 240)
	if len(bars) == 0 {
		return
	}
	last := bars[len(bars)-1]
	if last.OpenTime <= at.mentorLastTickOpen {
		return
	}
	at.mentorLastTickOpen = last.OpenTime

	if at.mentorEval == nil {
		cfg := mentor.DefaultConfig()
		cfg.Enabled = true
		at.mentorEval = mentor.New(cfg)
	}
	intents := at.mentorEval.Tick(bars, last.OpenTime)
	for _, in := range intents {
		switch in.Action {
		case mentor.PlaceStopEntry:
			choice, err := at.mentorSizeFor(in, mentorTierInputs{})
			if err != nil {
				continue
			}
			mentorCount("intent_" + in.Setup)
			if !mentorPlaceEnv() {
				at.logInfof("🧑‍🏫 mentor intent SIZED, NOT PLACED (MENTOR_PLACE off — P1 #309 first): %s %s %d contracts @ %.2f (stop %.2f, target %.2f, tier %s)",
					in.Setup, in.Side, choice.Contracts, in.Price, in.Stop, in.Target, choice.Tier)
				mentorCount("placement_held")
				continue
			}
			at.mentorPlaceIntent(in, choice)
		case mentor.CancelArm, mentor.LevelInvalid:
			// P3 scope: the arm lifecycle (cancel frames, level invalidation
			// bookkeeping) lands with P1; the intent is recorded, never silent.
			mentorCount("intent_" + string(in.Action))
			at.logInfof("🧑‍🏫 mentor %s intent recorded (arm lifecycle lands with P1): %s", in.Action, in.Reason)
		}
	}
}

// mentorPlaceIntent converts a sized intent into a synthetic decision and runs
// the FULL existing pipeline (every admission gate reused, SIM-only untouched).
func (at *AutoTrader) mentorPlaceIntent(in mentor.Intent, choice mentorSizeChoice) {
	d := mentor.BuildDecision(in, "MNQ", choice.Contracts)
	rec := &store.DecisionAction{
		Action:     d.Action,
		Symbol:     d.Symbol,
		Quantity:   0,
		Leverage:   d.Leverage,
		Price:      0,
		StopLoss:   d.StopLoss,
		TakeProfit: d.TakeProfit,
		Confidence: d.Confidence,
		Reasoning:  d.Reasoning,
		Timestamp:  time.Now().UTC(),
		Success:    false,
	}
	if err := at.executeDecisionWithRecord(d, rec); err != nil {
		mentorCount("placement_error")
		at.logErrorf("🧑‍🏫 mentor placement error: %v", err)
		return
	}
	mentorCount("placed_" + choice.Tier)
	at.logInfof("🧑‍🏫 mentor placed: %s %s %d contracts (tier %s)", in.Setup, in.Side, choice.Contracts, choice.Tier)
}
