package updateauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Single-use job ids. Every authorized install consumes its job id BEFORE
// anything else happens; a second use is ErrReplay (409), across restarts
// AND across clock step-backs.
//
// The store is {"v":2,"pruned_through":N,"ids":[{job_id,expires_at,consumed_at}…]}.
// pruned_through is the largest expires_at ever pruned from ids (0 = nothing
// pruned yet; every valid expires_at is > 0). A grant whose expires_at is at
// or below it is refused as a replay whatever the clock says: the store can
// no longer prove its id was never consumed (M3-RT-F1 — a prune followed by a
// clock step-back of more than SeenRetention used to put a consumed grant
// back inside its window with its id forgotten).
const (
	// MaxSeenEntries is the hard cap: past it Consume refuses (fail closed)
	// rather than evicting a live entry.
	MaxSeenEntries = 10000
	// SeenRetention: an entry is pruned only once its expires_at passed more
	// than this long ago. Pruning is safe at any clock because every pruned
	// expiry is covered by pruned_through.
	SeenRetention = 10 * time.Minute

	seenVersion      = 2
	maxSeenFileBytes = 4 << 20
)

var (
	// ErrReplay: the job id was already consumed.
	ErrReplay = errors.New("updateauth: job id already used")
	// ErrPrunedReplay (errors.Is ErrReplay): expires_at is at or below the
	// store's pruned_through watermark, so the id may have been consumed and
	// pruned. Refused as a replay (fail closed); a genuinely fresh grant only
	// lands here after the clock was stepped back past an earlier prune.
	ErrPrunedReplay = fmt.Errorf("%w: expires_at at or below the pruned-through watermark (consumed and pruned, or minted under a clock stepped back past a prune)", ErrReplay)
	// ErrSeenCorrupt: the seen file exists but cannot be read/parsed. It is
	// NEVER reset — an empty store would re-open every consumed id.
	ErrSeenCorrupt = errors.New("updateauth: seen-job store unreadable")
	// ErrSeenFull: MaxSeenEntries live entries.
	ErrSeenFull = errors.New("updateauth: seen-job store full")
	// ErrBadClock: the clock reads at or before the unix epoch. Consume
	// refuses before it locks or writes, because consumed_at <= 0 is a record
	// the store's own reader refuses — writing it would wedge the store
	// (corrupt is never reset) for every later install (M3-RT-F3).
	ErrBadClock = errors.New("updateauth: clock at or before the unix epoch — refusing to record a consumption")
)

type seenEntry struct {
	JobID      string `json:"job_id"`
	ExpiresAt  int64  `json:"expires_at"`
	ConsumedAt int64  `json:"consumed_at"`
}

// seenStore is the decoded store.
type seenStore struct {
	PrunedThrough int64
	IDs           []seenEntry
}

// seenFile is the on-disk shape (field order is the write order).
type seenFile struct {
	V             int         `json:"v"`
	PrunedThrough int64       `json:"pruned_through"`
	IDs           []seenEntry `json:"ids"`
}

// readSeen: absent ⇒ empty store; anything else unreadable ⇒ ErrSeenCorrupt.
func readSeen(dataDir string) (seenStore, error) {
	b, err := readPrivateFile(SeenPath(dataDir), maxSeenFileBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return seenStore{}, nil
		}
		return seenStore{}, errors.Join(ErrSeenCorrupt, err)
	}
	return parseSeen(b)
}

// parseSeen is THE store validator: the reader applies it to what it reads
// and the writer (encodeSeen) to what it is about to write.
func parseSeen(b []byte) (seenStore, error) {
	if len(b) > maxSeenFileBytes {
		return seenStore{}, ErrSeenCorrupt
	}
	m, err := decodeStrictObject(bytes.NewReader(b), "v", "pruned_through", "ids")
	if err != nil {
		return seenStore{}, errors.Join(ErrSeenCorrupt, err)
	}
	if string(bytes.TrimSpace(m["v"])) != strconv.Itoa(seenVersion) {
		return seenStore{}, ErrSeenCorrupt
	}
	var s seenStore
	if s.PrunedThrough, err = rawNonNegInt(m["pruned_through"]); err != nil {
		return seenStore{}, errors.Join(ErrSeenCorrupt, err)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(m["ids"], &raw); err != nil || raw == nil {
		return seenStore{}, ErrSeenCorrupt
	}
	s.IDs = make([]seenEntry, 0, len(raw))
	for _, r := range raw {
		em, err := decodeStrictObject(bytes.NewReader(r), "job_id", "expires_at", "consumed_at")
		if err != nil {
			return seenStore{}, errors.Join(ErrSeenCorrupt, err)
		}
		var e seenEntry
		if e.JobID, err = rawString(em["job_id"]); err != nil || !ValidJobID(e.JobID) {
			return seenStore{}, ErrSeenCorrupt
		}
		if e.ExpiresAt, err = rawUnixSeconds(em["expires_at"]); err != nil {
			return seenStore{}, ErrSeenCorrupt
		}
		if e.ConsumedAt, err = rawUnixSeconds(em["consumed_at"]); err != nil {
			return seenStore{}, ErrSeenCorrupt
		}
		s.IDs = append(s.IDs, e)
	}
	return s, nil
}

// encodeSeen marshals s and then runs the bytes through parseSeen: the writer
// never emits a store its own reader refuses (under never-reset such a store
// is a permanent wedge — M3-RT-F3's general rule).
func encodeSeen(s seenStore) ([]byte, error) {
	ids := s.IDs
	if ids == nil {
		ids = []seenEntry{}
	}
	b, err := json.Marshal(seenFile{V: seenVersion, PrunedThrough: s.PrunedThrough, IDs: ids})
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	if _, err := parseSeen(b); err != nil {
		return nil, fmt.Errorf("updateauth: refusing to write a seen-job store its reader would refuse: %w", err)
	}
	return b, nil
}

// Consume records jobID as used, atomically under .seen.lock. A job id
// already present ⇒ ErrReplay; expiresAt at or below pruned_through ⇒
// ErrPrunedReplay (errors.Is ErrReplay); a corrupt store ⇒ ErrSeenCorrupt
// and the file is left byte-identical; MaxSeenEntries live entries ⇒
// ErrSeenFull; a clock at or before the epoch ⇒ ErrBadClock before any lock
// or write. Only entries whose expires_at passed more than SeenRetention ago
// are pruned, and the largest pruned expires_at is kept as pruned_through.
func Consume(dataDir, jobID string, expiresAt int64, now time.Time) error {
	if err := checkDataDir(dataDir); err != nil {
		return err
	}
	if !ValidJobID(jobID) || expiresAt <= 0 {
		return malformed("job_id/expires_at")
	}
	if now.Unix() <= 0 {
		return ErrBadClock
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
	st, err := readSeen(dataDir)
	if err != nil {
		return err
	}
	for _, e := range st.IDs {
		if e.JobID == jobID {
			return ErrReplay
		}
	}
	if expiresAt <= st.PrunedThrough {
		return ErrPrunedReplay
	}
	cutoff := now.Add(-SeenRetention).Unix()
	watermark := st.PrunedThrough
	kept := make([]seenEntry, 0, len(st.IDs)+1)
	for _, e := range st.IDs {
		if e.ExpiresAt >= cutoff {
			kept = append(kept, e)
		} else if e.ExpiresAt > watermark {
			watermark = e.ExpiresAt
		}
	}
	if len(kept) >= MaxSeenEntries {
		return ErrSeenFull
	}
	kept = append(kept, seenEntry{JobID: jobID, ExpiresAt: expiresAt, ConsumedAt: now.Unix()})
	b, err := encodeSeen(seenStore{PrunedThrough: watermark, IDs: kept})
	if err != nil {
		return err
	}
	return writeAtomic(dir, SeenPath(dataDir), b)
}
