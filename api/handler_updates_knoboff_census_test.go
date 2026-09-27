package api

// #206 review fold (api/handler_updates_jobs_test.go:118): the knob-OFF pins
// only checked that NO LOG LINE appears — a silent job-file read, worker dial
// or jobs listing with the knob OFF stayed green. This is a CENSUS over the
// production branches: with the knob OFF, handleUpdatesJob's 404 branch may
// contain exactly the 404 emission (one c.JSON call) and a return, and
// configureUpdater's OFF branch may contain NOTHING but a return — any call
// added there (fs, dial, list) fails the census by construction, logged or
// not. The behavioural pin (a loose jobs dir is invisible) stays beside it.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func updatesHandlerAST(t *testing.T) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "handler_updates.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func funcDecl(t *testing.T, f *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == name {
			return fd
		}
	}
	t.Fatalf("func %s not found", name)
	return nil
}

// offCalls returns every CallExpr inside the OFF branch (the first if
// statement of the function) that is NOT the one c.JSON 404 emission.
func offCalls(t *testing.T, f *ast.File, fn string, allowJSON404 bool) []string {
	t.Helper()
	fd := funcDecl(t, f, fn)
	var off *ast.BlockStmt
	for _, st := range fd.Body.List {
		if is, ok := st.(*ast.IfStmt); ok {
			off = is.Body
			break
		}
	}
	if off == nil {
		t.Fatalf("%s has no OFF branch (no leading if)", fn)
	}
	var calls []string
	ast.Inspect(off, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := ce.Fun.(*ast.SelectorExpr)
		if allowJSON404 && ok {
			if x, isIdent := sel.X.(*ast.Ident); isIdent && x.Name == "c" && sel.Sel.Name == "JSON" {
				return true // the one permitted call
			}
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%s(", funcName(ce.Fun))
		for i, a := range ce.Args {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%T", a)
		}
		b.WriteString(")")
		calls = append(calls, b.String())
		return true
	})
	return calls
}

func funcName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return funcName(x.X) + "." + x.Sel.Name
	}
	return fmt.Sprintf("%T", e)
}

func TestJobRoutesKnobOffIsACensusPure404(t *testing.T) {
	f := updatesHandlerAST(t)

	// handleUpdatesJob: OFF = the literal 404, one c.JSON call, nothing else.
	if calls := offCalls(t, f, "handleUpdatesJob", true); len(calls) != 0 {
		t.Fatalf("handleUpdatesJob's knob-OFF branch calls %v — OFF must reach the 404 with no filesystem, dial or list", calls)
	}

	// configureUpdater: OFF = bare return, not even the glue's probe.
	if calls := offCalls(t, f, "configureUpdater", false); len(calls) != 0 {
		t.Fatalf("configureUpdater's knob-OFF branch calls %v — OFF must run nothing", calls)
	}
}
