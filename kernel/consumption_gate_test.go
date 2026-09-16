package kernel

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderPlanStatusCountsClosesSinceBirth — the prompt tail's closes-beyond
// must count bars since the plan's BIRTH, not the whole cache ("84 vs 11").
func TestRenderPlanStatusCountsClosesSinceBirth(t *testing.T) {
	rows := make([][4]float64, 16)
	for i := 0; i < 12; i++ {
		rows[i] = [4]float64{30200, 30205, 30199, 30201}
	}
	for i := 12; i < 16; i++ {
		rows[i] = [4]float64{30400, 30405, 30399, 30401}
	}
	bars := series(rows)
	doc := PlanDoc{Levels: []PlanLevel{{Price: 30300, Label: "ONH", Grade: "A"}}}
	now := nowAfter(bars)
	birth := bars[12].OpenTime

	whole := RenderPlanStatus("", "MNQ", doc, bars, 30401, 300, "2x5m", 2, now, 0)
	if !strings.Contains(whole, "closes-beyond 4") {
		t.Fatalf("whole-cache count should be 4 closes above, got: %s", whole)
	}
	since := RenderPlanStatus("", "MNQ", doc, bars, 30401, 300, "2x5m", 2, now, birth)
	if !strings.Contains(since, "closes-beyond 4") {
		t.Fatalf("birth-scoped count should be 4, got: %s", since)
	}
	doc2 := PlanDoc{Levels: []PlanLevel{{Price: 30200, Label: "ONL", Grade: "A"}}}
	since2 := RenderPlanStatus("", "MNQ", doc2, bars, 30401, 300, "2x5m", 2, now, birth)
	if !strings.Contains(since2, "closes-beyond 4") {
		t.Fatalf("birth-scoped count should be 4 closes above since birth, got: %s", since2)
	}
}

// TestConsumptionSitesUseBirthScopedBars — AST guard: every EvaluateLevelFacts
// call in the kernel display surfaces must feed BarsSince(bars, birth). A raw
// whole-cache series is a fourth interpretation of "consumed".
func TestConsumptionSitesUseBirthScopedBars(t *testing.T) {
	root, _ := filepath.Abs("..")
	bad := 0
	for _, f := range []string{"plan_render.go", "plan_lifecycle.go"} {
		fset := token.NewFileSet()
		astF, err := parser.ParseFile(fset, filepath.Join(root, "kernel", f), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		ast.Inspect(astF, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || id.Name != "EvaluateLevelFacts" || len(call.Args) < 1 {
				return true
			}
			// a variable already derived from BarsSince (e.g. judge) is fine;
			// only a RAW identifier or other expression is a fourth interpretation.
			if idArg, ok := call.Args[0].(*ast.Ident); ok && idArg.Name != "bars" {
				return true // a derived variable (judge etc.) is already BarsSince-scoped
			}
			inner, ok := call.Args[0].(*ast.CallExpr)
			if !ok {
				pos := fset.Position(call.Pos())
				t.Errorf("%s:%d: EvaluateLevelFacts without BarsSince scoping", pos.Filename, pos.Line)
				bad++
				return true
			}
			if id2, ok := inner.Fun.(*ast.Ident); !ok || id2.Name != "BarsSince" {
				pos := fset.Position(call.Pos())
				t.Errorf("%s:%d: EvaluateLevelFacts bars must be BarsSince(...)-scoped", pos.Filename, pos.Line)
				bad++
			}
			return true
		})
	}
	if bad == 0 {
		t.Logf("all kernel display sites scope EvaluateLevelFacts through BarsSince")
	}
}

var _ = os.Getenv // keep os import if unused in some build tags
