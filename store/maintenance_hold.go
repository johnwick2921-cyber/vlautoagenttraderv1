package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ── W-ONE-BUTTON M2 — THE INSTALLATION-WIDE MAINTENANCE HOLD ────────────────
//
// One file, <data dir>/updater/hold.json, outside every versioned release, so
// it survives restarts and release swaps and has no automatic expiry. The
// updater worker (M4) writes it; cmd/maintenance-hold writes it for ops/tests.
// Nothing in the trading API can write or clear it.
//
//	absent                       → NOT held (today's behaviour, byte-identical)
//	present, well-formed         → what it says ("held": true|false)
//	present, unreadable/corrupt  → HELD (fail closed): a hold we cannot read
//	                               is a hold we must honour
//
// Every entry gate reads it through ReadMaintenanceHold; a stat-keyed cache
// (device, inode, mtime, size) makes the per-send read cheap without ever
// serving a verdict for a file that has changed.

// MaintenanceHold is the on-disk schema.
type MaintenanceHold struct {
	Held   bool   `json:"held"`
	JobID  string `json:"job_id"`
	Since  string `json:"since"` // RFC3339
	Reason string `json:"reason,omitempty"`
	Owner  string `json:"owner,omitempty"` // "updater" | "cli"
}

// MaintenanceHoldState is what a reader learns. Held is the only field a gate
// decides on; the rest is for the boot line, the status surface and logs.
type MaintenanceHoldState struct {
	Held    bool
	Present bool
	Corrupt bool   // present but unusable → Held is true
	Err     string // why it is corrupt/unreadable ("" otherwise)
	Path    string
	Hold    MaintenanceHold // zero unless Present && !Corrupt
}

// MaintenanceHoldPath is <dataDir>/updater/hold.json.
func MaintenanceHoldPath(dataDir string) string {
	return filepath.Join(dataDir, "updater", "hold.json")
}

type holdCacheEntry struct {
	dev, ino uint64
	mtime    int64
	size     int64
	state    MaintenanceHoldState
}

var (
	holdCacheMu sync.Mutex
	holdCache   = map[string]holdCacheEntry{}
)

func resetMaintenanceHoldCacheForTest() {
	holdCacheMu.Lock()
	holdCache = map[string]holdCacheEntry{}
	holdCacheMu.Unlock()
}

// ReadMaintenanceHold reads the hold for the installation whose data dir is
// dataDir. Never panics; never returns "not held" for a file it could not read.
func ReadMaintenanceHold(dataDir string) MaintenanceHoldState {
	p := MaintenanceHoldPath(dataDir)
	fi, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			holdCacheMu.Lock()
			delete(holdCache, p)
			holdCacheMu.Unlock()
			return MaintenanceHoldState{Path: p}
		}
		// stat failed for another reason (permissions on the dir, I/O): we
		// cannot prove it is absent, so it holds.
		return MaintenanceHoldState{Held: true, Present: true, Corrupt: true, Path: p,
			Err: "stat: " + err.Error()}
	}
	var dev, ino uint64
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		dev, ino = uint64(st.Dev), uint64(st.Ino)
	}
	key := holdCacheEntry{dev: dev, ino: ino, mtime: fi.ModTime().UnixNano(), size: fi.Size()}
	holdCacheMu.Lock()
	if c, ok := holdCache[p]; ok && c.dev == key.dev && c.ino == key.ino && c.mtime == key.mtime && c.size == key.size {
		holdCacheMu.Unlock()
		return c.state
	}
	holdCacheMu.Unlock()

	st := parseMaintenanceHold(p)
	key.state = st
	holdCacheMu.Lock()
	holdCache[p] = key
	holdCacheMu.Unlock()
	return st
}

func parseMaintenanceHold(p string) MaintenanceHoldState {
	corrupt := func(why string) MaintenanceHoldState {
		return MaintenanceHoldState{Held: true, Present: true, Corrupt: true, Path: p, Err: why}
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return corrupt("read: " + err.Error())
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return corrupt("empty file")
	}
	var h MaintenanceHold
	if err := json.Unmarshal(b, &h); err != nil {
		return corrupt("parse: " + err.Error())
	}
	if h.Held {
		if strings.TrimSpace(h.JobID) == "" {
			return corrupt("held without job_id")
		}
		if _, err := time.Parse(time.RFC3339, h.Since); err != nil {
			return corrupt("held with an unparseable since: " + h.Since)
		}
	}
	return MaintenanceHoldState{Held: h.Held, Present: true, Path: p, Hold: h}
}

// WriteMaintenanceHold writes the hold atomically: temp file in the same dir,
// fsync, rename over hold.json, fsync the dir. A reader sees the old file or
// the new file, never a partial one.
func WriteMaintenanceHold(dataDir string, h MaintenanceHold) error {
	if h.Held {
		if strings.TrimSpace(h.JobID) == "" {
			return errors.New("maintenance hold: held requires a job_id")
		}
		if _, err := time.Parse(time.RFC3339, h.Since); err != nil {
			return fmt.Errorf("maintenance hold: since must be RFC3339: %w", err)
		}
	}
	dir := filepath.Dir(MaintenanceHoldPath(dataDir))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".hold-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op after a successful rename
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, MaintenanceHoldPath(dataDir)); err != nil {
		return err
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}

// ClearMaintenanceHold removes the hold only when it is held by jobID. A
// mismatched or empty job id, or a corrupt file, is refused: the job that
// holds is the only one that releases (a corrupt file is cleared by an
// operator with ForceClearMaintenanceHold, never silently).
func ClearMaintenanceHold(dataDir, jobID string) error {
	if strings.TrimSpace(jobID) == "" {
		return errors.New("maintenance hold: clear requires the holding job_id")
	}
	st := ReadMaintenanceHold(dataDir)
	if !st.Present {
		return nil // already clear — idempotent
	}
	if st.Corrupt {
		return fmt.Errorf("maintenance hold: file is unusable (%s); refusing to clear by job id — use the operator force-clear", st.Err)
	}
	if st.Hold.JobID != jobID {
		return fmt.Errorf("maintenance hold: held by job %q, not %q", st.Hold.JobID, jobID)
	}
	return removeMaintenanceHold(dataDir)
}

// ForceClearMaintenanceHold removes the file regardless of its content. Only
// the local operator CLI calls it (cmd/maintenance-hold clear --force).
func ForceClearMaintenanceHold(dataDir string) error { return removeMaintenanceHold(dataDir) }

func removeMaintenanceHold(dataDir string) error {
	p := MaintenanceHoldPath(dataDir)
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	holdCacheMu.Lock()
	delete(holdCache, p)
	holdCacheMu.Unlock()
	if d, err := os.Open(filepath.Dir(p)); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}
