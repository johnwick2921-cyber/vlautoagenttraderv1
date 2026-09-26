package updaterjob

// ── census: the job-file WRITERS belong to the worker ──────────────────────
//
// The worker's runner and socket are the only code that may create or
// transition a job (Write, New, (*Job).Enter). The app links this package
// (api/ reads jobs through Read) but must never WRITE one: an app-side edit
// could push a legal-edge transition (say requested → cancelled) into the
// worker's own job file, compile, and pass every test — nothing forbade it.
// This census pins the rule: no non-test file outside internal/updaterjob
// and internal/updaterworker names any of the three writers.
//
// The scanner asks the GO PARSER for calls and resolves import NAMES by
// PATH, so alias imports (`fw "nofx/internal/updaterjob"; fw.Write`) and
// var-declared or passed-around values (`var w = updaterjob.Write`) are
// seen, and string literals are not (the sqlitedriver census's lesson).

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestJobFileWritersBelongToTheWorker(t *testing.T) {
	offenders, scanned, err := jobFileWriterOffenders("")
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 {
		t.Fatalf("scanned %d files — the census is not covering the module", scanned)
	}
	for _, o := range offenders {
		t.Errorf("%s", o)
	}

	// the synthetic shapes the scanner must see (and not see)
	shapes := map[string]string{
		"a plain write": `package x
import "nofx/internal/updaterjob"
func a() error { return updaterjob.Write("/d", updaterjob.Job{}) }`,
		"an alias-imported write": `package x
import fw "nofx/internal/updaterjob"
func a() error { return fw.Write("/d", fw.Job{}) }`,
		"a var-declared writer value": `package x
import "nofx/internal/updaterjob"
var w = updaterjob.Write`,
		"a passed writer": `package x
import "nofx/internal/updaterjob"
func a(f func(string, updaterjob.Job) error) {}
func b() { a(updaterjob.Write) }`,
		"a New call": `package x
import "nofx/internal/updaterjob"
func a() { updaterjob.New("job", "rel", func() (t time.Time) { return t }) }`,
		"an Enter transition": `package x
import "nofx/internal/updaterjob"
func a(j updaterjob.Job) { j.Enter(updaterjob.StateRequested, timeNow()) }`,
		"a READ is not a write": `package x
import "nofx/internal/updaterjob"
func a() { updaterjob.Read("/d", "job") }`,
	}
	for name, src := range shapes {
		dir := t.TempDir()
		p := filepath.Join(dir, "probe.go")
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		hits := writerOffendersInFile("probe.go", "probe", src)
		if strings.HasPrefix(name, "a READ") {
			if len(hits) != 0 {
				t.Errorf("%s: scanner flagged a reader: %q", name, hits)
			}
			continue
		}
		if len(hits) != 1 {
			t.Errorf("%s: scanner found %q, want exactly one site", name, hits)
		}
	}

	// the MUTATION at a real module site: a non-test file of an app package
	// calling a writer is an offender the full census names.
	probe := filepath.Join("..", "..", "api", "zz_writers_census_probe.go")
	if err := os.WriteFile(probe, []byte(`package api

import "nofx/internal/updaterjob"

func zzWritersCensusProbe() error {
	return updaterjob.Write("/tmp/never", updaterjob.Job{})
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(probe)
	offenders, _, err = jobFileWriterOffenders("")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range offenders {
		if strings.Contains(o, "zz_writers_census_probe.go") {
			found = true
		}
	}
	if !found {
		t.Fatalf("an app-package writer call was not flagged: %v", offenders)
	}
}

// jobFileWriterOffenders scans the whole module (root "" ⇒ two levels up).
func jobFileWriterOffenders(root string) ([]string, int, error) {
	if root == "" {
		root = filepath.Join("..", "..")
	}
	var offenders []string
	scanned := 0
	err := filepath.Walk(root, func(p string, info os.FileInfo, werr error) error {
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
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		dir := filepath.Dir(filepath.ToSlash(rel))
		switch dir {
		case "internal/updaterjob", "internal/updaterworker":
			return nil // the declarations and the worker itself
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		scanned++
		for _, hit := range writerOffendersInFile(p, rel, string(src)) {
			offenders = append(offenders, fmt.Sprintf("%s: %s", rel, hit))
		}
		return nil
	})
	return offenders, scanned, err
}

// writerOffendersInFile parses src and returns every NAMING of the three
// job-file writers (call, value, argument) with the import name resolved by
// PATH. `Enter` is matched on any selector: updaterjob.Job is the only Enter
// in this module (verified 2026-09-25).
func writerOffendersInFile(name, rel, src string) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	// import path → the name it is imported as
	imports := map[string]string{}
	for _, im := range f.Imports {
		p, err := strconv.Unquote(im.Path.Value)
		if err != nil {
			continue
		}
		n := filepath.Base(p)
		if im.Name != nil {
			n = im.Name.Name
		}
		imports[p] = n
	}
	jname, hasUpdaterjob := imports["nofx/internal/updaterjob"]
	var hits []string
	pos := func(n ast.Node) string {
		return fmt.Sprintf("%s: line %d", rel, fset.Position(n.Pos()).Line)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		se, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		sel := se.Sel.Name
		if sel == "Enter" {
			// a method transition on a Job value: any .Enter in a file that
			// imports the package is one (the module has no other Enter).
			if hasUpdaterjob {
				hits = append(hits, pos(se)+" .Enter")
			}
			return true
		}
		if sel != "Write" && sel != "New" {
			return true
		}
		id, ok := se.X.(*ast.Ident)
		if !ok || !hasUpdaterjob || id.Name != jname {
			return true
		}
		hits = append(hits, pos(se)+" "+sel)
		return true
	})
	return hits
}
