package kernel

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/internal/censuswalk"
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
// acceptanceGuardOffenders is the shared scan: every non-test .go file
// under root (censuswalk) is checked for raw-counter call sites outside
// kernel/scenario_facts.go; offenders are returned as "rel:line name" strings.
func acceptanceGuardOffenders(root string) (offenders []string, err error) {
	rawCounters := map[string]bool{
		"Acceptance":          true,
		"ClosesBeyond":        true,
		"LevelStillValid":     true,
		"aggregateToMinutes":  true,
		"acceptanceTFMinutes": true,
		"acceptanceNeed":      true,
	}
	files, werr := censuswalk.NonTestGoFiles(root)
	if werr != nil {
		return nil, werr
	}
	for _, cf := range files {
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, cf.Path, nil, 0)
		if perr != nil {
			return nil, perr
		}
		if strings.HasSuffix(cf.Rel, "kernel/scenario_facts.go") {
			continue // the owner of the raw counters; composition lives here
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
				offenders = append(offenders, fmt.Sprintf("%s:%d: raw bar counter %s() outside kernel/scenario_facts.go — hardcoded acceptance interval; resolve via AcceptanceBars / LevelStillValidOn / EvaluateLevelFacts", cf.Rel, pos.Line, name))
			}
			return true
		})
	}
	return offenders, nil
}

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
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	offenders, err := acceptanceGuardOffenders(repoRoot)
	if err != nil {
		t.Fatalf("scan repo: %v", err)
	}
	for _, o := range offenders {
		t.Error(o)
	}
	if len(offenders) == 0 {
		t.Logf("all acceptance consumers resolve the rule timeframe (no raw-counter call sites outside kernel/scenario_facts.go)")
	}
}

// TestAcceptanceCensusSeesNestedSkipNamedDirs plants a raw-counter call in
// EVERY censuswalk.NestedProbeDirs directory of a synthetic module and asserts
// acceptanceGuardOffenders (the same scan) reports every one. With the old
// any-depth SkipDir the dirs named like a root skip were invisible (CLASS 258).
func TestAcceptanceCensusSeesNestedSkipNamedDirs(t *testing.T) {
	root := t.TempDir()
	dirs := censuswalk.NestedProbeDirs()
	for _, dir := range dirs {
		full := filepath.Join(root, filepath.FromSlash(dir))
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
		src := "package " + censuswalk.PackageName(dir) + "\n\nfunc offender() { _ = Acceptance() }\n"
		if err := os.WriteFile(filepath.Join(full, "offender.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	offenders, err := acceptanceGuardOffenders(root)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, o := range offenders {
		seen[strings.SplitN(o, ":", 2)[0]] = true
	}
	var missed []string
	for _, dir := range dirs {
		if !seen[dir+"/offender.go"] {
			missed = append(missed, dir)
		}
	}
	if len(missed) > 0 {
		t.Fatalf("the acceptance guard skipped %d of %d nested probe dirs — a skip by NAME at depth exempts compiled packages (CLASS 258):\n\t%s",
			len(missed), len(dirs), strings.Join(missed, "\n\t"))
	}
}
