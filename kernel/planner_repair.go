package kernel

import (
	"strings"
)

// repairReturnContract (REPAIR-PARSE E1, 2026-09-02) is the packaging contract,
// stated at the TOP and again at the BOTTOM of every repair prompt. The
// lost-in-the-middle rule: a single instruction ahead of a 4 KB rejected
// document and a wall of validator text is the one most likely to be ignored.
const repairReturnContract = "Return ONLY the COMPLETE corrected plan JSON document — the whole plan with the named defects fixed and nothing else changed. No prose before or after it. No markdown fences (no ```). No commentary. A fragment (one scenario, a patch, a diff) is NOT acceptable: return the entire document."

// BuildPlannerRepairPrompt (planner-speed wave 3, 2026-08-31) composes the
// attempt-≥2 EDIT call: a compact instruction header + the rejected output
// verbatim + ALL validator errors verbatim + minimal law excerpts for the
// violated rules only. NO candle tables, NO level map, NO full playbook —
// the repair is expected to cost a fraction of a full re-author's tokens.
//
// REPAIR-PARSE (2026-09-02): it now also carries the class-34 condition
// vocabulary (`live`), which only the RE-AUTHOR tail carried before — so the
// DEFAULT retry path had run without the legal condition list since class 34
// shipped — and the return contract is repeated at both ends.
func BuildPlannerRepairPrompt(rejectedOutput string, errors string, live []string) string {
	var b strings.Builder
	b.WriteString("You are repairing a rejected plan. Fix ONLY the named defects. Change nothing else.\n")
	b.WriteString(repairReturnContract)
	b.WriteString("\n\n## Validator errors (verbatim)\n")
	b.WriteString(errors)
	b.WriteString("\n\n## Rejected plan output (verbatim)\n")
	b.WriteString(rejectedOutput)
	b.WriteString("\n\n## Applicable law (excerpts for the violated rules only)\n")
	b.WriteString(lawExcerptsForDoc(errors, rejectedOutput))
	if line := LiveConditionsLine(live); line != "" {
		b.WriteString(line)
	}
	b.WriteString("\n\n## Return format (restated — this is the contract)\n")
	b.WriteString(repairReturnContract)
	b.WriteString("\n")
	return b.String()
}

// lawExcerptsFor maps a validator error string to the minimal law excerpts for
// the violated rules.
//
// REPAIR-PARSE (2026-09-02): this was a first-match `switch`, so a repair whose
// errors violated two laws was told about one; and its cases matched neither
// `fade_requires_touch` nor `invalid (`, the two most common confirm-rule
// defects, which therefore fell through to a GENERIC excerpt about level
// labels and targets. Measured over the 2026-09-01 journals: 11 of 17
// content-rejected repairs received an irrelevant excerpt. It now collects
// EVERY applicable excerpt, and the generic line is the fallback only when
// nothing matched.
func lawExcerptsFor(errors string) string {
	var out []string
	add := func(s string) {
		for _, have := range out {
			if have == s {
				return
			}
		}
		out = append(out, s)
	}
	// Arm / split-entry contract.
	if strings.Contains(errors, "EXACTLY 2 legs") || strings.Contains(errors, "split requires confirm=touch") ||
		strings.Contains(errors, "arm legs on") || strings.Contains(errors, "arm_legs_sweep_reclaim_only") ||
		strings.Contains(errors, "must equal leg 1") {
		add(RepairArmSplitLaw)
	}
	// Breakdown void.
	if strings.Contains(errors, "breakdown is void") || strings.Contains(errors, "came back across") {
		add(RepairBreakdownLaw)
	}
	// A confirm/confirm2 RULE token that is not in that field's enum, or a
	// word that is in no enum at all. This is the dominant defect class.
	if (strings.Contains(errors, "confirm.rule") || strings.Contains(errors, "confirm2.rule")) &&
		(strings.Contains(errors, "invalid (") || strings.Contains(errors, "fade_requires_touch") ||
			strings.Contains(errors, "not allowed for")) {
		add(RepairConfirmVocabLaw)
	}
	// CLASS 46 RIDER (owner ruling 2026-09-02) — see lawExcerptsForDoc: the
	// enum is also attached whenever the DOCUMENT carries a confirm object,
	// not only when the incoming error names one.
	// Entry-law: a confirm rule legal in the field but illegal for THIS play.
	if strings.Contains(errors, "not allowed for") || strings.Contains(errors, "fade_requires_touch") {
		add(RepairEntryConfirmLaw)
	}
	if len(out) == 0 {
		add("Copy the machine table's labels and prices; collapse duplicate seats; targets must sit within the proximity band of price.")
	}
	return strings.Join(out, "\n")
}

// lawExcerptsForDoc (CLASS 46 RIDER, owner ruling 2026-09-02) is lawExcerptsFor
// plus one document-driven rule: if the REJECTED DOCUMENT contains a confirm or
// confirm2 object, the confirm-rule vocabulary is attached regardless of what
// the incoming error was.
//
// Evidence (chain 4, 2026-09-02 14:23 CT): attempt 1 was rejected for a VOID
// BREAKDOWN, so the repair prompt correctly carried the breakdown law — and
// nothing about confirm rules, because the routing keys on the incoming error.
// The model fixed the breakdown and, in the same edit, wrote
// scenario[1].confirm.rule "1x5m_close" on a reject fade: a confirm-rule
// violation it had never been shown the enum for. A repair that rewrites a
// scenario can introduce the very defect class 44 exists to close, through a
// door class 44 did not cover. Cost of the fix: ~60 tokens on a ~1,200-token
// prompt.
func lawExcerptsForDoc(errors, rejectedOutput string) string {
	base := lawExcerptsFor(errors)
	if !docHasConfirmObject(rejectedOutput) {
		return base
	}
	// OWNER RULING 2026-09-02 — the ENUM is replaced by the per-condition
	// TABLE. Live counterexample (18:39 CT, planner_rejected_prompts row 104):
	// the enum reached the model and it still wrote `1x5m_close` on a `reject`
	// fade — and `1x5m_close` IS in the enum. Listing the words that exist
	// never addressed the failure; the failure is which word THIS CONDITION
	// permits. The table is generated from the validator's own entryLaw map.
	table := ConfirmRuleTable()
	if strings.Contains(base, table) {
		return base
	}
	if strings.TrimSpace(base) == "" {
		return table
	}
	return base + "\n" + table
}

// docHasConfirmObject reports whether the rejected document carries a confirm
// or confirm2 object at all — the trigger for the rider above.
func docHasConfirmObject(doc string) bool {
	return strings.Contains(doc, "\"confirm\"") || strings.Contains(doc, "\"confirm2\"")
}
