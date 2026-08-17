package kernel

import (
	"strings"
	"testing"

	"nofx/market"
)

// W11b — with a LevelStateProvider installed, ScoreLevels DROPS a consumed/burned
// level (freshMult 0) and LABELS a decayed one (B/tested); nil provider → all fresh.
func TestW11bScoreLevelsSurfacesPersistedState(t *testing.T) {
	defer func() { LevelStateProvider = nil }() // never leak into the golden tests

	burned := DetectedLevel{Kind: KindPDH, Price: 30050, Label: "PDH"}
	decayed := DetectedLevel{Kind: KindPDL, Price: 29970, Label: "PDL"}
	levels := []DetectedLevel{burned, decayed}

	// baseline: no provider → both fresh, both seated.
	LevelStateProvider = nil
	base := ScoreLevels(levels, 30000, 100, levelFreshnessFn("MNQ"), 8, 1.5)
	if len(base) != 2 {
		t.Fatalf("no provider → both levels fresh+seated, got %d", len(base))
	}

	// provider: PDH burned (done), PDL decayed (B).
	LevelStateProvider = func(symbol string, l DetectedLevel) string {
		switch l.Label {
		case "PDH":
			return "done"
		case "PDL":
			return "B"
		}
		return ""
	}
	got := ScoreLevels(levels, 30000, 100, levelFreshnessFn("MNQ"), 8, 1.5)
	if len(got) != 1 {
		t.Fatalf("burned PDH must be dropped → 1 level, got %d", len(got))
	}
	if got[0].Label != "PDL" || got[0].Fresh != "B" {
		t.Fatalf("PDL must survive labeled B, got label=%q fresh=%q", got[0].Label, got[0].Fresh)
	}
}

// W11b — PLAN STATUS annotates persisted state; nil provider → no annotation
// (byte-identical to the pre-W11b line, which the golden captures).
func TestW11bPlanStatusAnnotation(t *testing.T) {
	defer func() { LevelStateProvider = nil }()

	doc := PlanDoc{Levels: []PlanLevel{
		{Price: 30050, Label: "PDH"},
		{Price: 29970, Label: "PDL"},
	}}
	bars := []market.Kline{{OpenTime: 1, Open: 30000, High: 30010, Low: 29990, Close: 30000, CloseTime: 2}}

	// nil provider → no persisted annotation.
	LevelStateProvider = nil
	if s := RenderPlanStatus("MNQ", doc, bars, 30000, 100, "2x5m", 2, 100); strings.Contains(s, "BURNED") || strings.Contains(s, "state=") {
		t.Fatalf("nil provider must not annotate:\n%s", s)
	}

	// provider → burned/decayed annotations on the matching level lines.
	LevelStateProvider = func(symbol string, l DetectedLevel) string {
		if l.Label == "PDH" {
			return "done"
		}
		if l.Label == "PDL" {
			return "B"
		}
		return ""
	}
	s := RenderPlanStatus("MNQ", doc, bars, 30000, 100, "2x5m", 2, 100)
	if !strings.Contains(s, "BURNED(prior session)") {
		t.Fatalf("burned PDH must be annotated BURNED:\n%s", s)
	}
	if !strings.Contains(s, "state=B(tested)") {
		t.Fatalf("decayed PDL must be annotated state=B:\n%s", s)
	}
}
