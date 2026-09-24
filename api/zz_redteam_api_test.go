package api

// THROWAWAY red-team probes (M3 lens: MAC, replay, injection) against the
// PRODUCTION router built by NewServer. Deleted after the run.

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/internal/updateauth"
	"nofx/logger"

	"github.com/gin-gonic/gin"
)

func rtBody(t *testing.T, key []byte, rel, job string, exp int64) string {
	t.Helper()
	mac, err := updateauth.ComputeMAC(key, rel, job, exp)
	if err != nil {
		t.Fatal(err)
	}
	return grantBody(updateauth.Grant{ReleaseID: rel, JobID: job, ExpiresAt: exp, HMAC: mac})
}

// RT-A1: clock rollback after a prune replays a consumed job id through the
// production router.
func TestRTA_ClockRollbackAfterPruneReplaysThroughTheRouter(t *testing.T) {
	e := newUpdEnv(t)
	var mu sync.Mutex
	cur := time.Now()
	e.s.updatesNow = func() time.Time { mu.Lock(); defer mu.Unlock(); return cur }
	set := func(t2 time.Time) { mu.Lock(); cur = t2; mu.Unlock() }
	key := mustKey(t, e.dataDir)

	expA := cur.Unix() + 300
	bodyA := rtBody(t, key, updRelease, "rollback-job-a0001", expA)
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("first use = %d", w.Code)
	}
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusConflict {
		t.Fatalf("immediate replay = %d, want 409", w.Code)
	}
	// Rollback WITHIN retention: still 409 (the design's R5 claim).
	set(time.Unix(expA-200, 0))
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusConflict {
		t.Fatalf("replay after a small rollback = %d, want 409", w.Code)
	}
	// Clock moves (or is stepped) forward past retention; the owner's next
	// legitimate install prunes A.
	t2 := time.Unix(expA+601, 0)
	set(t2)
	if w := e.do("POST", "/api/updates/install", rtBody(t, key, updRelease, "rollback-job-b0001", t2.Unix()+300)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("B = %d", w.Code)
	}
	// Clock stepped back (NTP/WSL correction) to before A's expiry.
	set(time.Unix(expA-10, 0))
	w := e.do("POST", "/api/updates/install", bodyA)
	t.Logf("replay of A after prune + %ds rollback: %d %s", t2.Unix()-(expA-10), w.Code, w.Body.String())
	if w.Code != http.StatusConflict {
		t.Errorf("REPLAY PASSED THE SINGLE-USE GATE: job rollback-job-a0001 consumed twice (%d)", w.Code)
	}
}

// RT-A2: the same grant re-encoded (escapes, key order, whitespace) is the
// same job id → 409.
func TestRTA_ReencodedGrantIsStillAReplay(t *testing.T) {
	e := newUpdEnv(t)
	g := e.grant(updRelease)
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("first = %d", w.Code)
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
	w := e.do("POST", "/api/updates/install", re)
	if w.Code != http.StatusConflict {
		t.Errorf("re-encoded replay = %d %s, want 409", w.Code, w.Body.String())
	}
	// a NEW MAC over the same job id with a different exp/release is still a replay
	key := mustKey(t, e.dataDir)
	if w := e.do("POST", "/api/updates/install", rtBody(t, key, "other-release", g.JobID, g.ExpiresAt-1)); w.Code != http.StatusConflict {
		t.Errorf("same job id, different release/exp = %d, want 409", w.Code)
	}
}

// RT-A3: escaped duplicate key; oversize body; body refusals never consume.
func TestRTA_BodyAliasesAndSize(t *testing.T) {
	e := newUpdEnv(t)
	g := e.grant(updRelease)
	dup := fmt.Sprintf(`{"release_id":"%s","release\u005fid":"x","job_id":"%s","expires_at":%d,"hmac":"%s"}`, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC)
	if w := e.do("POST", "/api/updates/install", dup); w.Code != http.StatusBadRequest {
		t.Errorf("escaped duplicate key = %d, want 400", w.Code)
	}
	pad := grantBody(g) + strings.Repeat(" ", 5000)
	if w := e.do("POST", "/api/updates/install", pad); w.Code != http.StatusBadRequest {
		t.Errorf("valid grant + 5000 bytes of whitespace = %d, want 400", w.Code)
	}
	exact := grantBody(g)
	exact += strings.Repeat(" ", 4096-len(exact))
	w := e.do("POST", "/api/updates/install", exact)
	t.Logf("exactly 4096-byte body: %d", w.Code)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("4096-byte valid body = %d, want 422", w.Code)
	}
}

// RT-A4: every non-admitted install outcome has the gate's byte-identical
// body; the seen-store refusals (corrupt, full, unsafe mode) too.
func TestRTA_SeenStoreRefusalsLeakNothing(t *testing.T) {
	e := newUpdEnv(t)
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusUnprocessableEntity {
		t.Fatal(w.Code)
	}
	p := updateauth.SeenPath(e.dataDir)
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(p)
	w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease)))
	if w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("0644 seen store = %d %s", w.Code, w.Body.String())
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Error("0644 store rewritten")
	}
	_ = os.Chmod(p, 0o600)
	// symlinked seen store → refused, target untouched
	tgt := p + ".real"
	_ = os.Rename(p, tgt)
	_ = os.Symlink(tgt, p)
	w = e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease)))
	if w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("symlinked seen store = %d %s", w.Code, w.Body.String())
	}
	// .seen.lock replaced by a symlink → refused
	_ = os.Remove(p)
	_ = os.Rename(tgt, p)
}

// RT-A5: logs never carry MAC/token/key in any refusal path, including the
// seen-store error path, the 422 path and malformed bodies that embed them.
func TestRTA_LogsOnEveryPath(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex
	sink := writerFunc(func(p []byte) (int, error) { mu.Lock(); defer mu.Unlock(); return buf.Write(p) })
	prevGin := gin.DefaultWriter
	gin.DefaultWriter = sink
	logger.Log.SetOutput(sink)
	t.Cleanup(func() { gin.DefaultWriter = prevGin; logger.Log.SetOutput(os.Stdout) })
	e := newUpdEnv(t)
	g := e.grant(updRelease)
	key := mustKey(t, e.dataDir)
	// malformed body embedding the MAC in an unknown field NAME (the parser
	// formats unknown field names into its error)
	e.do("POST", "/api/updates/install", `{"`+g.HMAC+`":1}`)
	e.do("POST", "/api/updates/install", `{"release_id":"`+g.HMAC+`/x","job_id":"a","expires_at":1,"hmac":"x"}`)
	// the refusal log formats the URL path: a MAC or token in the path
	e.do("GET", "/api/updates/jobs/"+g.HMAC, "", withToken("garbage"))
	// the corrupt-store Errorf path
	_ = os.WriteFile(updateauth.SeenPath(e.dataDir), []byte("junk"), 0o600)
	e.do("POST", "/api/updates/install", grantBody(g))
	mu.Lock()
	logs := buf.String()
	mu.Unlock()
	for name, s := range map[string]string{"token": e.tok, "key hex": fmt.Sprintf("%x", key), "raw key": string(key)} {
		if strings.Contains(logs, s) {
			t.Errorf("the logs contain the %s", name)
		}
	}
	if strings.Contains(logs, g.HMAC) {
		idx := strings.Index(logs, g.HMAC)
		lo := idx - 120
		if lo < 0 {
			lo = 0
		}
		t.Logf("MAC appears in logs (context): %q", logs[lo:idx+64])
	}
	for _, line := range strings.Split(logs, "\n") {
		if strings.Contains(line, "updates") {
			t.Logf("LOG: %.200s", line)
		}
	}
}

// RT-A6: the 422/503 paths consume the job id; a verifier that ERRORS after
// consumption leaves the id spent (by design) and nothing re-opens it.
type rtAcceptVerifier struct{}

func (rtAcceptVerifier) VerifiedManifest(r string) (updateauth.Manifest, error) {
	return updateauth.Manifest{ReleaseID: r}, nil
}

type rtLyingVerifier struct{}

func (rtLyingVerifier) VerifiedManifest(r string) (updateauth.Manifest, error) {
	return updateauth.Manifest{ReleaseID: r + "x"}, nil
}

func TestRTA_PostConsumePaths(t *testing.T) {
	e := newUpdEnv(t)
	for name, v := range map[string]updateauth.Verifier{"accept": rtAcceptVerifier{}, "lying": rtLyingVerifier{}} {
		e.s.SetUpdateVerifier(v)
		g := e.grant(updRelease)
		w1 := e.do("POST", "/api/updates/install", grantBody(g))
		w2 := e.do("POST", "/api/updates/install", grantBody(g))
		t.Logf("%s verifier: first %d %s, replay %d", name, w1.Code, w1.Body.String(), w2.Code)
		if w2.Code != http.StatusConflict {
			t.Errorf("%s: replay %d", name, w2.Code)
		}
	}
	started := 0
	e.s.SetUpdateVerifier(rtAcceptVerifier{})
	e.s.SetUpdateStarter(func(updateauth.Grant, updateauth.Manifest) error { started++; return errors.New("boom") })
	g := e.grant(updRelease)
	w1 := e.do("POST", "/api/updates/install", grantBody(g))
	w2 := e.do("POST", "/api/updates/install", grantBody(g))
	t.Logf("starter error: first %d %s; replay %d; starts=%d", w1.Code, w1.Body.String(), w2.Code, started)
	if started != 1 || w2.Code != http.StatusConflict {
		t.Errorf("starter ran %d times / replay %d", started, w2.Code)
	}
}

// RT-A1b: after a prune WITHOUT rollback the pruned grant is expired (403).
func TestRTA_PrunedGrantWithoutRollbackIsExpired(t *testing.T) {
	e := newUpdEnv(t)
	cur := time.Now()
	e.s.updatesNow = func() time.Time { return cur }
	key := mustKey(t, e.dataDir)
	expA := cur.Unix() + 300
	bodyA := rtBody(t, key, updRelease, "prune-job-a00001", expA)
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusUnprocessableEntity {
		t.Fatal(w.Code)
	}
	cur = time.Unix(expA+601, 0)
	if w := e.do("POST", "/api/updates/install", rtBody(t, key, updRelease, "prune-job-b00001", cur.Unix()+300)); w.Code != http.StatusUnprocessableEntity {
		t.Fatal(w.Code)
	}
	b, _ := os.ReadFile(updateauth.SeenPath(e.dataDir))
	if strings.Contains(string(b), "prune-job-a00001") {
		t.Fatal("A not pruned")
	}
	if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("pruned grant without rollback = %d %s, want 403", w.Code, w.Body.String())
	}
}

// RT-A7: Q8 (iat >= updated_at) under clock skew: a token minted while the
// clock ran AHEAD survives a later password change.
func TestRTA_FutureIatTokenSurvivesPasswordChange(t *testing.T) {
	e := newUpdEnv(t)
	skewed := mintJWT(t, updAdminID, updAdminEmail, time.Now().Add(time.Hour), time.Now().Add(2*time.Hour), updSecret)
	if w := e.do("GET", "/api/updates", "", withToken(skewed)); w.Code != http.StatusOK {
		t.Logf("future-iat token before the change: %d (ValidateJWT refuses future iat?)", w.Code)
	}
	r := httptest.NewRequest("PUT", "/api/user/password", strings.NewReader(`{"new_password":"another-long-pass"}`))
	r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
	r.Header.Set("Authorization", "Bearer "+e.tok)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.s.router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("change password = %d", w.Code)
	}
	if w := e.do("GET", "/api/updates", "", withToken(e.tok)); w.Code != http.StatusForbidden {
		t.Fatalf("control: pre-change token = %d, want 403", w.Code)
	}
	w2 := e.do("GET", "/api/updates", "", withToken(skewed))
	t.Logf("future-iat token (minted before the change, clock ahead) after the change: %d %s", w2.Code, w2.Body.String())
	if w2.Code != http.StatusForbidden {
		t.Errorf("Q8 BYPASS under clock skew: a token issued (in real time) before the password change is admitted (%d)", w2.Code)
	}
}

// RT-A8: an all-zero device.key (a zero-filled file after a bad restore or a
// torn write) is accepted as a real key: anyone who guesses "zeros" mints.
func TestRTA_AllZeroDeviceKeyIsAccepted(t *testing.T) {
	e := newUpdEnv(t)
	zero := make([]byte, 32)
	if err := os.WriteFile(updateauth.DeviceKeyPath(e.dataDir), zero, 0o600); err != nil {
		t.Fatal(err)
	}
	exp := time.Now().Unix() + 120
	w := e.do("POST", "/api/updates/install", rtBody(t, zero, updRelease, "zero-key-job-0001", exp))
	t.Logf("install with a MAC under the all-zero key: %d %s", w.Code, w.Body.String())
	if w.Code != http.StatusForbidden {
		t.Errorf("ALL-ZERO device.key accepted: a MAC anyone can compute passed (%d)", w.Code)
	}
}
