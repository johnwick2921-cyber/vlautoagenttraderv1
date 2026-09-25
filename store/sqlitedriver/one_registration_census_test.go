package sqlitedriver

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This package's doc comment states the rule: every package that needs SQLite
// imports THIS package, never a driver directly, because database/sql panics
// ("sql: Register called twice for driver sqlite") when two register the same
// name in one binary. The partner machine hit exactly that panic on its first
// boot after a second blank import appeared.
//
// TestSingleRegistration could not enforce it. sql.Drivers() is per-BINARY, and
// that test's binary links only this package — so it cannot see a violation in
// any other package. The rule was documented and unenforced, which is how
// internal/activation/steps.go came to blank-import github.com/glebarez/go-sqlite
// directly: nothing failed.
//
// This is the census the rule needed. It ASKS THE GO PARSER for each file's
// imports rather than grepping, which matters: cmd/nofx-activate/main_test.go
// contains a driver import inside a STRING LITERAL — the source of a throwaway
// program handed to `go run -` — and a grep-based census would flag that as a
// violation when it is a separate process that registers once and cannot
// conflict with anything. A census that cannot tell an import from a string
// produces false accusations, and a false accusation is how a real rule gets
// ignored.
var forbiddenDirect = []string{
	"modernc.org/sqlite",
	"github.com/glebarez/go-sqlite",
	"github.com/glebarez/sqlite",
	"gorm.io/driver/sqlite",
	"github.com/mattn/go-sqlite3",
}

func TestNoPackageImportsASQLiteDriverDirectly(t *testing.T) {
	root := repoRootFromHere(t)
	var offenders []string
	fset := token.NewFileSet()

	err := filepath.Walk(root, func(p string, info os.FileInfo, werr error) error {
		if werr != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", "vendor", "web", ".understand-anything":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		// This package IS the registration site. It is the only exemption, and
		// it is exempt by identity rather than by name-matching a path prefix.
		if filepath.Dir(rel) == filepath.Join("store", "sqlitedriver") {
			return nil
		}
		f, perr := parser.ParseFile(fset, p, nil, parser.ImportsOnly)
		if perr != nil {
			return nil // unparseable files are not this test's business
		}
		// THE DOMAIN IS LIBRARIES, and the reason is what makes this a census
		// rather than a blanket ban. The panic needs TWO registrants in ONE
		// binary. A `package main` cannot be imported, so its driver choice
		// affects only itself and can never collide with anything — the
		// standalone research harnesses under docs/ are each their own binary
		// with one registrant. A LIBRARY is different: anything may link it,
		// so a driver import there is a landmine for every future importer,
		// which is exactly how internal/activation became one for the M4
		// worker.
		if f.Name != nil && f.Name.Name == "main" {
			return nil
		}
		for _, im := range f.Imports {
			path, uerr := strconv.Unquote(im.Path.Value)
			if uerr != nil {
				continue
			}
			for _, bad := range forbiddenDirect {
				if path == bad {
					offenders = append(offenders, rel+" imports "+path)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("%d package(s) import a SQLite driver directly instead of %s:\n  %s\n\n"+
			"database/sql panics when two drivers register the name %q in one binary. "+
			"Import nofx/store/sqlitedriver instead — it is the ONE registration site.",
			len(offenders), "nofx/store/sqlitedriver", strings.Join(offenders, "\n  "), DriverName)
	}
}

// repoRootFromHere walks up to the module root so the census covers the WHOLE
// repo. An earlier census of mine scanned three hand-listed directories and
// declared itself complete while omitting a producer; a census is either
// exhaustive over its domain or it is a list.
func repoRootFromHere(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for dir := wd; ; {
		if _, serr := os.Stat(filepath.Join(dir, "go.mod")); serr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", wd)
		}
		dir = parent
	}
}
