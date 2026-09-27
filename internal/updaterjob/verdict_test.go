package updaterjob

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The verdict reader is app-linkable, so it must not be able to WRITE a
// verdict: no call in verdict.go may create, link, rename, chmod or remove a
// file, and it imports no package that could (the worker's FetchRelease is the
// one writer — internal/updaterworker, which the trading app cannot link).
// The behaviour (FetchRelease writes, ReadVerdict reads) is pinned at the
// worker's call sites in internal/updaterworker/release_test.go.
func TestVerdictFileHasNoWriter(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "verdict.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	allowedImports := map[string]bool{
		"bytes": true, "encoding/json": true, "errors": true, "fmt": true, "io": true, "io/fs": true,
		"os": true, "path/filepath": true, "regexp": true, "strings": true, "syscall": true, "time": true,
		"nofx/internal/updaterwire": true,
	}
	for _, im := range f.Imports {
		p, _ := strconv.Unquote(im.Path.Value)
		if !allowedImports[p] {
			t.Errorf("verdict.go imports %s — the verdict reader imports only the standard library it reads with and updaterwire", p)
		}
	}
	writers := map[string]bool{
		"WriteFile": true, "Create": true, "CreateTemp": true, "Link": true, "Symlink": true, "Rename": true,
		"Mkdir": true, "MkdirAll": true, "MkdirTemp": true, "Remove": true, "RemoveAll": true, "Chmod": true,
		"Chown": true, "Lchown": true, "Chtimes": true, "Truncate": true, "Write": true, "WriteString": true,
		"Mkfifo": true, "Mknod": true, "Unlink": true,
	}
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if ok && writers[sel.Sel.Name] {
			t.Errorf("verdict.go references %s — the app-linkable verdict reader may not write", sel.Sel.Name)
		}
		if id, ok := n.(*ast.Ident); ok && (id.Name == "O_WRONLY" || id.Name == "O_RDWR" || id.Name == "O_CREATE" || id.Name == "O_CREAT" || id.Name == "O_TRUNC" || id.Name == "O_APPEND") {
			t.Errorf("verdict.go opens with %s — the app-linkable verdict reader may not write", id.Name)
		}
		return true
	})
	t.Run("the package: no file but verdict.go names the verdict path", testVerdictPathOnlyInVerdictGo)
}

// verdictPathNames are the identifiers that name WHERE a verdict lives.
var verdictPathNames = map[string]bool{"VerdictPath": true, "verdictsDirName": true}

// testVerdictPathOnlyInVerdictGo — CTO ruling 1790279155144 (3): the no-writer
// pin covers the whole internal/updaterjob PACKAGE, not one file. verdict.go
// is judged above to write nothing; a SECOND file that could reach the
// verdict's path could write there and the pin above would never see it
// (the class-88 shape: a second writer beside the one that was checked). So
// every NON-test .go file in the package is parsed — whatever its build tags
// — and any file other than verdict.go that references VerdictPath or
// verdictsDirName as an identifier or selector (a call, a function-value
// alias, a Join argument), or spells the "verdicts" directory in a string
// literal, is refused. A comment mention is not a reference and passes.
// Inside verdict.go, VerdictPath may only be CALLED, and only inside
// ReadVerdict; verdictsDirName is used only in VerdictPath's own body; and the
// "verdicts" literal appears only as verdictsDirName's value — so verdict.go
// cannot hand a second name for either (a wrapper, a second const) to another
// file (verifier f13 defects 2+3). Limits are named at checkVerdictGoPathUses.
func testVerdictPathOnlyInVerdictGo(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var walked []string
	sawInVerdictGo := false
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		walked = append(walked, name)
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if name == "verdict.go" {
			sawInVerdictGo = checkVerdictGoPathUses(t, f)
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident: // also the Sel of every selector
				if verdictPathNames[n.Name] {
					t.Errorf("%s references %s — only verdict.go may name the verdict path (the verdict's one writer is internal/updaterworker)", name, n.Name)
				}
			case *ast.BasicLit:
				if spellsVerdictsDir(n) {
					t.Errorf("%s spells the verdict directory %q in a string literal (%s) — only verdict.go may name the verdict path", name, verdictsDirName, n.Value)
				}
			}
			return true
		})
	}
	sort.Strings(walked)
	if !sawInVerdictGo || len(walked) < 2 {
		t.Fatalf("the walk is vacuous: files %v, verdict.go references seen = %v", walked, sawInVerdictGo)
	}
}

// spellsVerdictsDir reports whether a string literal names the verdict
// directory as one of its path segments.
func spellsVerdictsDir(n *ast.BasicLit) bool {
	if n.Kind != token.STRING {
		return false
	}
	v, err := strconv.Unquote(n.Value)
	if err != nil {
		return false
	}
	for _, seg := range strings.Split(filepath.ToSlash(v), "/") {
		if seg == verdictsDirName {
			return true
		}
	}
	return false
}

// checkVerdictGoPathUses holds verdict.go to CALLING VerdictPath only inside
// ReadVerdict (its one reader today — a wrapper elsewhere in verdict.go would
// hand the path to another file under a name the walk does not know), using
// verdictsDirName only inside VerdictPath, and spelling the "verdicts"
// directory in exactly ONE string literal: the verdictsDirName const's own
// value (a second const/var with that value would be a second name the walk
// does not know). It reports whether it saw the call and the use.
//
// Named limit [B]: a directory name BUILT from pieces ("verdict"+"s", a
// runtime concatenation, a byte slice) is not a literal that spells it and is
// not caught here; neither is a caller OUTSIDE this package that takes
// VerdictPath's result and writes (only internal/updaterworker does today).
func checkVerdictGoPathUses(t *testing.T, f *ast.File) bool {
	t.Helper()
	called, used := false, false
	allowed := map[*ast.Ident]bool{}
	allowedLit := map[*ast.BasicLit]bool{}
	var readVerdict *ast.FuncDecl
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == "ReadVerdict" && d.Recv == nil {
				readVerdict = d
			}
			if d.Name.Name == "VerdictPath" && d.Recv == nil {
				allowed[d.Name] = true
				ast.Inspect(d.Body, func(n ast.Node) bool {
					if id, ok := n.(*ast.Ident); ok && id.Name == "verdictsDirName" {
						allowed[id], used = true, true
					}
					return true
				})
			}
		case *ast.GenDecl:
			for _, sp := range d.Specs {
				if vs, ok := sp.(*ast.ValueSpec); ok && d.Tok == token.CONST {
					for i, nm := range vs.Names {
						if nm.Name == "verdictsDirName" {
							allowed[nm] = true
							if i < len(vs.Values) {
								if lit, ok := vs.Values[i].(*ast.BasicLit); ok {
									allowedLit[lit] = true
								}
							}
						}
					}
				}
			}
		}
	}
	if readVerdict == nil || readVerdict.Body == nil {
		t.Errorf("verdict.go has no ReadVerdict — the one place VerdictPath may be called is gone; re-judge this pin")
	} else {
		ast.Inspect(readVerdict.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "VerdictPath" {
					allowed[id], called = true, true
				}
			}
			return true
		})
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			if verdictPathNames[n.Name] && !allowed[n] {
				t.Errorf("verdict.go uses %s other than by a call inside ReadVerdict (VerdictPath) or inside VerdictPath (verdictsDirName) — a second name for the verdict path could reach another file", n.Name)
			}
		case *ast.BasicLit:
			if spellsVerdictsDir(n) && !allowedLit[n] {
				t.Errorf("verdict.go spells the verdict directory %q in a string literal (%s) other than verdictsDirName's own value — a second name for the verdict path could reach another file", verdictsDirName, n.Value)
			}
		}
		return true
	})
	return called && used
}
