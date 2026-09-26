// Settings truth W1 (g) — the per-session DayPlanConfig resolvers with their
// source, so the Settings page can say WHERE a per-session value came from.
// The methods (MinScenarioQualityFor, MinGradeFor, MaxTradesFor,
// LastEntryOffsetFor, EODFlatOffsetFor) delegate here — one rule, one place
// (resolve_source.go's pattern); effective_sources_test.go pins the pairs.

package store

import "strings"

// MinScenarioQualityForWithSource is DayPlanConfig.MinScenarioQualityFor with its
// source: per-session override → strategy value → shipped default "C".
func MinScenarioQualityForWithSource(c *DayPlanConfig, session string) (string, string) {
	floor, src := "C", SourceShippedDefault
	if c != nil && strings.TrimSpace(c.MinScenarioQuality) != "" {
		floor, src = strings.ToUpper(strings.TrimSpace(c.MinScenarioQuality)), SourceStrategyValue
	}
	if ov := c.SessionOverride(session); ov != nil && ov.MinScenarioQuality != nil && strings.TrimSpace(*ov.MinScenarioQuality) != "" {
		floor, src = strings.ToUpper(strings.TrimSpace(*ov.MinScenarioQuality)), SourceSessionOverride
	}
	return floor, src
}

// MinGradeForWithSource is DayPlanConfig.MinGradeFor with its source: the
// per-session override, else "" (no floor) from the shipped default.
func MinGradeForWithSource(c *DayPlanConfig, session string) (string, string) {
	if ov := c.SessionOverride(session); ov != nil && ov.MinGrade != nil {
		return strings.ToUpper(strings.TrimSpace(*ov.MinGrade)), SourceSessionOverride
	}
	return "", SourceShippedDefault
}

// MaxTradesForWithSource is DayPlanConfig.MaxTradesFor with its source. ok=false
// means no per-session cap (the strategy-level daily guardrail still applies).
func MaxTradesForWithSource(c *DayPlanConfig, session string) (int, bool, string) {
	if ov := c.SessionOverride(session); ov != nil && ov.MaxTrades != nil && *ov.MaxTrades >= 0 {
		return *ov.MaxTrades, true, SourceSessionOverride
	}
	return 0, false, SourceShippedDefault
}

// LastEntryOffsetForWithSource is DayPlanConfig.LastEntryOffsetFor with its source.
func LastEntryOffsetForWithSource(c *DayPlanConfig, session string) (int, string) {
	if ov := c.SessionOverride(session); ov != nil && ov.LastEntryOffsetMin != nil && *ov.LastEntryOffsetMin >= 0 {
		return *ov.LastEntryOffsetMin, SourceSessionOverride
	}
	return DefaultLastEntryOffsetMin, SourceShippedDefault
}

// EODFlatOffsetForWithSource is DayPlanConfig.EODFlatOffsetFor with its source.
func EODFlatOffsetForWithSource(c *DayPlanConfig, session string) (int, string) {
	if ov := c.SessionOverride(session); ov != nil && ov.EODFlatOffsetMin != nil && *ov.EODFlatOffsetMin >= 0 {
		return *ov.EODFlatOffsetMin, SourceSessionOverride
	}
	return DefaultEODFlatOffsetMin, SourceShippedDefault
}
