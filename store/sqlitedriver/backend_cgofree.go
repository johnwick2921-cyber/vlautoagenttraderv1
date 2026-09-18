//go:build cgofree

package sqlitedriver

import (
	_ "github.com/glebarez/go-sqlite"       // registers database/sql driver "sqlite" (pure Go)
	glebsqlite "github.com/glebarez/sqlite" // GORM dialector over glebarez/go-sqlite (pure Go)
	"gorm.io/gorm"
)

const backendName = "github.com/glebarez/go-sqlite + github.com/glebarez/sqlite (cgofree)"

func gormDialector(dsn string) gorm.Dialector {
	return glebsqlite.Open(dsn)
}
