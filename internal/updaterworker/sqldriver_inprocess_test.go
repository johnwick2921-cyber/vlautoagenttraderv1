package updaterworker

import (
	"database/sql"
	"testing"
)

// PIN (D4, in-process half): THIS test binary links the whole worker set —
// nofx/store (the hold, modernc via store/sqlitedriver) and, through
// library_activation.go, nofx/internal/activation. It got here, so no second
// "sqlite" registration panicked at init; and the registry it built has
// "sqlite" exactly once and no duplicate name at all. (A duplicate never
// reaches this line — database/sql panics in init — which is why the naming
// half, the toolchain's dependency list, lives in the worker-free test-only
// package internal/updaterworker/sqldriverpin.)
func TestTheWorkerTestBinaryRegistersSQLiteOnce(t *testing.T) {
	seen := map[string]int{}
	for _, d := range sql.Drivers() {
		seen[d]++
	}
	if seen["sqlite"] != 1 {
		t.Fatalf("database/sql drivers = %v; want \"sqlite\" registered exactly once", sql.Drivers())
	}
	for d, n := range seen {
		if n > 1 {
			t.Fatalf("database/sql driver %q registered %d times: %v", d, n, sql.Drivers())
		}
	}
	// and it is the adapter's library that opens it: Backup's sql.Open("sqlite")
	// resolves to this one registration (TestAdapterDelegatesToActivation/Backup)
	var _ Library = activationLibrary{}
}
