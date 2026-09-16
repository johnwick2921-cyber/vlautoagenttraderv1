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

// TestAcceptanceIntervalNoHardcodedSites is the H10 guard: the raw bar counters
// count BARS of whatever series they are handed, so a call site that feeds them
// its own series IS a hardcoded acceptance interval. They may only be composed
// inside kernel/scenario_facts.go, which owns the timeframe resolution. Every
// other site must go through AcceptanceBars / LevelStillValidOn /
// EvaluateLevelFacts — if any file outside scenario_facts.go calls a raw counter,
// this test fails and names the site.
//
// This is the same class of test as TestSessionEndSingleSourceOfTruth: it fails
// on future hardcoding, not just on today's bug.
func TestAcceptanceIntervalNoHardcodedSites(t *testing.T) {
	rawCounters := map[string]bool{
		"Acceptance":          true,
		"ClosesBeyond":        true,
		"LevelStillValid":     true,
		"aggregateToMinutes":  true,
		"acceptanceTFMinutes": true,
		"acceptanceNeed":      true,
	}
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	bad := 0
	err = filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", "data", "web", "sandbox", "docs", ".understand-anything", ".claude", "ninjascript", "screenshots":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(path)
		if strings.HasSuffix(rel, "kernel/scenario_facts.go") {
			return nil // the owner of the raw counters; composition lives here
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				name = fn.Name
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			}
			if rawCounters[name] {
				pos := fset.Position(call.Pos())
				t.Errorf("%s:%d: raw bar counter %s() outside kernel/scenario_facts.go — hardcoded acceptance interval; resolve via AcceptanceBars / LevelStillValidOn / EvaluateLevelFacts", pos.Filename, pos.Line, name)
				bad++
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("scan repo: %v", err)
	}
	if bad == 0 {
		t.Logf("all acceptance consumers resolve the rule timeframe (no raw-counter call sites outside kernel/scenario_facts.go)")
	}
}
