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
// The store is
// {"v":3,"pruned_through":N,"clock_floor":F,"ids":[{job_id,expires_at,consumed_at}…]}.
// pruned_through is the largest expires_at ever pruned from ids (0 = nothing
// pruned yet; every valid expires_at is > 0). A grant whose expires_at is at
// or below it is refused as a replay whatever the clock says: the store can
// no longer prove its id was never consumed (M3-RT-F1 — a prune followed by a
// clock step-back of more than SeenRetention used to put a consumed grant
// back inside its window with its id forgotten).
//
// clock_floor is the latest server clock reading (unix seconds) the store has
// recorded — at every consumption and at every refusal of a genuine expired
// code (NoteExpired) — 0 = none yet. A grant whose expires_at is at or below
// it is EXPIRED whatever the clock says afterwards: the server's own
// "expired" verdict is sticky across a clock step-back (red-team red-3 #2 — a
// code refused as expired used to be admitted once the clock stepped back
// inside its window). A step-back smaller than MaxAuthorizationWindow costs
// nothing; a larger one refuses installs until the clock passes the floor.
const (
	// MaxSeenEntries is the hard cap: past it Consume refuses (fail closed)
	// rather than evicting a live entry.
	MaxSeenEntries = 10000
	// SeenRetention: an entry is pruned only once its expires_at passed more
	// than this long ago. Pruning is safe at any clock because every pruned
	// expiry is covered by pruned_through.
	SeenRetention = 10 * time.Minute

	// seenVersion 2 added pruned_through (red-team F1); 3 added clock_floor
	// (red-team red-3 #2). A v1 or v2 file reads ErrSeenCorrupt under
	// never-reset, and that is safe ONLY because M3 never shipped: no v1/v2
	// store was ever written by a shipped binary, so no box holds one.
	seenVersion      = 3
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
	// ErrExpiredAtFloor (errors.Is ErrExpired): expires_at is at or below the
	// store's clock_floor — the server has already seen a clock past it.
	ErrExpiredAtFloor = fmt.Errorf("%w: expires_at at or below the seen store's clock floor (the server already saw a later clock; was it stepped back?)", ErrExpired)
)

type seenEntry struct {
	JobID      string `json:"job_id"`
	ExpiresAt  int64  `json:"expires_at"`
	ConsumedAt int64  `json:"consumed_at"`
}

// seenStore is the decoded store.
type seenStore struct {
	PrunedThrough int64
	ClockFloor    int64
	IDs           []seenEntry
}

// seenFile is the on-disk shape (field order is the write order).
type seenFile struct {
	V             int         `json:"v"`
	PrunedThrough int64       `json:"pruned_through"`
	ClockFloor    int64       `json:"clock_floor"`
	IDs           []seenEntry `json:"ids"`
}

// errSeenMissing: the store is absent although the installation is enrolled.
var errSeenMissing = errors.New("seen-job store missing although the installation is enrolled (Enroll creates it) — never treated as empty")

// readSeen: anything unreadable ⇒ ErrSeenCorrupt. Absent is empty ONLY while
// nothing is enrolled (neither admin.json nor device.key exists — Consume is
// reachable only behind the gate, which requires both). Once enrolled, Enroll
// has created the store, so an absent one was moved, deleted or rolled back
// with its directory — reading it as empty would re-open every consumed job
// id (L7: absent ≠ []; red-team red-3 #4).
func readSeen(dataDir string) (seenStore, error) {
	b, err := readPrivateFile(SeenPath(dataDir), maxSeenFileBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if enrolledOnDisk(dataDir) {
				return seenStore{}, errors.Join(ErrSeenCorrupt, errSeenMissing)
			}
			return seenStore{}, nil
		}
		return seenStore{}, errors.Join(ErrSeenCorrupt, err)
	}
	return parseSeen(b)
}

// enrolledOnDisk: admin.json or device.key exists in any form (a stat error
// other than not-exist counts as present — fail closed).
func enrolledOnDisk(dataDir string) bool {
	for _, p := range []string{AdminPath(dataDir), DeviceKeyPath(dataDir)} {
		if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			return true
		}
	}
	return false
}

// ensureSeenStore creates an empty seen-job store when none exists, under
// .seen.lock, through the writer that runs its reader's validator. An
// existing store — valid or not — is never touched (never-reset).
func ensureSeenStore(dir, dataDir string) error {
	unlock, err := lockFile(seenLockPath(dataDir))
	if err != nil {
		return err
	}
	defer unlock()
	if _, err := os.Lstat(SeenPath(dataDir)); !errors.Is(err, os.ErrNotExist) {
		return err // present (nil) or unstatable (refuse)
	}
	b, err := encodeSeen(seenStore{})
	if err != nil {
		return err
	}
	return writeAtomic(dir, SeenPath(dataDir), b)
}

// parseSeen is THE store validator: the reader applies it to what it reads
// and the writer (encodeSeen) to what it is about to write.
func parseSeen(b []byte) (seenStore, error) {
	if len(b) > maxSeenFileBytes {
		return seenStore{}, ErrSeenCorrupt
	}
	m, err := decodeStrictObject(bytes.NewReader(b), "v", "pruned_through", "clock_floor", "ids")
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
	if s.ClockFloor, err = rawNonNegInt(m["clock_floor"]); err != nil {
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
	b, err := json.Marshal(seenFile{V: seenVersion, PrunedThrough: s.PrunedThrough, ClockFloor: s.ClockFloor, IDs: ids})
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	if _, err := parseSeen(b); err != nil {
		return nil, fmt.Errorf("updateauth: refusing to write a seen-job store its reader would refuse: %w", err)
	}
	return b, nil
}

// Consume records jobID as used, atomically under .seen.lock. It reads the
// clock (clock()) AFTER the lock is held and judges everything on that
// reading (red-team red-3 #3: a request parked on the lock must not be
// judged, or recorded, on a reading taken before the wait): expiresAt outside
// CheckExpiry's window ⇒ ErrExpired (an expired genuine code also raises
// clock_floor, as NoteExpired does); a job id already present ⇒ ErrReplay;
// expiresAt at or below pruned_through ⇒ ErrPrunedReplay (errors.Is
// ErrReplay); at or below clock_floor ⇒ ErrExpiredAtFloor (errors.Is
// ErrExpired); a corrupt store ⇒ ErrSeenCorrupt and the file is left
// byte-identical; MaxSeenEntries live entries ⇒ ErrSeenFull; a clock at or
// before the epoch ⇒ ErrBadClock (checked before any lock or write, and
// again under the lock). Only entries whose expires_at passed more than
// SeenRetention ago are pruned, and the largest pruned expires_at is kept as
// pruned_through. consumed_at is the reading taken under the lock.
//
// Callers pass a clock, never a time: the install handler passes its clock
// seam. Known limit: the lock wait itself is unbounded (flock LOCK_EX); a
// hung holder parks the request, but can no longer get a stale verdict.
func Consume(dataDir, jobID string, expiresAt int64, clock func() time.Time) error {
	if err := checkDataDir(dataDir); err != nil {
		return err
	}
	if !ValidJobID(jobID) || expiresAt <= 0 {
		return malformed("job_id/expires_at")
	}
	if clock == nil {
		return errors.New("updateauth: Consume needs a clock")
	}
	if clock().Unix() <= 0 {
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
	now := clock() // red-3 #3: the reading every verdict below is taken on
	if now.Unix() <= 0 {
		return ErrBadClock
	}
	st, err := readSeen(dataDir)
	if err != nil {
		return err
	}
	if err := CheckExpiry(expiresAt, now); err != nil {
		if expiresAt <= now.Unix() {
			// a genuine code (the caller verified its MAC) expired while it
			// waited: its refusal is sticky, like NoteExpired's
			if werr := raiseClockFloor(dir, dataDir, st, now); werr != nil {
				return errors.Join(err, werr)
			}
		}
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
	if expiresAt <= st.ClockFloor {
		return ErrExpiredAtFloor
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
	b, err := encodeSeen(seenStore{PrunedThrough: watermark, ClockFloor: max(st.ClockFloor, now.Unix()), IDs: kept})
	if err != nil {
		return err
	}
	return writeAtomic(dir, SeenPath(dataDir), b)
}

// raiseClockFloor rewrites st with clock_floor = now when now is above it
// (the caller holds .seen.lock and read st under it).
func raiseClockFloor(dir, dataDir string, st seenStore, now time.Time) error {
	if now.Unix() <= st.ClockFloor {
		return nil
	}
	st.ClockFloor = now.Unix()
	b, err := encodeSeen(st)
	if err != nil {
		return err
	}
	return writeAtomic(dir, SeenPath(dataDir), b)
}

// NoteExpired records that the server refused a GENUINE code (its MAC
// verified) as expired at now: it raises clock_floor to now, so that code —
// and every code expiring at or before now — stays expired after a clock
// step-back (red-team red-3 #2). It writes nothing when now is not above the
// floor, and never writes a store its reader would refuse (a clock at or
// before the epoch is ErrBadClock; a corrupt or, once enrolled, missing store
// is ErrSeenCorrupt and left untouched). The caller refuses the request
// whatever this returns. Callers: the install handler, ONLY after VerifyMAC —
// a caller without the key can never move the floor.
func NoteExpired(dataDir string, now time.Time) error {
	if err := checkDataDir(dataDir); err != nil {
		return err
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
	return raiseClockFloor(dir, dataDir, st, now)
}
