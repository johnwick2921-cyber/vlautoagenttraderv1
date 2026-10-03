package trader

import (
	"math"
	"testing"

	"vl/kernel"
	"vl/kernel/mentor"
	"vl/store"
	ntTrader "vl/trader/ninjatrader"
)

// mentoredTrader builds a minimal AutoTrader with the given risk-control
// config — enough for the mentor mode switch surfaces.
func mentoredTrader(t *testing.T, rc store.RiskControlConfig) *AutoTrader {
	t.Helper()
	return &AutoTrader{
		id: "t-mentor",
		config: AutoTraderConfig{
			StrategyConfig: &store.StrategyConfig{RiskControl: rc},
		},
	}
}

// TestMentorContractsForSizeTable pins EVERY tier at the production call site
// (mentorContractsFor). A size tier off by one — the CTO's mutant — fails this.
func TestMentorContractsForSizeTable(t *testing.T) {
	base, conf, big, reduced, swing4h, spentCap, mx := 5, 10, 20, 3, 3, 2, 20
	cases := []struct {
		name string
		in   mentorTierInputs
		want mentorSizeChoice
	}{
		{"base ISB", mentorTierInputs{Setup: "ISB", StopPts: 5.75, TargetPts: 11.5}, mentorSizeChoice{5, "base", ""}},
		{"base PHL", mentorTierInputs{Setup: "PHL", StopPts: 11.25, TargetPts: 22.5}, mentorSizeChoice{5, "base", ""}},
		{"confluence", mentorTierInputs{Setup: "PHL", StopPts: 11.25, Confluence: true}, mentorSizeChoice{10, "confluence", ""}},
		{"big", mentorTierInputs{Setup: "PHL", StopPts: 12, TargetPts: 35, RoomMultiple: 2.5, Confluence: true, HTFAgree: true}, mentorSizeChoice{20, "big", ""}},
		{"big needs 4h AND 1h", mentorTierInputs{Setup: "PHL", StopPts: 12, TargetPts: 35, RoomMultiple: 2.5, Confluence: true, HTFAgree: false}, mentorSizeChoice{10, "confluence", ""}},
		{"big needs room", mentorTierInputs{Setup: "PHL", StopPts: 20, TargetPts: 35, RoomMultiple: 1.5, Confluence: true, HTFAgree: true}, mentorSizeChoice{10, "confluence", ""}},
		{"big needs target", mentorTierInputs{Setup: "PHL", StopPts: 12, TargetPts: 29, RoomMultiple: 2.5, Confluence: true, HTFAgree: true}, mentorSizeChoice{10, "confluence", ""}},
		{"stop in the twenties", mentorTierInputs{Setup: "PLH", StopPts: 23, TargetPts: 46}, mentorSizeChoice{3, "reduced", ""}},
		{"stop at 20 boundary", mentorTierInputs{Setup: "PLH", StopPts: 20, TargetPts: 40}, mentorSizeChoice{3, "reduced", ""}},
		{"stop below twenties", mentorTierInputs{Setup: "PLH", StopPts: 19.5, TargetPts: 40}, mentorSizeChoice{5, "base", ""}},
		{"spent day beats twenties", mentorTierInputs{Setup: "PLH", StopPts: 23, TargetPts: 46, SpentDay: true}, mentorSizeChoice{2, "spent_day", ""}},
		{"SWING4H", mentorTierInputs{Setup: "SWING4H", StopPts: 12, TargetPts: 24}, mentorSizeChoice{3, "swing4h", ""}},
		{"hard cap", mentorTierInputs{Setup: "PHL", StopPts: 12, TargetPts: 35, RoomMultiple: 2.5, Confluence: true, HTFAgree: true}, mentorSizeChoice{20, "big", ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := mentorContractsFor(c.in, base, conf, big, reduced, swing4h, spentCap, mx)
			if err != nil {
				t.Fatalf("mentorContractsFor: %v", err)
			}
			if got.Contracts != c.want.Contracts || got.Tier != c.want.Tier {
				t.Fatalf("got (%d, %s, %q), want (%d, %s)", got.Contracts, got.Tier, got.Why, c.want.Contracts, c.want.Tier)
			}
		})
	}
}

// TestMentorContractsKnobCap: the owner's per-strategy cap wins over the big
// tier (never above mentor_max_contracts).
func TestMentorContractsKnobCap(t *testing.T) {
	in := mentorTierInputs{Setup: "PHL", StopPts: 12, TargetPts: 35, RoomMultiple: 2.5, Confluence: true, HTFAgree: true}
	got, err := mentorContractsFor(in, 5, 10, 20, 3, 3, 2, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got.Contracts != 12 || got.Tier != "big" {
		t.Fatalf("cap: got %d %s, want 12 big", got.Contracts, got.Tier)
	}
	if _, err := mentorContractsFor(in, 5, 10, 20, 3, 3, 2, 0); err == nil {
		t.Fatal("a non-positive max must refuse (fail-closed)")
	}
}

// TestMentorModeSwitchBothDirections: mentor ON gets the mentor ceiling (the
// 0B clamp does NOT apply); mentor OFF keeps the 2-contract AI clamp
// byte-identical (the mentor size must never leak into AI mode).
func TestMentorModeSwitchBothDirections(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	if !at.mentorEnabled() {
		t.Fatal("mentor_mode on must enable")
	}
	if mx, ok := at.mentorMaxContracts(); !ok || mx != mentorMaxContractsDefault {
		t.Fatalf("mentor max = %d %v, want %d true", mx, ok, mentorMaxContractsDefault)
	}
	if got := at.resolveMaxContracts(); got != mentorMaxContractsDefault {
		t.Fatalf("resolveMaxContracts in mentor mode = %d, want %d (the 0B clamp must not apply)", got, mentorMaxContractsDefault)
	}

	off := mentoredTrader(t, store.RiskControlConfig{})
	if off.mentorEnabled() {
		t.Fatal("mentor_mode off must disable (default)")
	}
	if _, ok := off.mentorMaxContracts(); ok {
		t.Fatal("mentor max must not apply with mentor_mode off")
	}
	if got := off.resolveMaxContracts(); got != 1 {
		t.Fatalf("AI-mode resolveMaxContracts = %d, want 1 (the 0B Stage-A size-1 cap, unchanged)", got)
	}
}

// TestMentorAIEntriesOffButMentorDecisionsPass: the mode switch suppresses
// AI opens only; mentor-sourced decisions and closes pass.
func TestMentorAIEntriesOffButMentorDecisionsPass(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	aiOpen := func(action string) *kernel.Decision {
		return &kernel.Decision{Action: action}
	}
	if r := at.mentorSuppressAIEntry(aiOpen("open_long")); r == "" {
		t.Fatal("an AI open must be refused in mentor mode")
	}
	if r := at.mentorSuppressAIEntry(aiOpen("close_short")); r != "" {
		t.Fatalf("a close must never be suppressed: %q", r)
	}
	m := aiOpen("open_long")
	m.MentorSourced = true
	if r := at.mentorSuppressAIEntry(m); r != "" {
		t.Fatalf("a mentor-sourced open must pass: %q", r)
	}
	off := mentoredTrader(t, store.RiskControlConfig{})
	if r := off.mentorSuppressAIEntry(aiOpen("open_long")); r != "" {
		t.Fatalf("AI mode must never suppress: %q", r)
	}
}

// TestMentorNotionalRoundTripsThroughTheExecutorSizing: the synthetic
// decision's notional must come back out of futuresOrderQuantity as exactly
// the mentor contract count at the mentor ceiling (the mutant that restores
// the 2-contract clamp in resolveMaxContracts fails this).
func TestMentorNotionalRoundTripsThroughTheExecutorSizing(t *testing.T) {
	in := mentor.Intent{Action: mentor.PlaceStopEntry, Side: mentor.SideLong,
		Price: 21000, Stop: 20988, Target: 21024, Setup: "ISB", StopPts: 12}
	d := mentor.BuildDecision(in, "MNQ", 17)
	if !d.MentorSourced || d.EntryPrice != in.Price {
		t.Fatalf("decision not mentor-marked: %+v", d)
	}
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	q := futuresOrderQuantity(d.Symbol, d.PositionSizeUSD, in.Price, at.resolveMaxContracts())
	if math.Abs(q-17) > 1e-9 {
		t.Fatalf("mentor quantity = %f, want 17 (the 0B clamp must not touch mentor sizing)", q)
	}
}

// TestMentorScaleOutWireAndFallback: ceil(n/2) goes through the
// reduce_position frame; a missing frame falls back to all-or-nothing B,
// logged + counted (the spec's explicit fallback).
func TestMentorScaleOutWireAndFallback(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	ResetMentorCountersForTest()
	pos := &mentorPosition{Symbol: "MNQ", Side: "long", Entry: 21000, Stop: 20988, R: 12, Contracts: 5, Mode: "B"}

	var took int
	reducePositionWire = func(quantity int) error { took = quantity; return nil }
	if out := at.mentorScaleOut(pos); out != scaleOutSent {
		t.Fatalf("scale-out outcome = %s, want sent", out)
	}
	if took != 3 || pos.Contracts != 2 || !pos.Scaled {
		t.Fatalf("scale-out: took %d, remaining %d, scaled %v — want took 3, remaining 2, scaled", took, pos.Contracts, pos.Scaled)
	}

	// fallback: no frame
	reducePositionWire = nil
	pos2 := &mentorPosition{Symbol: "MNQ", Side: "short", Entry: 21000, Stop: 21012, R: 12, Contracts: 5, Mode: "B"}
	if out := at.mentorScaleOut(pos2); out != scaleOutFallback {
		t.Fatalf("missing frame outcome = %s, want fallback_all_or_nothing", out)
	}
	if pos2.Contracts != 5 {
		t.Fatalf("fallback must keep every contract riding: %d", pos2.Contracts)
	}
	if got := MentorCountSnapshot()["scaleout_fallback"]; got != 1 {
		t.Fatalf("the fallback must be counted once, got %d", got)
	}
}

// TestMentorExitRules: B arms BE at +1R and trails behind closed candles
// (no look-ahead: the arming candle is never trailed against); same-bar
// stop+1R is worse (stop); C never moves the stop.
func TestMentorExitRules(t *testing.T) {
	long := mentorPosition{Symbol: "MNQ", Side: "long", Entry: 100, Stop: 90, R: 10, Contracts: 4, Mode: "B"}
	// arming candle reaches +1R (high 110): arm BE, NO exit, NO trail of THIS
	// candle (its low 95 is below entry — trailing it would look ahead).
	stop, px, why, exited := mentorExitB(long, 110, 95)
	if exited || px != 0 || why != "" {
		t.Fatalf("arming candle: exited=%v px=%v why=%q — must only arm", exited, px, why)
	}
	if stop != long.Entry {
		t.Fatalf("arming candle must move the stop to entry (BE), got %.2f", stop)
	}
	// a candle that does NOT touch the BE stop trails behind its close, no exit.
	long.ArmedBE = true
	long.Stop = long.Entry
	stop, _, _, exited = mentorExitB(long, 103, 101)
	if exited || stop != 101 {
		t.Fatalf("trail candle: exited=%v stop=%.2f — want stop trailed to the closed low 101 (no touch, no exit)", exited, stop)
	}
	// a candle touching the trailed stop exits at the stop.
	long.Stop = 101
	_, px, why, exited = mentorExitB(long, 102, 100.5)
	if !exited || px != 101 || why != "trail" {
		t.Fatalf("trail exit: exited=%v px=%.2f why=%q — want exited at 101 trail", exited, px, why)
	}
	// same-bar worse before arming.
	pre := mentorPosition{Symbol: "MNQ", Side: "long", Entry: 100, Stop: 90, R: 10, Contracts: 4, Mode: "B"}
	_, px, why, exited = mentorExitB(pre, 110, 89)
	if !exited || px != 90 || why != "stop(worse-same-bar)" {
		t.Fatalf("same-bar worse: exited=%v px=%.2f why=%q", exited, px, why)
	}
	// C: the stop never moves.
	c := mentorPosition{Symbol: "MNQ", Side: "short", Entry: 100, Stop: 112, R: 12, Contracts: 4, Mode: "C"}
	for i := 0; i < 5; i++ {
		if _, why, exited := mentorExitC(c, 108, 109); exited || why != "" {
			t.Fatalf("C hold: exited=%v why=%q on a non-stop candle", exited, why)
		}
	}
	if px, why, exited := mentorExitC(c, 111, 113); !exited || px != 112 || why != "stop" {
		t.Fatalf("C stop: exited=%v px=%.2f why=%q", exited, px, why)
	}
}

// TestMentorExitMechSuspensionAppliesToAIOnly: EXIT_MECHS_SUSPENDED (0B)
// suspends the AI mechanisms but never the mentor stop moves — the same last
// hop, two different gates.
func TestMentorExitMechSuspensionAppliesToAIOnly(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	ResetExitMechSuspendNoticeForTest()

	var sent string
	var sentPx float64
	moveStopWire = func(nt *ntTrader.TCPTrader, side string, newStop float64) error {
		sent, sentPx = side, newStop
		return nil
	}
	t.Cleanup(func() { moveStopWire = nil })

	if err := at.mentorMoveStop(&ntTrader.TCPTrader{}, "long", 100.25); err != nil {
		t.Fatal(err)
	}
	if sent != "long" || sentPx != 100.25 {
		t.Fatalf("mentor stop move did not reach the wire (suspended leak): sent=%q px=%.2f", sent, sentPx)
	}
	if !at.exitMechSuspendedRefuse("be40", "test trigger") {
		t.Fatal("the AI suspension must STILL apply to the AI mechanisms")
	}
}

// TestMentorRuleGateR8R9: the injector-side R8/R9 gate (RULES-FIX v3).
// SWING4H: 30–60 allowed, ~100 refused, EXEMPT from the 25-pt ceiling.
// R9: target never smaller than stop; spent-day cap 15 skips a stop over 15.
func TestMentorRuleGateR8R9(t *testing.T) {
	if why := mentorRuleGate(mentor.Intent{Setup: "SWING4H", StopPts: 40, TargetPts: 80}, mentorTierInputs{}); why != "" {
		t.Fatalf("SWING4H 40-pt stop must pass (30–60 allowed): %q", why)
	}
	if why := mentorRuleGate(mentor.Intent{Setup: "SWING4H", StopPts: 60, TargetPts: 120}, mentorTierInputs{}); why != "" {
		t.Fatalf("SWING4H 60-pt stop must pass: %q", why)
	}
	if why := mentorRuleGate(mentor.Intent{Setup: "SWING4H", StopPts: 100, TargetPts: 200}, mentorTierInputs{}); why == "" {
		t.Fatal("SWING4H ~100-pt stop must be refused (R8)")
	}
	// the swing is the ONLY exemption from the 25-pt ceiling
	if why := mentorRuleGate(mentor.Intent{Setup: "PHL", StopPts: 30, TargetPts: 60}, mentorTierInputs{}); why == "" {
		t.Fatal("a non-swing stop over 25 must be refused (R8 ceiling)")
	}
	// R9: target never smaller than stop
	if why := mentorRuleGate(mentor.Intent{Setup: "PHL", StopPts: 12, TargetPts: 10}, mentorTierInputs{}); why == "" {
		t.Fatal("a target smaller than the stop must be refused (R9)")
	}
	if why := mentorRuleGate(mentor.Intent{Setup: "PHL", StopPts: 12, TargetPts: 24}, mentorTierInputs{}); why != "" {
		t.Fatalf("target ≥ stop must pass: %q", why)
	}
	// R9 spent-day cap: stop over 15 skips
	if why := mentorRuleGate(mentor.Intent{Setup: "PHL", StopPts: 18, TargetPts: 36}, mentorTierInputs{SpentDay: true}); why == "" {
		t.Fatal("a spent day must skip any stop over 15 (R9 cap)")
	}
	if why := mentorRuleGate(mentor.Intent{Setup: "PHL", StopPts: 12, TargetPts: 24}, mentorTierInputs{SpentDay: true}); why != "" {
		t.Fatalf("a spent day with a 12-pt stop must pass: %q", why)
	}
	// ISB intents carry no target (0): the target check never fires on them
	if why := mentorRuleGate(mentor.Intent{Setup: "ISB", StopPts: 5.75}, mentorTierInputs{}); why != "" {
		t.Fatalf("a targetless ISB must pass the gate: %q", why)
	}
}

// TestMentorSpentDayClamp: §7 — at most 2 contracts running on a spent day.
func TestMentorSpentDayClamp(t *testing.T) {
	if got := mentorSpentDayClamp(20, 2); got != 2 {
		t.Fatalf("spent-day clamp 20->%d, want 2", got)
	}
	if got := mentorSpentDayClamp(1, 2); got != 1 {
		t.Fatalf("spent-day clamp 1->%d, want 1 (never up-size)", got)
	}
}

// TestMentorSizeForCountsAndLogsTier: the AutoTrader entry point counts the
// tier ("which tier and why") — a silent size choice fails this.
func TestMentorSizeForCountsAndLogsTier(t *testing.T) {
	ResetMentorCountersForTest()
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	in := mentor.Intent{Action: mentor.PlaceStopEntry, Side: mentor.SideLong,
		Price: 21000, Stop: 20988, Target: 21024, Setup: "ISB", StopPts: 12}
	choice, err := at.mentorSizeFor(in, mentorTierInputs{})
	if err != nil {
		t.Fatal(err)
	}
	if choice.Contracts != mentorBaseContractsDefault || choice.Tier != "base" {
		t.Fatalf("choice = %+v", choice)
	}
	if got := MentorCountSnapshot()["tier_base"]; got != 1 {
		t.Fatalf("the tier must be counted once, got %d", got)
	}
	if choice.Why == "" {
		t.Fatal("the tier must carry its why")
	}
}
