package api

// PR #200 fold F2 (CTO 1790252194343, csrf-origin lens): the H1/H2 audit
// lines name the caller with c.ClientIP(). gin.Default() trusts
// X-Forwarded-For / X-Real-IP from EVERY peer (gin's defaultTrustedCIDRs are
// 0.0.0.0/0 and ::/0), so any client could write the address of its choice
// into the forensic log — forged attribution on exactly the lines an owner
// reads after an incident. NewServer now calls router.SetTrustedProxies(nil):
// ClientIP() is the socket peer (RemoteAddr); the loopback bind has no
// legitimate proxy. The /updates gate never read ClientIP (F4 judges
// RemoteAddr) and is unchanged.
//
// Driven at the PRODUCTION router (canon 53) with the production logger AND
// gin's own access log (gin.Default's Logger, bound at NewServer) captured:
// every audit line names the socket peer, and the forged address appears
// nowhere in either log.

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/logger"

	"github.com/gin-gonic/gin"
)

// captureLogs points the production logger and gin's access log at a buffer
// for the rest of the test. Call it BEFORE the server is built: gin.Default()
// binds its Logger's writer at construction.
func captureLogs(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	var mu sync.Mutex
	sink := writerFunc(func(p []byte) (int, error) { mu.Lock(); defer mu.Unlock(); return buf.Write(p) })
	prevGin := gin.DefaultWriter
	gin.DefaultWriter = sink
	logger.Log.SetOutput(sink)
	t.Cleanup(func() { gin.DefaultWriter = prevGin; logger.Log.SetOutput(os.Stdout) })
	return func() string { mu.Lock(); defer mu.Unlock(); return buf.String() }
}

const forgedClientIP = "203.0.113.77" // TEST-NET-3: never a real peer

// forgedCall is credCall from the same loopback socket peer, carrying every
// header gin reads a client address from, all naming forgedClientIP.
func forgedCall(t *testing.T, e *updEnv, method, path, tok, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
	r.Header.Set("X-Forwarded-For", forgedClientIP)
	r.Header.Set("X-Real-IP", forgedClientIP)
	if tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	e.s.router.ServeHTTP(w, r)
	return w
}

func TestForwardedForNeverRewritesTheLoggedCaller(t *testing.T) {
	logs := captureLogs(t)
	e := newUpdEnv(t)
	owner, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}

	// Each audit site, driven to its line through the production router.
	steps := []struct {
		site, method, path, tok, body string
		env                           string // ALLOW_ACCOUNT_RESET for this step
		code                          int
		line                          string // the audit line, naming the socket peer
	}{
		{"credential_guard.go credentialForbid", "PUT", "/api/user/password", mustBotToken(t),
			`{"current_password":"` + updAdminPass + `","new_password":"machine-chosen-pass-1"}`, "", http.StatusForbidden,
			"[credentials] refused PUT /api/user/password from 127.0.0.1: machine token"},
		{"server.go authMiddleware tokenRetirement", "GET", "/api/my-traders",
			mintNoIAT(t, updAdminID, updAdminEmail, time.Now().Add(time.Hour), updSecret), "", "", http.StatusUnauthorized,
			"[auth] refused GET /api/my-traders from 127.0.0.1: token carries no iat"},
		{"handler_user.go wrong current_password", "PUT", "/api/user/password", owner,
			`{"current_password":"not-the-password","new_password":"owner-new-password-7"}`, "", http.StatusForbidden,
			"[credentials] refused PUT /api/user/password from 127.0.0.1: current password is incorrect"},
		{"handler_user.go reset-password 410", "POST", "/api/reset-password", "",
			`{"email":"x@example.test","new_password":"whatever-pass-1"}`, "", http.StatusGone,
			"blocked POST /api/reset-password from 127.0.0.1"},
		{"handler_user.go reset-account env gate", "POST", "/api/reset-account", owner,
			`{"confirm":"RESET-ALL-DATA"}`, "", http.StatusForbidden,
			"blocked POST /api/reset-account from 127.0.0.1 (user " + updAdminID + ") — ALLOW_ACCOUNT_RESET is not enabled"},
		{"handler_user.go reset-account confirm", "POST", "/api/reset-account", owner,
			`{"confirm":"nope"}`, "1", http.StatusBadRequest,
			"blocked POST /api/reset-account from 127.0.0.1 (user " + updAdminID + ") — missing/incorrect confirm token"},
		// LAST: this one deletes every user row.
		{"handler_user.go reset-account authorized", "POST", "/api/reset-account", owner,
			`{"confirm":"RESET-ALL-DATA"}`, "1", http.StatusOK,
			"ACCOUNT RESET authorized by user " + updAdminID + " from 127.0.0.1"},
	}
	for _, s := range steps {
		t.Setenv("ALLOW_ACCOUNT_RESET", s.env)
		if w := forgedCall(t, e, s.method, s.path, s.tok, s.body); w.Code != s.code {
			t.Fatalf("%s: %s %s = %d %s — want %d (the harness did not reach the audit line)", s.site, s.method, s.path, w.Code, w.Body.String(), s.code)
		}
	}

	all := logs()
	for _, s := range steps {
		if !strings.Contains(all, s.line) {
			t.Errorf("%s: no audit line %q naming the socket peer", s.site, s.line)
		}
	}
	// gin's access log (gin.Default's Logger) prints ClientIP() too.
	if !strings.Contains(all, "[GIN]") {
		t.Fatal("positive control: gin's access log was not captured")
	}
	for _, line := range strings.Split(all, "\n") {
		if strings.Contains(line, forgedClientIP) {
			t.Errorf("a client-supplied X-Forwarded-For/X-Real-IP was logged as the caller: %q", strings.TrimSpace(line))
		}
	}
}
