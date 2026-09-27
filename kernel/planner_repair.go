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
	if omitted := RenderOmittedSeatedLevels(errors); omitted != "" {
		b.WriteString("\n")
		b.WriteString(omitted)
	}
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
	// FOLD (2) — re-aimed item 3: the confirm SIDE shape (the real entry-shape
	// kill, ids 412-414: `scenario[0].confirm.side "" invalid (above|below)`).
	// The author prompt states it (PromptContract row 346); the repair had no
	// case and fell to the generic excerpt.
	if strings.Contains(errors, "confirm.side") {
		add(`"confirm": {"side": "above" | "below"} — side is a STRING enum, exactly one of above|below (never empty, never a number).`)
	}
	// CLASS 46 RIDER (owner ruling 2026-09-02) — see lawExcerptsForDoc: the
	// enum is also attached whenever the DOCUMENT carries a confirm object,
	// not only when the incoming error names one.
	// Entry-law: a confirm rule legal in the field but illegal for THIS play.
	if strings.Contains(errors, "not allowed for") || strings.Contains(errors, "fade_requires_touch") {
		add(RepairEntryConfirmLaw)
	}
	// FIX-PLANNER (2026-09-26, item 3) — the entry-policy shape refusal
	// (EntryPolicyLegal: "entry policy planned_order is not legal on …"): the
	// repair carries the GENERATED shape table, never hand-rettyped prose.
	if strings.Contains(errors, "entry policy") || strings.Contains(errors, "not legal on") {
		add("ENTRY POLICY SHAPES (from the validator's own entry-law table):\n" + EntryPolicyShapeTable())
	}
	// FOLD from DS-104 cross-check (PR #242) — schema_json killer: a plan JSON
	// unmarshal failure carries an unmarshal-specific excerpt that QUOTES the
	// exact decode error (field path + expected type) and the minimal schema
	// for that field — never the generic excerpt (ids 357-359 / 391-393).
	if strings.Contains(errors, "plan JSON unmarshal") || strings.Contains(errors, "cannot unmarshal") {
		add(repairUnmarshalExcerpt(errors))
	}
	// W-FLIP-DIRECTION (2026-09-17): a flip side that points the wrong way for
	// the bias it flips from (plan_doc.go FlipDirectionContradiction).
	if strings.Contains(errors, "contradicts bias") {
		add(RepairFlipDirectionLaw)
	}
	// W-FLIP-LINE-SIDE-OF-PRICE (2026-09-17): a flip or death line already
	// beyond price on its own side (plan_doc.go FlipLineBeyondPrice /
	// DeathLineBeyondPrice — both rejections end in "far side of price").
	if strings.Contains(errors, "far side of price") {
		add(RepairFlipSideOfPriceLaw)
	}
	// W-EXEC-TRUTH W2 A5 (2026-09-23): a time_hold whose prose minutes are not
	// stored as confirm.hold_min (confirm_resolver.go ValidateConfirmHoldProse).
	if strings.Contains(errors, "hold_min") {
		add(RepairHoldMinLaw)
	}
	// W-EXEC-TRUTH W2 A1 (2026-09-23): scenario.invalid outside the grammar —
	// the combined refusal opens with AuthoredGrammarRefusalMarker.
	if strings.Contains(errors, AuthoredGrammarRefusalMarker) {
		add(RepairInvalidationGrammarLaw)
	}
	// W-EXEC-TRUTH W2 A3/A4 (scenario_write_truth.go) — identity ≠ price /
	// unresolved id, and the obstacle-chain contract.
	if strings.Contains(errors, "identity≠price") || strings.Contains(errors, "identity unresolved") {
		add(RepairIdentityPriceLaw)
	}
	if strings.Contains(errors, "obstacle chain:") {
		add(RepairObstacleChainLaw)
	}
	// W-EXEC-TRUTH W3 (2026-09-23): the market_in_zone zone refusal (trader
	// write-time hook) and the armable hold floor (ValidateArmableHoldFloor) —
	// each routes on its OWN marker, never on a shared field token.
	if strings.Contains(errors, EntryZoneRefusalMarker) {
		add(RepairEntryZoneLaw)
	}
	if strings.Contains(errors, ArmableHoldFloorMarker) {
		add(RepairArmableHoldFloorLaw)
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

// RenderOmittedSeatedLevels (FIX-PLANNER 2026-09-26, item 2) extracts every
// "obstacle chain: omits …" clause from the verbatim validator errors and
// renders the omitted seated levels as a per-level copy list — each with the
// id, price and the three legal roles the validator accepts. The text is
// parsed from the PRODUCTION error builder's output (scenario_write_truth.go
// obstacleChainWriteIssues), never retyped: a change to that format is a change
// here, pinned by TestFpRepairPromptListsOmittedSeatedLevelsWithRoles.
// No matching clause (or a malformed one) renders nothing.
func RenderOmittedSeatedLevels(errors string) string {
	const clause = "obstacle chain: omits "
	const clauseEnd = " — list every seated level between entry "
	var items []string
	rest := errors
	for {
		i := strings.Index(rest, clause)
		if i < 0 {
			break
		}
		tail := rest[i+len(clause):]
		j := strings.Index(tail, clauseEnd)
		if j < 0 {
			break
		}
		for _, item := range strings.Split(tail[:j], ", ") {
			if item = strings.TrimSpace(item); item != "" {
				items = append(items, item)
			}
		}
		rest = tail[j:]
	}
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Omitted seated levels (copy EXACTLY, with their ids, into economics.path_levels)\n")
	for _, item := range items {
		b.WriteString("- " + item + " → legal roles: pass_through | reduce | exit\n")
	}
	return b.String()
}

// repairUnmarshalSchemaFor maps the durable tail of a decode-error field path
// to the minimal schema for that field, taken from the PlanDoc JSON contract
// (the SAME struct tags the unmarshal reads). Unknown fields get the generic
// type-only line — the decode error above still quotes the exact path and type.
var repairUnmarshalSchemaFor = []struct {
	path, schema string
}{
	{"breakdown.level", `"breakdown": {"level": <number>, "entry_mode": "pullback" | "immediate"} — level is a NUMBER (never a string); write the level as a plain numeric price.`},
}

// repairUnmarshalExcerpt quotes the exact decode error (field path + expected
// type, parsed from the production fmt.Errorf("plan JSON unmarshal: %w") text)
// and attaches the minimal schema for the offending field when one is known.
// Parsed, never retyped: a change to the decode error's shape is a change here,
// pinned by TestFpUnmarshalRepairExcerptQuotesDecodeErrorAndSchema.
func repairUnmarshalExcerpt(errors string) string {
	head := "PLAN JSON SHAPE: the model output failed to decode. Fix the field exactly as named."
	i := strings.Index(errors, "cannot unmarshal")
	if i < 0 {
		return head
	}
	field := ""
	tail := errors[i:]
	if j := strings.Index(tail, " of type "); j >= 0 {
		field = strings.TrimSpace(tail[strings.Index(tail, " into Go struct field ")+len(" into Go struct field ") : j])
	}
	if field == "" {
		return head
	}
	quote := strings.TrimSpace(strings.SplitN(tail, " of type ", 2)[0])
	schema := "make that field the type the decode error names: the value is " + strings.TrimSpace(strings.SplitN(tail, " of type ", 2)[1]) + "."
	for _, s := range repairUnmarshalSchemaFor {
		if strings.Contains(field, s.path) {
			schema = "minimal schema for " + s.path + ": " + s.schema
			break
		}
	}
	return head + "\n" + "decode error (verbatim): " + quote + " of type " + strings.TrimSpace(strings.SplitN(tail, " of type ", 2)[1]) + "\nfield: " + field + " → " + schema
}
