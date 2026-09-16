package trader

import (
	"encoding/json"
	"testing"
	"time"

	"nofx/kernel"
)

// NO-TRADE BAND (2026-09-02) — the WRITE SITE. The card can only be honest if
// the machine actually stamps the windows on the doc; before this wave the doc
// carried the model's prose and nothing else, so the read-time evaluator had
// nothing to evaluate and the card fell back to the prose it was written with.
func TestNoTradeBandWriteSiteStampsMachineWindows(t *testing.T) {
	at := plannerTestTrader(t)
	facts := kernel.PlanFacts{Price: 15550, DATR: 300}
	machine := map[float64]string{15480: "PWL", 15700: "RN 15700"}

	ver, lc, err := at.runPlannerReadCoreWithFactsGrades("NY", "2026-09-01", "owner_reset",
		"deepseek-v4-pro", "hashBand1", "", "", "", "PROMPT", facts, nil, machine, nil, true,
		func(string) (string, error) { return class39LegsPlanJSON("15550"), nil })
	if err != nil || lc != "active" {
		t.Fatalf("write must land: ver=%d lc=%q err=%v", ver, lc, err)
	}

	row, err := at.store.Plan().GetLatestPlanForSession("2026-09-01", "NY")
	if err != nil || row == nil {
		t.Fatalf("stored plan row: %v", err)
	}
	var doc kernel.PlanDoc
	if err := json.Unmarshal([]byte(row.Doc), &doc); err != nil {
		t.Fatalf("stored doc: %v", err)
	}

	if len(doc.NoTradeWindows) == 0 {
		t.Fatal("the machine wrote NO no_trade_windows — the card has nothing to evaluate and falls back to prose")
	}
	kinds := map[string]kernel.NoTradeWindow{}
	for _, w := range doc.NoTradeWindows {
		kinds[w.Kind] = w
	}
	first, okF := kinds[kernel.KindFirstN]
	if !okF {
		t.Fatal("no first-N window on the doc")
	}
	lunch, okL := kinds[kernel.KindLunch]
	if !okL {
		t.Fatal("no lunch window on the doc")
	}

	// Every bound must match the enforcing definition, not a copy.
	sess, ok := at.sessionRegistry(time.Now()).SessionByName("NY")
	if !ok {
		t.Fatal("NY missing from the registry")
	}
	wantFirst := kernel.BuildMachineNoTradeWindows(*sess)
	for _, w := range wantFirst {
		if w.Kind == kernel.KindFirstN && (w.StartMin != first.StartMin || w.EndMin != first.EndMin) {
			t.Errorf("first-N on the doc is %d–%d, the gate refuses %d–%d", first.StartMin, first.EndMin, w.StartMin, w.EndMin)
		}
	}
	ls, le := kernel.LunchWindowCT()
	if kernel.HHMM(lunch.StartMin) != ls || kernel.HHMM(lunch.EndMin) != le {
		t.Errorf("lunch on the doc is %s–%s, the gate refuses %s–%s",
			kernel.HHMM(lunch.StartMin), kernel.HHMM(lunch.EndMin), ls, le)
	}
	if first.Source != kernel.SourceCodeConstant || lunch.Source != kernel.SourceCodeConstant {
		t.Errorf("both fixed windows are code constants; the card must not imply a knob: %q / %q", first.Source, lunch.Source)
	}

	// The model's prose is left exactly as authored — this wave adds a field,
	// it does not edit what the model wrote.
	if len(doc.NoTrade) != 1 || doc.NoTrade[0] != "first 5m" {
		t.Errorf("the model's no_trade prose was modified: %+v", doc.NoTrade)
	}
}
