package trader

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// The dispatch's claim list is pinned to production files. Test/helper callers
// cannot make a disconnected feature pass this wiring gate.
func TestIdentityProductionWiring(t *testing.T) {
	claims := map[string][]string{
		"kernel/levels_assemble.go":        {"CaptureIdentityContext"},
		"kernel/levels_intraday.go":        {"WithFormationClose"},
		"kernel/map_candidates.go":         {"CandidateIdentity"},
		"kernel/planner_prompt.go":         {"RenderIdentityMapBlock"},
		"trader/auto_trader_planner.go":    {"stampPlanIdentity", "recordPlanIdentity"},
		"trader/auto_trader_levelstate.go": {"observeScenarioIdentity"},
		"trader/detector_record.go":        {"EpisodeLevelID"},
		"trader/episode_close_wiring.go":   {"EpisodeScenarioByID"},
		"trader/desk_facts.go":             {"ScenarioIdentities"},
		"api/handler_plan.go":              {"ScenarioIdentities"},
		"trader/auto_trader.go":            {"logLevelIdentityBootAt"},
		"trader/research_snapshot.go":      {"recordResearchCandidateIdentity"},
	}
	for path, names := range claims {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", path), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		calls := map[string]int{}
		ast.Inspect(f, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fn := c.Fun.(type) {
			case *ast.Ident:
				calls[fn.Name]++
			case *ast.SelectorExpr:
				calls[fn.Sel.Name]++
			}
			return true
		})
		for _, name := range names {
			if calls[name] == 0 {
				t.Errorf("production call sites: 0 — %s → %s", path, name)
			} else {
				t.Logf("%s → %s: %d", path, name, calls[name])
			}
		}
	}
}
