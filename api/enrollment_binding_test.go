package api

// M3 red-team H1/H2, the M3-local belt: the enrollment (admin.json) is bound
// to the admin row's password_hash by HMAC under device.key, so ANY password
// change — the owner's own, a bot's, a thief's with a retired token — leaves
// /api/updates* refusing until the owner re-enrolls on the box (attended,
// `updater-bootstrap enroll --replace`), exactly as a reset-account does.
// Driven at the production router (canon 53).

import (
	"net/http"
	"testing"
	"time"

	"nofx/internal/updateauth"
)

func TestPasswordChangeUnbindsTheEnrollment(t *testing.T) {
	e := newUpdEnv(t)
	e.expectAllAdmitted("before any password change")

	// The owner's own change through the web flow (login, PUT new_password).
	owner, code := credLogin(t, e, updAdminEmail, updAdminPass)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}
	if w := credCall(t, e, "PUT", "/api/user/password", owner, `{"new_password":"owner-rotated-pass-1"}`); w.Code != http.StatusOK {
		t.Fatalf("positive control: the owner's password change = %d %s", w.Code, w.Body.String())
	}
	// A token minted AFTER the change passes Q8 and names the enrolled
	// admin — but the enrollment was bound to the previous password.
	fresh := mintJWT(t, updAdminID, updAdminEmail, time.Now().Add(2*time.Second), time.Now().Add(time.Hour), updSecret)
	e.expectAllForbidden("post-change token under a pre-change enrollment", withToken(fresh))

	// The owner re-enrolls on the box (the CLI's own writer, --replace) with
	// the row's CURRENT hash: the same fresh token is admitted again.
	u, err := e.st.User().GetByID(updAdminID)
	if err != nil {
		t.Fatal(err)
	}
	if err := updateauth.Enroll(e.dataDir, updAdminID, updAdminEmail, u.PasswordHash, time.Now(), true); err != nil {
		t.Fatalf("re-enroll --replace: %v", err)
	}
	e.expectAllAdmitted("after re-enrolling with the new password", withToken(fresh))

	// And the binding is not the only thing that moved: a SECOND change (the
	// owner again, with the fresh session) un-enrolls again.
	if w := credCall(t, e, "PUT", "/api/user/password", fresh, `{"new_password":"owner-rotated-pass-2"}`); w.Code != http.StatusOK {
		t.Fatalf("second change = %d %s", w.Code, w.Body.String())
	}
	fresher := mintJWT(t, updAdminID, updAdminEmail, time.Now().Add(3*time.Second), time.Now().Add(time.Hour), updSecret)
	e.expectAllForbidden("after a second password change", withToken(fresher))
}
