package api

// Verifier gap (M3 ha, defect 2): the /updates Q8 gate is documented as
// STRICTER than the shared H2 predicate every other surface uses
// (auth.RetiredBy — authMiddleware, the credential guard, the bot). Two
// cases separate them, and nothing pinned either, so a refactor that
// "unifies" Q8 onto the shared helper stayed green:
//
//   - a row with NO updated_at (NULL or zero — a legacy row): RetiredBy finds
//     no epoch and admits every token; Q8 refuses every token (it cannot
//     tell how old the row's last change is).
//   - a row that never changed (updated_at == created_at): RetiredBy finds no
//     epoch, so a token from the row's creation second — or older — is
//     admitted everywhere else; Q8 still reads updated_at as the epoch and
//     refuses a token issued at or before it (same second included).
//
// Each case beside its contrast on the same router: GET /api/my-traders
// admits the same token (the shared rule), and a token issued the second
// after the row is admitted on /updates (so the refusal is Q8's, not a
// broken fixture).

import (
	"net/http"
	"testing"
	"time"
)

func TestUpdatesQ8StaysStricterThanTheSharedRetireRule(t *testing.T) {
	e := newUpdEnv(t) // enrolled admin row: created_at == updated_at, an hour ago
	row, err := e.st.User().GetByID(updAdminID)
	if err != nil {
		t.Fatal(err)
	}
	if !row.UpdatedAt.Equal(row.CreatedAt) {
		t.Fatalf("fixture: the admin row must never have changed (created %v, updated %v)", row.CreatedAt, row.UpdatedAt)
	}
	sec := row.UpdatedAt.Truncate(time.Second)
	tok := func(iat time.Time) string {
		return mintJWT(t, updAdminID, updAdminEmail, iat, time.Now().Add(time.Hour), updSecret)
	}
	sharedAdmits := func(label, tk string) {
		t.Helper()
		if w := credCall(t, e, "GET", "/api/my-traders", tk, ""); w.Code != http.StatusOK {
			t.Fatalf("contrast (%s): GET /api/my-traders = %d %s — the shared rule admits this token", label, w.Code, w.Body.String())
		}
	}

	// A never-changed row: a token from its creation second, and one older.
	for label, tk := range map[string]string{
		"iat in the row's creation second (the registration token's shape)": tok(sec.Add(300 * time.Millisecond)),
		"iat a minute before the row's updated_at":                          tok(sec.Add(-time.Minute)),
	} {
		e.expectAllForbidden("never-changed row, "+label, withToken(tk))
		sharedAdmits("never-changed row, "+label, tk)
	}
	e.expectAllAdmitted("never-changed row, iat the second after updated_at", withToken(tok(sec.Add(time.Second))))

	// A row with no updated_at: NULL and zero both read as zero.
	for _, v := range []any{nil, time.Time{}} {
		if err := e.st.GormDB().Exec(`UPDATE users SET updated_at = ? WHERE id = ?`, v, updAdminID).Error; err != nil {
			t.Fatal(err)
		}
		if u, err := e.st.User().GetByID(updAdminID); err != nil || !u.UpdatedAt.IsZero() {
			t.Fatalf("fixture: updated_at = %v did not read back as zero (err %v)", v, err)
		}
		e.expectAllForbidden("row with no updated_at", withToken(e.tok))
		sharedAdmits("row with no updated_at", e.tok)
	}
}
