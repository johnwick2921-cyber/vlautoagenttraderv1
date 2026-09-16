package kernel

import (
	"fmt"
	"strings"
)

// CLASS 38 (2026-09-01) — THE PROMPT/VALIDATOR CONTRACT.
//
// The defect class: the prompt offers or names something the validator
// rejects. reject_retest (class 34), the bare "2x5m" confirm token, and the
// unqualified "legs" schema field are all instances. The model is not the
// defect; the contract is.
//
// This registry is the machine-checkable half of that contract. Every
// validator branch that restricts a field BY CONDITION is listed here with the
// sentence(s) the rendered prompt must carry. ValidatePromptContracts asserts
// the sentences are present — "validator forbids X" without "prompt says X is
// forbidden" fails the build (table test) and shouts at boot.
//
// Add a row whenever a condition-keyed restriction is added to the validator.
// The list is deliberately literal: it is a contract, not a heuristic.

// PromptContract is one validator restriction and the prompt text that states
// it to the author.
type PromptContract struct {
	// Rule is the human name of the restriction.
	Rule string
	// Site is the validator branch that enforces it (file:line at authoring).
	Site string
	// MustAppear are fragments the rendered prompt must contain. ALL must be
	// present — a restriction stated only half-way is how row 80 happened.
	MustAppear []string
}

// PromptContracts is the C5 enumeration: every condition-keyed restriction in
// ValidatePlanDoc / ArmSpecValid / ValidateEntryLaw.
func PromptContracts() []PromptContract {
	return []PromptContract{
		// CLASS 45 (2026-09-02) — the two feed-forward facts. They are
		// CONDITIONAL sections (rendered only when there is something to say),
		// so the contract asserts the ORDER text that always ships with them.
		{
			// CHOP COLLAPSE (owner ruling 2026-09-03): levels void BOTH ways carry
			// no direction, so the eighteen-line list measured live at 00:00:56
			// CT renders as ONE aggregated CHOP line naming the alternative
			// ("prefer touch/fade plays there"); one-sided voids keep their own
			// line with side and reclaim time. MustAppear stays the ORDER text
			// that ALWAYS ships — the chop line itself is conditional, and a
			// fragment that cannot always appear would make this guard unusable.
			Rule:       "void breakdown levels are named in the facts, not discovered at write (both-way voids collapse into one CHOP line that names the alternative)",
			Site:       "kernel/class45_feeds_forward.go ComputeVoidBreakdownLevels → BreakdownContinueState; RenderVoidBreakdownLevels (chop collapse)",
			MustAppear: []string{"if a breakdown level is listed as VOID above, author a different condition there"},
		},
		{
			Rule:       "the gap-side order names a DIRECTION, never a condition",
			Site:       "kernel/plan_doc.go hasDirection + gapDownDirectionMessage",
			MustAppear: []string{"MUST include a SHORT-direction scenario", "ANY legal condition"},
		},
		{
			Rule:       "arm{} legal only on armable conditions",
			Site:       "plan_doc.go ArmSpecValid (arm enabled on non-armable condition)",
			MustAppear: []string{"legal ONLY on " + ArmableConditionsPipe()},
		},
		{
			Rule:       "legs[] only on sweep_reclaim (arm_legs_sweep_reclaim_only)",
			Site:       "plan_doc.go ArmSpecValid (arm legs on %s)",
			MustAppear: []string{"ONLY if condition is sweep_reclaim", "legs[] are the sweep_reclaim SPLIT contract"},
		},
		{
			Rule:       "legs[] must be EXACTLY 2 when present",
			Site:       "plan_doc.go ArmSpecValid (needs EXACTLY 2 legs)",
			MustAppear: []string{"EXACTLY 2 legs"},
		},
		{
			Rule:       "every non-sweep condition arms SINGLE (no legs)",
			Site:       "plan_doc.go ArmSpecValid (arm legs on %s — other conditions arm single)",
			MustAppear: []string{"must arm SINGLE", "no legs"},
		},
		{
			Rule:       "split requires confirm=touch at the sweep ref",
			Site:       "plan_doc.go ArmSpecValid (split requires confirm=touch)",
			MustAppear: []string{"confirm=touch at the sweep ref"},
		},
		{
			Rule:       "split leg 1 rests (wait_confirm false), leg 2 chains (wait_confirm true)",
			Site:       "plan_doc.go ArmSpecValid (leg 1 must rest / leg 2 must chain)",
			MustAppear: []string{"leg 1 rests there (wait_confirm false)", "leg 2 chains (wait_confirm true)"},
		},
		{
			Rule:       "split leg 2 rule ∈ {1m_mss, 1x5m_close} and equals confirm2.rule",
			Site:       "plan_doc.go ArmSpecValid (sweep_leg2_requires_mss_or_1x5m / must match confirm2.rule)",
			MustAppear: []string{"confirm2 = 1m_mss or 1x5m_close", "EQUAL to confirm2.rule"},
		},
		{
			Rule:       "split top-level entry/stop/target mirror leg 1",
			Site:       "plan_doc.go ArmSpecValid (top-level must equal leg 1's)",
			MustAppear: []string{"top-level entry/stop/target mirror leg 1"},
		},
		{
			Rule:       "breakdown/breakup arm requires breakdown{} with entry_mode=pullback",
			Site:       "plan_doc.go ArmSpecValid (arm requires entry_mode=pullback)",
			MustAppear: []string{"entry_mode=pullback", "entry_mode=immediate is AI-path ONLY"},
		},
		{
			Rule:       "sweep_reclaim single arm requires wait_confirm:true",
			Site:       "plan_doc.go ArmSpecValid (sweep_reclaim arm requires wait_confirm:true)",
			MustAppear: []string{"wait_confirm:true"},
		},
		{
			Rule:       "fvg{} REQUIRED iff condition==fvg_entry",
			Site:       "plan_doc.go ValidatePlanDoc (fvg required)",
			MustAppear: []string{`fvg{} REQUIRED iff condition=="fvg_entry"`},
		},
		{
			Rule:       "breakdown{} REQUIRED iff waterfall-class condition",
			Site:       "plan_doc.go ValidatePlanDoc (breakdown required)",
			MustAppear: []string{"breakdown{} REQUIRED iff waterfall-class"},
		},
		{
			Rule:       "fades (reject|fvg_entry) are touch-only (fade_requires_touch)",
			Site:       "entry_law.go ValidateEntryLaw (fade_requires_touch)",
			MustAppear: []string{"touch ONLY (fade_requires_touch)"},
		},
		{
			Rule:       "2x5m_close legal ONLY on breakdown_continue|breakup_continue (2x5m_reserved)",
			Site:       "entry_law.go ValidateEntryLaw (2x5m_reserved)",
			MustAppear: []string{"2x5m_close is legal ONLY here"},
		},
		{
			Rule:       "armed fade needs a structure stop ≥2 ticks beyond the level",
			Site:       "entry_law.go ValidateEntryLaw (structure stop required)",
			MustAppear: []string{"structure stop ≥2 ticks beyond the level"},
		},
		{
			Rule:       "breakout_retest never arms (GAR-F4)",
			Site:       "armed.go ArmableCondition (breakout_retest excluded)",
			MustAppear: []string{"breakout_retest stays a normal AI play"},
		},
		{
			Rule:       "death/flip.rule is a SEPARATE enum from confirm.rule",
			Site:       "plan_doc.go conditionRules vs confirmRules",
			MustAppear: []string{"death/flip rules use their OWN vocabulary"},
		},
	}
}

// ValidatePromptContracts asserts every enumerated restriction is stated in the
// rendered prompt. Returns the first restriction that is not.
func ValidatePromptContracts(prompt string) error {
	for _, c := range PromptContracts() {
		for _, frag := range c.MustAppear {
			if !strings.Contains(prompt, frag) {
				return fmt.Errorf("restriction %q (enforced at %s) is NOT stated in the prompt — missing fragment %q", c.Rule, c.Site, frag)
			}
		}
	}
	return nil
}

// PromptContractBootLine renders the F8 boot line. The prompt is rendered from
// the pure output-contract block (no market data needed), so the guard runs at
// every boot exactly as the table test runs it in CI.
func PromptContractBootLine() string {
	n := len(PromptContracts())
	// 0/0 → resolvePlanCaps supplies the RESOLVED caps (A11). The contract
	// sentences are static text, so the cap values never change the verdict.
	if err := ValidatePromptContracts(plannerOutputContract(0, 0, true, true)); err != nil {
		return fmt.Sprintf("📜 prompt/validator contract: BROKEN — %v (class 38 guard)", err)
	}
	return fmt.Sprintf("📜 prompt/validator contract: %d restrictions, all stated in prompt (class 38 guard)", n)
}
