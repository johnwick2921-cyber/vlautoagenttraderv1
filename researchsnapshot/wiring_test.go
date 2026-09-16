package researchsnapshot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// Behavioral pins exercise these producers too. This census additionally guards
// installation order and explicit successful/failing outcome hooks that need a
// broker or owner to exercise live. Comments do not count as call sites.
func TestStageAProductionWriterCensus(t *testing.T) {
	cases := []struct{ file, caller, call string }{
		{"main.go", "main", "Start"},
		{"store/armed_orders.go", "ExpirePlacement", "recordResearchPlacementTimeout"},
		{"trader/auto_trader_planner.go", "assemblePlannerInputWithCtx", "recordResearchCandidates"},
		{"trader/auto_trader_planner.go", "runPlannerReadCoreObserved", "Begin"},
		{"trader/auto_trader_planner.go", "runPlannerReadCoreObserved", "Reply"},
		{"trader/auto_trader_planner.go", "runPlannerReadCoreObserved", "Published"},
		{"trader/auto_trader_levelstate.go", "recordScenarioStateAt", "recordResearchPermissions"},
		{"trader/auto_trader_clock.go", "recordClosedTradeAnalyticsAt", "recordResearchOutcomeAt"},
		{"trader/auto_trader_orders.go", "executeDecisionWithRecord", "recordResearchGate"},
		{"trader/armed_executor.go", "maybeManageArmedOrdersAt", "recordResearchGate"},
		{"trader/detector_record.go", "recordDetectorOutputs", "recordResearchEpisodes"},
		{"provider/ninjatrader/tcp_server.go", "readLoop", "observe"},
		{"provider/ninjatrader/tcp_server.go", "SendSignal", "recordResearchSignal"},
		{"cmd/research_export/main.go", "run", "Export"},
		{"researchsnapshot/recorder.go", "perform", "Save"},
	}
	for _, c := range cases {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", c.file), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.Name != c.caller {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := ""
				switch x := call.Fun.(type) {
				case *ast.Ident:
					name = x.Name
				case *ast.SelectorExpr:
					name = x.Sel.Name
				}
				if name == c.call {
					n++
				}
				return true
			})
		}
		if n == 0 {
			t.Errorf("production call sites: 0 — %s:%s → %s", c.file, c.caller, c.call)
		}
	}
	f, err := parser.ParseFile(token.NewFileSet(), "../main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var install, load token.Pos
	ast.Inspect(f, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if s.Sel.Name == "Start" {
			if x, ok := s.X.(*ast.Ident); ok && x.Name == "researchsnapshot" {
				install = c.Pos()
			}
		}
		if s.Sel.Name == "LoadTradersFromStore" {
			load = c.Pos()
		}
		return true
	})
	if install == 0 || load == 0 || install >= load {
		t.Fatal("research recorder must install before trader/TCP producers start")
	}
}
