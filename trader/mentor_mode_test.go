package trader

import (
	"fmt"
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

// TestMentorSplitLegs (CTO ruling 2026-10-03 05:42Z): n → leg 1 = ceil(n/2)
// with its own TP; leg 2 = the runner. n = 1 → a single leg. runnerCap (spent
// day: 2) caps the runner. Parity: ceil(n/2) equals DS-108's replay half.
func TestMentorSplitLegs(t *testing.T) {
	cases := []struct {
		n, cap, want1, want2 int
	}{
		{0, 0, 0, 0},
		{1, 0, 1, 0},
		{2, 0, 1, 1},
		{3, 0, 2, 1},
		{4, 0, 2, 2},
		{5, 0, 3, 2},
		{20, 0, 10, 10},
		{5, 2, 3, 2}, // spent day: at most 2 run after leg 1
		{6, 2, 3, 2}, // the runner is capped, not the total
		{20, 2, 10, 2},
	}
	for _, c := range cases {
		l1, l2 := mentorSplitLegs(c.n, c.cap)
		if l1 != c.want1 || l2 != c.want2 {
			t.Fatalf("mentorSplitLegs(%d, %d) = (%d, %d), want (%d, %d)", c.n, c.cap, l1, l2, c.want1, c.want2)
		}
	}
}

// TestMentorLeg1TPForC: C sets leg 1's TP at ≥1:2 AT ENTRY.
func TestMentorLeg1TPForC(t *testing.T) {
	if got := mentorLeg1TPForC(100, 10, "long"); got != 120 {
		t.Fatalf("C leg-1 TP long = %.2f, want 120 (2R)", got)
	}
	if got := mentorLeg1TPForC(100, 10, "short"); got != 80 {
		t.Fatalf("C leg-1 TP short = %.2f, want 80 (2R)", got)
	}
}

// TestMentorApplyExitResult: a failed or unwired modify_bracket / move_stop is
// logged and counted; wired actions reach their wires per leg.
func TestMentorApplyExitResult(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	ResetMentorCountersForTest()
	tp := 103.0

	// unwired: both actions counted, never sent.
	at.mentorApplyExitResult(&ntTrader.TCPTrader{}, "long", mentorExitResult{
		MoveStops: []string{"leg1", "leg2"}, NewStop: 100, ModifyTP: &tp,
	})
	snap := MentorCountSnapshot()
	if snap["move_stop_unwired"] != 2 || snap["modify_bracket_unwired"] != 1 {
		t.Fatalf("unwired actions must be counted (2 move_stop, 1 modify): %v", snap)
	}
	// wired: each leg moves and the TP modifies.
	ResetMentorCountersForTest()
	var moved []string
	var modLeg string
	var modTP float64
	moveStopWire = func(nt *ntTrader.TCPTrader, side string, newStop float64) error {
		moved = append(moved, side)
		return nil
	}
	modifyBracketWire = func(leg string, newTP float64) error { modLeg, modTP = leg, newTP; return nil }
	t.Cleanup(func() { moveStopWire = nil; modifyBracketWire = nil })
	at.mentorApplyExitResult(&ntTrader.TCPTrader{}, "long", mentorExitResult{
		MoveStops: []string{"leg1", "leg2"}, NewStop: 100, ModifyTP: &tp,
	})
	if len(moved) != 2 || modLeg != "leg1" || modTP != 103 {
		t.Fatalf("wired actions: moved=%v modLeg=%q modTP=%.2f", moved, modLeg, modTP)
	}
	// a failing modify is counted, not silent.
	ResetMentorCountersForTest()
	modifyBracketWire = func(leg string, newTP float64) error { return fmt.Errorf("boom") }
	at.mentorApplyExitResult(&ntTrader.TCPTrader{}, "long", mentorExitResult{ModifyTP: &tp})
	if got := MentorCountSnapshot()["modify_bracket_failed"]; got != 1 {
		t.Fatalf("the failed modify must be counted once, got %d", got)
	}
}

// TestMentorConfirmLegProtection: a leg whose protection cannot be confirmed
// on the next snapshot is FLATTENED (that leg only, fail-closed).
func TestMentorConfirmLegProtection(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	ResetMentorCountersForTest()
	var flattened []string
	flattenLegWire = func(leg string) error { flattened = append(flattened, leg); return nil }
	t.Cleanup(func() { flattenLegWire = nil })

	// no confirmation source → EVERY leg is flattened.
	at.mentorConfirmLegProtection([]string{"leg1", "leg2"})
	if len(flattened) != 2 {
		t.Fatalf("nil protection source must flatten both legs, flattened=%v", flattened)
	}
	snap := MentorCountSnapshot()
	if snap["leg_protection_lost_leg1"] != 1 || snap["leg_protection_lost_leg2"] != 1 {
		t.Fatalf("the lost protection must be counted per leg: %v", snap)
	}
	// leg1 confirmed, leg2 not → only leg2 flattens.
	ResetMentorCountersForTest()
	flattened = nil
	mentorLegProtectedSource = func(leg string) bool { return leg == "leg1" }
	t.Cleanup(func() { mentorLegProtectedSource = nil })
	at.mentorConfirmLegProtection([]string{"leg1", "leg2"})
	if len(flattened) != 1 || flattened[0] != "leg2" {
		t.Fatalf("only the unconfirmed leg flattens, flattened=%v", flattened)
	}
}

// TestMentorExitRules (EXIT-SPEC-v3 on SPLIT LEGS): at +0.5R BOTH legs' stops
// arm BE; leg 1's +1R TP candle marks the runner's trail start (the bracket
// closes leg 1 — no wire action); the runner trails behind closed candles only
// AFTER that (no look-ahead); same-bar stop+level is worse (stop); a BE touch
// exits "be", a trailed-stop touch "trail"; trail off (video-8 knob) never
// moves the stop; C never moves the stop, even at +2R.
func TestMentorExitRules(t *testing.T) {
	long := mentorPosition{Symbol: "MNQ", Side: "long", Entry: 100, Stop: 90, R: 10, Leg1: 2, Leg2: 2, Mode: "B"}
	// +0.5R candle (high 105, low 95): arm BE for BOTH legs, no exit, no leg-1
	// crossing, no trail of THIS candle (its low 95 is below entry).
	res := mentorExitB(long, 104, 105, 95, true)
	if res.Exited || res.ExitPrice != 0 || res.ExitReason != "" || res.Leg1AtTarget {
		t.Fatalf("arming candle: %+v — must only arm BE", res)
	}
	if res.NewStop != long.Entry || len(res.MoveStops) != 2 {
		t.Fatalf("arming candle must move BOTH legs to BE 100: %+v", res)
	}
	long.ArmedBE = true
	long.Stop = long.Entry
	// +1R candle (high 110, low above BE): leg 1's TP is hit — the bracket
	// closes leg 1 (no wire action); the runner trails from the NEXT candle.
	res = mentorExitB(long, 109, 110, 100.5, true)
	if res.Exited || !res.Leg1AtTarget || res.ModifyTP != nil {
		t.Fatalf("leg-1-at-target candle: %+v", res)
	}
	if res.NewStop != long.Entry {
		t.Fatalf("the +1R candle must keep the stop at BE, got %.2f", res.NewStop)
	}
	long.Scaled = true
	// a candle that does NOT touch the BE stop trails the RUNNER behind its close.
	res = mentorExitB(long, 101.5, 102, 100.5, true)
	if res.Exited || res.NewStop != 100.5 {
		t.Fatalf("trail candle: %+v — want the runner trailed to the closed low 100.5", res)
	}
	if len(res.MoveStops) != 1 || res.MoveStops[0] != "leg2" {
		t.Fatalf("only the runner trails, got %v", res.MoveStops)
	}
	// a candle touching the trailed stop exits at the stop.
	long.Stop = 100.5
	res = mentorExitB(long, 101.4, 101.5, 100.4, true)
	if !res.Exited || res.ExitPrice != 100.5 || res.ExitReason != "trail" {
		t.Fatalf("trail exit: %+v — want exited at 100.5 trail", res)
	}
	// BE touch (after leg 1's target, never trailed): exits "be" at entry.
	be := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 100, R: 10, Mode: "B", ArmedBE: true, Scaled: true}
	res = mentorExitB(be, 100.9, 101, 99.9, true)
	if !res.Exited || res.ExitPrice != 100 || res.ExitReason != "be" {
		t.Fatalf("BE exit: %+v — want exited at 100 be", res)
	}
	// trail off (video-8 knob): after leg 1's target the stop NEVER moves.
	off := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 100, R: 10, Mode: "B", ArmedBE: true, Scaled: true}
	res = mentorExitB(off, 101.5, 102, 101, false)
	if res.Exited || res.NewStop != 100 || len(res.MoveStops) != 0 {
		t.Fatalf("trail off: %+v — want the stop pinned at BE 100", res)
	}
	// same-bar worse before arming (stop + halfR in one candle).
	pre := mentorPosition{Symbol: "MNQ", Side: "long", Entry: 100, Stop: 90, R: 10, Mode: "B"}
	res = mentorExitB(pre, 104, 105, 89, true)
	if !res.Exited || res.ExitPrice != 90 || res.ExitReason != "stop(worse-same-bar)" {
		t.Fatalf("same-bar worse: %+v", res)
	}
	// short side: BE arm at −0.5R for both legs; leg 1 at −1R; runner trails
	// over the high.
	short := mentorPosition{Symbol: "MNQ", Side: "short", Entry: 100, Stop: 110, R: 10, Mode: "B"}
	res = mentorExitB(short, 95, 96, 94.5, true)
	if res.Exited || res.NewStop != 100 || len(res.MoveStops) != 2 {
		t.Fatalf("short arming candle: %+v — want both stops at BE 100", res)
	}
	short.ArmedBE = true
	short.Stop = 100
	res = mentorExitB(short, 90, 96, 89.5, true)
	if res.Exited || !res.Leg1AtTarget {
		t.Fatalf("short leg-1 candle: %+v — want the −1R leg-1 mark", res)
	}
	short.Scaled = true
	res = mentorExitB(short, 98, 99.5, 98, true)
	if res.Exited || res.NewStop != 99.5 || len(res.MoveStops) != 1 || res.MoveStops[0] != "leg2" {
		t.Fatalf("short trail: %+v — want the runner trailed to the closed high 99.5", res)
	}
	// C: the stop never moves, even at +0.5R/+1R/+2R (leg 1's TP is set at
	// entry — mentorLeg1TPForC; nothing modifies it).
	c := mentorPosition{Symbol: "MNQ", Side: "short", Origin: "ISB", Entry: 100, Stop: 112, R: 12, Mode: "C"}
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

// TestMentorISBFillCandleExit [D1.4 p1 @12:23–13:27]: for an ISB trade leg 1
// is closed at the FILL candle's close — modify_bracket of leg 1's TP to the
// current price (a limit at or through the market). The runner continues.
// Non-ISB origins keep leg 1's +1R TP.
func TestMentorISBFillCandleExit(t *testing.T) {
	// fill candle closes at 103 — between entry and +0.5R: leg 1's TP is
	// modified to the close; the runner's stop does NOT move.
	pos := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 90, R: 10, Mode: "B"}
	res := mentorExitB(pos, 103, 103, 101, true)
	if res.Exited || res.ModifyTP == nil || *res.ModifyTP != 103 {
		t.Fatalf("ISB fill candle close: %+v — want leg1 TP modified to the close 103", res)
	}
	if res.NewStop != pos.Stop || len(res.MoveStops) != 0 {
		t.Fatalf("a sub-+0.5R fill candle must NOT move the stops: %+v", res)
	}
	// fill candle closes above +0.5R: BE for both legs AND the modify.
	pos2 := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 90, R: 10, Mode: "B"}
	res = mentorExitB(pos2, 105, 105, 101, true)
	if res.ModifyTP == nil || *res.ModifyTP != 105 || res.NewStop != 100 || len(res.MoveStops) != 2 {
		t.Fatalf("ISB fill candle above +0.5R: %+v — want the modify AND both stops at BE", res)
	}
	// the stop on the fill candle still wins (worse same bar: stop + halfR).
	pos3 := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "ISB", Entry: 100, Stop: 90, R: 10, Mode: "B"}
	res = mentorExitB(pos3, 104, 105, 89, true)
	if !res.Exited || res.ExitPrice != 90 || res.ExitReason != "stop(worse-same-bar)" || res.ModifyTP != nil {
		t.Fatalf("ISB fill candle through the stop: %+v — the stop must win", res)
	}
	// a non-ISB origin keeps leg 1's +1R TP: the same 103 candle only holds.
	pos4 := mentorPosition{Symbol: "MNQ", Side: "long", Origin: "PHL", Entry: 100, Stop: 90, R: 10, Mode: "B"}
	res = mentorExitB(pos4, 103, 103, 101, true)
	if res.Exited || res.ModifyTP != nil || res.NewStop != 90 || len(res.MoveStops) != 0 {
		t.Fatalf("PHL origin on the same candle: %+v — no modify below +1R", res)
	}
}

// TestMentorResonanceFork (EXIT-SPEC-v3 A on SPLIT LEGS): a PHL/PLH fill
// followed by an ISB in the SAME direction within 3 candles flips the position
// to A — both stops to BE at that moment and leg 1's TP modified OUT to the
// runner's target (no 1:1 scale-out). Then A NEVER trails (candles that would
// trail in B are ignored); it exits only on a BE touch.
func TestMentorResonanceFork(t *testing.T) {
	ResetMentorCountersForTest()
	pos := &mentorPosition{Symbol: "MNQ", Side: "long", Origin: "PHL", Entry: 100, Stop: 90, Target: 130, R: 10, Mode: "B"}
	armed, modifyTP := mentorMaybeArmResonance(pos, "long", 2)
	if !armed {
		t.Fatal("ISB same side within 3 candles of a PHL fill must arm resonance")
	}
	if modifyTP != 130 {
		t.Fatalf("the resonance modify must push leg 1's TP to the runner's target 130, got %.2f", modifyTP)
	}
	if pos.Mode != "A-resonance" || !pos.ArmedBE || pos.Stop != pos.Entry {
		t.Fatalf("resonance arm: mode=%s armed=%v stop=%.2f — want A-resonance, BE stop 100", pos.Mode, pos.ArmedBE, pos.Stop)
	}
	if got := MentorCountSnapshot()["resonance_armed"]; got != 1 {
		t.Fatalf("resonance_armed must be counted once, got %d", got)
	}
	// A holds through candles that would trail in B (+1R, +2R highs).
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
	if armed, _ := mentorMaybeArmResonance(pos, "long", 4); armed {
		t.Fatal("an ISB 4 candles after the fill must NOT arm resonance")
	}
	if armed, _ := mentorMaybeArmResonance(&mentorPosition{Origin: "PHL", Mode: "B"}, "short", 2); armed {
		t.Fatal("an ISB against the position must NOT arm resonance")
	}
	if armed, _ := mentorMaybeArmResonance(&mentorPosition{Origin: "PHL", Mode: "B"}, "long", 0); armed {
		t.Fatal("the fill bar itself must NOT arm resonance (within 3 candles AFTER the fill)")
	}
	if armed, _ := mentorMaybeArmResonance(&mentorPosition{Origin: "ISB", Mode: "B"}, "long", 2); armed {
		t.Fatal("a non-PHL/PLH origin must NOT arm resonance")
	}
	if armed, _ := mentorMaybeArmResonance(&mentorPosition{Origin: "PLH", Mode: "A-resonance"}, "long", 2); armed {
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
