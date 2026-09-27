package store

import (
	"strings"
	"testing"

	"nofx/internal/censuswalk"
)

// ── M3 integration (ha2 verify note A, CTO 1790242842706) — the users-table
// writer census walks with censuswalk, root-only skips ──
//
// FOLD-M3-A's census walked the tree itself and SkipDir'd any directory whose
// name starts with "." or "_", or is testdata, at ANY depth — the M5 class the
// other censuses had already closed. The toolchain compiles and links a
// package in api/.hidden, _x or x/testdata/y like any other, so a raw
// `UPDATE users` placed there was invisible while it moved the credential
// epoch every session is judged by (auth/retire.go). The verifier's probe
// (api/.hidden/w.go, linked by `go build ./api/` and named by `go list -deps`)
// passed the census. This pin plants the same writer in every
// censuswalk.NestedProbeDirs() directory of a synthetic module (never the
// real tree) and drives the production census function.
func TestUsersWriterCensusSeesNestedSkipNamedDirs(t *testing.T) {
	base := func() string {
		root := t.TempDir()
		censusWrite(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
		censusWrite(t, root, "store/user.go", "package store\n\ntype User struct{ ID string }\n\ntype UserStore struct{}\n")
		return root
	}
	const writer = "func fix(db dbish) { db.Exec(`UPDATE users SET updated_at = ?`, 1) }\n"
	// positive control: the same writer in api/ is caught
	root := base()
	censusWrite(t, root, "api/w.go", "package api\n\n"+writer)
	if sites, _, err := usersTableWriters(root); err != nil || !censusHas(sites, "api/w.go · fix · raw SQL UPDATE users") {
		t.Fatalf("positive control: api/w.go's raw UPDATE users must be reported (sites=%v err=%v)", sites, err)
	}
	for _, dir := range censuswalk.NestedProbeDirs() {
		t.Run(dir, func(t *testing.T) {
			root := base()
			rel := dir + "/w.go"
			censusWrite(t, root, rel, "package "+censuswalk.PackageName(dir)+"\n\n"+writer)
			sites, _, err := usersTableWriters(root)
			if err != nil {
				t.Fatal(err)
			}
			if want := rel + " · fix · raw SQL UPDATE users"; !censusHas(sites, want) {
				t.Fatalf("%s writes the users table (a compiled, importable package) and the users-writer census reports %v — want one == %q", rel, sites, want)
			}
		})
	}
	// The root-level skips stay skipped (web/ is the frontend, not Go the app links).
	root = base()
	censusWrite(t, root, "web/w.go", "package web\n\n"+writer)
	if sites, _, _ := usersTableWriters(root); strings.Join(sites, "") != "" {
		t.Fatalf("a root-level skip dir (web/) must stay outside the walk, as censuswalk rules: %v", sites)
	}
}
