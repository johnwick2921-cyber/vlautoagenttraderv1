//go:build !cgofree

package sqlitedriver

import (
	gormsqlite "gorm.io/driver/sqlite" // GORM dialector — mattn/go-sqlite3 (cgo)
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // registers database/sql driver "sqlite" (pure Go)
)

const backendName = "modernc.org/sqlite + gorm.io/driver/sqlite (cgo)"

func gormDialector(dsn string) gorm.Dialector {
	return gormsqlite.Open(dsn)
}

func dialectorConn(conn gorm.ConnPool) gorm.Dialector {
	return gormsqlite.New(gormsqlite.Config{Conn: conn})
}
