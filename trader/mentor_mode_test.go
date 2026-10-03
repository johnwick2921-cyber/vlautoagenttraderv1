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
		// S9 (D5.2 p2 @05:21): a strong day cuts EVERY tier to 1–2, even a big
		// confluence setup — checked FIRST.
		{"strong day cuts big to 2", mentorTierInputs{Setup: "PHL", StopPts: 12, TargetPts: 35, RoomMultiple: 2.5, Confluence: true, HTFAgree: true, StrongDay: true}, mentorSizeChoice{2, "strong_day", ""}},
		{"strong day cuts base to 2", mentorTierInputs{Setup: "ISB", StopPts: 5.75, TargetPts: 11.5, StrongDay: true}, mentorSizeChoice{2, "strong_day", ""}},
		// ISB at an old high/low → reduce size, tier 3 (owner ruling 00:1x CT,
		// D4.1 p1 written rule 2).
		{"ISB at old extreme → 3", mentorTierInputs{Setup: "ISB", StopPts: 5.75, TargetPts: 11.5, ISBOldExtreme: true}, mentorSizeChoice{3, "isb_old_extreme", ""}},
		{"old extreme beats confluence", mentorTierInputs{Setup: "ISB", StopPts: 11.25, Confluence: true, ISBOldExtreme: true}, mentorSizeChoice{3, "isb_old_extreme", ""}},
		{"strong day beats old extreme", mentorTierInputs{Setup: "ISB", StopPts: 5.75, TargetPts: 11.5, StrongDay: true, ISBOldExtreme: true}, mentorSizeChoice{2, "strong_day", ""}},
		{"spent day beats old extreme", mentorTierInputs{Setup: "ISB", StopPts: 5.75, TargetPts: 11.5, SpentDay: true, ISBOldExtreme: true}, mentorSizeChoice{2, "spent_day", ""}},
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

// TestMentorExitRules (EXIT-SPEC-v3): B arms BE at +0.5R, scales at +1R, and
// trails behind closed candles only AFTER the scale (no look-ahead: the
// arming/scale candles are never trailed against); same-bar stop+level is
// worse (stop); a BE touch exits "be", a trailed-stop touch exits "trail";
// trail off (video-8 knob) never moves the stop; C never moves the stop and
// never scales even at +2R.
func TestMentorExitRules(t *testing.T) {
	long := mentorPosition{Symbol: "MNQ", Side: "long", Entry: 100, Stop: 90, R: 10, Contracts: 4, Mode: "B"}
	// +0.5R candle (high 105, low 95): arm BE, NO exit, NO scale, NO trail of
	// THIS candle (its low 95 is below entry — trailing it would look ahead).
	stop, px, why, exited, scaled := mentorExitB(long, 105, 95, true)
	if exited || px != 0 || why != "" || scaled {
		t.Fatalf("arming candle: exited=%v px=%v why=%q scaled=%v — must only arm BE", exited, px, why, scaled)
	}
	if stop != long.Entry {
		t.Fatalf("arming candle must move the stop to entry (BE), got %.2f", stop)
	}
	long.ArmedBE = true
	long.Stop = long.Entry
	// +1R candle (high 110, low above BE): scale signal, stop stays at BE.
	stop, px, why, exited, scaled = mentorExitB(long, 110, 100.5, true)
	if exited || px != 0 || why != "" || !scaled {
		t.Fatalf("scale candle: exited=%v px=%v why=%q scaled=%v — want the +1R scale signal only", exited, px, why, scaled)
	}
	if stop != long.Entry {
		t.Fatalf("the scale candle must keep the stop at BE, got %.2f", stop)
	}
	long.Scaled = true
	// a candle that does NOT touch the BE stop trails behind its close, no exit.
	stop, _, _, exited, _ = mentorExitB(long, 102, 100.5, true)
	if exited || stop != 100.5 {
		t.Fatalf("trail candle: exited=%v stop=%.2f — want stop trailed to the closed low 100.5 (no touch, no exit)", exited, stop)
	}
	// a candle touching the trailed stop exits at the stop.
	long.Stop = 100.5
	_, px, why, exited, _ = mentorExitB(long, 101.5, 100.4, true)
	if !exited || px != 100.5 || why != "trail" {
		t.Fatalf("trail exit: exited=%v px=%.2f why=%q — want exited at 100.5 trail", exited, px, why)
	}
	// BE touch (scaled, never trailed): exits "be" at entry.
	be := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 100, R: 10, Contracts: 2, Mode: "B", ArmedBE: true, Scaled: true}
	_, px, why, exited, _ = mentorExitB(be, 101, 99.9, true)
	if !exited || px != 100 || why != "be" {
		t.Fatalf("BE exit: exited=%v px=%.2f why=%q — want exited at 100 be", exited, px, why)
	}
	// trail off (video-8 knob): after the scale the stop NEVER moves.
	off := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 100, R: 10, Contracts: 2, Mode: "B", ArmedBE: true, Scaled: true}
	stop, _, _, exited, _ = mentorExitB(off, 102, 101, false)
	if exited || stop != 100 {
		t.Fatalf("trail off: exited=%v stop=%.2f — want the stop pinned at BE 100", exited, stop)
	}
	// same-bar worse before arming (stop + halfR in one candle).
	pre := mentorPosition{Symbol: "MNQ", Side: "long", Entry: 100, Stop: 90, R: 10, Contracts: 4, Mode: "B"}
	_, px, why, exited, _ = mentorExitB(pre, 105, 89, true)
	if !exited || px != 90 || why != "stop(worse-same-bar)" {
		t.Fatalf("same-bar worse: exited=%v px=%.2f why=%q", exited, px, why)
	}
	// short side: BE arm at −0.5R, scale at −1R, trail over the high.
	short := mentorPosition{Symbol: "MNQ", Side: "short", Entry: 100, Stop: 110, R: 10, Contracts: 4, Mode: "B"}
	stop, _, _, exited, _ = mentorExitB(short, 96, 94.5, true)
	if exited || stop != 100 {
		t.Fatalf("short arming candle: exited=%v stop=%.2f — want BE at 100", exited, stop)
	}
	short.ArmedBE = true
	short.Stop = 100
	_, _, _, exited, scaled = mentorExitB(short, 96, 89.5, true)
	if exited || !scaled {
		t.Fatalf("short scale candle: exited=%v scaled=%v — want the −1R scale signal", exited, scaled)
	}
	short.Scaled = true
	stop, _, _, exited, _ = mentorExitB(short, 99.5, 98, true)
	if exited || stop != 99.5 {
		t.Fatalf("short trail: exited=%v stop=%.2f — want stop trailed to the closed high 99.5", exited, stop)
	}
	// C: the stop never moves and nothing scales, even at +0.5R/+1R/+2R.
	c := mentorPosition{Symbol: "MNQ", Side: "short", Origin: "ISB", Entry: 100, Stop: 112, R: 12, Contracts: 4, Mode: "C"}
	for i := 0; i < 5; i++ {
		if _, why, exited := mentorExitC(c, 108, 109); exited || why != "" {
			t.Fatalf("C hold: exited=%v why=%q on a non-stop candle", exited, why)
		}
	}
	for _, cc := range [][2]float64{{100, 95}, {95, 90}, {90, 75}} {
		if _, why, exited := mentorExitC(c, cc[0], cc[1]); exited || why != "" {
			t.Fatalf("C must hold through +0.5R/+1R/+2R candles, exited=%v why=%q", exited, why)
		}
	}
	if px, why, exited := mentorExitC(c, 111, 113); !exited || px != 112 || why != "stop" {
		t.Fatalf("C stop: exited=%v px=%.2f why=%q", exited, px, why)
	}
}

// TestMentorISBFillCandleExit [D1.4 p1 @12:23–13:27]: for an ISB trade the
// first partial is MANDATORY when the candle that FILLED you closes — it
// REPLACES the +1R scale for ISB trades only; the trail rules after it are
// unchanged. Non-ISB origins keep the +1R scale.
func TestMentorISBFillCandleExit(t *testing.T) {
	// fill candle closes at 103 — between entry and +0.5R: the partial fires
	// anyway (bắt buộc), the stop does NOT move.
	pos := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 90, R: 10, Contracts: 4, Mode: "B"}
	stop, px, why, exited, scaled := mentorExitB(pos, 103, 101, true)
	if exited || px != 0 || why != "" || !scaled {
		t.Fatalf("ISB fill candle close: exited=%v px=%v why=%q scaled=%v — want the mandatory partial only", exited, px, why, scaled)
	}
	if stop != pos.Stop {
		t.Fatalf("a sub-+0.5R fill candle must NOT move the stop, got %.2f", stop)
	}
	// fill candle closes above +0.5R: BE arm and the partial happen together.
	pos2 := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 90, R: 10, Contracts: 4, Mode: "B"}
	stop, _, _, _, scaled = mentorExitB(pos2, 105, 101, true)
	if !scaled || stop != 100 {
		t.Fatalf("ISB fill candle above +0.5R: scaled=%v stop=%.2f — want the partial AND the BE arm", scaled, stop)
	}
	// the stop on the fill candle still wins (worse same bar: stop + halfR).
	pos3 := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 90, R: 10, Contracts: 4, Mode: "B"}
	_, px, why, exited, scaled = mentorExitB(pos3, 105, 89, true)
	if !exited || px != 90 || why != "stop(worse-same-bar)" || scaled {
		t.Fatalf("ISB fill candle through the stop: exited=%v px=%.2f why=%q scaled=%v — the stop must win", exited, px, why, scaled)
	}
	// a non-ISB origin keeps the +1R scale: the same 103 candle only holds.
	pos4 := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "PHL", Entry: 100, Stop: 90, R: 10, Contracts: 4, Mode: "B"}
	stop, _, _, exited, scaled = mentorExitB(pos4, 103, 101, true)
	if exited || scaled || stop != 90 {
		t.Fatalf("PHL origin on the same candle: exited=%v scaled=%v stop=%.2f — no partial below +1R", exited, scaled, stop)
	}
}

// TestMentorResonanceFork (EXIT-SPEC-v3 A): a PHL/PLH fill followed by an ISB
// in the SAME direction within 3 candles flips the position to A — stop to BE
// at that moment, then A NEVER trails and NEVER scales (candles that would
// scale/trail in B are ignored); it exits only on a BE touch.
func TestMentorResonanceFork(t *testing.T) {
	ResetMentorCountersForTest()
	pos := &mentorPosition{Symbol: "MNQ", Side: "long", Origin: "PHL", Entry: 100, Stop: 90, R: 10, Contracts: 5, Mode: "B"}
	if !mentorMaybeArmResonance(pos, "long", 2) {
		t.Fatal("ISB same side within 3 candles of a PHL fill must arm resonance")
	}
	if pos.Mode != "A-resonance" || !pos.ArmedBE || pos.Stop != pos.Entry {
		t.Fatalf("resonance arm: mode=%s armed=%v stop=%.2f — want A-resonance, BE stop 100", pos.Mode, pos.ArmedBE, pos.Stop)
	}
	if got := MentorCountSnapshot()["resonance_armed"]; got != 1 {
		t.Fatalf("resonance_armed must be counted once, got %d", got)
	}
	// A holds through candles that would arm/scale/trail in B (+1R, +2R highs).
	for _, cc := range [][2]float64{{105, 101}, {110, 101}, {120, 102}} {
		if px, why, exited := mentorExitA(*pos, cc[1], cc[0]); exited || px != 0 || why != "" {
			t.Fatalf("A must hold a %v candle: exited=%v px=%.2f why=%q", cc, exited, px, why)
		}
	}
	// only the BE touch exits, at entry, with the resonance reason.
	if px, why, exited := mentorExitA(*pos, 99.5, 102); !exited || px != 100 || why != "resonance_be" {
		t.Fatalf("A BE exit: exited=%v px=%.2f why=%q — want exited at 100 resonance_be", exited, px, why)
	}
	// refused arms: too late (4th candle), wrong side, non-PHL/PLH origin, the
	// fill bar itself, already-armed.
	if mentorMaybeArmResonance(pos, "long", 4) {
		t.Fatal("an ISB 4 candles after the fill must NOT arm resonance")
	}
	if mentorMaybeArmResonance(&mentorPosition{Origin: "PHL", Mode: "B"}, "short", 2) {
		t.Fatal("an ISB against the position must NOT arm resonance")
	}
	if mentorMaybeArmResonance(&mentorPosition{Origin: "PHL", Mode: "B"}, "long", 0) {
		t.Fatal("the fill bar itself must NOT arm resonance (within 3 candles AFTER the fill)")
	}
	if mentorMaybeArmResonance(&mentorPosition{Origin: "ISB", Mode: "B"}, "long", 2) {
		t.Fatal("a non-PHL/PLH origin must NOT arm resonance")
	}
	if mentorMaybeArmResonance(&mentorPosition{Origin: "PLH", Mode: "A-resonance"}, "long", 2) {
		t.Fatal("an already-resonant position must NOT re-arm")
	}
}

// TestMentorTrailTFKnob: the B candle trail runs on 1m (default) and 30s/45s;
// off = the video-8 legacy (superseded); an unknown value refuses to trail
// fail-closed and is counted.
func TestMentorTrailTFKnob(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	if got := at.mentorTrailTF(); got != "1m" {
		t.Fatalf("unset knob → %q, want the 1m default", got)
	}
	for _, tc := range []struct{ in, want string }{
		{"1m", "1m"}, {"30s", "30s"}, {"45s", "45s"}, {"off", "off"}, {" OFF ", "off"},
	} {
		at.config.StrategyConfig.RiskControl.MentorTrailTF = tc.in
		if got := at.mentorTrailTF(); got != tc.want {
			t.Fatalf("trail_tf %q → %q, want %q", tc.in, got, tc.want)
		}
	}
	ResetMentorCountersForTest()
	at.config.StrategyConfig.RiskControl.MentorTrailTF = "garbage"
	if got := at.mentorTrailTF(); got != "off" {
		t.Fatalf("bad trail_tf → %q, want off (fail-closed, no trail)", got)
	}
	if got := MentorCountSnapshot()["trail_tf_bad_value"]; got != 1 {
		t.Fatalf("the bad trail_tf must be counted once, got %d", got)
	}
	if !mentorTrailEnabled("1m") || mentorTrailEnabled("off") {
		t.Fatal("mentorTrailEnabled: 1m must trail, off must not")
	}
}

// TestMentorExitMechSuspensionAppliesToAIOnly: EXIT_MECHS_SUSPENDED (0B)
// suspends the AI mechanisms but never the mentor stop moves — the same last
// hop, two different gates.
func TestMentorExitMechSuspensionAppliesToAIOnly(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	ResetExitMechSuspendNoticeForTest()

	// the never-widen guard needs a wired open stop (99 → 100.25 tightens).
	mentorOpenStopSource = func() (float64, bool) { return 99, true }
	t.Cleanup(func() { mentorOpenStopSource = nil })

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
