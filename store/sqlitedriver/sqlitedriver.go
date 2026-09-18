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

// Backend names the compiled-in backend (for boot lines / diagnostics).
func Backend() string {
	return backendName
}
