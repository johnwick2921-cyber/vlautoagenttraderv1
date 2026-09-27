package updaterworker

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"nofx/internal/censuswalk"
)

// sqliteDriverPackages register (or wrap) the "sqlite" database/sql driver.
var sqliteDriverPackages = map[string]bool{
	"modernc.org/sqlite": true, "github.com/glebarez/go-sqlite": true, "github.com/mattn/go-sqlite3": true,
	"gorm.io/driver/sqlite": true, "github.com/glebarez/sqlite": true,
}

// GUARD: the worker binary links ONE sqlite driver registration — the one
// store/sqlitedriver owns (its package doc: every other package imports IT;
// "sql: Register called twice for driver sqlite" otherwise). The worker links
// nofx/store (hold.go) and, once library_activation.go lands, nofx/internal/
// activation — whose steps.go:14 at #201 (dev c62a35dc) blank-imports
// github.com/glebarez/go-sqlite itself. In the default build that is a
// SECOND registration and the worker panics at init [A: reproduced at
// merge-tree(this branch, c62a35dc) with the drafted adapter; the one-line fix
// — import _ "nofx/store/sqlitedriver" there — turns it green]. This guard
// goes RED at the fold the moment the adapter links such a package.
func TestWorkerBinaryLinksOneSqliteDriverRegistrationGuard(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	module, err := censuswalk.ModulePath(root)
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := censuswalk.ListPackages(root, true, "./cmd/nofx-updater")
	if err != nil {
		t.Fatal(err)
	}
	owner := module + "/store/sqlitedriver"
	var offenders []string
	sawOwner := false
	for _, p := range pkgs {
		if p.ImportPath == owner {
			sawOwner = true
		}
		if p.Module != module || p.ImportPath == owner {
			continue
		}
		for _, f := range p.Files {
			af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range af.Imports {
				if path, _ := strconv.Unquote(imp.Path.Value); sqliteDriverPackages[path] {
					offenders = append(offenders, p.ImportPath+" ("+filepath.Base(f)+") imports "+path)
				}
			}
		}
	}
	if !sawOwner {
		t.Fatalf("the worker binary does not link %s — the guard is not seeing the binary (%d packages)", owner, len(pkgs))
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("the worker binary links a second sqlite driver registration (init panics \"sql: Register called twice for driver sqlite\"):\n%s", strings.Join(offenders, "\n"))
	}
}
