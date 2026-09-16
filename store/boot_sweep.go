package store

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// ── CLASS 33 (2026-09-02) — PRE-BOOT ARM DECIDABILITY ────────────────────────
// The armed_orders ledger persists across restarts; the ORDERS it describes
// live at NinjaTrader, not in this process. 2026-09-02 00:16 CT a cutover
// landed with S1 @29044 and S3 @29068.05 resting: the old process died, its
// broker orders did not, and they sat with NO listener for 15 minutes until
// the stale-window reconcile cancelled them at 00:31:48 — while the new binary
// re-armed its own S1/S3, so for minutes TWO S3 orders existed at the broker.
// A boot sweep can only act if "pre-boot" is DECIDABLE, hence this stamp.

var (
	bootIDOnce sync.Once
	bootIDVal  string
)

// ProcessBootID returns this process's boot identity: "<pid>-<unix-ms at first
// call>". Stable for the process lifetime, distinct across restarts (a reused
// PID cannot collide because the start stamp differs).
func ProcessBootID() string {
	bootIDOnce.Do(func() {
		bootIDVal = fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixMilli())
	})
	return bootIDVal
}

// ListPreBoot returns ONE trader's sweepable rows from a DIFFERENT process.
// Sweepable means non-terminal EXCEPT cancel_pending: those rows belong to
// confirmPendingCancels, which settles them only with a persisted snapshot id.
// An EMPTY boot_id (written before this column existed) counts as pre-boot.
func (s *ArmedOrderStore) ListPreBoot(traderID, bootID string) ([]ArmedOrderDB, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("store required")
	}
	var out []ArmedOrderDB
	err := s.db.Where("trader_id = ? AND (boot_id IS NULL OR boot_id <> ?)",
		traderID, bootID).Where(SweepableArmStateSQL()).Order("id").Find(&out).Error
	return out, err
}

// DB exposes the underlying handle for fixtures that need to seed or inspect
// rows the public API deliberately does not expose (e.g. a foreign boot stamp).
func (s *ArmedOrderStore) DB() *gorm.DB { return s.db }

// BootSweptKey — the RECORDED counter for pre-boot arms cancelled by the boot
// sweep (class 35 lesson: a log-only tally evaporates at the next boot).
const BootSweptKey = "arms_boot_swept_class33"

// IncBootSwept bumps the counter atomically by n and returns the new value.
func IncBootSwept(st *Store, n int) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	if n <= 0 {
		return BootSweptCount(st)
	}
	if err := st.gdb.Exec(
		`INSERT INTO system_config (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + ? AS TEXT)`,
		BootSweptKey, strconv.Itoa(n), n).Error; err != nil {
		return 0, err
	}
	return BootSweptCount(st)
}

// BootSweptCount reads the counter (0 when unset).
func BootSweptCount(st *Store) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	var v string
	if err := st.gdb.Raw(`SELECT value FROM system_config WHERE key = ?`, BootSweptKey).Scan(&v).Error; err != nil {
		return 0, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n, nil
}
