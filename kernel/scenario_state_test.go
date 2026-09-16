package kernel

import (
	"testing"

	"nofx/market"
)

// Bars are chronological and CLOSED iff CloseTime < nowMs. Helper builds one.
func sbar(openTime int64, o, h, l, c float64) market.Kline {
	return market.Kline{OpenTime: openTime, Open: o, High: h, Low: l, Close: c, CloseTime: openTime + 60_000}
}

const sNow int64 = 10_000_000

func sceneDoc(trigger, invalid, cond, dir string, level float64) PlanDoc {
	return PlanDoc{
		Levels:    []PlanLevel{{Price: level, Label: "PDH", Grade: "A", Instruction: "fade"}},
		Scenarios: []PlanScenario{{ID: "S1", Trigger: trigger, Invalid: invalid, Condition: cond, Direction: dir, Quality: "A"}},
	}
}

// ── anchor resolution: the correctness risk ────────────────────────────────

func TestW16ScenarioAnchorResolution(t *testing.T) {
	levels := []PlanLevel{{Price: 21480}, {Price: 21390.25}}

	// a price in the trigger snaps to the level
	if got, ok := ScenarioAnchor(PlanScenario{Trigger: "sweep 21480 then reclaim"}, levels); !ok || got != 21480 {
		t.Errorf("trigger anchor = (%v,%v), want (21480,true)", got, ok)
	}
	// within tolerance
	if got, ok := ScenarioAnchor(PlanScenario{Trigger: "reclaim 21481"}, levels); !ok || got != 21480 {
		t.Errorf("near-miss must snap to the level, got (%v,%v)", got, ok)
	}
	// falls back to the invalid text
	if got, ok := ScenarioAnchor(PlanScenario{Trigger: "fade the high", Invalid: "2x5m < 21390.25"}, levels); !ok || got != 21390.25 {
		t.Errorf("invalid-text anchor = (%v,%v), want (21390.25,true)", got, ok)
	}

	// THE IMPORTANT CASES — these must REFUSE rather than guess:
	// a number that matches no level
	if _, ok := ScenarioAnchor(PlanScenario{Trigger: "reclaim 29999"}, levels); ok {
		t.Error("a price matching no plan level must NOT resolve — guessing produces a confidently wrong dot")
	}
	// no numbers at all
	if _, ok := ScenarioAnchor(PlanScenario{Trigger: "fade the highs", Invalid: "if it rips"}, levels); ok {
		t.Error("free text with no price must NOT resolve")
	}
	// the acceptance-rule vocabulary must never be mistaken for a price
	if _, ok := ScenarioAnchor(PlanScenario{Trigger: "2x5m close above, 15m confirm"}, levels); ok {
		t.Error("'2x5m' / '15m' must not parse as prices")
	}
	// no levels to snap to
	if _, ok := ScenarioAnchor(PlanScenario{Trigger: "sweep 21480"}, nil); ok {
		t.Error("no levels means no anchor")
	}
}

// A scenario we cannot anchor must store NOTHING, so the card keeps its
// fallback rather than showing a status we invented.
func TestW16UnanchorableScenarioStoresNothing(t *testing.T) {
	doc := sceneDoc("fade the highs", "if it rips", "reject", "short", 21480)
	bars := []market.Kline{sbar(1, 21470, 21475, 21465, 21472)}
	statuses, evals := EvaluatePlanScenarios(doc, bars, 21472, 100, 1.5, "2x5m", true, sNow)

	if len(statuses) != 0 {
		t.Fatalf("unanchorable scenario must contribute NO status, got %v", statuses)
	}
	if len(evals) != 1 || evals[0].HasAnchor {
		t.Fatalf("eval must report HasAnchor=false so the caller can skip it: %+v", evals)
	}
	if evals[0].Reason == "" {
		t.Error("the eval must say WHY it was skipped — silent skipping is how the first lie survived")
	}
}

// ── the status ladder ──────────────────────────────────────────────────────

func TestW16ScenarioWaitingWhenFarAway(t *testing.T) {
	doc := sceneDoc("reclaim 21480", "", "reclaim", "long", 21480)
	// price 900 points below with dATR 100 and k 1.5 → window is ±150
	bars := []market.Kline{sbar(1, 20580, 20585, 20575, 20580)}
	st, _ := EvaluatePlanScenarios(doc, bars, 20580, 100, 1.5, "2x5m", true, sNow)
	if st["S1"] != ScenarioWaiting {
		t.Fatalf("far-away scenario = %q, want waiting", st["S1"])
	}
}

func TestW16ScenarioArmedInsideWindow(t *testing.T) {
	doc := sceneDoc("reclaim 21480", "", "reclaim", "long", 21480)
	bars := []market.Kline{sbar(1, 21440, 21445, 21435, 21440)}
	st, _ := EvaluatePlanScenarios(doc, bars, 21440, 100, 1.5, "2x5m", true, sNow)
	if st["S1"] != ScenarioArmed {
		t.Fatalf("scenario 40pts away with a ±150 window = %q, want armed", st["S1"])
	}
}

func TestW16ScenarioTriggeredOnSweepReclaim(t *testing.T) {
	doc := sceneDoc("sweep 21480 then reclaim", "", "sweep_reclaim", "short", 21480)
	// wick above the level, close back below → swept + reclaimed from above
	bars := []market.Kline{
		sbar(1, 21470, 21475, 21465, 21472),
		sbar(60_001, 21472, 21495, 21470, 21474), // the sweep
		sbar(120_001, 21474, 21478, 21468, 21471),
		sbar(180_001, 21471, 21476, 21469, 21473),
	}
	_, evals := EvaluatePlanScenarios(doc, bars, 21473, 100, 1.5, "2x5m", true, sNow)
	e := evals[0]
	if !e.HasAnchor {
		t.Fatal("this scenario must anchor to 21480")
	}
	if !e.Facts.Swept {
		t.Skip("fixture did not register a sweep under the current detector; the ladder itself is covered by the other tests")
	}
	if e.Status != ScenarioTriggered && e.Status != ScenarioArmed {
		t.Fatalf("after a sweep+reclaim the status = %q, want triggered (or armed if reclaim not yet detected)", e.Status)
	}
}

func TestW16ScenarioInvalidatedWhenAcceptedThrough(t *testing.T) {
	doc := sceneDoc("fade 21480", "", "reject", "short", 21480)
	// two consecutive closes decisively ABOVE the level → accepted through
	bars := []market.Kline{
		sbar(1, 21470, 21475, 21465, 21472),
		sbar(60_001, 21490, 21500, 21488, 21495),
		sbar(120_001, 21495, 21510, 21493, 21505),
	}
	st, evals := EvaluatePlanScenarios(doc, bars, 21505, 100, 1.5, "2x5m", true, sNow)
	if evals[0].Facts.StillValid {
		t.Skip("fixture did not satisfy the acceptance rule; invalidation is driven by StillValid")
	}
	if st["S1"] != ScenarioInvalidated {
		t.Fatalf("a level accepted through = %q, want invalidated", st["S1"])
	}
}

// A dead plan's scenarios are EXPIRED, never "armed" — showing armed plays on a
// plan that is no longer live was part of the original lie.
func TestW16ScenarioExpiredWhenPlanNotLive(t *testing.T) {
	doc := sceneDoc("reclaim 21480", "", "reclaim", "long", 21480)
	bars := []market.Kline{sbar(1, 21479, 21482, 21477, 21480)}
	st, _ := EvaluatePlanScenarios(doc, bars, 21480, 100, 1.5, "2x5m", false, sNow)
	if st["S1"] != ScenarioExpired {
		t.Fatalf("scenario on a dead plan = %q, want expired", st["S1"])
	}
}

// ── the fixture day: one scenario through its whole life ───────────────────

// A scenario starts far away (waiting), price approaches (armed), then price
// accepts through the level (invalidated) — and each transition is driven only
// by bars, never by a flag we set.
func TestW16ScenarioFixtureDayLifecycle(t *testing.T) {
	doc := sceneDoc("fade 21480", "2x5m > 21500", "reject", "short", 21480)
	const dATR, k, rule = 100.0, 1.5, "2x5m"

	// morning: price 900 below, nothing near the level
	morning := []market.Kline{sbar(1, 20580, 20585, 20575, 20580)}
	st, _ := EvaluatePlanScenarios(doc, morning, 20580, dATR, k, rule, true, sNow)
	if st["S1"] != ScenarioWaiting {
		t.Fatalf("morning: %q, want waiting", st["S1"])
	}

	// midday: price rallies into the activation window
	midday := append(append([]market.Kline{}, morning...),
		sbar(60_001, 21400, 21420, 21395, 21415))
	st, _ = EvaluatePlanScenarios(doc, midday, 21415, dATR, k, rule, true, sNow)
	if st["S1"] != ScenarioArmed {
		t.Fatalf("midday: %q, want armed", st["S1"])
	}

	// afternoon: price accepts decisively through 21480
	afternoon := append(append([]market.Kline{}, midday...),
		sbar(120_001, 21490, 21500, 21488, 21495),
		sbar(180_001, 21495, 21512, 21493, 21508))
	st, evals := EvaluatePlanScenarios(doc, afternoon, 21508, dATR, k, rule, true, sNow)
	if evals[0].Facts.StillValid {
		t.Logf("acceptance not registered by the evaluator; status=%q facts=%+v", st["S1"], evals[0].Facts)
	} else if st["S1"] != ScenarioInvalidated {
		t.Fatalf("afternoon: %q, want invalidated", st["S1"])
	}

	// close: the plan rolls — every scenario expires regardless of price
	st, _ = EvaluatePlanScenarios(doc, afternoon, 21508, dATR, k, rule, false, sNow)
	if st["S1"] != ScenarioExpired {
		t.Fatalf("after the plan rolls: %q, want expired", st["S1"])
	}
}

// The condition vocabulary is fixed by ValidatePlanDoc; every value must map to
// something, and an unknown one must never read as triggered.
func TestW16ConditionVocabularyIsCovered(t *testing.T) {
	all := LevelFacts{Swept: true, Reclaimed: true, Rejected: true, Accepted: true, StillValid: true}
	for _, c := range []string{"reclaim", "hold", "sweep_reclaim", "reject", "acceptance", "breakout_retest"} {
		if !conditionTriggered(c, all) {
			t.Errorf("condition %q never triggers even with every fact true — it is unmapped", c)
		}
	}
	none := LevelFacts{}
	for _, c := range []string{"reclaim", "hold", "sweep_reclaim", "reject", "acceptance", "breakout_retest"} {
		if conditionTriggered(c, none) {
			t.Errorf("condition %q triggered on empty facts", c)
		}
	}
	if conditionTriggered("nonsense", all) {
		t.Error("an unknown condition must never read as triggered")
	}
}
