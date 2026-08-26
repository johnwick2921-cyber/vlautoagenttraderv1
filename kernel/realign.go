package kernel

import (
	"fmt"
	"strings"
)

// W13 — PLAN RE-ALIGNMENT ON OWNER EDIT (owner decision 2026-08-16: AUTO trigger).
//
// When the owner saves an overlay edit, the planner re-examines the WHOLE current
// plan in light of that change and returns a PROPOSAL — never a silent mutation.
// It reuses the Ask-Planner path VERBATIM: same AskPlannerSystemPrompt (the
// anti-sycophancy contract), same ParsePlannerReply, same verdict vocabulary, same
// KPI table. Only the USER prompt differs: instead of an owner question, it carries
// the owner's structural change plus the four re-examination questions.
//
// Keeping the contract byte-identical is deliberate: the sycophancy KPI must stay
// ONE comparable series across typed questions and auto re-aligns.

// RealignTrigger values (stored on the QA row).
const (
	TriggerOwnerEdit = "owner-edit"     // auto: fired by an overlay save
	TriggerManual    = "manual-realign" // owner tapped "Re-align plan"
)

// OwnerChange describes the edit that triggered the re-align. Every field is
// optional except Kind + Summary; the prompt renders only what is present, so a
// delete (no grade/instruction) reads as cleanly as a full add.
type OwnerChange struct {
	Kind        string  // add-level | edit-level | delete-level | bulk-add
	Summary     string  // human line, e.g. "added D-4h 30156 (A) — sweep+reclaim = entry"
	Price       float64 // 0 = omit
	Label       string  // provenance / type chip
	Grade       string  // A | B | C
	Instruction string  // instruction verb
	Note        string  // the owner's note — their voice into the AI (any language)
	ScenarioTag string  // S1 / S2 / "+ new play"
	BatchCount  int     // bulk-add: how many levels landed in the ONE batch
}

// Describe renders the change as the prompt's owner-input block.
func (c OwnerChange) Describe() string {
	var b strings.Builder
	kind := strings.TrimSpace(c.Kind)
	if kind == "" {
		kind = "edit"
	}
	fmt.Fprintf(&b, "change_kind: %s\n", kind)
	if c.BatchCount > 1 {
		fmt.Fprintf(&b, "batch: %d levels saved together (ONE re-examination for the whole batch)\n", c.BatchCount)
	}
	if s := strings.TrimSpace(c.Summary); s != "" {
		fmt.Fprintf(&b, "what_changed: %s\n", s)
	}
	if c.Price > 0 {
		fmt.Fprintf(&b, "price: %.2f\n", c.Price)
	}
	for _, kv := range [][2]string{
		{"type/label", c.Label}, {"grade", c.Grade},
		{"instruction", c.Instruction}, {"scenario_tag", c.ScenarioTag},
	} {
		if v := strings.TrimSpace(kv[1]); v != "" {
			fmt.Fprintf(&b, "%s: %s\n", kv[0], v)
		}
	}
	if n := strings.TrimSpace(c.Note); n != "" {
		fmt.Fprintf(&b, "owner_note (their reasoning, any language): %s\n", n)
	}
	return strings.TrimRight(b.String(), "\n")
}

// realignQuestions is the fixed re-examination framing (owner-specified, W13 §2).
const realignQuestions = `The owner just changed the plan (below). Re-examine the WHOLE plan in light of it:
(a) does this change the bias or the flip line?
(b) does it warrant a NEW scenario, or does it fit an existing one?
(c) do any scenarios need re-targeting, re-grading, or removal?
(d) does it conflict with an AI level?
DEFEND the existing structure unless the owner's input is genuinely new information.
If nothing should change, say so plainly with DEFEND and emit no patch. If something
should change, use PROPOSE-MERGE and put EVERY change in the RFC-6902 patch — the
patch may touch any part of the plan (bias, conviction, flip_condition, scenarios,
target chains, grades, level instructions). Never give vague advice: the patch IS
the proposal.`

// BuildRealignUserPrompt assembles the re-align user prompt from the SAME context
// blocks Ask-Planner uses (overlay-resolved plan block + live Go facts) plus the
// owner's change. The system prompt stays AskPlannerSystemPrompt (unchanged).
func BuildRealignUserPrompt(planBlock, liveStatus string, change OwnerChange) string {
	var b strings.Builder
	b.WriteString("# CURRENT PLAN (overlay-resolved — this is what the card and the executor see)\n")
	b.WriteString(planBlock + "\n\n")
	if strings.TrimSpace(liveStatus) != "" {
		b.WriteString("# LIVE FACTS (Go-computed)\n" + liveStatus + "\n\n")
	}
	b.WriteString("# OWNER'S CHANGE\n" + change.Describe() + "\n\n")
	b.WriteString("# YOUR TASK\n" + realignQuestions + "\n")
	return b.String()
}

// IsNoChange reports whether a reply means "the plan stands" — the UI renders this
// as the quiet auto-fading chip rather than a proposal card. A reply is no-change
// when it is not a PROPOSE-MERGE, or when it carries no patch ops at all (a
// PROPOSE-MERGE without a patch cannot change anything, so it is not a proposal).
func IsNoChange(r *PlannerReply) bool {
	if r == nil {
		return true
	}
	if r.Verdict != "PROPOSE-MERGE" {
		return true
	}
	return !hasPatchOps(r.Patch)
}
