package api

// W-ONE-BUTTON M3 red-team fold (red-4 #4, second half): the install MAC is
// bound to the enrolled administrator's identity — the message carries the
// admin's user_id — so a code minted while one administrator was enrolled
// never authorizes an install for another, even under the same device key
// (admin.json restored or swapped beside a key it was not enrolled with).
// Fail-closed only: it adds a refusal, never an admission.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"nofx/internal/updateauth"
	"nofx/store"
)

func TestInstallMACIsBoundToTheEnrolledAdminIdentity(t *testing.T) {
	e := newUpdEnv(t)
	gA := e.grant(updRelease) // minted while A is the enrolled administrator
	past := time.Now().Add(-time.Hour).UTC()
	if err := e.st.User().Create(&store.User{ID: updOtherID, Email: updOtherEmail, PasswordHash: "x", CreatedAt: past, UpdatedAt: past}); err != nil {
		t.Fatal(err)
	}
	// admin.json now names B; device.key is unchanged (no enroll ran).
	// The H1/H2 belt recomputes the binding at the gate, so the fixture
	// computes the SAME HMAC Enroll would have minted: hex-HMAC-SHA256 of
	// "nofx-updater/password-binding/v1\x00" + user_id + "\x00" + hash
	// under device.key (the production call site PasswordStillBound compares
	// against it).
	key, err := updateauth.LoadDeviceKey(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	m := hmac.New(sha256.New, key)
	m.Write([]byte("nofx-updater/password-binding/v1\x00" + updOtherID + "\x00" + "x"))
	b, _ := json.Marshal(map[string]string{"user_id": updOtherID, "email": updOtherEmail, "enrolled_at": past.Format(time.RFC3339), "password_binding": hex.EncodeToString(m.Sum(nil))})
	if err := os.WriteFile(updateauth.AdminPath(e.dataDir), b, 0o600); err != nil {
		t.Fatal(err)
	}
	tokB := mintJWT(t, updOtherID, updOtherEmail, time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)
	if w := e.do("POST", "/api/updates/install", grantBody(gA), withToken(tokB)); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("a code minted for administrator A, presented by administrator B under the same key = %d %s, want 403 %s", w.Code, w.Body.String(), forbiddenBody)
	}
	// positive control: B's own code (minted with B enrolled) is authorized
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease)), withToken(tokB)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: B's own code = %d %s, want 422", w.Code, w.Body.String())
	}
}
