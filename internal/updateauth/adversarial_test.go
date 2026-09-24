package updateauth

// W-ONE-BUTTON M3 — adversarial pins kept from the stop-snapshot red-team
// probes (zz_redteam_test.go, triaged 2026-09-24). Each test is named by the
// property it pins; each was run and refused the attack it describes.
//
// The three triage findings M3-RT-F1..F3 were RED here (gated on
// NOFX_M3_OPEN_FINDINGS) until the fix commit removed the gate; they are
// now ordinary pins.

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// For every (release, job, exp) that Message accepts, the message splits on
// '|' back into exactly that triple — no two accepted triples share a MAC
// input. Seeded, so deterministic; the rare-character alphabet carries the
// separator, path, control and look-alike characters an attacker would try.
func TestMessageSplitsBackIntoItsExactTriple(t *testing.T) {
	rare := []rune("aZ09._-|/\\:%\x00\n\r\t \u200b\u00c5\uff21\uff10Kk\u212a" + "..")
	r := rand.New(rand.NewSource(1))
	safe := []rune("abcdef0123456789-._XY")
	gen := func() string {
		n := 1 + r.Intn(70)
		var b strings.Builder
		for i := 0; i < n; i++ {
			if r.Intn(40) == 0 {
				b.WriteRune(rare[r.Intn(len(rare))])
			} else {
				b.WriteRune(safe[r.Intn(len(safe))])
			}
		}
		return b.String()
	}
	accepted := 0
	for i := 0; i < 300000; i++ {
		rel, job := gen(), gen()
		exp := r.Int63n(1 << 40)
		m, err := Message(rel, job, exp)
		if err != nil {
			continue
		}
		accepted++
		parts := strings.Split(string(m), "|")
		if len(parts) != 3 || parts[0] != rel || parts[1] != job || parts[2] != strconv.FormatInt(exp, 10) {
			t.Fatalf("ambiguous message %q from (%q,%q,%d)", m, rel, job, exp)
		}
	}
	if accepted == 0 {
		t.Fatal("positive control: no generated triple was accepted — the property was never exercised")
	}
}

// Whitespace, newline, control, zero-width and Unicode look-alike variants of
// otherwise valid ids are refused (Go's `$` is end-of-text, so a trailing
// newline cannot ride through the anchors).
func TestIDValidatorsRefuseWhitespaceControlAndLookalikes(t *testing.T) {
	for _, s := range []string{"v1\n", "v1\r", "\nv1", "v1 ", " v1", "v1\u200b", "\uff561", "v\u0131", "v1\x00", "v1.\u0000",
		"v1/..", "v1\\", "v1%2f", "V1..", "a..b", ".", "..", "-v1", "_v1", ".v1"} {
		if ValidReleaseID(s) {
			t.Errorf("release %q accepted", s)
		}
	}
	for _, s := range []string{"0123456789abcdef\n", "0123456789ABCDEF", "0123456789abcde\uff26", "-123456789abcdef", "0123456\u200b789abcdef", "1234567"} {
		if ValidJobID(s) {
			t.Errorf("job %q accepted", s)
		}
	}
	if !ValidReleaseID("v1") || !ValidJobID("0123456789abcdef") { // positive control
		t.Fatal("positive control: a plain id was refused")
	}
}

// JSON-level aliasing cannot smuggle a second value or an invisible byte past
// the strict parser: an escaped duplicate key (release\u005fid IS
// release_id), a UTF-8 BOM and invalid UTF-8 are malformed; an hmac with a
// JSON-escaped trailing newline never verifies, even when the 64 hex chars
// before it are the real MAC.
func TestParseInstallRequestRefusesEscapedAliasesBOMAndInvalidUTF8(t *testing.T) {
	mac := strings.Repeat("a", 64)
	body := func(rel, h string) string {
		return `{"release_id":"` + rel + `","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + h + `"}`
	}
	if _, err := ParseInstallRequest(strings.NewReader(body("v1", mac))); err != nil { // positive control
		t.Fatalf("positive control: %v", err)
	}
	dupEsc := `{"release_id":"v1","release\u005fid":"v2","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`
	if _, err := ParseInstallRequest(strings.NewReader(dupEsc)); !errors.Is(err, ErrMalformed) {
		t.Errorf("escaped duplicate key: err = %v, want ErrMalformed", err)
	}
	if _, err := ParseInstallRequest(strings.NewReader("\xef\xbb\xbf" + body("v1", mac))); err == nil {
		t.Error("UTF-8 BOM accepted")
	}
	if _, err := ParseInstallRequest(strings.NewReader(body("v1\xff", mac))); err == nil {
		t.Error("invalid UTF-8 release_id accepted")
	}
	key := make([]byte, DeviceKeyLen)
	for i := range key {
		key[i] = byte(i + 1)
	}
	good, err := ComputeMAC(key, "v1", "0123456789abcdef", 1800000300)
	if err != nil {
		t.Fatal(err)
	}
	g, err := ParseInstallRequest(strings.NewReader(body("v1", good+`\n`)))
	if err == nil && VerifyMAC(key, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC) {
		t.Error("an hmac with a JSON-escaped trailing newline verified")
	}
	if g2, err := ParseInstallRequest(strings.NewReader(body("v1", good))); err != nil || !VerifyMAC(key, g2.ReleaseID, g2.JobID, g2.ExpiresAt, g2.HMAC) {
		t.Fatalf("positive control: the real MAC did not verify (%v)", err)
	}
}

// The expiry window is judged in whole unix seconds even when now carries a
// sub-second part: at n.999 the edge n+300 is inside and n+301 outside, and
// the grant Authorize mints at that instant is accepted at that instant.
func TestCheckExpiryJudgesWholeSecondsAtASubSecondNow(t *testing.T) {
	base := time.Unix(1_800_000_000, 999_999_999)
	n := base.Unix()
	if CheckExpiry(n+300, base) != nil {
		t.Error("n+300 refused at n.999")
	}
	if CheckExpiry(n+301, base) == nil {
		t.Error("n+301 accepted at n.999")
	}
	// the exp Authorize computes for a grant minted at base
	if exp := base.Add(MaxAuthorizationWindow).Unix(); exp != n+300 || CheckExpiry(exp, base) != nil {
		t.Errorf("a grant minted at n.999 carries exp=n%+d and is not accepted at the same instant", exp-n)
	}
}

// The seen store's entry cap binds before its byte cap (MaxSeenEntries
// max-length ids fit), the entry past the cap is ErrSeenFull, and every
// oversize or malformed shape is ErrSeenCorrupt with the file left
// byte-identical (never reset — an empty store would re-open every replay).
func TestSeenStoreEntryCapBindsBeforeTheByteCapAndMalformedShapesFailClosed(t *testing.T) {
	d := t.TempDir()
	if _, err := ensurePrivateDir(d); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString(`{"v":2,"pruned_through":0,"ids":[`)
	exp := tNow.Unix() + 300
	for i := 0; i < MaxSeenEntries-1; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"job_id":"%064d","expires_at":%d,"consumed_at":%d}`, i, exp, tNow.Unix())
	}
	b.WriteString("]}\n")
	if err := os.WriteFile(SeenPath(d), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, strings.Repeat("f", 64), exp, tNow); err != nil {
		t.Errorf("entry %d (the last under the cap): %v", MaxSeenEntries, err)
	}
	if err := Consume(d, strings.Repeat("e", 64), exp, tNow); !errors.Is(err, ErrSeenFull) {
		t.Errorf("entry %d: %v, want ErrSeenFull", MaxSeenEntries+1, err)
	}
	big := make([]byte, maxSeenFileBytes+1)
	for i := range big {
		big[i] = ' '
	}
	copy(big, []byte(`{"v":2,"pruned_through":0,"ids":[]}`))
	if err := os.WriteFile(SeenPath(d), big, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, strings.Repeat("d", 64), exp, tNow); !errors.Is(err, ErrSeenCorrupt) {
		t.Errorf("oversize store: %v, want ErrSeenCorrupt", err)
	}
	for name, body := range map[string]string{
		"v float":             `{"v":2.0,"pruned_through":0,"ids":[]}`,
		"ids null":            `{"v":2,"pruned_through":0,"ids":null}`,
		"dup v":               `{"v":2,"v":2,"pruned_through":0,"ids":[]}`,
		"entry extra":         `{"v":2,"pruned_through":0,"ids":[{"job_id":"0123456789abcdef","expires_at":1,"consumed_at":1,"x":1}]}`,
		"entry badid":         `{"v":2,"pruned_through":0,"ids":[{"job_id":"../x","expires_at":1,"consumed_at":1}]}`,
		"entry exp0":          `{"v":2,"pruned_through":0,"ids":[{"job_id":"0123456789abcdef","expires_at":0,"consumed_at":1}]}`,
		"entry consumed0":     `{"v":2,"pruned_through":0,"ids":[{"job_id":"0123456789abcdef","expires_at":1,"consumed_at":0}]}`,
		"ids object":          `{"v":2,"pruned_through":0,"ids":{}}`,
		"two objs":            `{"v":2,"pruned_through":0,"ids":[]}{"v":2,"pruned_through":0,"ids":[]}`,
		"v string":            `{"v":"2","pruned_through":0,"ids":[]}`,
		"ids trailing":        `{"v":2,"pruned_through":0,"ids":[] }garbage`,
		"v1 (no watermark)":   `{"v":1,"ids":[]}`,
		"watermark missing":   `{"v":2,"ids":[]}`,
		"watermark negative":  `{"v":2,"pruned_through":-1,"ids":[]}`,
		"watermark float":     `{"v":2,"pruned_through":1.5,"ids":[]}`,
		"watermark string":    `{"v":2,"pruned_through":"0","ids":[]}`,
		"watermark leading 0": `{"v":2,"pruned_through":01,"ids":[]}`,
		"watermark null":      `{"v":2,"pruned_through":null,"ids":[]}`,
		"watermark dup":       `{"v":2,"pruned_through":0,"pruned_through":0,"ids":[]}`,
	} {
		if err := os.WriteFile(SeenPath(d), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := Consume(d, strings.Repeat("c", 64), exp, tNow); !errors.Is(err, ErrSeenCorrupt) {
			t.Errorf("%s: %v, want ErrSeenCorrupt", name, err)
		}
		if got, _ := os.ReadFile(SeenPath(d)); string(got) != body {
			t.Errorf("%s: the store was rewritten", name)
		}
	}
	// positive control: a valid empty store accepts
	if err := os.WriteFile(SeenPath(d), []byte(`{"v":2,"pruned_through":0,"ids":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, strings.Repeat("c", 64), exp, tNow); err != nil {
		t.Fatalf("positive control: %v", err)
	}
}

// ── M3-RT-F1..F3 (were open findings; RED at 6de60f66, fixed after it) ────

// M3-RT-F1: a consumed job id stays single-use across a clock step-back.
// Consume prunes an entry once its expires_at is SeenRetention in the past;
// a later clock step-back of more than SeenRetention puts the pruned grant
// back inside its validity window. The pruned-through watermark refuses it
// as a replay; a grant minted at the stepped-back clock (expires_at above the
// watermark) is still admitted.
func TestConsumeRefusesAReplayAfterAClockRollbackPastRetention(t *testing.T) {
	d := t.TempDir()
	T := time.Unix(1_800_000_000, 0)
	jobA := "aaaaaaaaaaaaaaaa"
	expA := T.Unix() + 300
	if err := Consume(d, jobA, expA, T); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, jobA, expA, T); !errors.Is(err, ErrReplay) {
		t.Fatalf("positive control: immediate replay not refused: %v", err)
	}
	// the clock runs forward; a later install prunes A
	T2 := time.Unix(expA+601, 0)
	if err := Consume(d, "bbbbbbbbbbbbbbbb", T2.Unix()+300, T2); err != nil {
		t.Fatal(err)
	}
	// the clock is stepped back to before A's expiry: A's MAC is in-window again
	T3 := time.Unix(expA-10, 0)
	if CheckExpiry(expA, T3) != nil {
		t.Fatal("precondition: A is not inside its window at T3")
	}
	if err := Consume(d, jobA, expA, T3); !errors.Is(err, ErrReplay) || !errors.Is(err, ErrPrunedReplay) {
		t.Errorf("job %s after a prune + %ds clock step-back: err = %v, want ErrPrunedReplay (a replay)", jobA, T2.Unix()-T3.Unix(), err)
	}
	// positive control: a grant minted at the stepped-back clock carries
	// expires_at = T3+300 > the watermark (expA) and is admitted once
	fresh := "cccccccccccccccc"
	if err := Consume(d, fresh, T3.Unix()+300, T3); err != nil {
		t.Fatalf("positive control: a fresh grant at the stepped-back clock: %v", err)
	}
	if err := Consume(d, fresh, T3.Unix()+300, T3); !errors.Is(err, ErrReplay) || errors.Is(err, ErrPrunedReplay) {
		t.Fatalf("fresh id replay: err = %v, want the plain ErrReplay", err)
	}
}

// M3-RT-F3: Consume never writes a record its own reader refuses. At a
// clock reading <= the unix epoch it used to write consumed_at <= 0, which
// readSeen rejects as corrupt — and because a corrupt store is never reset,
// every later Consume failed until the file was repaired by hand.
func TestConsumeNeverWritesARecordItsReaderRefuses(t *testing.T) {
	for _, clock := range []time.Time{time.Unix(0, 0), time.Unix(-100, 0), time.Unix(0, 999_999_999)} {
		d := t.TempDir()
		_ = Consume(d, "0123456789abcdef", clock.Unix()+150, clock) // admitted or refused: either is fine (exp > 0 for all clocks)
		if err := Consume(d, "0123456789abcdee", tNow.Unix()+60, tNow); err != nil {
			t.Errorf("clock %d: the store no longer accepts a fresh id at a sane clock: %v", clock.Unix(), err)
		}
	}
}
