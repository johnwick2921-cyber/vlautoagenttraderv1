package api

// W-ONE-BUTTON M3 red-team fold (red-3 #3): expiry is judged AFTER the
// seen-store lock is acquired. The handler's early check reads the clock
// before Consume waits on .seen.lock (a slow disk, a peer, the M4 worker);
// a request parked on the lock while its grant expired must be refused
// under the lock — never consumed and handed to the verifier on a stale
// reading — and consumed_at records the reading taken under the lock.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"nofx/internal/updateauth"
)

func holdSeenLock(t *testing.T, dataDir string) (release func()) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(updateauth.Dir(dataDir), ".seen.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }
}

func TestInstallJudgesExpiryUnderTheSeenStoreLock(t *testing.T) {
	e := newUpdEnv(t)
	clk := newUpdClock(e)
	key := mustKey(t, e.dataDir)
	T := clk.now().Unix()

	install := func(body string) (chan *httptest.ResponseRecorder, func()) {
		release := holdSeenLock(t, e.dataDir)
		done := make(chan *httptest.ResponseRecorder, 1)
		go func() { done <- e.do("POST", "/api/updates/install", body) }()
		select {
		case w := <-done:
			release()
			t.Fatalf("precondition: install returned %d while the seen lock was held", w.Code)
		case <-time.After(700 * time.Millisecond):
		}
		return done, release
	}
	wait := func(done chan *httptest.ResponseRecorder) *httptest.ResponseRecorder {
		select {
		case w := <-done:
			return w
		case <-time.After(20 * time.Second):
			t.Fatal("install never returned after the lock was released")
			return nil
		}
	}

	// the grant expires while the request is parked on the lock
	done, release := install(grantBodyUnder(t, key, updRelease, "lockwait-job-00001", T+2))
	clk.set(T + 3600)
	release()
	if w := wait(done); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("a grant that expired while its request waited on the seen lock = %d %s, want 403 %s", w.Code, w.Body.String(), forbiddenBody)
	}
	if b, _ := os.ReadFile(updateauth.SeenPath(e.dataDir)); strings.Contains(string(b), "lockwait-job-00001") {
		t.Errorf("the expired grant was consumed: %s", b)
	}

	// that refusal was of a genuine code: it raised the clock floor to the
	// reading under the lock (red-3 #2's stickiness holds on this path too)
	if b, _ := os.ReadFile(updateauth.SeenPath(e.dataDir)); !strings.Contains(string(b), `"clock_floor":`+strconv.FormatInt(T+3600, 10)) {
		t.Errorf("the under-lock expiry refusal did not raise clock_floor to %d: %s", T+3600, b)
	}

	// positive control: a grant still valid after the wait is consumed, and
	// consumed_at is the reading taken UNDER the lock
	T2 := T + 3601
	clk.set(T2)
	done, release = install(grantBodyUnder(t, key, updRelease, "lockwait-job-00002", T2+300))
	clk.set(T2 + 100)
	release()
	if w := wait(done); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: a grant valid after the wait = %d %s, want 422", w.Code, w.Body.String())
	}
	b, _ := os.ReadFile(updateauth.SeenPath(e.dataDir))
	if !strings.Contains(string(b), `"job_id":"lockwait-job-00002","expires_at":`+strconv.FormatInt(T2+300, 10)+`,"consumed_at":`+strconv.FormatInt(T2+100, 10)) {
		t.Errorf("consumed_at is not the reading taken under the lock (T2+100 = %d): %s", T2+100, b)
	}
}
