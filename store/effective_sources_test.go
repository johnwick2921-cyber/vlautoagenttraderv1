package store

// W1 (g) — each (value, source) twin in effective_sources.go must equal the
// DayPlanConfig method it shadows, called as production calls it. At
// integration the methods delegate to the twins; this test keeps guarding them.

import "testing"

func TestEffectiveSourceTwinsMatchMethods(t *testing.T) {
	sp := func(s string) *string { return &s }
	ip := func(n int) *int { return &n }
	configs := []*DayPlanConfig{
		nil,
		{},
		{MinScenarioQuality: "b"},
		{MinScenarioQuality: "  ", Sessions: []DayPlanSessionOverride{{Session: "NY", MinScenarioQuality: sp("a")}}},
		{MinScenarioQuality: "B", Sessions: []DayPlanSessionOverride{{Session: "ny", MinScenarioQuality: sp(" "), MinGrade: sp(" a "),
			MaxTrades: ip(0), LastEntryOffsetMin: ip(30), EODFlatOffsetMin: ip(5)}}},
		{Sessions: []DayPlanSessionOverride{{Session: "NY", MaxTrades: ip(-1), LastEntryOffsetMin: ip(-5), EODFlatOffsetMin: ip(-1)}}},
	}
	for i, c := range configs {
		for _, s := range []string{"", "NY", "ny", "ASIA"} {
			if v, _ := MinScenarioQualityForWithSource(c, s); v != c.MinScenarioQualityFor(s) {
				t.Fatalf("cfg %d %q: MinScenarioQuality twin %q, method %q", i, s, v, c.MinScenarioQualityFor(s))
			}
			if v, _ := MinGradeForWithSource(c, s); v != c.MinGradeFor(s) {
				t.Fatalf("cfg %d %q: MinGrade twin %q, method %q", i, s, v, c.MinGradeFor(s))
			}
			n, ok, _ := MaxTradesForWithSource(c, s)
			if wn, wok := c.MaxTradesFor(s); n != wn || ok != wok {
				t.Fatalf("cfg %d %q: MaxTrades twin %d/%v, method %d/%v", i, s, n, ok, wn, wok)
			}
			if v, _ := LastEntryOffsetForWithSource(c, s); v != c.LastEntryOffsetFor(s) {
				t.Fatalf("cfg %d %q: LastEntry twin %d, method %d", i, s, v, c.LastEntryOffsetFor(s))
			}
			if v, _ := EODFlatOffsetForWithSource(c, s); v != c.EODFlatOffsetFor(s) {
				t.Fatalf("cfg %d %q: EODFlat twin %d, method %d", i, s, v, c.EODFlatOffsetFor(s))
			}
		}
	}
	// Sources name the layer that decided.
	c := configs[4]
	if _, src := MinScenarioQualityForWithSource(c, "NY"); src != SourceStrategyValue {
		t.Fatalf("blank session value falls through to the strategy: %q", src)
	}
	if _, src := MinGradeForWithSource(c, "NY"); src != SourceSessionOverride {
		t.Fatalf("min grade: %q", src)
	}
	if n, ok, src := MaxTradesForWithSource(c, "NY"); n != 0 || !ok || src != SourceSessionOverride {
		t.Fatalf("a 0 cap is meaningful: %d %v %q", n, ok, src)
	}
	if _, src := LastEntryOffsetForWithSource(configs[5], "NY"); src != SourceShippedDefault {
		t.Fatalf("a negative offset is ignored: %q", src)
	}
}
