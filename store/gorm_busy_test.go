package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// TestEveryPooledConnCarriesBusyTimeout is the P1-B pin (audit 2026-09-26):
// PRAGMA busy_timeout executed once on the pool reaches exactly ONE of the
// four connections; every other pooled connection opens with busy_timeout=0
// and fails a write with SQLITE_BUSY immediately under contention. The fix
// moves the pragma into the DSN so it applies at EVERY connection open.
//
// The test exhausts the pool deliberately: it checks out MaxOpenConns
// connections without returning them, forcing database/sql to OPEN new
// connections (until the pool cap), then reads PRAGMA busy_timeout on each.
func TestEveryPooledConnCarriesBusyTimeout(t *testing.T) {
	db, err := InitGorm(filepath.Join(t.TempDir(), "busy.db"))
	if err != nil {
		t.Fatalf("InitGorm: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	const want = 5000
	conns := make([]*sql.Conn, 0, 4)
	for i := 0; i < 4; i++ {
		c, err := sqlDB.Conn(context.Background())
		if err != nil {
			t.Fatalf("checkout conn %d: %v", i, err)
		}
		conns = append(conns, c)
	}
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()

	for i, c := range conns {
		var bt int
		if err := c.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&bt); err != nil {
			t.Fatalf("conn %d PRAGMA busy_timeout: %v", i, err)
		}
		if bt != want {
			t.Fatalf("conn %d: busy_timeout=%d, want %d — a PRAGMA run once on the pool reaches only one connection; the DSN must carry it per connection", i, bt, want)
		}
	}
}
