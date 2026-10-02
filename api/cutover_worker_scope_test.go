package api

// P-E E2 — a cutover-worker token is admitted ONLY on its two read-only
// routes, deny-by-default everywhere else. CENSUS over the PRODUCTION router
// (canon 53): every registered route is probed with a cutover-worker token —
// force-flat included — and must answer 403 except GET /api/maintenance and
// GET /api/installation-gate (200). E3: a worker token minted under an older
// worker_token_epoch is refused even on its two routes.

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"vl/auth"
)

// workerTokenClaims mints a cutover-worker token with the given epoch.
func workerTokenClaims(wte int64) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"user_id": updAdminID, "email": updAdminEmail, "iss": "vlAI",
		"scope": auth.ScopeCutoverWorker, "wte": wte,
		"iat": now.Add(-5 * time.Second).Unix(), "nbf": now.Add(-time.Minute).Unix(),
		"exp": now.Add(time.Hour).Unix(),
	}
}

func TestCutoverWorkerTokenIsRefusedOnEveryOtherRoute(t *testing.T) {
	e := newUpdEnv(t)
	auth.SetWorkerEpochReader(func() (int64, error) { return 0, nil })
	tok := mintMapJWT(t, workerTokenClaims(0))
	owner, _ := credLogin(t, e, updAdminEmail, updAdminPass)

	allowed := map[string]bool{
		"GET /api/maintenance":       true,
		"GET /api/installation-gate": true,
	}
	seenAllowed := map[string]bool{}
	walked := 0
	for _, r := range e.s.router.Routes() {
		key := r.Method + " " + r.Path
		path := concretePath(r.Path)
		// Only JWT-protected routes are in scope: 401 with no token, but the
		// owner's login token is admitted (not 401). Routes with their own
		// auth domain (the bars stream ticket) fail the second probe and are
		// skipped — they never read the JWT scope.
		if probe := credCall(t, e, r.Method, path, "", censusBody(r.Method, r.Path)); probe.Code != http.StatusUnauthorized {
			continue
		}
		if ownerProbe := credCall(t, e, r.Method, path, owner, censusBody(r.Method, r.Path)); ownerProbe.Code == http.StatusUnauthorized {
			continue
		}
		rec := credCall(t, e, r.Method, path, tok, censusBody(r.Method, r.Path))
		if allowed[key] {
			seenAllowed[key] = true
			if rec.Code != http.StatusOK {
				t.Fatalf("%s = %d %s — the worker's own route must admit it", key, rec.Code, rec.Body.String())
			}
			continue
		}
		walked++
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s = %d %s — want 403 (cutover-worker is denied by default)", key, rec.Code, rec.Body.String())
		}
	}
	if walked == 0 {
		t.Fatal("census walked no routes — the router is empty?")
	}
	for k := range allowed {
		if !seenAllowed[k] {
			t.Fatalf("census never walked %s — is it still registered?", k)
		}
	}
	// The ruling names force-flat explicitly: it must have been walked and refused.
	if rec := credCall(t, e, "POST", "/api/risk/force-flat", tok, `{}`); rec.Code != http.StatusForbidden {
		t.Fatalf("POST /api/risk/force-flat = %d — want 403 for a cutover-worker token", rec.Code)
	}
}

func TestCutoverWorkerTokenRefusedBelowTheEpoch(t *testing.T) {
	e := newUpdEnv(t)
	auth.SetWorkerEpochReader(func() (int64, error) { return 7, nil })
	stale := mintMapJWT(t, workerTokenClaims(6))
	for _, p := range []string{"/api/maintenance", "/api/installation-gate"} {
		if rec := credCall(t, e, "GET", p, stale, ""); rec.Code != http.StatusForbidden {
			t.Fatalf("GET %s with wte=6 under epoch 7 = %d — want 403", p, rec.Code)
		}
	}
	fresh := mintMapJWT(t, workerTokenClaims(7))
	if rec := credCall(t, e, "GET", "/api/maintenance", fresh, ""); rec.Code != http.StatusOK {
		t.Fatalf("GET /api/maintenance with wte=7 under epoch 7 = %d — want 200", rec.Code)
	}
}

func TestCutoverWorkerTokenRefusedWhenTheEpochReaderIsMissing(t *testing.T) {
	e := newUpdEnv(t)
	auth.SetWorkerEpochReader(nil) // fail closed: no reader, no worker token
	tok := mintMapJWT(t, workerTokenClaims(0))
	if rec := credCall(t, e, "GET", "/api/maintenance", tok, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("GET /api/maintenance with no epoch reader = %d — want 403 (fail closed)", rec.Code)
	}
}
