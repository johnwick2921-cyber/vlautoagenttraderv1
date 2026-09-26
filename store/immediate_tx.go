// W117 a3 — ApplyNT8Exit must take SQLite's write lock UP FRONT.
//
// GORM's db.Transaction opens a DEFERRED transaction: the first statement is a
// READ (the receipt lookup), and the lock is only sought at the first WRITE.
// Under WAL, a deferred read→write upgrade while another connection holds the
// write lock returns SQLITE_BUSY / SQLITE_BUSY_SNAPSHOT IMMEDIATELY — the busy
// handler is not consulted for an upgrade that risks deadlock, so busy_timeout
// cannot help, the transaction rolls back and the close is lost (live 08:55 CT
// 2026-09-25, row 618). BEGIN IMMEDIATE seeks the RESERVED lock before any
// statement, where the busy handler DOES apply: the transaction waits
// (busy_timeout) and proceeds the instant the holder commits.
//
// The pool-side PRAGMA wrinkle is the same class: store/gorm.go sets
// busy_timeout with ONE db.Exec on a pool of 4 — that pragma reaches exactly
// one pooled connection. Here the pragma is set on the DEDICATED connection
// this transaction borrows, so the busy wait is guaranteed for the one write
// path whose loss is a lost exit.

package store

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"nofx/store/sqlitedriver"
)

// immediateTxBusyTimeoutMs is the busy_timeout applied on the dedicated
// connection before BEGIN IMMEDIATE. Defaults to 5000 (the live pool's value);
// tests shrink it so the bounded-retry pins stay fast.
var immediateTxBusyTimeoutMs atomic.Int64

func init() {
	immediateTxBusyTimeoutMs.Store(5000)
}

// SetImmediateTxBusyTimeoutForTest shrinks the BEGIN IMMEDIATE busy wait so a
// busy-retry pin can exhaust its attempts in milliseconds instead of seconds.
func SetImmediateTxBusyTimeoutForTest(ms int64) { immediateTxBusyTimeoutMs.Store(ms) }

// ResetImmediateTxBusyTimeoutForTest restores the production value.
func ResetImmediateTxBusyTimeoutForTest() { immediateTxBusyTimeoutMs.Store(5000) }

// withImmediateWriteTx runs fn inside an explicit BEGIN IMMEDIATE transaction
// on ONE dedicated pooled connection, with the default busy wait. fn receives
// a GORM session bound to that connection with default auto-transactions
// disabled (the surrounding BEGIN/COMMIT is the transaction).
func withImmediateWriteTx(db *gorm.DB, fn func(tx *gorm.DB) error) (err error) {
	return withImmediateWriteTxBusy(db, immediateTxBusyTimeoutMs.Load(), fn)
}

// withImmediateWriteTxBusy is the same, with an explicit busy_timeout budget.
// The write lock is sought at BEGIN IMMEDIATE, where SQLite's busy handler
// applies — the transaction WAITS up to busyMs for the holder instead of
// returning SQLITE_BUSY at a deferred read→write upgrade.
func withImmediateWriteTxBusy(db *gorm.DB, busyMs int64, fn func(tx *gorm.DB) error) (err error) {
	sqlDB, serr := db.DB()
	if serr != nil {
		return serr
	}
	ctx := context.Background()
	conn, cerr := sqlDB.Conn(ctx)
	if cerr != nil {
		return cerr
	}
	defer conn.Close()

	if busyMs > 0 {
		if _, perr := conn.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", busyMs)); perr != nil {
			return perr
		}
	}
	if _, berr := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); berr != nil {
		return berr
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(ctx, "ROLLBACK")
		}
	}()

	gdb, gerr := gorm.Open(sqlitedriver.DialectorConn(conn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if gerr != nil {
		return gerr
	}
	if ferr := fn(gdb.Session(&gorm.Session{SkipDefaultTransaction: true})); ferr != nil {
		return ferr
	}
	if _, cerr2 := conn.ExecContext(ctx, "COMMIT"); cerr2 != nil {
		return cerr2
	}
	committed = true
	return nil
}

// immediateOrPlainTx chooses the immediate write-lock transaction on SQLite and
// falls back to GORM's default transaction on other backends (Postgres).
func (s *PositionStore) immediateOrPlainTx(fn func(tx *gorm.DB) error) error {
	if s.db.Dialector.Name() != sqlitedriver.DriverName {
		return s.db.Transaction(fn)
	}
	return withImmediateWriteTx(s.db, fn)
}
