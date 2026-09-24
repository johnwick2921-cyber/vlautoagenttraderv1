package api

// W-ONE-BUTTON M3 — adversarial pins kept from the stop-snapshot red-team
// probes (zz_redteam_api_test.go, triaged 2026-09-24), driven at the
// PRODUCTION router (NewServer + gin.Default + corsMiddleware + setupRoutes,
// canon 53) through the harness in handler_updates_test.go. Each test is
// named by the property it pins and carries a positive control.
//
// The triage findings M3-RT-F1 and M3-RT-F2 were RED here (gated on
// NOFX_M3_OPEN_FINDINGS) until the fix commit removed the gate; they are
// now ordinary pins.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/internal/updateauth"
	"nofx/logger"
	"nofx/manager"

	"github.com/gin-gonic/gin"
)

// grantBodyUnder is an install body whose MAC is computed under key.
func grantBodyUnder(t *testing.T, key []byte, rel, job string, exp int64) string {
	t.Helper()
	mac, err := updateauth.ComputeMAC(key, rel, job, exp)
	if err != nil {
		t.Fatal(err)
	}
	return grantBody(updateauth.Grant{ReleaseID: rel, JobID: job, ExpiresAt: exp, HMAC: mac})
}

// attackerBodyUnder is an install body whose MAC an attacker computes under
// key with a plain HMAC-SHA256 over the canonical tagged message — no
// updateauth minting rule applies (it is how a guessed key would be used).
func attackerBodyUnder(key []byte, rel, job string, exp int64) string {
	m := hmac.New(sha256.New, key)
	fmt.Fprintf(m, "%s|%s|%s|%d", updateauth.MACPurpose, rel, job, exp)
	return grantBody(updateauth.Grant{ReleaseID: rel, JobID: job, ExpiresAt: exp, HMAC: hex.EncodeToString(m.Sum(nil))})
}

// ── replay ───────────────────────────────────────────────────────────────

// A consumed id that a prune pass RETAINS (its expiry passed less than
// SeenRetention ago) is still a replay after the clock is stepped back to
// inside its validity window — the bound seen.go's retention rule promises.
func TestInstallReplayRefusedAfterAClockStepBackWhileTheIDIsRetained(t *testing.T) {
	e := newUpdEnv(t)
	cur := time.Now()
	e.s.updatesNow = func() time.Time { return cur }
	key := mustKey(t, e.dataDir)
	expA := cur.Unix() + 300
	bodyA := grantBodyUnder(t, key, updRelease, "retained-job-a0001", expA)
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: first use = %d", w.Code)
	}
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusConflict {
		t.Fatalf("immediate replay = %d, want 409", w.Code)
	}
	// a later install runs a prune pass while A's expiry is 599s old: A stays
	cur = time.Unix(expA+599, 0)
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "retained-job-b0001", cur.Unix()+300)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("B = %d", w.Code)
	}
	if b, _ := os.ReadFile(updateauth.SeenPath(e.dataDir)); !strings.Contains(string(b), "retained-job-a0001") {
		t.Fatal("precondition: A was pruned inside the retention window")
	}
	// the clock is stepped back to inside A's window
	cur = time.Unix(expA-10, 0)
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusConflict {
		t.Fatalf("replay of a retained id after a clock step-back = %d %s, want 409", w.Code, w.Body.String())
	}
}

// Pruning cannot re-open a replay while the clock only moves forward: once an
// id is pruned (expired more than SeenRetention ago) its grant is refused as
// expired with the gate's 403 body, never admitted.
func TestInstallPrunedGrantIsRefusedAsExpiredWhileTheClockMovesForward(t *testing.T) {
	e := newUpdEnv(t)
	cur := time.Now()
	e.s.updatesNow = func() time.Time { return cur }
	key := mustKey(t, e.dataDir)
	expA := cur.Unix() + 300
	bodyA := grantBodyUnder(t, key, updRelease, "prune-job-a00001", expA)
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: first use = %d", w.Code)
	}
	cur = time.Unix(expA+601, 0)
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "prune-job-b00001", cur.Unix()+300)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("B = %d", w.Code)
	}
	if b, _ := os.ReadFile(updateauth.SeenPath(e.dataDir)); strings.Contains(string(b), "prune-job-a00001") {
		t.Fatal("precondition: A was not pruned")
	}
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("pruned grant, clock moving forward = %d %s, want 403 %s", w.Code, w.Body.String(), forbiddenBody)
	}
}

// A replay is keyed on the DECODED job id, not the body bytes: the same grant
// re-encoded (unicode escapes in every value and in a key name, reordered
// keys, extra whitespace) is 409, and a fresh MAC over the same job id with
// another release and expiry is 409 too.
func TestInstallReplayIsKeyedOnTheDecodedJobIDNotTheBodyBytes(t *testing.T) {
	e := newUpdEnv(t)
	g := e.grant(updRelease)
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: first use = %d", w.Code)
	}
	esc := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			fmt.Fprintf(&b, `\u%04x`, r)
		}
		return b.String()
	}
	re := fmt.Sprintf(" {\n \"hmac\" : \"%s\" ,\"expires_at\":%d, \"job\\u005fid\":\"%s\",\"release_id\":\"%s\"}\n\t",
		esc(g.HMAC), g.ExpiresAt, esc(g.JobID), esc(g.ReleaseID))
	if w := e.do("POST", "/api/updates/install", re); w.Code != http.StatusConflict {
		t.Errorf("re-encoded replay = %d %s, want 409", w.Code, w.Body.String())
	}
	key := mustKey(t, e.dataDir)
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, "other-release", g.JobID, g.ExpiresAt-1)); w.Code != http.StatusConflict {
		t.Errorf("same job id, different release/expiry = %d, want 409", w.Code)
	}
}

// Body aliasing and size: an escaped duplicate key is 400; a valid grant
// padded past the 4 KiB cap is 400; neither spends the job id — the same
// grant padded to exactly 4096 bytes then reaches the stub (422, not 409).
func TestInstallBodyAliasesAndOversizeAreRefusedWithoutSpendingTheJob(t *testing.T) {
	e := newUpdEnv(t)
	g := e.grant(updRelease)
	dup := fmt.Sprintf(`{"release_id":"%s","release\u005fid":"x","job_id":"%s","expires_at":%d,"hmac":"%s"}`, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC)
	if w := e.do("POST", "/api/updates/install", dup); w.Code != http.StatusBadRequest {
		t.Errorf("escaped duplicate key = %d, want 400", w.Code)
	}
	if w := e.do("POST", "/api/updates/install", grantBody(g)+strings.Repeat(" ", 5000)); w.Code != http.StatusBadRequest {
		t.Errorf("valid grant + 5000 bytes of whitespace = %d, want 400", w.Code)
	}
	exact := grantBody(g)
	exact += strings.Repeat(" ", maxUpdateInstallBody-len(exact))
	if w := e.do("POST", "/api/updates/install", exact); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("valid grant padded to exactly %d bytes = %d, want 422 (the refusals above did not spend it)", maxUpdateInstallBody, w.Code)
	}
}

// An unsafe seen store — 0644, or a symlink — refuses the install with the
// gate's byte-identical 403 body and is never rewritten (nor is a symlink's
// target).
func TestInstallRefusesAnUnsafeSeenStoreUniformlyAndNeverRewritesIt(t *testing.T) {
	e := newUpdEnv(t)
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: first install = %d", w.Code)
	}
	p := updateauth.SeenPath(e.dataDir)
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("0644 seen store = %d %s, want 403 %s", w.Code, w.Body.String(), forbiddenBody)
	}
	if after, _ := os.ReadFile(p); !bytes.Equal(before, after) {
		t.Error("the 0644 seen store was rewritten")
	}
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: the restored 0600 store = %d, want 422", w.Code)
	}
	tgt := p + ".real"
	if err := os.Rename(p, tgt); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(tgt, p); err != nil {
		t.Fatal(err)
	}
	tgtBefore, err := os.ReadFile(tgt)
	if err != nil {
		t.Fatal(err)
	}
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("symlinked seen store = %d %s, want 403 %s", w.Code, w.Body.String(), forbiddenBody)
	}
	if fi, err := os.Lstat(p); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlinked seen store was replaced")
	}
	if tgtAfter, _ := os.ReadFile(tgt); !bytes.Equal(tgtBefore, tgtAfter) {
		t.Error("the symlink's target was rewritten")
	}
}

// ── logs ─────────────────────────────────────────────────────────────────

// A MAC a client embeds in a malformed body — as an unknown field NAME (the
// strict parser quotes it into its error) or inside release_id — never
// reaches the logs, and the corrupt-store error path logs neither the MAC,
// the token nor the key. (A MAC the client puts in the URL path is logged as
// the path it chose; that is not a server leak and is not pinned.)
func TestUpdatesNeverLogAMACEmbeddedInAMalformedBodyOrTheStoreError(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex
	sink := writerFunc(func(p []byte) (int, error) { mu.Lock(); defer mu.Unlock(); return buf.Write(p) })
	prevGin := gin.DefaultWriter
	gin.DefaultWriter = sink // gin.Default()'s Logger binds it at NewServer
	logger.Log.SetOutput(sink)
	t.Cleanup(func() { gin.DefaultWriter = prevGin; logger.Log.SetOutput(os.Stdout) })

	e := newUpdEnv(t)
	g := e.grant(updRelease)
	key := mustKey(t, e.dataDir)
	if w := e.do("POST", "/api/updates/install", `{"`+g.HMAC+`":1}`); w.Code != http.StatusBadRequest {
		t.Fatalf("MAC as a field name = %d, want 400", w.Code)
	}
	if w := e.do("POST", "/api/updates/install", `{"release_id":"`+g.HMAC+`/x","job_id":"a","expires_at":1,"hmac":"x"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("MAC inside release_id = %d, want 400", w.Code)
	}
	if err := os.WriteFile(updateauth.SeenPath(e.dataDir), []byte("junk"), 0o600); err != nil {
		t.Fatal(err)
	}
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusForbidden {
		t.Fatalf("corrupt store = %d, want 403", w.Code)
	}
	mu.Lock()
	logs := buf.String()
	mu.Unlock()
	if !strings.Contains(logs, "job-id store refused") {
		t.Fatal("positive control: the corrupt-store error line was not captured")
	}
	for name, s := range map[string]string{"mac": g.HMAC, "token": e.tok, "key hex": hex.EncodeToString(key), "raw key": string(key)} {
		if strings.Contains(logs, s) {
			t.Errorf("the logs contain the %s", name)
		}
	}
}

// ── post-consume outcomes ────────────────────────────────────────────────

// lyingVerifier "verifies" a manifest for a release other than the one asked.
type lyingVerifier struct{}

func (lyingVerifier) VerifiedManifest(id string) (updateauth.Manifest, error) {
	return updateauth.Manifest{ReleaseID: id + "x"}, nil
}

// The job id is spent the moment authorization passes, whatever follows: a
// verifier that answers with a DIFFERENT release's manifest is not a
// verification (422, the starter never runs); a verified manifest with no
// starter is 503; a starter that errors runs once and is 503 — and in every
// case the replay is 409.
func TestInstallJobStaysSpentAfterEveryPostConsumeOutcome(t *testing.T) {
	e := newUpdEnv(t)
	started := 0
	failing := UpdateStarter(func(updateauth.Grant, updateauth.Manifest) error { started++; return errors.New("boom") })
	for _, c := range []struct {
		name   string
		v      updateauth.Verifier
		start  UpdateStarter
		first  int
		starts int
	}{
		{"verifier answers another release", lyingVerifier{}, failing, http.StatusUnprocessableEntity, 0},
		{"verified, no starter", fakeVerifier{}, nil, http.StatusServiceUnavailable, 0},
		{"starter errors", fakeVerifier{}, failing, http.StatusServiceUnavailable, 1},
	} {
		started = 0
		e.s.SetUpdateVerifier(c.v)
		e.s.SetUpdateStarter(c.start)
		g := e.grant(updRelease)
		w1 := e.do("POST", "/api/updates/install", grantBody(g))
		w2 := e.do("POST", "/api/updates/install", grantBody(g))
		if w1.Code != c.first {
			t.Errorf("%s: first use = %d %s, want %d", c.name, w1.Code, w1.Body.String(), c.first)
		}
		if w2.Code != http.StatusConflict {
			t.Errorf("%s: replay = %d, want 409", c.name, w2.Code)
		}
		if started != c.starts {
			t.Errorf("%s: starter ran %d times, want %d", c.name, started, c.starts)
		}
	}
}

// ── M3-RT-F1 / F2 (were open findings; RED at 6de60f66, fixed after it) ──

// M3-RT-F1 at the router: a consumed job id stays single-use across a clock
// step-back. The id is pruned once its expiry is SeenRetention old; a later
// step-back of more than SeenRetention puts the grant back in its window.
// The seen store's pruned-through watermark answers 409 — across a restart
// too — while a grant minted at the stepped-back clock still reaches the
// stub.
func TestInstallReplayRefusedAfterAClockStepBackPastRetention(t *testing.T) {
	e := newUpdEnv(t)
	cur := time.Now()
	e.s.updatesNow = func() time.Time { return cur }
	key := mustKey(t, e.dataDir)
	expA := cur.Unix() + 300
	bodyA := grantBodyUnder(t, key, updRelease, "rollback-job-a0001", expA)
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: first use = %d", w.Code)
	}
	cur = time.Unix(expA+601, 0) // a later install prunes A
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "rollback-job-b0001", cur.Unix()+300)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("B = %d", w.Code)
	}
	cur = time.Unix(expA-10, 0) // clock stepped back 611s (NTP/WSL correction)
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusConflict || w.Body.String() != `{"error":"job already used"}` {
		t.Errorf("job rollback-job-a0001 admitted twice after a prune + 611s step-back: %d %s, want 409", w.Code, w.Body.String())
	}
	// a restarted app over the same installation still refuses it
	e.s = NewServer(manager.NewTraderManager(), e.st, nil, "127.0.0.1", 0)
	e.s.updatesNow = func() time.Time { return cur }
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusConflict {
		t.Errorf("after a restart: %d %s, want 409", w.Code, w.Body.String())
	}
	// a grant minted at the stepped-back clock (expires_at above the
	// watermark, but at or below the clock floor the store recorded when it
	// consumed B) is expired: the uniform 403 (red-3 #2)
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "rollback-job-c0001", cur.Unix()+300)); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Fatalf("a fresh grant at the stepped-back clock = %d %s, want 403 (below the clock floor)", w.Code, w.Body.String())
	}
	// positive control: once the clock passes the floor a fresh grant is
	// authorized and reaches the stub
	cur = time.Unix(expA+602, 0)
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "rollback-job-d0001", cur.Unix()+300)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: a fresh grant past the clock floor = %d %s, want 422", w.Code, w.Body.String())
	}
}

// M3-RT-F2: an all-zero device.key (32 zero bytes, 0600, our uid — e.g. a
// zero-filled restore) is a key anyone can compute a MAC under; the
// possession factor refuses it rather than verifying against it — and so
// does every other /api/updates route (the enrollment is unusable), with
// the byte-identical 403. An all-0xFF key is refused the same way; the
// restored real key is admitted again (positive control).
func TestInstallRefusesAMACUnderAnAllZeroDeviceKey(t *testing.T) {
	e := newUpdEnv(t)
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: real key = %d", w.Code)
	}
	realKey := mustKey(t, e.dataDir)
	for _, fill := range []byte{0x00, 0xff} {
		weak := bytes.Repeat([]byte{fill}, updateauth.DeviceKeyLen)
		if err := os.WriteFile(updateauth.DeviceKeyPath(e.dataDir), weak, 0o600); err != nil {
			t.Fatal(err)
		}
		// allRoutes mints through the production minter, which (correctly)
		// cannot mint under a degenerate key — so the routes are listed here
		// with the attacker's MAC.
		for _, rt := range []updRoute{
			{"POST", "/api/updates/install", attackerBodyUnder(weak, updRelease, fmt.Sprintf("weak-key-job-%04x", fill), time.Now().Unix()+120)},
			{"GET", "/api/updates", ""},
			{"POST", "/api/updates/check", "{}"},
			{"GET", "/api/updates/jobs/0123456789abcdef", ""},
			{"GET", "/api/updates/jobs/0123456789abcdef/receipt", ""},
		} {
			if w := e.do(rt.method, rt.path, rt.body); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
				t.Errorf("%#02x-filled device.key: %s %s = %d %s, want 403 %s", fill, rt.method, rt.path, w.Code, w.Body.String(), forbiddenBody)
			}
		}
	}
	if err := os.WriteFile(updateauth.DeviceKeyPath(e.dataDir), realKey, 0o600); err != nil {
		t.Fatal(err)
	}
	e.expectAllAdmitted("real key restored")
}
