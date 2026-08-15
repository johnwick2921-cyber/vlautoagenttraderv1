package kernel

import (
	"strings"
	"testing"
)

func samplePlannerInput() PlannerInput {
	return PlannerInput{
		TradeDate: "2026-08-14",
		Session:   "NY",
		ReadKind:  "read 08:25 CT (live)",
		Price:     15600,
		DATR:      120,
		Regime:    RegimeBlock{TrendDaily: "up", Trend1h: "up", ATR14: 118, ATRRegime: "NORMAL", ATRPercentile: 55, VIXRegime: "unavailable"},
		Levels: []ScoredLevel{
			{DetectedLevel: DetectedLevel{Kind: KindPDH, Price: 15620, Label: "PDH"}, Grade: "A", Fresh: "fresh", Distance: 20},
			{DetectedLevel: DetectedLevel{Kind: KindPDL, Price: 15450, Label: "PDL"}, Grade: "A", Fresh: "fresh", Distance: -150},
		},
		StructureSummary: []string{"D: up, above EMA200", "1h: up, holding VWAP"},
		OvernightStory:   "swept ONH, reclaimed",
		PriorDayStory:    "trend up into the close",
		Calendar:         []PlannerCalendarEvent{{TimeCT: "13:00", Currency: "USD", Title: "FOMC", Impact: "T1"}},
		DigestChain:      []string{"yesterday: trend day, +2R captured", "Mon: balance"},
		OwnerNote:        "respect PDH, don't chase",
	}
}

func TestBuildPlannerPrompt(t *testing.T) {
	p := BuildPlannerPrompt(samplePlannerInput())
	for _, want := range []string{
		"DAY-PLAN READER", "trade_date 2026-08-14 · session NY",
		"REGIME: trend D=up", "Ranked levels", "PDH", "Calendar", "FOMC", "HARD no-trade blackout",
		"Owner note", "respect PDH", `"reasoning"`, "sweep_reclaim", "death_condition",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("planner prompt missing %q\n---\n%s", want, p)
		}
	}
}

// TestSamplePlannerPrompt logs the assembled prompt for the exit-bar report.
func TestSamplePlannerPrompt(t *testing.T) {
	t.Logf("\n%s", BuildPlannerPrompt(samplePlannerInput()))
}
