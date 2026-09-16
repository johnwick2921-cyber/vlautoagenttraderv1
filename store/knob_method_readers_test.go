package store

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// ── CLEANUP BATCH 2, B1 — THE REGISTRY'S FIELD GREP MISSED METHOD READERS ─────
//
// Six wake knobs rendered "no known reader — pending verification" in the
// settings panel while being fully wired: each is read through a DayPlanConfig
// ACCESSOR METHOD (WakeOn15mZoneEnabled(), WakeMinIntervalMinutes(), …) and the
// 2026-09-03 sweep grepped for the FIELD. The registry's own note said a
// method-based reader would not appear. This is the method-level detector that
// note asked for, as code: for every candidate-unverified knob, find the struct
// field behind its json leaf, find methods on the owning type whose body reads
// that field, and find production call sites of those methods. A candidate with
// a method reader is misclassified, and this pin names the reader.

// knobFieldOwner walks StrategyConfig by reflection and returns, for a json
// leaf, every (ownerTypeName, GoFieldName) that carries it.
func knobFieldOwners(leaf string) [][2]string {
	var out [][2]string
	seen := map[string]bool{}
	var walk func(t reflect.Type, depth int)
	walk = func(t reflect.Type, depth int) {
		if depth > 6 || t == nil {
			return
		}
		for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct || seen[t.Name()] {
			return
		}
		seen[t.Name()] = true
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")[0]
			ft := f.Type
			for ft.Kind() == reflect.Ptr || ft.Kind() == reflect.Slice {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				walk(ft, depth+1)
				continue
			}
			if tag == leaf {
				out = append(out, [2]string{t.Name(), f.Name})
			}
		}
	}
	walk(reflect.TypeOf(StrategyConfig{}), 0)
	return out
}

// repoGoFiles lists every non-test .go file under the module root.
func repoGoFiles(t *testing.T) (root string, files []string) {
	t.Helper()
	root = ".."
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := info.Name()
			if base == "node_modules" || base == ".git" || base == "web" || (strings.HasPrefix(base, ".") && base != "." && base != "..") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return root, files
}

// receiverTypeName returns the bare type name of a method receiver ("*DayPlanConfig" → "DayPlanConfig").
func receiverTypeName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	switch t := fd.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// methodReadsField reports whether a method body contains <recv>.<field>.
func methodReadsField(fd *ast.FuncDecl, field string) bool {
	if fd.Body == nil || fd.Recv == nil || len(fd.Recv.List) == 0 || len(fd.Recv.List[0].Names) == 0 {
		return false
	}
	recv := fd.Recv.List[0].Names[0].Name
	found := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == recv && sel.Sel.Name == field {
				found = true
			}
		}
		return !found
	})
	return found
}

// MethodReadersOfKnob is the detector. It returns "file:line" production call
// sites of accessor methods that read the knob's field, plus the accessor
// names, or nothing.
func methodReadersOfKnob(t *testing.T, leaf string) (accessors []string, callSites []string) {
	t.Helper()
	owners := knobFieldOwners(leaf)
	if len(owners) == 0 {
		return nil, nil
	}
	root, files := repoGoFiles(t)
	fset := token.NewFileSet()
	parsed := map[string]*ast.File{}
	for _, f := range files {
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			continue
		}
		parsed[f] = af
	}
	// pass 1: accessor methods on the owning type that read the field
	acc := map[string]bool{}
	for _, af := range parsed {
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil {
				continue
			}
			for _, o := range owners {
				if receiverTypeName(fd) == o[0] && methodReadsField(fd, o[1]) {
					acc[fd.Name.Name] = true
				}
			}
		}
	}
	if len(acc) == 0 {
		return nil, nil
	}
	// pass 2: call sites of those accessors, anywhere in production code,
	// excluding the accessors' own package-internal definitions.
	for f, af := range parsed {
		ast.Inspect(af, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !acc[sel.Sel.Name] {
				return true
			}
			pos := fset.Position(call.Pos())
			rel, _ := filepath.Rel(root, f)
			callSites = append(callSites, fmt.Sprintf("%s:%d", rel, pos.Line))
			return true
		})
	}
	for a := range acc {
		accessors = append(accessors, a)
	}
	sort.Strings(accessors)
	sort.Strings(callSites)
	return accessors, callSites
}

// A CANDIDATE-UNVERIFIED KNOB HAS NO METHOD READER. If it has one, it is
// misclassified: the panel would tell the operator a wired switch does nothing.
func TestCandidateKnobsHaveNoMethodReaders(t *testing.T) {
	for _, e := range AllKnobs() {
		if e.Status != KnobCandidate {
			continue
		}
		acc, sites := methodReadersOfKnob(t, e.Path)
		if len(sites) > 0 {
			t.Errorf("knob %q is candidate-unverified but is READ through %v at %v — reclassify it live with those consumers", e.Path, acc, sites)
		}
	}
}

// THE SIX ARE LIVE, AND THE REGISTRY NAMES THEIR METHOD READERS. The detector
// must find at least one production call site for each, and the registry's
// Consumers must include it — so the entry is a reader, not a claim (A24).
func TestWakeKnobsAreLiveThroughTheirAccessors(t *testing.T) {
	for _, leaf := range []string{"wake_min_interval_min", "wake_on_15m_zone", "wake_on_htf_ob", "wake_on_htf_zone", "wake_on_ifvg", "wake_on_seated_invalidation"} {
		e, ok := LookupKnob(leaf)
		if !ok {
			t.Fatalf("%s not in the registry", leaf)
		}
		acc, sites := methodReadersOfKnob(t, leaf)
		if len(sites) == 0 {
			t.Fatalf("%s: the detector found no method reader — the wiring this pin exists for is gone", leaf)
		}
		if e.Status != KnobLive {
			t.Errorf("%s: status %q, want live — read via %v at %v", leaf, e.Status, acc, sites)
		}
		named := false
		for _, c := range e.Consumers {
			for _, s := range sites {
				if c == s {
					named = true
				}
			}
		}
		if !named {
			t.Errorf("%s: registry Consumers %v name none of the detector's call sites %v", leaf, e.Consumers, sites)
		}
		if e.UILabel() == "no known reader — pending verification" {
			t.Errorf("%s renders %q", leaf, e.UILabel())
		}
	}
}
