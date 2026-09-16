package kernel

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// CLASS 38 (2026-09-01) — PROMPT/VALIDATOR CONTRACT MISMATCH.
//
// Evidence: planner_rejected_prompts rows 78, 79, 80 — one ASIA read, three
// attempts, three DIFFERENT rejects, each one a place where the prompt (or a
// hint it quotes) offers something the validator refuses:
//
//	78  17:47:32 attempt 1  scenario[0].confirm2.rule "1m_mss" not allowed for
//	                        breakdown_continue — entry law: … "2x5m legal ONLY here"
//	                        → the hint names the DEATH/FLIP spelling of the token
//	                          in a message about the CONFIRM field.
//	79  17:53:33 attempt 2  scenario[0].confirm2.rule "2x5m" invalid
//	                        (touch|1x5m_close|2x5m_close|1m_mss|time_hold)
//	                        → the model copied the hint's token verbatim.
//	80  18:00:41 attempt 3  arm legs on breakdown_continue —
//	                        arm_legs_sweep_reclaim_only
//	                        → the schema line offers "legs":[…] on EVERY scenario
//	                          with no condition qualifier; the sweep_reclaim-only
//	                          rule lives ONLY in plan_doc.go.
//
// The model is not the defect; the contract is. These assertions fail on the
// pre-class-38 tree.

// bareRuleToken matches a confirm-rule token spelled in a form the confirm enum
// does NOT contain. \b…\b means "2x5m_close" never matches \b2x5m\b (the "_" is
// a word character), so only genuinely bare spellings trip this.
var bareRuleToken = regexp.MustCompile(`\b(2x5m|1x5m|2x|15m|5m_close|5m-close|2x_5m)\b`)

// TestClass38PinRows78to80 is the row-78/79/80 pin: the three exact defects,
// asserted against the live entry-law table and the live rendered prompt.
func TestClass38PinRows78to80(t *testing.T) {
	// ---- row 78 + 79: the confirm-field hint must speak the confirm enum ----
	// entry_law.go Style strings are quoted VERBATIM into the rejection the
	// model reads ("… not allowed for %s — entry law: %s"). A Style naming a
	// token that confirm.rule/confirm2.rule cannot hold is an instruction the
	// model is punished for following.
	for _, cond := range KnownConditions() {
		law, ok := EntryLawFor(cond)
		if !ok {
			continue
		}
		if m := bareRuleToken.FindString(law.Style); m != "" {
			t.Errorf("entry law Style for %q names %q — not a confirm.rule enum member "+
				"(touch|1x5m_close|2x5m_close|1m_mss|time_hold). Row 78/79: the model copied this token into confirm2.rule and was rejected.\n  Style: %s",
				cond, m, law.Style)
		}
	}

	// The breakdown law must name the token in its enum form explicitly.
	bd, ok := EntryLawFor("breakdown_continue")
	if !ok {
		t.Fatal("breakdown_continue missing from the entry law table")
	}
	if !strings.Contains(bd.Style, "2x5m_close") {
		t.Errorf("breakdown_continue Style must name 2x5m_close (the confirm enum form), got: %s", bd.Style)
	}

	// The repair excerpt the model reads on attempt 2 (row 79's prompt) must
	// speak the same enum.
	if m := bareRuleToken.FindString(RepairEntryConfirmLaw); m != "" {
		t.Errorf("RepairEntryConfirmLaw names bare token %q — row 79 is exactly this token echoed back into confirm2.rule.\n  %s", m, RepairEntryConfirmLaw)
	}

	// ---- row 80: the schema must qualify legs, and the prose must forbid it ----
	prompt := plannerOutputContract(8, 5, true, true)

	idx := strings.Index(prompt, `"legs"`)
	if idx < 0 {
		t.Fatal(`the rendered output contract has no "legs" field — locate the schema line before editing`)
	}
	// The qualifier must ride WITH the field, the way fvg{}/breakdown{} carry
	// theirs. Checking the whole line would pass on the unrelated
	// "condition": "…|sweep_reclaim|…" enum earlier in the same line, so the
	// window starts at the field itself.
	legsWindow := prompt[idx:min(idx+400, len(prompt))]
	if !strings.Contains(legsWindow, "ONLY if condition is sweep_reclaim") {
		t.Errorf(`the "legs" schema field carries NO condition qualifier, while its siblings fvg{} and breakdown{} on the same line DO ("REQUIRED iff …"). `+
			`plan_doc.go rejects legs on every non-sweep_reclaim condition (arm_legs_sweep_reclaim_only) — row 80.`+"\n  field renders as: %s", legsWindow[:min(200, len(legsWindow))])
	}

	// The ARMED ORDERS prose pushes breakdown_continue toward arm{}; it must
	// also say those arms are SINGLE.
	for _, want := range []string{"arm SINGLE", "no legs"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the rendered prompt never says %q — breakdown_continue/breakup_continue arms are single-leg, and nothing in the prompt states it (row 80: 24 breakdown + 11 reject instances in 72h)", want)
		}
	}

	// The tokens the dispatch counted as absent from the rendered prompt
	// (counts before this wave: "EXACTLY 2 legs" 0 · "split contract" 0).
	for _, want := range []string{"EXACTLY 2 legs", "split contract"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the rendered prompt never states %q — the split contract is invisible to the author", want)
		}
	}
}

// TestClass38DeathFlipVocabularyIsDeclared — death/flip.rule genuinely has its
// OWN enum (2x5m|5m_close, plan_doc.go conditionRules) which is NOT the confirm
// enum. Two vocabularies for one concept is the trap that produced row 79, so
// the prompt must say so beside the death/flip lines rather than leave the
// reader to infer it.
func TestClass38DeathFlipVocabularyIsDeclared(t *testing.T) {
	prompt := plannerOutputContract(8, 5, true, true)
	if !strings.Contains(prompt, `"rule": "2x5m|5m_close"`) {
		t.Fatal("death/flip schema line not found — locate before editing")
	}
	if !strings.Contains(prompt, "death/flip rules use their OWN vocabulary") {
		t.Error("the prompt offers death/flip rule tokens 2x5m|5m_close and confirm rule tokens touch|1x5m_close|2x5m_close|1m_mss|time_hold with no statement that they are DIFFERENT enums — row 79 is the model moving a token between them")
	}
}

// NO-TRADE BAND (2026-09-02, rewritten under the 2026-09-03 ruling) — the
// prompt must STATE the machine's lunch window, resolved, so the author knows
// it exists.
//
// SUPERSEDED SPEC: this used to require the OUTPUT CONTRACT to state the
// windows, on the reasoning that "the model cannot list a window it was not
// shown". The ruling inverted that — the model must not list them at all — so
// what survives is the weaker and still necessary claim: the window is stated
// somewhere in the prompt, and it comes from the resolver rather than a typed
// copy. Where it is stated is the no-trade gate block.
func TestNoTradeContractRendersResolvedWindows(t *testing.T) {
	ls, le := LunchWindowCT()
	prompt := BuildPlannerPrompt(PlannerInput{
		TradeDate: "2026-09-03", Session: SessionNY, Price: 29000, DATR: 300,
		Now: time.Date(2026, 9, 3, 8, 30, 0, 0, CTLocation()),
	})
	if !strings.Contains(prompt, "lunch "+ls+"–"+le+" CT") {
		t.Errorf("the no-trade gate block does not state the lunch window %s–%s CT", ls, le)
	}
	// Derived, not typed: moving the definition must move the prompt.
	if !strings.Contains(prompt, ls) || !strings.Contains(prompt, le) {
		t.Fatal("the prompt's lunch window is not derived from LunchWindowCT")
	}
}

// F4 — THE LITERAL SCAN. Every no-trade window on every surface must resolve
// through the shared definitions. A file in this list holding a bare "12:00"
// or "13:30" is a fifth copy waiting to drift.
func TestNoTradeWindowsHaveNoSurfaceLiterals(t *testing.T) {
	ls, le := LunchWindowCT()
	for _, f := range []string{
		"planner_prompt.go",
		"adherence.go",
		"no_trade_band.go",
	} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		// Comments are stripped first: several of these files EXPLAIN the old
		// literals in prose, and a scan that cannot tell code from commentary
		// forces the history out of the file to stay green.
		src := stripLineComments(string(b))
		// no_trade_band.go IS the definition site; the bounds live there once.
		if f == "no_trade_band.go" {
			if strings.Count(src, `"`+ls+`"`) != 1 || strings.Count(src, `"`+le+`"`) != 1 {
				t.Errorf("%s must hold EXACTLY one copy of each lunch bound (it is the definition)", f)
			}
			continue
		}
		// The WINDOW, not a bare bound. The first cut looked only for the
		// quoted forms ("12:00") and so missed two copies written as prose:
		//   "the lunch no-trade (12:00–13:30 CT)"                      (clock line)
		//   "lunch 11:30–13:30 ET … (the system hard-gates 12:00–13:30 CT)"
		// Scanning for a bare bound instead over-fires: "13:30 ET" is a
		// different time from "13:30 CT", and the ET lull is deliberately not
		// the machine's window. So the unit is the pair, in either dash.
		for _, lit := range []string{ls + "–" + le, ls + "-" + le, `"` + ls + `"`, `"` + le + `"`} {
			if strings.Contains(src, lit) {
				t.Errorf("%s hardcodes the lunch window %q — read LunchWindowCT() instead", f, lit)
			}
		}
	}
}

// stripLineComments removes // commentary so the literal scan reads code only.
// Crude by design: no window bound in this package is written on a line that
// also carries a // sequence inside a string.
func stripLineComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// RIDER (owner ruling 2026-09-03) — the no_trade example must not demonstrate
// the machine's own windows.
//
// ASIA v14, written 44 minutes after the band shipped, carried
//
//	"no_trade": ["first 5m (CT)", "12:00-13:30 CT lunch"]
//
// which is the schema example verbatim, minus its placeholder. The sentence
// below the example asked the model not to write those windows; the example
// above it showed them as the expected content. An example is a demonstration
// and a sentence is a request — the example won.
func TestNoTradeExampleDoesNotDemonstrateMachineWindows(t *testing.T) {
	ex := NoTradeSchemaExample()
	ls, le := LunchWindowCT()
	for _, banned := range []string{
		fmt.Sprintf("first %dm", FirstNoTradeMinutes()),
		ls, le, "lunch",
	} {
		if strings.Contains(strings.ToLower(ex), strings.ToLower(banned)) {
			t.Errorf("the no_trade example demonstrates %q — the machine writes that window, and the model copies whatever the example shows.\n  example: %s", banned, ex)
		}
	}
	if !strings.Contains(ex, "no_trade") {
		t.Errorf("the example must still name the field: %s", ex)
	}
	// The sentence stays — in its 2026-09-03 wording, which no longer names a
	// window either (see TestNoTradeInstructionNamesNoMachineWindow).
	if !strings.Contains(NoTradeInstruction(), "do not list them") {
		t.Error("the instruction sentence must survive the example change")
	}
}

// RIDER — the prompt declares that EVERY time in it is CT, then printed ET
// times. A model reading "10:30 ET" as CT is an hour out. One clock.
func TestPromptStatesNoUntypedEasternTimes(t *testing.T) {
	prompt := plannerOutputContract(8, 5, true, true)
	// the whole prompt, not just the output contract — the ET times lived in
	// the no-trade gate and killzone blocks
	full := BuildPlannerPrompt(PlannerInput{
		TradeDate: "2026-09-03", Session: SessionNY, Price: 29000, DATR: 300,
		Now: time.Date(2026, 9, 3, 8, 30, 0, 0, CTLocation()),
	})
	for name, src := range map[string]string{"output contract": prompt, "no-trade gates": full} {
		if etTime.MatchString(src) {
			t.Errorf("%s prints an ET wall-clock time (%q) while the clock line declares every time in this prompt CT — the model is an hour out on it",
				name, etTime.FindString(src))
		}
	}
}

// etTime matches an "HH:MM ET" / "HH:MM–HH:MM ET" wall clock.
var etTime = regexp.MustCompile(`\d{1,2}:\d{2}(\s*[–-]\s*\d{1,2}:\d{2})?\s*ET\b`)

// RIDER (owner ruling 2026-09-03, seam closed) — the no_trade INSTRUCTION must
// not name a machine window either.
//
// The example stopped demonstrating them, but the sentence still opened
// "no_trade may contain ONLY the fixed session windows (first 5m, 12:00-13:30
// CT lunch) plus T1 HARD-blackout lines" — naming as permitted content exactly
// what the machine writes. Example and sentence were saying different things,
// which is the same trap one layer down.
func TestNoTradeInstructionNamesNoMachineWindow(t *testing.T) {
	ins := NoTradeInstruction()
	ls, le := LunchWindowCT()
	for _, tok := range []string{
		fmt.Sprintf("first %dm", FirstNoTradeMinutes()),
		ls, le, "lunch",
	} {
		if strings.Contains(strings.ToLower(ins), strings.ToLower(tok)) {
			t.Errorf("the no_trade instruction names the machine window token %q — the machine writes it, so naming it here invites the model to restate it.\n  instruction: %s", tok, ins)
		}
	}
	// What it MUST still say: the windows apply regardless, and the field is
	// the model's own.
	for _, want := range []string{"enforces", "regardless", "do not list them", "your OWN"} {
		if !strings.Contains(ins, want) {
			t.Errorf("the instruction dropped %q — without it the author is not told the windows apply anyway.\n  instruction: %s", want, ins)
		}
	}
	// The window itself is still STATED in the prompt, in the no-trade gate
	// block — the author must know it exists, just not be told to write it.
	prompt := BuildPlannerPrompt(PlannerInput{
		TradeDate: "2026-09-03", Session: SessionNY, Price: 29000, DATR: 300,
		Now: time.Date(2026, 9, 3, 8, 30, 0, 0, CTLocation()),
	})
	if !strings.Contains(prompt, ls+"–"+le+" CT") {
		t.Errorf("the rendered prompt no longer states the lunch window ANYWHERE — the author must still know it exists")
	}
}

// RIDER — the no-trade GATE BLOCK must not order the model to declare the
// machine's own gates.
//
// Found while closing the instruction seam: the block header read "## No-trade
// gates (advisory — declare in no_trade or skip the day)" over a list whose
// items include the machine's hard-gated lunch window and Tier-1 news. Telling
// the author to declare that list in no_trade is the same instruction the
// sentence below now forbids, one layer up, and a header is read before a rule.
func TestNoTradeGateBlockDoesNotOrderDeclaration(t *testing.T) {
	prompt := BuildPlannerPrompt(PlannerInput{
		TradeDate: "2026-09-03", Session: SessionNY, Price: 29000, DATR: 300,
		Now: time.Date(2026, 9, 3, 8, 30, 0, 0, CTLocation()),
	})
	if strings.Contains(prompt, "declare in no_trade") {
		t.Error(`the gate block still says "declare in no_trade" over a list containing machine-enforced gates — the instruction says do not list them`)
	}
	// The gates themselves must still be visible; only the order to restate
	// them is gone.
	for _, want := range []string{"No-trade gates", "balance-day", "Tier-1 news"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the gate block lost %q — the author must still see the gates", want)
		}
	}
}

// DISPLACEMENT FEEDS FORWARD (owner ruling 2026-09-03) — contract row.
//
// The waterfall floor is enforced at write and was never stated. Two reads on
// 2026-09-03 burned attempt 1 authoring breakdown_continue at levels with 0.00
// measured displacement. The schema line must qualify the field, and the facts
// block must carry both the floor and the per-level measurement.
func TestBreakdownSchemaRequiresMeasuredDisplacement(t *testing.T) {
	contract := plannerOutputContract(8, 5, true, true)
	idx := strings.Index(contract, `"breakdown"`)
	if idx < 0 {
		t.Fatal(`the rendered contract has no "breakdown" field`)
	}
	window := contract[idx:min(idx+520, len(contract))]
	for _, want := range []string{"MEASURED displacement", "≥ the stated floor", "REFUSED at write"} {
		if !strings.Contains(window, want) {
			t.Errorf(`the "breakdown" schema field never says %q — the floor is enforced and unstated, which is what rejected S3 at 00:07 and S2 at 00:33.`+"\n  field renders as: %s", want, window[:min(260, len(window))])
		}
	}

	// The facts block states the floor and the per-level measurement, and both
	// come from the resolvers rather than a literal.
	full := BuildPlannerPrompt(PlannerInput{
		TradeDate: "2026-09-03", Session: SessionNY, Price: 29000, DATR: 300,
		Now:            time.Date(2026, 9, 3, 8, 30, 0, 0, CTLocation()),
		StopFloorATR5m: 15.2, StopFloorMult: 1.5,
		LevelDisplacements: []LevelDisplacement{
			{Price: 29100, Label: "PDL", Broken: false},
			{Price: 28900, Label: "ONL", Broken: true, Short: true, Pts: 40},
			{Price: 29050, Label: "VWAP", Broken: true, Short: true, Pts: 3}, // broken, but a nudge
		},
	})
	for _, want := range []string{
		"Waterfall displacement floor this cycle",
		"Measured displacement per level",
		"none — no break",
		"BELOW the floor — closed-5m displacement does not permit pullback authoring",
		"at or above the floor — closed-5m displacement permits pullback authoring",
	} {
		if !strings.Contains(full, want) {
			t.Errorf("the rendered prompt never states %q", want)
		}
	}

	// No ATR reading → no claim, rather than a zero floor.
	bare := BuildPlannerPrompt(PlannerInput{
		TradeDate: "2026-09-03", Session: SessionNY, Price: 29000, DATR: 300,
		Now:                time.Date(2026, 9, 3, 8, 30, 0, 0, CTLocation()),
		LevelDisplacements: []LevelDisplacement{{Price: 29100, Label: "PDL"}},
	})
	if strings.Contains(bare, "Waterfall displacement floor") {
		t.Error("with no ATR the floor must not be printed at all — a 0.0 pt floor is a lie the model would author against")
	}
}
