package updateauth

// W-ONE-BUTTON M3 — pins for the fixes of the red-team triage findings
// M3-RT-F1 (single-use across a clock step-back), M3-RT-F2 (degenerate
// device key) and M3-RT-F3 (a writer that emits what its reader refuses).
// The RED proofs themselves live in adversarial_test.go; these pin each
// check of the fix on its own, so a mutation of one check is not masked by
// another.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// seqKey is a non-degenerate 32-byte test key: start, start+1, …
func seqKey(start byte) []byte {
	k := make([]byte, DeviceKeyLen)
	for i := range k {
		k[i] = start + byte(i)
	}
	return k
}

// ── M3-RT-F2: degenerate device keys ─────────────────────────────────────

// A key whose bytes are all equal (all-zero from a zero-filled restore or a
// sparse copy, all-0xFF, any fill) is refused by the loader, the verifier
// and the minter: a MAC under it is computable by anyone in ≤256 guesses.
func TestDegenerateDeviceKeyIsRefusedByTheLoaderVerifierAndMinter(t *testing.T) {
	const rel, job, exp = "v1.2.3", "0123456789abcdef", int64(1800000300)
	for _, fill := range []byte{0x00, 0xff, 0x07} {
		key := bytes.Repeat([]byte{fill}, DeviceKeyLen)
		d := enrolled(t)
		mustLoad(t, d) // positive control: the enrolled random key loads
		if err := os.WriteFile(DeviceKeyPath(d), key, 0o600); err != nil {
			t.Fatal(err)
		}
		if k, err := LoadDeviceKey(d); !errors.Is(err, ErrUnsafe) || k != nil {
			t.Errorf("fill %#02x: LoadDeviceKey = (%d bytes, %v), want ErrUnsafe and no key", fill, len(k), err)
		}
		if _, err := Authorize(d, rel, tNow); !errors.Is(err, ErrUnsafe) {
			t.Errorf("fill %#02x: Authorize = %v, want ErrUnsafe", fill, err)
		}
		m := hmac.New(sha256.New, key)
		m.Write([]byte(MACPurpose + "|" + tUser + "|" + rel + "|" + job + "|1800000300"))
		if VerifyMAC(key, tUser, rel, job, exp, hex.EncodeToString(m.Sum(nil))) {
			t.Errorf("fill %#02x: VerifyMAC accepted a MAC under a degenerate key", fill)
		}
		if mac, err := ComputeMAC(key, tUser, rel, job, exp); err == nil || mac != "" {
			t.Errorf("fill %#02x: ComputeMAC minted %q under a degenerate key", fill, mac)
		}
	}
	// positive controls: the rule is "every byte equal", not "contains a
	// zero": one differing byte is not degenerate by this rule, and a
	// sequential key loads, mints and verifies.
	one := make([]byte, DeviceKeyLen)
	one[DeviceKeyLen-1] = 1
	if degenerateKey(one) {
		t.Error("a key with one differing byte classed degenerate")
	}
	d := enrolled(t)
	if err := os.WriteFile(DeviceKeyPath(d), seqKey(1), 0o600); err != nil {
		t.Fatal(err)
	}
	k, err := LoadDeviceKey(d)
	if err != nil {
		t.Fatalf("positive control: sequential key: %v", err)
	}
	mac, err := ComputeMAC(k, tUser, rel, job, exp)
	if err != nil || !VerifyMAC(k, tUser, rel, job, exp, mac) {
		t.Fatalf("positive control: sequential key mint/verify: %v", err)
	}
	if !degenerateKey(nil) || !degenerateKey([]byte{}) {
		t.Error("an empty key is not degenerate")
	}
}

// Enroll never writes a key LoadDeviceKey would refuse: a random source that
// returns a degenerate key makes Enroll fail with nothing written.
func TestEnrollNeverWritesADegenerateKey(t *testing.T) {
	prev := randRead
	t.Cleanup(func() { randRead = prev })
	randRead = func(b []byte) (int, error) {
		for i := range b {
			b[i] = 0
		}
		return len(b), nil
	}
	d := t.TempDir()
	if err := Enroll(d, tUser, tEmail, tHash, tNow, false); err == nil {
		t.Fatal("Enroll accepted an all-zero key from the random source")
	}
	for _, p := range []string{AdminPath(d), DeviceKeyPath(d)} {
		if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s written by a refused enroll (%v)", p, err)
		}
	}
	randRead = prev // positive control
	if err := Enroll(d, tUser, tEmail, tHash, tNow, false); err != nil {
		t.Fatalf("positive control: %v", err)
	}
	mustLoad(t, d)
}

// ── M3-RT-F3: the writer passes its reader's validator ──────────────────

// At a clock reading at or before the unix epoch Consume refuses with
// ErrBadClock BEFORE it creates the updater dir, takes the lock or writes:
// an existing store stays byte-identical, and a fresh dir stays empty.
func TestConsumeAtAClockAtOrBeforeTheEpochRefusesAndTouchesNothing(t *testing.T) {
	for _, clock := range []time.Time{time.Unix(0, 0), time.Unix(-1, 0), time.Unix(0, 999_999_999), time.Unix(-100, 0)} {
		fresh := t.TempDir()
		if err := Consume(fresh, "0123456789abcdef", 150, clockAt(clock)); !errors.Is(err, ErrBadClock) {
			t.Errorf("clock %v: err = %v, want ErrBadClock", clock.Unix(), err)
		}
		if _, err := os.Lstat(Dir(fresh)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("clock %v: the refused Consume created the updater dir", clock.Unix())
		}
		d := t.TempDir()
		if err := Consume(d, "0123456789abcdee", tNow.Unix()+60, clockAt(tNow)); err != nil {
			t.Fatal(err)
		}
		before, _ := os.ReadFile(SeenPath(d))
		if err := Consume(d, "0123456789abcdef", 150, clockAt(clock)); !errors.Is(err, ErrBadClock) {
			t.Errorf("clock %v on an existing store: err = %v, want ErrBadClock", clock.Unix(), err)
		}
		if after, _ := os.ReadFile(SeenPath(d)); !bytes.Equal(before, after) {
			t.Errorf("clock %v: the store was rewritten", clock.Unix())
		}
	}
	// positive control: the first second after the epoch is a clock
	if err := Consume(t.TempDir(), "0123456789abcdef", 301, clockAt(time.Unix(1, 0))); err != nil {
		t.Fatalf("positive control: %v", err)
	}
}

// encodeSeen refuses every store parseSeen would refuse, and what it emits
// parses back to exactly what it was given.
func TestEncodeSeenRefusesWhatItsReaderRefuses(t *testing.T) {
	good := seenEntry{JobID: "0123456789abcdef", ExpiresAt: 1800000300, ConsumedAt: 1800000000}
	for name, s := range map[string]seenStore{
		"consumed_at 0":        {IDs: []seenEntry{{JobID: good.JobID, ExpiresAt: good.ExpiresAt, ConsumedAt: 0}}},
		"consumed_at negative": {IDs: []seenEntry{{JobID: good.JobID, ExpiresAt: good.ExpiresAt, ConsumedAt: -5}}},
		"expires_at 0":         {IDs: []seenEntry{{JobID: good.JobID, ExpiresAt: 0, ConsumedAt: good.ConsumedAt}}},
		"path-shaped id":       {IDs: []seenEntry{{JobID: "../x", ExpiresAt: good.ExpiresAt, ConsumedAt: good.ConsumedAt}}},
		"negative watermark":   {PrunedThrough: -1, IDs: []seenEntry{good}},
		"negative clock floor": {ClockFloor: -1, IDs: []seenEntry{good}},
	} {
		if b, err := encodeSeen(s); err == nil {
			t.Errorf("%s: encodeSeen emitted %q", name, b)
		}
	}
	for name, s := range map[string]seenStore{
		"empty":          {},
		"one entry":      {IDs: []seenEntry{good}},
		"with watermark": {PrunedThrough: 1799999000, IDs: []seenEntry{good}},
		"with floor":     {PrunedThrough: 1799999000, ClockFloor: 1800000000, IDs: []seenEntry{good}},
	} {
		b, err := encodeSeen(s)
		if err != nil {
			t.Fatalf("positive control %s: %v", name, err)
		}
		back, err := parseSeen(b)
		if err != nil {
			t.Fatalf("positive control %s: parse back: %v", name, err)
		}
		want := s
		if want.IDs == nil {
			want.IDs = []seenEntry{}
		}
		if !reflect.DeepEqual(back, want) {
			t.Fatalf("%s: round trip %+v, want %+v", name, back, want)
		}
	}
}

// ── M3-RT-F1: the pruned-through watermark ───────────────────────────────

// pruned_through is the largest expires_at a prune has removed; it persists
// across Consume calls that prune nothing; a grant at or below it is
// ErrPrunedReplay (the boundary is inclusive) whatever the clock, one above
// it is admitted.
func TestPrunedThroughIsTheLargestPrunedExpiryAndBindsAtAnyClock(t *testing.T) {
	d := t.TempDir()
	T := tNow
	read := func() seenStore {
		t.Helper()
		st, err := readSeen(d)
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
	if err := Consume(d, "aaaaaaaaaaaaaaaa", T.Unix()+100, clockAt(T)); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, "bbbbbbbbbbbbbbbb", T.Unix()+300, clockAt(T)); err != nil {
		t.Fatal(err)
	}
	if st := read(); st.PrunedThrough != 0 {
		t.Fatalf("nothing pruned yet: pruned_through = %d, want 0", st.PrunedThrough)
	}
	T2 := T.Add(901 * time.Second) // cutoff = T+301: both a and b prune
	if err := Consume(d, "cccccccccccccccc", T2.Unix()+300, clockAt(T2)); err != nil {
		t.Fatal(err)
	}
	st := read()
	if st.PrunedThrough != T.Unix()+300 || len(st.IDs) != 1 || st.IDs[0].JobID != "cccccccccccccccc" {
		t.Fatalf("after the prune: %+v, want pruned_through=%d and only c", st, T.Unix()+300)
	}
	back := T.Add(50 * time.Second) // the clock is stepped back
	for id, exp := range map[string]int64{
		"aaaaaaaaaaaaaaaa": T.Unix() + 100, // a pruned id, replayed
		"bbbbbbbbbbbbbbbb": T.Unix() + 300, // the id AT the watermark
		"dddddddddddddddd": T.Unix() + 300, // a never-seen id at the watermark
	} {
		if err := Consume(d, id, exp, clockAt(back)); !errors.Is(err, ErrPrunedReplay) || !errors.Is(err, ErrReplay) {
			t.Errorf("%s exp=%d at a stepped-back clock: err = %v, want ErrPrunedReplay", id, exp, err)
		}
	}
	// one above the watermark is not a replay; at this stepped-back clock it
	// is at or below the clock floor (T2, recorded when c was consumed), so
	// it is expired (red-3 #2) — the watermark boundary is still exact
	if err := Consume(d, "eeeeeeeeeeeeeeee", T.Unix()+301, clockAt(back)); errors.Is(err, ErrReplay) || !errors.Is(err, ErrExpiredAtFloor) {
		t.Fatalf("exp one above the watermark at a stepped-back clock: %v, want ErrExpiredAtFloor (not a replay)", err)
	}
	if err := Consume(d, "ffffffffffffffff", T2.Unix()+300, clockAt(T2)); err != nil { // prunes nothing new
		t.Fatal(err)
	}
	if st := read(); st.PrunedThrough != T.Unix()+300 {
		t.Fatalf("a Consume that pruned nothing moved pruned_through to %d", st.PrunedThrough)
	}
	b, _ := os.ReadFile(SeenPath(d))
	if !strings.HasPrefix(string(b), `{"v":3,"pruned_through":`) {
		t.Fatalf("store shape %q", b)
	}
}
