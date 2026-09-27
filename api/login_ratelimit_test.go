package api

// P2-11 (audit 0926-system): /login had no rate limit and its unknown-email
// path was a timing oracle (fast 401 vs bcrypt). Fix: per-IP AND per-account
// failure backoff (429 + Retry-After) and a dummy-bcrypt burn on the
// unknown-email path. Pins drive the PRODUCTION router (canon 53).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nofx/auth"
	"nofx/store"
)

func loginCall(t *testing.T, e *updEnv, email, pass string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"email": email, "password": pass})
	r := httptest.NewRequest("POST", "/api/login", strings.NewReader(string(b)))
	r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.s.router.ServeHTTP(w, r)
	return w
}

// resetLoginLimiter clears the package limiter between tests.
func resetLoginLimiter() {
	apiLoginLimiter.mu.Lock()
	apiLoginLimiter.m = make(map[string]*loginLimiterEntry)
	apiLoginLimiter.mu.Unlock()
}

func TestLoginBlocksAfterRepeatedFailures(t *testing.T) {
	e := newUpdEnv(t)
	resetLoginLimiter()

	// Five wrong-password attempts (the production bcrypt compare runs — the
	// known-email path). The block arms ON the fifth failure; the next request
	// is refused even with the CORRECT password.
	for i := 0; i < loginBlockAfterFails; i++ {
		if w := loginCall(t, e, updAdminEmail, "wrong-pass"); w.Code != http.StatusUnauthorized {
			t.Fatalf("fail %d: want 401, got %d (%s)", i+1, w.Code, w.Body.String())
		}
	}
	w := loginCall(t, e, updAdminEmail, updAdminPass)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("6th attempt with correct creds: want 429, got %d (%s)", w.Code, w.Body.String())
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("429 must carry Retry-After")
	}
}

func TestLoginBlockAppliesPerIPToo(t *testing.T) {
	e := newUpdEnv(t)
	resetLoginLimiter()

	// Fail from one account, then try a DIFFERENT valid account from the same
	// IP: the per-IP key blocks it.
	hashB := mustHash(t, "b-pass-123")
	now := time.Now().UTC()
	if err := e.st.User().Create(&store.User{ID: "user-b", Email: "b@example.com", PasswordHash: hashB, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("seed user-b: %v", err)
	}
	for i := 0; i < loginBlockAfterFails; i++ {
		if w := loginCall(t, e, updAdminEmail, "wrong-pass"); w.Code != http.StatusUnauthorized {
			t.Fatalf("fail %d: want 401, got %d", i+1, w.Code)
		}
	}
	if w := loginCall(t, e, "b@example.com", "b-pass-123"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("other account from same IP: want 429 (per-IP block), got %d", w.Code)
	}
}

func TestLoginSuccessResetsLimiter(t *testing.T) {
	e := newUpdEnv(t)
	resetLoginLimiter()

	// Three failures, then success. Without the reset the block would arm at
	// the 5th failure; with it, only the 5th failure AFTER the reset arms it.
	for i := 0; i < 3; i++ {
		if w := loginCall(t, e, updAdminEmail, "wrong-pass"); w.Code != http.StatusUnauthorized {
			t.Fatalf("fail %d: want 401, got %d", i+1, w.Code)
		}
	}
	if w := loginCall(t, e, updAdminEmail, updAdminPass); w.Code != http.StatusOK {
		t.Fatalf("correct creds after 3 fails: want 200, got %d (%s)", w.Code, w.Body.String())
	}
	for i := 0; i < loginBlockAfterFails; i++ {
		w := loginCall(t, e, updAdminEmail, "wrong-pass")
		if i < loginBlockAfterFails-1 && w.Code != http.StatusUnauthorized {
			t.Fatalf("fail %d post-reset: want 401, got %d", i+1, w.Code)
		}
	}
	if w := loginCall(t, e, updAdminEmail, updAdminPass); w.Code != http.StatusTooManyRequests {
		t.Fatalf("after 5 post-reset fails the correct creds must be blocked (429), got %d", w.Code)
	}
}

// countingPasswordCheck records (password, hash) pairs and returns false — a
// seam over auth.CheckPassword for the constant-time pins.
type countingPasswordCheck struct {
	calls [][2]string
}

func (c *countingPasswordCheck) Check(pass, hash string) bool {
	c.calls = append(c.calls, [2]string{pass, hash})
	return false
}

func TestLoginUnknownEmailBurnsDummyHash(t *testing.T) {
	e := newUpdEnv(t)
	resetLoginLimiter()

	counter := &countingPasswordCheck{}
	prev := loginCheckPassword
	loginCheckPassword = counter.Check
	t.Cleanup(func() { loginCheckPassword = prev })

	// Unknown email: the constant-time branch must run ONE password check
	// against the dummy hash.
	if w := loginCall(t, e, "nobody@example.com", "whatever-123"); w.Code != http.StatusUnauthorized {
		t.Fatalf("unknown email: want 401, got %d", w.Code)
	}
	if len(counter.calls) != 1 {
		t.Fatalf("unknown-email path must burn exactly one password check, got %d", len(counter.calls))
	}
	if counter.calls[0][1] != loginDummyHash {
		t.Fatal("unknown-email path did not check against the dummy hash — timing oracle reopened")
	}

	// Known email, wrong password: the check runs against the REAL stored hash.
	counter.calls = nil
	if w := loginCall(t, e, updAdminEmail, "wrong-pass"); w.Code != http.StatusUnauthorized {
		t.Fatalf("known email wrong pass: want 401, got %d", w.Code)
	}
	if len(counter.calls) != 1 || counter.calls[0][1] == loginDummyHash {
		t.Fatal("known-email path must check the real stored hash")
	}
}

func mustHash(t *testing.T, pass string) string {
	t.Helper()
	h, err := auth.HashPassword(pass)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
