package kernel

import (
	"strings"
	"testing"
)

// PARITY — the prompt's table and the validator's enforcement come from ONE
// map. If a condition's allowed set ever changes in entryLaw, the table moves
// with it; if someone hand-writes a row, this fails.
func TestConfirmRuleTableParityWithValidator(t *testing.T) {
	rows := ConfirmRuleRows()
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.Condition] = true
		law, ok := EntryLawFor(r.Condition)
		if !ok {
			t.Fatalf("row %q has no entry law", r.Condition)
		}
		if len(r.Allowed) != len(law.Allowed) {
			t.Fatalf("%s: table lists %v, validator allows %d rules", r.Condition, r.Allowed, len(law.Allowed))
		}
		for _, rule := range r.Allowed {
			if !law.Allowed[rule] {
				t.Fatalf("%s: table lists %q which the validator does NOT allow", r.Condition, rule)
			}
		}
	}
	// Every condition the schema accepts and that HAS a law must appear.
	for cond := range scenarioConds {
		if _, ok := EntryLawFor(cond); ok && !seen[cond] {
			t.Fatalf("condition %q has an entry law but no table row", cond)
		}
	}
}

// THE RULING'S OWN ROW: reject → [touch]. This is the pairing that failed live.
func TestConfirmRuleTableRejectIsTouchOnly(t *testing.T) {
	tbl := ConfirmRuleTable()
	if !strings.Contains(tbl, "reject → [touch]") {
		t.Fatalf("the ruling's row is missing:\n%s", tbl)
	}
	// And the table must say WHY a legal-looking token can be wrong.
	if !strings.Contains(tbl, "1x5m_close") || !strings.Contains(tbl, "ILLEGAL for another") {
		t.Fatalf("the table must explain that a token legal elsewhere is illegal here:\n%s", tbl)
	}
	// breakdown_continue is the only condition allowed 2x5m_close.
	for _, r := range ConfirmRuleRows() {
		has2x := false
		for _, x := range r.Allowed {
			if x == "2x5m_close" {
				has2x = true
			}
		}
		if has2x && !strings.HasSuffix(r.Condition, "_continue") {
			t.Fatalf("2x5m_close leaked to %q", r.Condition)
		}
	}
}

// The repair prompt carries the TABLE, not the bare enum, whenever the
// document has a confirm object — including when the incoming error is about
// something else entirely (the chain-4 / row-104 shape).
func TestRepairPromptCarriesTableNotEnum(t *testing.T) {
	doc := `{"reasoning":"r","bias":{"direction":"short"},"levels":[],"scenarios":[` +
		`{"id":"S2","condition":"reject","direction":"short",` +
		`"confirm":{"rule":"1x5m_close","ref_price":29149,"side":"below"}}]}`
	errs := "S3 breakdown_continue: a close came back across 29149.00 — the breakdown is void"
	p := BuildPlannerRepairPrompt(doc, errs, []string{"reject", "hold"})
	if !strings.Contains(p, "reject → [touch]") {
		t.Fatalf("the per-condition table must be attached:\n%s", rpTrunc(p, 500))
	}
	if !strings.Contains(p, "BREAKDOWN-CONTINUE LAW") {
		t.Fatal("the incoming defect's own law must still be attached")
	}
	if strings.Count(p, "CONFIRM RULES PER CONDITION") != 1 {
		t.Fatalf("the table must appear exactly once, got %d", strings.Count(p, "CONFIRM RULES PER CONDITION"))
	}
	// No confirm object → no table, no wasted tokens.
	noConf := `{"reasoning":"r","bias":{},"levels":[],"scenarios":[{"id":"S1","condition":"hold"}]}`
	if q := BuildPlannerRepairPrompt(noConf, errs, nil); strings.Contains(q, "CONFIRM RULES PER CONDITION") {
		t.Fatal("no confirm object — the table must not be attached")
	}
}
