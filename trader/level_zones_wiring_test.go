package trader

import (
	"os"
	"strings"
	"testing"
)

// Pin actual dataflow statements, not presence of a helper name in comments.
// The separate kernel chart pin executes BuildPlannerPrompt with the full map.
func TestLevelZonesProductionWiring(t *testing.T) {
	claims := map[string][]string{
		"auto_trader_planner.go": {
			"kernel.LevelZoneInputs(researchRaw, zoneSeries, now)",
			"kernel.BuildLevelZones(researchRaw, price, in.ATR5m,",
			"in.Zones = &zoneView",
			"Zones: input.Zones",
			"doc.Zones = facts.Zones",
			"zoneSeries[tf] = series",
		},
		"auto_trader.go":                                 {"at.logLevelZonesBootAt(time.Now())"},
		"level_zones.go":                                 {"at.store.Trader().GetByID(at.id)", "kernel.ZoneBootLine(kernel.ResolveZoneOptions(cap))"},
		"../kernel/planner_prompt.go":                    {"b.WriteString(in.Zones.Render())"},
		"../kernel/levels_swing.go":                      {"out[len(out)-1].ZoneDefiningWick = &wick"},
		"../web/src/components/plan/SessionPlanCard.tsx": {"<LevelZoneMap map={doc.zone_map} />"},
	}
	for path, need := range claims {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var code []string
		for _, line := range strings.Split(string(b), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "//") {
				code = append(code, line)
			}
		}
		for _, s := range need {
			if !strings.Contains(strings.Join(code, "\n"), s) {
				t.Errorf("missing production dataflow %s: %s", path, s)
			}
		}
	}
}
