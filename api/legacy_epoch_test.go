package api

// M3 FOLD-M3-B (CTO 1790239512054) — the LEGACY / phantom epoch, documented
// behaviour rather than a surprise. auth.CredentialEpoch reads ANY users row
// whose updated_at differs from created_at as a credential change: it cannot
// tell a password change from a value some older writer left there (the
// Postgres initTables ALTER … updated_at DEFAULT CURRENT_TIMESTAMP gives a
// row that predates the column the migration instant; a pre-M3 binary's
// writes). So such a row retires every token issued at or before its
// updated_at, on every protected route, although the password never changed
// — a one-time sign-in with the SAME password clears it. Driven at the
// production router (canon 53).

import (
	"net/http"
	"testing"
	"time"
)

func TestLegacyUpdatedAtIsAnEpochWithoutAPasswordChange(t *testing.T) {
	e := newUpdEnv(t) // admin row created an hour ago, updated_at == created_at
	before := e.adminRow()
	row, err := e.st.User().GetByID(updAdminID)
	if err != nil {
		t.Fatal(err)
	}
	created := row.CreatedAt

	// Tokens issued between created_at and the legacy value: fine today.
	legacy := time.Now().Add(-10 * time.Minute).Truncate(time.Second).Add(137 * time.Millisecond).UTC()
	older := mintJWT(t, updAdminID, updAdminEmail, legacy.Add(-5*time.Minute), time.Now().Add(time.Hour), updSecret)
	sameSecond := mintJWT(t, updAdminID, updAdminEmail, legacy.Add(-100*time.Millisecond), time.Now().Add(time.Hour), updSecret)
	for name, tok := range map[string]string{"older": older, "same-second": sameSecond} {
		if w := credCall(t, e, "GET", "/api/my-traders", tok, ""); w.Code != http.StatusOK {
			t.Fatalf("control: the %s token before any updated_at move = %d %s", name, w.Code, w.Body.String())
		}
	}

	// The legacy shape: updated_at later than created_at, NO password change
	// (the hash is untouched; nothing called UpdatePassword).
	if err := e.st.GormDB().Exec(`UPDATE users SET updated_at = ? WHERE id = ?`, legacy, updAdminID).Error; err != nil {
		t.Fatal(err)
	}
	after := e.adminRow()
	if after.hash != before.hash {
		t.Fatal("the fixture changed the password hash — this pin is about a row with NO password change")
	}
	if !after.updated.After(created) {
		t.Fatalf("fixture: updated_at %v is not after created_at %v", after.updated, created)
	}

	// Documented: a token issued before (or in the second of) that updated_at
	// is refused on every protected route — 401, the web UI signs out.
	for name, tok := range map[string]string{"issued 5 min before updated_at": older, "issued in updated_at's second": sameSecond} {
		for _, p := range []string{"/api/my-traders", "/api/config/resolved", "/api/telegram"} {
			if w := credCall(t, e, "GET", p, tok, ""); w.Code != http.StatusUnauthorized {
				t.Fatalf("legacy epoch, token %s: GET %s = %d %s — want 401 (a row with updated_at > created_at retires older tokens even with NO password change)", name, p, w.Code, w.Body.String())
			}
		}
	}
	// A token issued after it is admitted, and the ONE re-login uses the SAME
	// password (nothing changed it).
	fresh := mintJWT(t, updAdminID, updAdminEmail, legacy.Add(time.Second), time.Now().Add(time.Hour), updSecret)
	if w := credCall(t, e, "GET", "/api/my-traders", fresh, ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: a token issued the second after updated_at = %d %s", w.Code, w.Body.String())
	}
	relog, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("the one re-login with the unchanged password = %d", code)
	}
	if w := credCall(t, e, "GET", "/api/my-traders", relog, ""); w.Code != http.StatusOK {
		t.Fatalf("the re-login token on GET /api/my-traders = %d %s", w.Code, w.Body.String())
	}
}
