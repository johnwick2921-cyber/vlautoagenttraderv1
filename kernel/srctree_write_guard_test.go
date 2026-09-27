package kernel

// test-srctree-race GUARD — a test must NEVER write into the package source
// tree. A test that creates or removes a .go file under the repo (outside a
// TempDir/testdata fixture) races every other package's source scan in the
// same `go test -p N` run (the ENOENT DS-102 found: the writers-census test
// planted api/zz_writers_census_probe.go while this package's
// TestAcceptanceIntervalNoHardcodedSites walked api/*.go).
//
// This guard greps every *_test.go in the module for os.WriteFile / os.Create /
// ioutil.WriteFile calls whose FIRST argument is a repo-relative path (a
// literal or filepath.Join argument containing ".." traversal) and FAILS on
// any of them. Writes into t.TempDir()/testdata paths do not carry the
// traversal shape and pass.
//
// RED: this test fails on the pre-fix tree (the census probe wrote into api/).
// GREEN: the probe moved to a t.TempDir() fixture tree.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// srctreeWriteOffenderNames are the os functions whose first argument is the
// write target path.
var srctreeWriteOffenderNames = map[string]bool{
	"WriteFile": true,
	"Create":    true,
	"MkdirAll":  true,
	"Remove":    true,
	"Rename":    true,
}

// pathExprGoesUp reports whether a path expression starts with a ".."
// traversal (repo-relative) — the ONLY shape that can touch the source tree
// from a test — either directly or via a filepath.Join call.
func pathExprGoesUp(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return false
		}
		return strings.HasPrefix(strings.Trim(e.Value, `"`), "..")
	case *ast.CallExpr:
		// filepath.Join("..", "..", "api", …) — the census probe's shape.
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Join" {
			for _, arg := range e.Args {
				if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING &&
					strings.HasPrefix(strings.Trim(lit.Value, `"`), "..") {
					return true
				}
			}
		}
	}
	return false
}

// collectTraversingIdents finds `x := filepath.Join("..", …)` / `x = …`
// assignments in one file, so a WriteFile call whose target is that variable
// (the census probe's shape) is still caught.
func collectTraversingIdents(f *ast.File) map[string]bool {
	got := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			for i, rhs := range s.Rhs {
				if i >= len(s.Lhs) || !pathExprGoesUp(rhs) {
					continue
				}
				if id, ok := s.Lhs[i].(*ast.Ident); ok {
					got[id.Name] = true
				}
			}
		}
		return true
	})
	return got
}

// TestNoTestWritesIntoTheSourceTree is the guard itself.
func TestNoTestWritesIntoTheSourceTree(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	var offenders []string
	err = filepath.Walk(root, func(p string, info os.FileInfo, werr error) error {
		if werr != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", "vendor", "web", ".understand-anything", ".Codex":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		src, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, p, src, 0)
		if perr != nil {
			return nil
		}
		traversing := collectTraversingIdents(f)
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !srctreeWriteOffenderNames[sel.Sel.Name] {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			goesUp := pathExprGoesUp(call.Args[0])
			if id, ok := call.Args[0].(*ast.Ident); ok && traversing[id.Name] {
				goesUp = true
			}
			if !goesUp {
				return true
			}
			pos := fset.Position(call.Pos())
			rel, _ := filepath.Rel(root, p)
			offenders = append(offenders, rel+":"+pos.String()+": test calls os."+sel.Sel.Name+" with a repo-relative path — a test must never write into the source tree (use t.TempDir()/testdata)")
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range offenders {
		t.Errorf("%s", o)
	}
	if len(offenders) == 0 {
		t.Log("no test writes into the package source tree")
	}
}
