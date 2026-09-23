package store

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// ── W-ONE-BUTTON M2 site 6 — nothing in the trading API writes the hold ────
//
// The hold is written and cleared ONLY by the local operator CLI (and, from
// M4, the updater worker — which must be added to this list deliberately, in
// the PR that builds it). No API route — resume, clear-freeze, anything
// future — may lift an installation-wide hold. This scans every non-test Go
// file in the module for calls to the three writers.
func TestOnlyTheOperatorCLIWritesTheMaintenanceHold(t *testing.T) {
	allowed := map[string]bool{
		"store/maintenance_hold.go":   true, // the definitions
		"internal/holdcli/holdcli.go": true, // cmd/maintenance-hold
	}
	writers := map[string]bool{"WriteMaintenanceHold": true, "ClearMaintenanceHold": true, "ForceClearMaintenanceHold": true}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	var offenders []string
	scanned := 0
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "web", "vendor", ".claude", ".Codex", ".understand-anything":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		f, perr := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if perr != nil {
			return nil // not ours to judge; go build will
		}
		scanned++
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			case *ast.Ident:
				name = fn.Name
			}
			if writers[name] && !allowed[rel] {
				offenders = append(offenders, rel+": "+name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 {
		t.Fatalf("scan saw only %d files — the walk is not covering the module", scanned)
	}
	if len(offenders) > 0 {
		t.Fatalf("the maintenance hold may be written/cleared only by the operator CLI (and the M4 updater, when it lands):\n%s", strings.Join(offenders, "\n"))
	}
}
