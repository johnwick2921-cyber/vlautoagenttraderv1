package updaterworker

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// PIN (U4F verify note 5, the one-canonicalizer canon): internal/updaterworker
// and cmd/nofx-updater have ONE containment helper, PathWithin. An AST scan of
// their non-test code finds every containment-by-text shape —
//   - a filepath.Rel result compared (== / !=) with a string literal starting
//     with "..", or handed to strings.HasPrefix;
//   - strings.HasPrefix(…, os.TempDir()) (the test guard U4F defect 5 retired)
//
// — and the ONLY one allowed is inside PathWithin itself. The scanner is proven
// on the two helpers it replaced (worker.go's within, releaseroot.go's outside)
// so a scan that sees nothing cannot pass by being blind.
func TestContainmentCensusOneHelper(t *testing.T) {
	var got []string
	for _, root := range []string{".", filepath.Join("..", "..", "cmd", "nofx-updater")} {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			src, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			for _, h := range containmentShapes(t, p, string(src)) {
				got = append(got, filepath.ToSlash(filepath.Clean(p))+":"+h)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(got)
	want := []string{"containment.go:PathWithin"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("containment-by-text sites = %q; want only %q — ask PathWithin (containment.go) instead", got, want)
	}
}

// The scanner sees both shapes it replaced, and a TempDir prefix guard.
func TestContainmentCensusScannerSeesTheRetiredShapes(t *testing.T) {
	for name, src := range map[string]string{
		"worker.go within (a \"../\" literal)": `package x
import ("path/filepath"; "strings")
func within(p, dir string) bool {
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(p))
	if err != nil { return false }
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, "../"))
}`,
		"releaseroot.go outside (the separator)": `package x
import ("path/filepath"; "strings")
func outside(p, dir string) bool {
	rel, err := filepath.Rel(dir, p)
	if err != nil { return false }
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}`,
		"a Rel result compared alone": `package x
import "path/filepath"
func esc(p, dir string) bool { r, _ := filepath.Rel(dir, p); return r == ".." }`,
		"a func literal in a package-level var": `package x
import ("path/filepath"; "strings")
var in = func(p, dir string) bool { r, _ := filepath.Rel(dir, p); return !strings.HasPrefix(r, "..") }`,
		"a var-declared Rel result (H1)": `package x
import ("path/filepath"; "strings")
func esc(p, dir string) bool {
	var r, _ = filepath.Rel(dir, p)
	return r == ".." || strings.HasPrefix(r, "../")
}`,
		"an ALIASED Rel import (H3)": `package x
import fp "path/filepath"
func esc(p, dir string) bool { r, _ := fp.Rel(dir, p); return r == ".." }`,
		"a TempDir prefix guard": `package x
import ("os"; "strings")
func guard(p string) bool { return strings.HasPrefix(p, os.TempDir()) }`,
	} {
		if hits := containmentShapes(t, "probe.go", src); len(hits) != 1 {
			t.Errorf("%s: scanner found %q, want exactly one site", name, hits)
		}
	}
	// and a Rel that is not compared (snapshot.go's relative names) is not containment
	if hits := containmentShapes(t, "probe.go", `package x
import "path/filepath"
func name(p, dir string) string { r, _ := filepath.Rel(dir, p); return filepath.ToSlash(r) }`); len(hits) != 0 {
		t.Errorf("scanner flagged a Rel that is only a relative name: %q", hits)
	}
}

// containmentShapes returns the enclosing function name of every
// containment-by-text shape in src (one entry per function; a package-level
// declaration — a func literal in a var — is scanned too). The scanner resolves
// the import NAMES itself (U4G defect 2, H3: `fp "path/filepath"; fp.Rel` is a
// Rel), and binds Rel results from var declarations too (H1: `var r, _ =
// filepath.Rel(...)` is not an AssignStmt).
func containmentShapes(t *testing.T, name, src string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), name, src, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	// the import names for the three packages the shapes name, by PATH.
	pkgs := map[string]map[string]bool{} // import path → the names it is imported as
	for _, im := range f.Imports {
		path, err := strconv.Unquote(im.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		if im.Name != nil {
			name = im.Name.Name
		}
		if pkgs[path] == nil {
			pkgs[path] = map[string]bool{}
		}
		pkgs[path][name] = true
	}
	isCall := func(e ast.Expr, pkgPath, fn string) bool {
		c, ok := e.(*ast.CallExpr)
		if !ok {
			return false
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		x, ok := sel.X.(*ast.Ident)
		return ok && sel.Sel.Name == fn && pkgs[pkgPath][x.Name]
	}
	relOf := func(e ast.Expr) (string, bool) {
		c, ok := e.(*ast.CallExpr)
		if !ok {
			return "", false
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return "", false
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok || sel.Sel.Name != "Rel" || !pkgs["path/filepath"][x.Name] {
			return "", false
		}
		return x.Name + ".Rel", true
	}
	dotdot := func(e ast.Expr) bool {
		l, ok := e.(*ast.BasicLit)
		if !ok || l.Kind != token.STRING {
			return false
		}
		s, err := strconv.Unquote(l.Value)
		return err == nil && strings.HasPrefix(s, "..")
	}
	var out []string
	for _, d := range f.Decls {
		// a function, or a package-level declaration (a func literal in a var)
		unit, unitName := ast.Node(d), "<package-level declaration>"
		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Body == nil {
				continue
			}
			unit, unitName = fd.Body, fd.Name.Name
		}
		rel := map[string]bool{} // identifiers bound to a filepath.Rel result
		bind := func(lhs ast.Expr) {
			if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" {
				rel[id.Name] = true
			}
		}
		ast.Inspect(unit, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				if len(x.Rhs) == 1 && len(x.Lhs) > 0 {
					if _, ok := relOf(x.Rhs[0]); ok {
						bind(x.Lhs[0])
					}
				}
			case *ast.GenDecl: // H1: a var-declared Rel result
				for _, spec := range x.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok && len(vs.Values) == 1 && len(vs.Names) > 0 {
						if _, ok := relOf(vs.Values[0]); ok {
							bind(vs.Names[0])
						}
					}
				}
			}
			return true
		})
		isRel := func(e ast.Expr) bool {
			if id, ok := e.(*ast.Ident); ok {
				return rel[id.Name]
			}
			_, ok := relOf(e)
			return ok
		}
		hit := false
		ast.Inspect(unit, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BinaryExpr:
				if (x.Op == token.EQL || x.Op == token.NEQ) && ((isRel(x.X) && dotdot(x.Y)) || (isRel(x.Y) && dotdot(x.X))) {
					hit = true
				}
			case *ast.CallExpr:
				if isCall(x, "strings", "HasPrefix") && len(x.Args) == 2 && (isRel(x.Args[0]) || isCall(x.Args[1], "os", "TempDir")) {
					hit = true
				}
			}
			return true
		})
		if hit {
			out = append(out, unitName)
		}
	}
	return out
}
