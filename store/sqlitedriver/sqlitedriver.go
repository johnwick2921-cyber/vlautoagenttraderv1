// Package sqlitedriver is the ONE place in this repo that registers the
// "sqlite" database/sql driver and exposes the GORM SQLite dialector.
//
// Every other package that needs SQLite imports THIS package — never
// modernc.org/sqlite, github.com/glebarez/go-sqlite, gorm.io/driver/sqlite or
// github.com/glebarez/sqlite directly. database/sql panics
// ("sql: Register called twice for driver sqlite") when two drivers register
// the same name in one binary; a single registration site makes that
// impossible by construction (W-CGOFREE-SQLITE-UPSTREAM, 2026-09-17: the
// partner machine "Binnie" hit exactly that panic on its first boot after
// researchsnapshot/archive.go added a second blank import).
//
// The backend is selected by build tag:
//
//	default        modernc.org/sqlite for database/sql (pure Go) and
//	               gorm.io/driver/sqlite (mattn/go-sqlite3, needs cgo) for GORM
//	               — the owner's existing build, unchanged.
//	-tags cgofree  github.com/glebarez/go-sqlite for database/sql and
//	               github.com/glebarez/sqlite for GORM. Both are pure Go
//	               (glebarez/go-sqlite is a thin fork of modernc.org/sqlite's
//	               driver layer over the same modernc.org/sqlite/lib C
//	               translation), so machines without a C compiler build with
//	               `go build -tags cgofree`.
package sqlitedriver

import (
	"database/sql"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

// DriverName is the database/sql driver name this package registers.
const DriverName = "sqlite"

// Open opens a SQLite database through database/sql using the registered
// driver. The DSN is passed through unchanged.
func Open(dsn string) (*sql.DB, error) {
	return sql.Open(DriverName, dsn)
}

// GormDialector returns the GORM dialector for the selected backend.
func GormDialector(dsn string) gorm.Dialector {
	return gormDialector(dsn)
}

// DialectorConn returns the GORM dialector for the compiled-in backend bound to
// an EXISTING database/sql connection (or *sql.Tx) instead of opening a DSN.
// Callers use it for transactions whose BEGIN must be issued manually — e.g.
// BEGIN IMMEDIATE, which GORM's own Transaction cannot express.
func DialectorConn(conn gorm.ConnPool) gorm.Dialector {
	return dialectorConn(conn)
}

// IsBusy reports whether err is a SQLite lock-contention error
// (SQLITE_BUSY / SQLITE_BUSY_SNAPSHOT: "database is locked"), across both
// backends. The mattn text is "database is locked"; modernc (and glebarez over
// it) appends the code, e.g. "database is locked (5) (SQLITE_BUSY)".
func IsBusy(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "database is locked") || strings.Contains(s, "SQLITE_BUSY")
}

// Backend names the compiled-in backend (for boot lines / diagnostics).
func Backend() string {
	return backendName
}

// busyTimeoutDSN makes the per-connection busy_timeout guarantee EXPLICIT in
// the DSN. Both compiled-in backends happen to carry PRAGMA busy_timeout=5000
// on every connection open today (mattn/go-sqlite3 applies it by default,
// sqlite3.go:1098,1491; glebarez/go-sqlite bakes it in), but the guarantee is
// a driver default, not a contract — a driver upgrade could silently drop it
// and the other 3 pooled connections would then fail writes with SQLITE_BUSY
// immediately under contention (P1-B, audit 2026-09-26). The DSN parameter is
// backend-specific: _busy_timeout for mattn, _pragma=busy_timeout(5000) for
// modernc/glebarez.
func busyTimeoutDSN(dsn, param string) string {
	if strings.Contains(dsn, "_busy_timeout") || strings.Contains(dsn, "_pragma") || strings.Contains(dsn, ":memory:") {
		// already explicit, or an in-memory database whose driver-level default
		// covers every connection; never rewrite special DSN forms (":memory:"
		// must not become a file on disk).
		return dsn
	}
	if !strings.HasPrefix(dsn, "file:") {
		if abs, err := filepath.Abs(dsn); err == nil {
			dsn = "file:" + abs
		}
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + param
}
