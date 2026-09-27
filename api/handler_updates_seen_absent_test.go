package api

// W-ONE-BUTTON M3 red-team fold (red-3 #4, L7 absent ≠ []): once an
// installation is enrolled, a MISSING seen-job store is not an empty one —
// moving it aside, deleting it or restoring data/updater from an older copy
// would otherwise re-open every consumed job id inside its window. Enroll
// creates the store; absent afterwards is ErrSeenCorrupt → the uniform 403,
// and the refusal never re-creates it.

import (
	"net/http"
	"os"
	"testing"

	"nofx/internal/updateauth"
)

func TestInstallRefusesWhenTheSeenStoreIsMissingAfterEnrollment(t *testing.T) {
	e := newUpdEnv(t)
	p := updateauth.SeenPath(e.dataDir)
	// Enroll created the store: present, private, before any install
	fi, err := os.Lstat(p)
	if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o600 {
		t.Fatalf("seen store right after enroll: %v %v, want a 0600 regular file", fi, err)
	}
	g := e.grant(updRelease)
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control (fresh enroll): first use = %d %s, want 422", w.Code, w.Body.String())
	}
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusConflict {
		t.Fatalf("positive control: replay = %d, want 409", w.Code)
	}
	fresh := e.grant(updRelease) // minted while the store is present (the minter refuses without one)
	if err := os.Rename(p, p+".aside"); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"the consumed grant": grantBody(g), "a fresh grant": grantBody(fresh)} {
		if w := e.do("POST", "/api/updates/install", body); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
			t.Errorf("seen store missing after enrollment, %s = %d %s, want 403 %s", name, w.Code, w.Body.String(), forbiddenBody)
		}
	}
	if _, err := os.Lstat(p); err == nil {
		t.Fatal("a refused install re-created the missing seen store (a silent reset)")
	}
	// restored: the replay is a replay again, a fresh grant is authorized
	if err := os.Rename(p+".aside", p); err != nil {
		t.Fatal(err)
	}
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusConflict {
		t.Fatalf("restored store: replay = %d, want 409", w.Code)
	}
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("restored store: fresh grant = %d, want 422", w.Code)
	}
}
