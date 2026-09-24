package updateauth

// THROWAWAY red-team probes (M3 adversarial lens: MAC, replay, injection).
// Deleted after the run.

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

// RT1: Message is injective — for every accepted (release, job, exp) the
// message splits back into exactly the same triple.
func TestRT_MessageInjective(t *testing.T) {
	alpha := []rune("aZ09._-|/\\:%\x00\n\r\t ​ÅＡ０KkK" + "..")
	r := rand.New(rand.NewSource(1))
	safe := []rune("abcdef0123456789-._XY")
	gen := func() string {
		n := 1 + r.Intn(70)
		var b strings.Builder
		for i := 0; i < n; i++ {
			if r.Intn(40) == 0 {
				b.WriteRune(alpha[r.Intn(len(alpha))])
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
	t.Logf("accepted %d random triples; all split back uniquely", accepted)
}

// RT2: validators vs whitespace / newline / unicode tricks.
func TestRT_IDValidatorEdges(t *testing.T) {
	for _, s := range []string{"v1\n", "v1\r", "\nv1", "v1 ", " v1", "v1​", "ｖ1", "vı", "v1\x00", "v1.\u0000",
		"v1/..", "v1\\", "v1%2f", "V1..", "a..b", ".", "..", "-v1", "_v1", ".v1"} {
		if ValidReleaseID(s) {
			t.Errorf("release %q accepted", s)
		}
	}
	for _, s := range []string{"0123456789abcdef\n", "0123456789ABCDEF", "0123456789abcdeＦ", "-123456789abcdef", "0123456​789abcdef", "1234567"} {
		if ValidJobID(s) {
			t.Errorf("job %q accepted", s)
		}
	}
	// accepted-but-notable shapes (reported, not failures)
	for _, s := range []string{"v1.", "v1-", "CON", "NUL", "aux.txt", "v1._"} {
		t.Logf("release %q accepted=%v", s, ValidReleaseID(s))
	}
}

// RT3: JSON-level aliasing: escaped duplicate keys, escaped values.
func TestRT_ParseAliases(t *testing.T) {
	mac := strings.Repeat("a", 64)
	dupEsc := `{"release_id":"v1","release\u005fid":"v2","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`
	if _, err := ParseInstallRequest(strings.NewReader(dupEsc)); !errors.Is(err, ErrMalformed) {
		t.Errorf("escaped duplicate key accepted: %v", err)
	}
	escVal := `{"release_id":"\u0076\u0031","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"\u0061` + mac[1:] + `"}`
	g, err := ParseInstallRequest(strings.NewReader(escVal))
	t.Logf("escaped values decode to %+v err=%v (same decoded triple → same MAC and same seen key)", g, err)
	bom := "\xef\xbb\xbf" + `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`
	if _, err := ParseInstallRequest(strings.NewReader(bom)); err == nil {
		t.Errorf("BOM accepted")
	}
	badUTF8 := `{"release_id":"v1` + "\xff" + `","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`
	if _, err := ParseInstallRequest(strings.NewReader(badUTF8)); err == nil {
		t.Errorf("invalid UTF-8 release accepted")
	}
	hmacNL := `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `\n"}`
	g2, err := ParseInstallRequest(strings.NewReader(hmacNL))
	if err == nil && VerifyMAC(make([]byte, 32), g2.ReleaseID, g2.JobID, g2.ExpiresAt, g2.HMAC) {
		t.Errorf("hmac with trailing newline verified")
	}
}

// RT4: expiry edges with sub-second now.
func TestRT_ExpiryEdges(t *testing.T) {
	base := time.Unix(1_800_000_000, 999_999_999)
	n := base.Unix()
	if CheckExpiry(n+300, base) != nil {
		t.Errorf("n+300 refused at .999")
	}
	if CheckExpiry(n+301, base) == nil {
		t.Errorf("n+301 accepted")
	}
	// window seen by the grant minted at `base` by Authorize
	exp := base.Add(MaxAuthorizationWindow).Unix()
	t.Logf("Authorize at %v mints exp=%d (= n+%d)", base, exp, exp-n)
}

// RT5: CLOCK ROLLBACK after pruning re-opens a consumed job id.
func TestRT_ClockRollbackAfterPruneReopensReplay(t *testing.T) {
	d := t.TempDir()
	T := time.Unix(1_800_000_000, 0)
	jobA := "aaaaaaaaaaaaaaaa"
	expA := T.Unix() + 300
	if err := Consume(d, jobA, expA, T); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, jobA, expA, T); !errors.Is(err, ErrReplay) {
		t.Fatalf("replay not refused: %v", err)
	}
	// clock runs forward; some later install prunes A
	T2 := time.Unix(expA+601, 0)
	if err := Consume(d, "bbbbbbbbbbbbbbbb", T2.Unix()+300, T2); err != nil {
		t.Fatal(err)
	}
	// clock stepped back to before A's expiry
	T3 := time.Unix(expA-10, 0)
	if CheckExpiry(expA, T3) != nil {
		t.Fatalf("expiry refused at T3")
	}
	err := Consume(d, jobA, expA, T3)
	t.Logf("replay of A after prune + rollback: err=%v", err)
	if err == nil {
		t.Errorf("REPLAY ACCEPTED: job %s consumed twice (clock rollback of %ds after a prune)", jobA, T2.Unix()-T3.Unix())
	}
}

// RT6: seen-store oversize and near-cap sizes.
func TestRT_SeenStoreSizes(t *testing.T) {
	d := t.TempDir()
	if _, err := ensurePrivateDir(d); err != nil {
		t.Fatal(err)
	}
	// 10000 live entries with the longest job id: does it fit maxSeenFileBytes?
	var b strings.Builder
	b.WriteString(`{"v":1,"ids":[`)
	exp := tNow.Unix() + 300
	for i := 0; i < MaxSeenEntries-1; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		id := fmt.Sprintf("%064d", i)
		fmt.Fprintf(&b, `{"job_id":"%s","expires_at":%d,"consumed_at":%d}`, id, exp, tNow.Unix())
	}
	b.WriteString("]}\n")
	t.Logf("9999 max-length entries = %d bytes (cap %d)", b.Len(), maxSeenFileBytes)
	if err := os.WriteFile(SeenPath(d), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, strings.Repeat("f", 64), exp, tNow); err != nil {
		t.Errorf("9999+1: %v", err)
	}
	if err := Consume(d, strings.Repeat("e", 64), exp, tNow); !errors.Is(err, ErrSeenFull) {
		t.Errorf("10000+1: %v want ErrSeenFull", err)
	}
	// oversize by padding
	big := make([]byte, maxSeenFileBytes+1)
	for i := range big {
		big[i] = ' '
	}
	copy(big, []byte(`{"v":1,"ids":[]}`))
	if err := os.WriteFile(SeenPath(d), big, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, strings.Repeat("d", 64), exp, tNow); !errors.Is(err, ErrSeenCorrupt) {
		t.Errorf("oversize: %v want ErrSeenCorrupt", err)
	}
	for name, body := range map[string]string{
		"v float":      `{"v":1.0,"ids":[]}`,
		"ids null":     `{"v":1,"ids":null}`,
		"dup v":        `{"v":1,"v":1,"ids":[]}`,
		"entry extra":  `{"v":1,"ids":[{"job_id":"0123456789abcdef","expires_at":1,"consumed_at":1,"x":1}]}`,
		"entry badid":  `{"v":1,"ids":[{"job_id":"../x","expires_at":1,"consumed_at":1}]}`,
		"entry exp0":   `{"v":1,"ids":[{"job_id":"0123456789abcdef","expires_at":0,"consumed_at":1}]}`,
		"ids object":   `{"v":1,"ids":{}}`,
		"two objs":     `{"v":1,"ids":[]}{"v":1,"ids":[]}`,
		"v string":     `{"v":"1","ids":[]}`,
		"ids trailing": `{"v":1,"ids":[] }garbage`,
	} {
		if err := os.WriteFile(SeenPath(d), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := Consume(d, strings.Repeat("c", 64), exp, tNow); !errors.Is(err, ErrSeenCorrupt) {
			t.Errorf("%s: %v want ErrSeenCorrupt", name, err)
		}
		if got, _ := os.ReadFile(SeenPath(d)); string(got) != body {
			t.Errorf("%s: store rewritten", name)
		}
	}
}

// RT7: a seen store that is ABSENT while the enrollment exists — any deletion
// silently resets single-use (documented same-uid; recorded for the report).
func TestRT_SeenStoreDeletionResets(t *testing.T) {
	d := t.TempDir()
	exp := tNow.Unix() + 60
	if err := Consume(d, "0123456789abcdef", exp, tNow); err != nil {
		t.Fatal(err)
	}
	os.Remove(SeenPath(d))
	err := Consume(d, "0123456789abcdef", exp, tNow)
	t.Logf("after deleting seen_job_ids.json: replay err=%v", err)
}

// RT8: consume with the clock BEFORE 1970 / consumed_at <= 0 poisons the store?
func TestRT_ConsumedAtNonPositivePoisons(t *testing.T) {
	d := t.TempDir()
	past := time.Unix(0, 0)
	err := Consume(d, "0123456789abcdef", 100, past)
	t.Logf("consume at epoch 0: %v", err)
	err2 := Consume(d, "0123456789abcdee", 100, time.Unix(1, 0))
	t.Logf("next consume: %v", err2)
}
