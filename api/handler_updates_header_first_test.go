package api

// PR #200 fold F8 (CTO 1790252194343, review item #16): the update header
// (X-NOFX-Update: 1 — the CSRF factor) is judged BEFORE the JWT is parsed.
// A request without it is refused on transport grounds with no
// token-derived work: no blacklist verdict, no signature verdict, no
// enrollment read, no users-row read — so a cross-site page that cannot set
// the header learns nothing about any token it rides on. The earlier pins
// only removed the header from a VALID token, which cannot tell "header
// first" from "header after the token checks". Here each token below is one
// the gate WOULD refuse for a token reason (shown by the positive controls,
// which add the header back); without the header the logged category must
// still be the header one. A users-table read counter (a gorm Query
// callback on the test store) shows no users row is read either.
//
// Driven at the PRODUCTION router (canon 53) over all 5 /updates routes.

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"nofx/logger"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func countUsersReads(t *testing.T, e *updEnv) *atomic.Int64 {
	t.Helper()
	var n atomic.Int64
	db := e.st.GormDB()
	const name = "f8:count_users_reads"
	if err := db.Callback().Query().After("gorm:query").Register(name, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "users" {
			n.Add(1)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(name) })
	return &n
}

func TestUpdatesHeaderIsJudgedBeforeTheToken(t *testing.T) {
	logs := captureLogs(t)
	// CTO fold 1790280466263: a (route, category) WARNs once per process and
	// every repeat logs at DEBUG with the same "refused … : <category>" text.
	// This pin reads the category of EVERY request, so it reads at DEBUG.
	prevLevel := logger.Log.GetLevel()
	logger.Log.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() { logger.Log.SetLevel(prevLevel) })
	e := newUpdEnv(t)
	reads := countUsersReads(t, e)
	now := time.Now()

	// A distinctive exp keeps this string unique in the process-wide blacklist.
	revoked := mintJWT(t, updAdminID, updAdminEmail, now.Add(-6*time.Second), now.Add(time.Hour+53*time.Second), updSecret)
	if w := credCall(t, e, "POST", "/api/logout", revoked, ""); w.Code != http.StatusOK {
		t.Fatalf("logout = %d %s", w.Code, w.Body.String())
	}
	other := mintJWT(t, updOtherID, updOtherEmail, now.Add(-5*time.Second), now.Add(time.Hour), updSecret)

	const headerWhy = "update header missing or wrong"
	tokenCategories := []string{"authorization missing", "authorization malformed", "token revoked", "token invalid",
		"machine token", "not enrolled", "enrollment unreadable", "device key unreadable", "not the enrolled admin",
		"admin user row absent", "password changed since enrollment", "token older than the user row", "no user store"}
	cases := []struct {
		name, tok, tokenWhy string // tokenWhy: the category WITH the header ("" = admitted)
	}{
		{"valid enrolled-admin token", e.tok, ""},
		{"logged-out (blacklisted) admin token", revoked, "token revoked"},
		{"unsigned garbage", "not.a.jwt", "token invalid"},
		{"Telegram bot token (machine)", mustBotToken(t), "machine token"},
		{"another user's token", other, "not the enrolled admin"},
	}
	noHeader := func(r *http.Request) { r.Header.Del(UpdateHeader) }

	for _, tc := range cases {
		for _, rt := range e.allRoutes() {
			// Positive control: WITH the header this token reaches its own
			// verdict (a token category, or the handler) — so the header
			// refusal below is not a token refusal in disguise.
			mark := len(logs())
			w := e.do(rt.method, rt.path, rt.body, withToken(tc.tok))
			tail := logs()[mark:]
			if tc.tokenWhy == "" {
				if w.Code != admittedStatus[rt.path] {
					t.Fatalf("positive control %s: %s %s = %d %s, want %d", tc.name, rt.method, rt.path, w.Code, w.Body.String(), admittedStatus[rt.path])
				}
			} else if w.Code != http.StatusForbidden || !strings.Contains(tail, ": "+tc.tokenWhy) {
				t.Fatalf("positive control %s: %s %s = %d, log %q — want 403 %q", tc.name, rt.method, rt.path, w.Code, tail, tc.tokenWhy)
			}

			// Without the header: the header category, nothing token-derived.
			mark = len(logs())
			before := reads.Load()
			w = e.do(rt.method, rt.path, rt.body, withToken(tc.tok), noHeader)
			tail = logs()[mark:]
			if w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
				t.Fatalf("%s, no header: %s %s = %d %s — want 403 %s", tc.name, rt.method, rt.path, w.Code, w.Body.String(), forbiddenBody)
			}
			if !strings.Contains(tail, "[updates] refused") || !strings.Contains(tail, ": "+headerWhy) {
				t.Fatalf("%s, no header: %s %s logged %q — want the category %q", tc.name, rt.method, rt.path, tail, headerWhy)
			}
			for _, tw := range tokenCategories {
				if strings.Contains(tail, ": "+tw) {
					t.Fatalf("%s, no header: %s %s logged the TOKEN category %q — the token was judged before the header:\n%s", tc.name, rt.method, rt.path, tw, tail)
				}
			}
			if d := reads.Load() - before; d != 0 {
				t.Fatalf("%s, no header: %s %s read the users table %d time(s) — a header-less request must do no token-derived work", tc.name, rt.method, rt.path, d)
			}
		}
	}
	// The counter counts: an admitted request DOES read the users row.
	before := reads.Load()
	if w := e.do("GET", "/api/updates", ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: admin GET /api/updates = %d", w.Code)
	}
	if reads.Load() == before {
		t.Fatal("positive control: the users-read counter saw no read on an admitted request — it would pass vacuously")
	}
}
