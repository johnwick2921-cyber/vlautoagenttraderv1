package updateauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"time"
)

// Single-use job ids. Every authorized install consumes its job id BEFORE
// anything else happens; a second use is ErrReplay (409), across restarts.
const (
	// MaxSeenEntries is the hard cap: past it Consume refuses (fail closed)
	// rather than evicting a live entry.
	MaxSeenEntries = 10000
	// SeenRetention: an entry is pruned only once its expires_at passed more
	// than this long ago (its MAC can no longer pass the window, and a clock
	// stepped back by less than this still sees the replay).
	SeenRetention = 10 * time.Minute

	maxSeenFileBytes = 4 << 20
)

var (
	// ErrReplay: the job id was already consumed.
	ErrReplay = errors.New("updateauth: job id already used")
	// ErrSeenCorrupt: the seen file exists but cannot be read/parsed. It is
	// NEVER reset — an empty store would re-open every consumed id.
	ErrSeenCorrupt = errors.New("updateauth: seen-job store unreadable")
	// ErrSeenFull: MaxSeenEntries live entries.
	ErrSeenFull = errors.New("updateauth: seen-job store full")
)

type seenEntry struct {
	JobID      string `json:"job_id"`
	ExpiresAt  int64  `json:"expires_at"`
	ConsumedAt int64  `json:"consumed_at"`
}

type seenFile struct {
	V   int         `json:"v"`
	IDs []seenEntry `json:"ids"`
}

// readSeen: absent ⇒ empty; anything else unreadable ⇒ ErrSeenCorrupt.
func readSeen(dataDir string) ([]seenEntry, error) {
	b, err := readPrivateFile(SeenPath(dataDir), maxSeenFileBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, errors.Join(ErrSeenCorrupt, err)
	}
	m, err := decodeStrictObject(bytes.NewReader(b), "v", "ids")
	if err != nil {
		return nil, errors.Join(ErrSeenCorrupt, err)
	}
	if string(bytes.TrimSpace(m["v"])) != "1" {
		return nil, ErrSeenCorrupt
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(m["ids"], &raw); err != nil || raw == nil {
		return nil, ErrSeenCorrupt
	}
	out := make([]seenEntry, 0, len(raw))
	for _, r := range raw {
		em, err := decodeStrictObject(bytes.NewReader(r), "job_id", "expires_at", "consumed_at")
		if err != nil {
			return nil, errors.Join(ErrSeenCorrupt, err)
		}
		var e seenEntry
		if e.JobID, err = rawString(em["job_id"]); err != nil || !ValidJobID(e.JobID) {
			return nil, ErrSeenCorrupt
		}
		if e.ExpiresAt, err = rawUnixSeconds(em["expires_at"]); err != nil {
			return nil, ErrSeenCorrupt
		}
		if e.ConsumedAt, err = rawUnixSeconds(em["consumed_at"]); err != nil {
			return nil, ErrSeenCorrupt
		}
		out = append(out, e)
	}
	return out, nil
}

// Consume records jobID as used, atomically under .seen.lock. A job id
// already present ⇒ ErrReplay; a corrupt store ⇒ ErrSeenCorrupt and the
// file is left byte-identical; MaxSeenEntries live entries ⇒ ErrSeenFull.
// Only entries whose expires_at passed more than SeenRetention ago are pruned.
func Consume(dataDir, jobID string, expiresAt int64, now time.Time) error {
	if err := checkDataDir(dataDir); err != nil {
		return err
	}
	if !ValidJobID(jobID) || expiresAt <= 0 {
		return malformed("job_id/expires_at")
	}
	dir, err := ensurePrivateDir(dataDir)
	if err != nil {
		return err
	}
	unlock, err := lockFile(seenLockPath(dataDir))
	if err != nil {
		return err
	}
	defer unlock()
	entries, err := readSeen(dataDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.JobID == jobID {
			return ErrReplay
		}
	}
	cutoff := now.Add(-SeenRetention).Unix()
	kept := make([]seenEntry, 0, len(entries)+1)
	for _, e := range entries {
		if e.ExpiresAt >= cutoff {
			kept = append(kept, e)
		}
	}
	if len(kept) >= MaxSeenEntries {
		return ErrSeenFull
	}
	kept = append(kept, seenEntry{JobID: jobID, ExpiresAt: expiresAt, ConsumedAt: now.Unix()})
	b, err := json.Marshal(seenFile{V: 1, IDs: kept})
	if err != nil {
		return err
	}
	return writeAtomic(dir, SeenPath(dataDir), append(b, '\n'))
}
