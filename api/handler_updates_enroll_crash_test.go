package api

// W-ONE-BUTTON M3 red-team fold (red-4 #4): the updateauth.Enroll doc comment
// claimed that a crash between its two renames leaves "every MAC failing
// until enrollment is re-run". It does not: a replace writes device.key
// FIRST, so a replace that dies before admin.json is renamed leaves the NEW
// key beside the OLD admin.json — a working enrollment of the old admin.
// This is a comment-truth pin at the production router: it asserts exactly
// what the corrected comment says, so the comment cannot drift from the code
// again. The day the enrollment is bound to its key (CTO ruling (4): admin.json
// carries HMAC(device.key, password_hash)), assertion 3 goes RED on purpose:
// rewrite the Enroll comment and flip this test in the same commit.

import (
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"nofx/internal/updateauth"
)

func TestEnrollCommentTruthACrashBetweenTheTwoRenames(t *testing.T) {
	e := newUpdEnv(t)
	oldGrant := e.grant(updRelease) // minted under the pre-replace key
	// Enroll(replace)'s FIRST write, byte-for-byte its temp+fsync+chmod+rename
	// sequence; the process then dies — admin.json still names the incumbent.
	newKey := make([]byte, updateauth.DeviceKeyLen)
	for i := range newKey {
		newKey[i] = byte(i*7 + 3)
	}
	f, err := os.CreateTemp(updateauth.Dir(e.dataDir), ".updateauth-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(newKey); err != nil {
		t.Fatal(err)
	}
	_ = f.Sync()
	_ = f.Close()
	if err := os.Chmod(f.Name(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(f.Name(), updateauth.DeviceKeyPath(e.dataDir)); err != nil {
		t.Fatal(err)
	}

	// 1. the incumbent is still the enrolled administrator
	if a, err := updateauth.LoadAdmin(e.dataDir); err != nil || a.UserID != updAdminID {
		t.Fatalf("admin.json after the crash = %+v %v, want the incumbent", a, err)
	}
	// 2. every grant minted under the OLD key dies (the key rotated)
	if w := e.do("POST", "/api/updates/install", grantBody(oldGrant)); w.Code != http.StatusForbidden {
		t.Fatalf("a grant under the pre-replace key = %d, want 403", w.Code)
	}
	// 3. the corrected comment's claim: the pair is a WORKING enrollment of
	// the incumbent — authorize mints under the new key and the incumbent's
	// install is authorized (the stub's 422)
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("new key + old admin.json: incumbent's install = %d %s — the Enroll comment says it is a working enrollment; fix the comment with the code", w.Code, w.Body.String())
	}

	// A FIRST enroll that dies after the key rename: a lone key, nobody
	// enrolled, and a plain enroll refuses until --replace.
	d := t.TempDir()
	if err := os.MkdirAll(updateauth.Dir(d), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(updateauth.DeviceKeyPath(d), newKey, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := updateauth.LoadAdmin(d); !errors.Is(err, updateauth.ErrNotEnrolled) {
		t.Fatalf("lone key: LoadAdmin = %v, want ErrNotEnrolled", err)
	}
	if err := updateauth.Enroll(d, updAdminID, updAdminEmail, time.Now(), false); !errors.Is(err, updateauth.ErrAlreadyEnrolled) {
		t.Fatalf("lone key: plain Enroll = %v, want ErrAlreadyEnrolled (re-run with --replace)", err)
	}
}
