package trader

import (
	"errors"
	"path/filepath"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// F4 — the outcome counters persist across a restart and the classes are
// distinguished. Before this every failure was logged "UNPARSEABLE", which was
// wrong for 17 of the 18 measured cases.
func TestRepairOutcomeCountersPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.db")
	st, err := store.New(path)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	for _, o := range []string{"content", "content", "packaging", "ok"} {
		if _, err := store.IncRepairOutcome(st, o); err != nil {
			t.Fatalf("inc %s: %v", o, err)
		}
	}
	if n, _ := store.RepairOutcomeCount(st, "content"); n != 2 {
		t.Fatalf("content = %d, want 2", n)
	}
	if _, err := store.IncRepairOutcome(st, "  "); err == nil {
		t.Fatal("an empty outcome must be refused, never counted as \"\"")
	}
	st.Close()

	st2, err := store.New(path) // the restart
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	if n, _ := store.RepairOutcomeCount(st2, "content"); n != 2 {
		t.Fatalf("counter did not survive the restart: %d", n)
	}
	if s := store.RepairOutcomeSummary(st2); s != "ok=1 content=2 packaging=1 fragment=0" {
		t.Fatalf("summary: %q", s)
	}
}

// The classifier separates what the old single label conflated.
func TestRepairOutcomeClassification(t *testing.T) {
	full := `{"bias":{},"levels":[],"scenarios":[]}`
	cases := []struct {
		name string
		raw  string
		err  error
		want kernel.RepairOutcome
	}{
		{"landed", full, nil, kernel.RepairOK},
		{"empty output", "   ", nil, kernel.RepairNoOutcome},
		{"no json at all", "I cannot do that.", errors.New("no JSON object found in planner output"), kernel.RepairPackaging},
		// The ONE real packaging failure in the journals (2026-09-01 04:24:17):
		// a fractional contract size where an int is required.
		{"type error", full, errors.New(`plan JSON unmarshal: json: cannot unmarshal number 0.5 into Go struct field PlanArmLeg.scenarios.arm.legs.size of type int`), kernel.RepairPackaging},
		// The dominant real class: parsed fine, rejected on field values.
		{"vocabulary reject", full, errors.New(`scenario[3].confirm2.rule "displacement" invalid (touch|1x5m_close|2x5m_close|1m_mss|time_hold)`), kernel.RepairContent},
		{"scenario fragment", `{"id":"S1","condition":"reject","direction":"long"}`, errors.New("some schema error"), kernel.RepairFragment},
	}
	for _, c := range cases {
		if got := kernel.ClassifyRepairOutcome(c.raw, c.err); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

// F3 — packaging tolerance was ALREADY present (extractJSONObject scans to the
// first '{' and walks to its match), which is why no fenced/prose failure
// appears in the journals. This pins that it stays true AND that extraction
// never alters plan content.
func TestRepairExtractionToleratesPackagingWithoutAlteringContent(t *testing.T) {
	doc := `{"reasoning":"r","bias":{"direction":"long","conviction":"low","flip_condition":"x"},` +
		`"levels":[{"price":29500,"label":"PDL","grade":"A","instruction":"reclaim"}],` +
		`"scenarios":[{"id":"S1","trigger":"reclaim of 29500 with a 5m close back above","condition":"reclaim","direction":"long",` +
		`"target_chain":[29700],"invalid":"2x5m below 29500","quality":"B","economics":{"entry_zone":[29500,29500],"geometry":{"entry":29500,"stop":29480,"target":29700},"first_obstacle":{"price":29700,"level":"fixture reference","family":"reference","response":"pass_through"},"r_to_obstacle":10.0,"r_to_arm_target":10.0},` +
		`"confirm":{"rule":"1x5m_close","ref_price":29500,"side":"above"}}],` +
		`"no_trade":["first 5m"],"death_condition":"d"}`
	wrappers := map[string]string{
		"bare":          doc,
		"fenced":        "```json\n" + doc + "\n```",
		"prose before":  "Here is the corrected plan:\n\n" + doc,
		"prose after":   doc + "\n\nI fixed the confirm rule as instructed.",
		"prose + fence": "Sure — fixed.\n```\n" + doc + "\n```\nLet me know if anything else is wrong.",
	}
	var first string
	for name, raw := range wrappers {
		d, err := kernel.ParsePlanDocCapped(raw, 8, 5)
		if err != nil {
			t.Fatalf("%s: must parse, got %v", name, err)
		}
		if kernel.IsPlanFragment(raw) {
			t.Fatalf("%s: a full document must never be called a fragment", name)
		}
		got := d.Scenarios[0].Confirm.Rule + "|" + d.Levels[0].Label + "|" + d.Bias.Direction
		if first == "" {
			first = got
		} else if got != first {
			t.Fatalf("%s: packaging altered CONTENT: %q vs %q", name, got, first)
		}
	}
	if first == "" {
		t.Fatal("no wrapper exercised")
	}
}
