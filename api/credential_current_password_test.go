package api

// M3 red-team H1 — CTO ruling 1790231205208 item (1): PUT /api/user/password
// REQUIRES current_password, verified against the stored hash, on top of the
// email rule (the token's email must equal its row's). A bearer token alone —
// the bot's, a stolen session, an XSS'd page's — can no longer set the
// account's password. Driven at the PRODUCTION router (canon 53).

import (
	"net/http"
	"strings"
	"testing"
)

func TestPasswordChangeRequiresTheCurrentPassword(t *testing.T) {
	e := newUpdEnv(t)
	owner, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}
	before := e.adminRow()

	// The owner's OWN token with a WRONG current password: refused, row untouched.
	w := credCall(t, e, "PUT", "/api/user/password", owner, `{"current_password":"not-the-password","new_password":"owner-new-password-7"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("owner token with a WRONG current_password: PUT /api/user/password = %d %s — want 403", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "current password is incorrect") {
		t.Fatalf("wrong current_password body = %s — want the reason the UI shows", w.Body.String())
	}
	if e.adminRow() != before {
		t.Fatal("a wrong current_password still wrote the admin row")
	}
	if _, code := credLogin(t, e, updAdminEmail, "owner-new-password-7"); code != http.StatusUnauthorized {
		t.Fatalf("login with the refused new password = %d, want 401", code)
	}

	// No current password at all (the pre-ruling web body) and an empty one: 400.
	for _, body := range []string{
		`{"new_password":"owner-new-password-7"}`,
		`{"current_password":"","new_password":"owner-new-password-7"}`,
	} {
		w := credCall(t, e, "PUT", "/api/user/password", owner, body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("PUT /api/user/password %s = %d %s — want 400", body, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "current_password") {
			t.Fatalf("PUT /api/user/password %s body = %s — want it to name current_password", body, w.Body.String())
		}
	}
	if e.adminRow() != before {
		t.Fatal("a body without current_password still wrote the admin row")
	}

	// The right current password: 200, the new password logs in, the old does not.
	w = credCall(t, e, "PUT", "/api/user/password", owner, `{"current_password":"`+updAdminPass+`","new_password":"owner-new-password-7"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("owner token with the RIGHT current_password = %d %s — want 200", w.Code, w.Body.String())
	}
	if _, code := credLogin(t, e, updAdminEmail, "owner-new-password-7"); code != http.StatusOK {
		t.Fatalf("login with the new password = %d, want 200", code)
	}
	if _, code := credLogin(t, e, updAdminEmail, updAdminPass); code != http.StatusUnauthorized {
		t.Fatalf("login with the old password = %d, want 401", code)
	}
}
