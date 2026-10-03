package api

// Owner order 10-02 07:3x CT — POST /api/updates/install-with-password, the
// server half. Driven at the PRODUCTION router (canon 53): the SAME updates
// gate admits the caller (enrolled admin session only), then the PASSWORD
// proof gates the server-side grant mint; the mint hands to runInstall — the
// one install path both flows share.

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"vl/internal/updateauth"
)

// installPasswordBody builds the request body.
func installPasswordBody(release, password string) string {
	b, _ := json.Marshal(map[string]string{"release_id": release, "password": password})
	return string(b)
}

// fakeVerifier is the package fixture (admits every release) — see
// handler_updates_test.go.

func TestInstallWithPasswordRightPasswordStartsTheJob(t *testing.T) {
	e := newUpdEnv(t)
	e.s.SetUpdateVerifier(fakeVerifier{})
	type started struct {
		g updateauth.Grant
		m updateauth.Manifest
	}
	var got []started
	e.s.SetUpdateStarter(func(g updateauth.Grant, m updateauth.Manifest) error {
		got = append(got, started{g, m})
		return nil
	})

	w := e.do("POST", "/api/updates/install-with-password", installPasswordBody(updRelease, updAdminPass))
	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), "job_id") {
		t.Fatalf("right password = %d %s — want 202 with job_id", w.Code, w.Body.String())
	}
	if len(got) != 1 || got[0].m.ReleaseID != updRelease {
		t.Fatalf("starter calls = %+v — want exactly one for %s", got, updRelease)
	}
	// The server-side mint must verify under the SAME path the terminal
	// authorize produces: the device key + the enrolled admin id.
	key, err := updateauth.LoadDeviceKey(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if !updateauth.VerifyMAC(key, updAdminID, got[0].g.ReleaseID, got[0].g.JobID, got[0].g.ExpiresAt, got[0].g.HMAC) {
		t.Fatal("the server-minted grant does not verify against the device key + enrolled admin")
	}
}

func TestInstallWithPasswordWrongPasswordIsDelayedCounted403(t *testing.T) {
	// Production delay values, like the F9 pin.
	if currentPasswordFailDelay != time.Second {
		t.Fatalf("currentPasswordFailDelay = %v — want 1s", currentPasswordFailDelay)
	}
	if reflect.ValueOf(prodCurrentPasswordFailSleep).Pointer() != reflect.ValueOf(time.Sleep).Pointer() {
		t.Fatal("the production delay seam is not time.Sleep")
	}
	e := newUpdEnv(t)
	var calls []time.Duration
	prev := currentPasswordFailSleep
	currentPasswordFailSleep = func(d time.Duration) { calls = append(calls, d) }
	t.Cleanup(func() { currentPasswordFailSleep = prev })

	owner, _ := credLogin(t, e, updAdminEmail, updAdminPass)
	before := wrongPasswordCount(t, e, owner)
	// wrongPasswordCount reads currentPasswordWrongGate; this handler bumps its
	// own gate — read it through the same production route.
	readGate := func(tok, gate string) int {
		t.Helper()
		w := credCall(t, e, "GET", "/api/risk/gate-blocks", tok, "")
		var out struct {
			ByTrader map[string]map[string]int `json:"by_trader"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out.ByTrader[""][gate]
	}
	beforeInstall := readGate(owner, installPasswordWrongGate)

	w := e.do("POST", "/api/updates/install-with-password", installPasswordBody(updRelease, "wrong-password"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("wrong password = %d %s — want 403", w.Code, w.Body.String())
	}
	if len(calls) != 1 || calls[0] != time.Second {
		t.Fatalf("delay seam calls = %v — want exactly one 1s delay", calls)
	}
	if got := readGate(owner, installPasswordWrongGate); got != beforeInstall+1 {
		t.Fatalf("gate-block %s = %d — want %d", installPasswordWrongGate, got, beforeInstall+1)
	}
	_ = before
}

func TestInstallWithPasswordLocksOutAfterFiveWrong(t *testing.T) {
	e := newUpdEnv(t)
	for i := 0; i < 4; i++ {
		if w := e.do("POST", "/api/updates/install-with-password", installPasswordBody(updRelease, "wrong")); w.Code != http.StatusForbidden {
			t.Fatalf("wrong #%d = %d — want 403", i+1, w.Code)
		}
	}
	w := e.do("POST", "/api/updates/install-with-password", installPasswordBody(updRelease, "wrong"))
	if w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), "unlock_in_seconds") {
		t.Fatalf("5th wrong = %d %s — want 429 with unlock_in_seconds", w.Code, w.Body.String())
	}
	// Locked out: even the RIGHT password is refused until the window passes.
	if w := e.do("POST", "/api/updates/install-with-password", installPasswordBody(updRelease, updAdminPass)); w.Code != http.StatusTooManyRequests {
		t.Fatalf("right password while locked out = %d — want 429", w.Code)
	}
}

func TestInstallWithPasswordRefusesMachineAndNonAdminTokens(t *testing.T) {
	e := newUpdEnv(t)
	// Machine token: refused by the gate before the handler (uniform 403).
	machineClaims := ownerClaims(updAdminEmail)
	machineClaims["scope"] = "gate-jwt"
	machineTok := mintMapJWT(t, machineClaims)
	w := e.do("POST", "/api/updates/install-with-password", installPasswordBody(updRelease, updAdminPass),
		func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+machineTok) })
	if w.Code != http.StatusForbidden {
		t.Fatalf("machine token = %d — want 403", w.Code)
	}
	// A real user who is not the enrolled admin: 403 from the gate.
	other := mintJWT(t, "someone-else", "other@example.test", time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)
	w = e.do("POST", "/api/updates/install-with-password", installPasswordBody(updRelease, updAdminPass),
		func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+other) })
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-admin session = %d — want 403", w.Code)
	}
}

func TestInstallWithPasswordUnverifiedReleaseIs422(t *testing.T) {
	e := newUpdEnv(t) // default StubVerifier refuses every release
	w := e.do("POST", "/api/updates/install-with-password", installPasswordBody(updRelease, updAdminPass))
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "release not verified") {
		t.Fatalf("unverified release = %d %s — want 422 release not verified", w.Code, w.Body.String())
	}
}

func TestInstallWithPasswordBadBodyAndBadReleaseID(t *testing.T) {
	e := newUpdEnv(t)
	if w := e.do("POST", "/api/updates/install-with-password", `{"release_id":"x"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("missing password = %d — want 400", w.Code)
	}
	if w := e.do("POST", "/api/updates/install-with-password", installPasswordBody("../etc", updAdminPass)); w.Code != http.StatusBadRequest {
		t.Fatalf("bad release id = %d — want 400", w.Code)
	}
}

// The paste flow stays byte-identical: /api/updates/install still takes the
// full grant and reaches the same hand-off (runInstall is the ONE path).
func TestPasteInstallStillWorksAfterTheRefactor(t *testing.T) {
	e := newUpdEnv(t)
	e.s.SetUpdateVerifier(fakeVerifier{})
	var n int
	e.s.SetUpdateStarter(func(g updateauth.Grant, m updateauth.Manifest) error { n++; return nil })
	if w := e.do("POST", "/api/updates/install", grantBody(e.grant(updRelease))); w.Code != http.StatusAccepted {
		t.Fatalf("paste install = %d %s — want 202", w.Code, w.Body.String())
	}
	if n != 1 {
		t.Fatalf("paste install started %d jobs — want 1", n)
	}
}
