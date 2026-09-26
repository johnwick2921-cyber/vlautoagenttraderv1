package api

// PR #200 fold F5 (CTO 1790252194343, token-retire lens): the 410 body of
// POST /api/reset-password told a locked-out owner to reset the password
// "directly in the database". A hash-only UPDATE does NOT move the
// credential epoch (users.updated_at — auth.CredentialEpoch), so every
// session signed in before the reset — the very sessions a locked-out owner
// may be resetting AGAINST — stayed valid. The advice now carries the whole
// statement, updated_at=CURRENT_TIMESTAMP included, and says why.
//
// Two pins at the PRODUCTION router (canon 53):
//   - the body, byte for byte (a change to the advice is deliberate), with no
//     real email or hash in it;
//   - the advice EXECUTED: the SQL is cut out of the served body, its two
//     placeholders bound, and run against a temp SQLite store opened by the
//     production store.New. SQLite's CURRENT_TIMESTAMP writes second-precision
//     UTC text; the store must read it back as that instant (UTC, not local
//     time), CredentialEpoch must see it as an epoch after created_at, and a
//     token issued before it must be refused on the protected routes. The
//     hash-only UPDATE — the old advice — is run first as the WHY: it leaves
//     the pre-reset token admitted.

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"nofx/auth"
)

const resetPasswordGoneBody = `{"error":"Password reset by email is disabled. Sign in and use PUT /api/user/password. ` +
	`Locked out? Set the new hash AND the credential epoch in ONE statement: ` +
	`UPDATE users SET password_hash='NEW_BCRYPT_HASH', updated_at=CURRENT_TIMESTAMP WHERE email='YOUR_ACCOUNT_EMAIL'; ` +
	`— moving updated_at is what signs out every session issued before the reset; a hash-only UPDATE leaves those sessions valid."}`

func resetPasswordAdvice(t *testing.T, e *updEnv) string {
	t.Helper()
	w := credCall(t, e, "POST", "/api/reset-password", "", `{"email":"x@example.test","new_password":"whatever-pass-1"}`)
	if w.Code != http.StatusGone {
		t.Fatalf("POST /api/reset-password = %d %s — want 410", w.Code, w.Body.String())
	}
	return w.Body.String()
}

func TestResetPasswordGoneBodyCarriesTheEpochMovingStatement(t *testing.T) {
	e := newUpdEnv(t)
	body := resetPasswordAdvice(t, e)
	// gin's c.JSON writes <, > and & as JSON unicode escapes; compare the decoded text.
	var got, want map[string]string
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("410 body is not JSON: %s", body)
	}
	if err := json.Unmarshal([]byte(resetPasswordGoneBody), &want); err != nil {
		t.Fatal(err)
	}
	if got["error"] != want["error"] || len(got) != 1 {
		t.Fatalf("410 body:\n got %q\nwant %q", got["error"], want["error"])
	}
	// fapi verify note 3: the RAW bytes are what an owner reading curl sees.
	// gin's c.JSON escapes <, > and & (\u003c …), so a bracketed placeholder
	// arrives as "\u003cbcrypt hash …\u003e" and a literal fill-in stores a
	// broken hash. The statement must read the same raw and decoded.
	for _, esc := range []string{`\u003c`, `\u003e`, `\u0026`} {
		if strings.Contains(body, esc) {
			t.Fatalf("the raw 410 body carries %s — an owner reading it with curl sees an escaped placeholder; use placeholders gin does not escape:\n%s", esc, body)
		}
	}
	for _, leak := range []string{"@", "$2a$", "$2b$", "$2y$", updAdminEmail} {
		if strings.Contains(got["error"], leak) {
			t.Fatalf("the 410 body carries %q — it must hold placeholders only, never a real email or hash", leak)
		}
	}
}

var adviceSQL = regexp.MustCompile(`UPDATE users SET [^;]+;`)

func TestResetPasswordAdviceRetiresPreResetSessionsOnSQLite(t *testing.T) {
	e := newUpdEnv(t) // admin row created an hour ago, updated_at == created_at
	now := time.Now()
	preReset := mintJWT(t, updAdminID, updAdminEmail, now.Add(-5*time.Second), now.Add(time.Hour), updSecret)
	if w := credCall(t, e, "GET", "/api/my-traders", preReset, ""); w.Code != http.StatusOK {
		t.Fatalf("control: the pre-reset session = %d %s", w.Code, w.Body.String())
	}

	// WHY: the OLD advice (hash only) moves no epoch — the pre-reset session
	// survives a reset done that way.
	oldWay, err := auth.HashPassword("owner-reset-old-way-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.st.GormDB().Exec(`UPDATE users SET password_hash = ? WHERE email = ?`, oldWay, updAdminEmail).Error; err != nil {
		t.Fatal(err)
	}
	if w := credCall(t, e, "GET", "/api/my-traders", preReset, ""); w.Code != http.StatusOK {
		t.Fatalf("hash-only UPDATE: the pre-reset session = %d — this pin documents that it SURVIVES (why the advice moves updated_at)", w.Code)
	}

	// The advice, cut from the served 410 body, placeholders bound.
	var gone map[string]string
	if err := json.Unmarshal([]byte(resetPasswordAdvice(t, e)), &gone); err != nil {
		t.Fatal(err)
	}
	stmt := adviceSQL.FindString(gone["error"])
	if stmt == "" {
		t.Fatalf("the 410 body carries no UPDATE statement: %q", gone["error"])
	}
	const newPass = "owner-reset-pass-1"
	newHash, err := auth.HashPassword(newPass)
	if err != nil {
		t.Fatal(err)
	}
	for ph, v := range map[string]string{"NEW_BCRYPT_HASH": newHash, "YOUR_ACCOUNT_EMAIL": updAdminEmail} {
		if !strings.Contains(stmt, ph) {
			t.Fatalf("the advice lost its placeholder %q: %s", ph, stmt)
		}
		stmt = strings.Replace(stmt, ph, v, 1)
	}
	if err := e.st.GormDB().Exec(stmt).Error; err != nil {
		t.Fatalf("the advice does not run on SQLite: %v", err)
	}

	// What CURRENT_TIMESTAMP wrote, and how the store reads it back.
	// (an expression has no declared type, so the driver hands back the
	// stored text instead of parsing it as a DATETIME column)
	var raw string
	if err := e.st.GormDB().Raw(`SELECT typeof(updated_at) || '|' || updated_at FROM users WHERE id = ?`, updAdminID).Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^text\|\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`).MatchString(raw) {
		t.Fatalf("SQLite CURRENT_TIMESTAMP stored %q — want second-precision 'YYYY-MM-DD HH:MM:SS' UTC text", raw)
	}
	u, err := e.st.User().GetByID(updAdminID)
	if err != nil {
		t.Fatal(err)
	}
	if u.PasswordHash != newHash {
		t.Fatal("the advice did not set the hash")
	}
	if u.UpdatedAt.IsZero() || u.UpdatedAt.Nanosecond() != 0 {
		t.Fatalf("store read updated_at %v — want a non-zero, whole-second instant", u.UpdatedAt)
	}
	if d := time.Now().UTC().Sub(u.UpdatedAt); d < -2*time.Second || d > 5*time.Second {
		t.Fatalf("store read updated_at %v, %v from now — CURRENT_TIMESTAMP (UTC text) was not read back as UTC", u.UpdatedAt, d)
	}
	ep := auth.CredentialEpoch(u.CreatedAt, u.UpdatedAt)
	if ep.IsZero() || !ep.After(u.CreatedAt) {
		t.Fatalf("CredentialEpoch(created %v, updated %v) = %v — want an epoch after created_at", u.CreatedAt, u.UpdatedAt, ep)
	}

	// The pre-reset session is refused on the protected routes.
	for _, p := range []struct{ method, path, body string }{
		{"GET", "/api/my-traders", ""},
		{"GET", "/api/config/resolved", ""},
		{"PUT", "/api/user/password", `{"current_password":"` + newPass + `","new_password":"pre-reset-session-9"}`},
	} {
		if w := credCall(t, e, p.method, p.path, preReset, p.body); w.Code != http.StatusUnauthorized {
			t.Fatalf("after the advised UPDATE, the pre-reset session on %s %s = %d %s — want 401", p.method, p.path, w.Code, w.Body.String())
		}
	}
	// Positive controls: the new password signs in; a session issued after
	// the epoch's second is admitted.
	if _, code := credLogin(t, e, updAdminEmail, newPass); code != http.StatusOK {
		t.Fatalf("login with the reset password = %d, want 200", code)
	}
	fresh := mintJWT(t, updAdminID, updAdminEmail, u.UpdatedAt.Add(time.Second), time.Now().Add(time.Hour), updSecret)
	if w := credCall(t, e, "GET", "/api/my-traders", fresh, ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: a session issued after the epoch = %d %s", w.Code, w.Body.String())
	}
}
