package api

// N-1 (verify/0926-system, P2): POST /api/onboarding/beginner wrote
// CLAW402_* into the process env AND the .env file for ANY authenticated
// user. The fix: owner-only (the first-created account, non-machine token),
// and refused outright on the futures build. These pins drive the PRODUCTION
// router (NewServer) — canon 53.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"nofx/auth"
	"nofx/store"
)

// onboardingCall drives POST /api/onboarding/beginner through the production
// router from a loopback peer.
func onboardingCall(t *testing.T, e *updEnv, tok, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("POST", "/api/onboarding/beginner", strings.NewReader(body))
	r.RemoteAddr, r.Host = "127.0.0.1:52000", "127.0.0.1:8080"
	if tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	e.s.router.ServeHTTP(w, r)
	return w
}

// seedSecondUser creates a non-owner account (created AFTER the admin) and
// returns its JWT.
func seedSecondUser(t *testing.T, e *updEnv) string {
	t.Helper()
	now := time.Now().UTC()
	if err := e.st.User().Create(&store.User{
		ID:           "second-user",
		Email:        "second@example.com",
		PasswordHash: "x",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("seed second user: %v", err)
	}
	return mintJWT(t, "second-user", "second@example.com",
		time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)
}

// captureEnvProbe records whether CLAW402_WALLET_KEY is set in the process env.
func captureEnvProbe(t *testing.T) (probe func() bool, restore func()) {
	t.Helper()
	before, wasSet := os.LookupEnv("CLAW402_WALLET_KEY")
	restore = func() {
		if wasSet {
			os.Setenv("CLAW402_WALLET_KEY", before)
		} else {
			os.Unsetenv("CLAW402_WALLET_KEY")
		}
	}
	os.Unsetenv("CLAW402_WALLET_KEY")
	return func() bool {
		_, ok := os.LookupEnv("CLAW402_WALLET_KEY")
		return ok
	}, restore
}

func TestBeginnerOnboardingNonOwnerRefusedAndEnvUntouched(t *testing.T) {
	e := newUpdEnv(t)
	secondTok := seedSecondUser(t, e)

	// Run in an isolated cwd so a .env write, if it happened, would be visible.
	prevDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	probe, restore := captureEnvProbe(t)
	defer restore()

	w := onboardingCall(t, e, secondTok, "{}")
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-owner onboarding: want 403, got %d (%s)", w.Code, w.Body.String())
	}
	if probe() {
		t.Fatal("non-owner onboarding set CLAW402_WALLET_KEY — env mutated")
	}
	if _, err := os.Stat(".env"); err == nil {
		t.Fatal("non-owner onboarding wrote a .env file")
	}
}

func TestBeginnerOnboardingMachineTokenRefused(t *testing.T) {
	e := newUpdEnv(t)
	botTok := mintJWT(t, updAdminID, auth.BotInternalEmail,
		time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)

	prevDir, _ := os.Getwd()
	work := t.TempDir()
	_ = os.Chdir(work)
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	probe, restore := captureEnvProbe(t)
	defer restore()

	w := onboardingCall(t, e, botTok, "{}")
	if w.Code != http.StatusForbidden {
		t.Fatalf("machine-token onboarding: want 403, got %d (%s)", w.Code, w.Body.String())
	}
	if probe() {
		t.Fatal("machine token set CLAW402_WALLET_KEY — env mutated")
	}
}

func TestBeginnerOnboardingRefusedOnFuturesBuild(t *testing.T) {
	e := newUpdEnv(t)
	prevMode := apiTradingMode
	apiTradingMode = func() string { return "futures" }
	t.Cleanup(func() { apiTradingMode = prevMode })
	prevBalance := queryUSDCBalanceStr
	queryUSDCBalanceStr = func(string) string { return "0.00" }
	t.Cleanup(func() { queryUSDCBalanceStr = prevBalance })

	prevDir, _ := os.Getwd()
	work := t.TempDir()
	_ = os.Chdir(work)
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	probe, restore := captureEnvProbe(t)
	defer restore()

	w := onboardingCall(t, e, e.tok, "{}")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("futures onboarding: want 400, got %d (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "crypto-era") {
		t.Fatalf("futures refusal must carry the clear message; got %s", w.Body.String())
	}
	if probe() {
		t.Fatal("futures refusal set CLAW402_WALLET_KEY — env mutated")
	}
}

func TestBeginnerOnboardingOwnerStillWorksOnCryptoBuild(t *testing.T) {
	e := newUpdEnv(t)
	prevBalance := queryUSDCBalanceStr
	queryUSDCBalanceStr = func(string) string { return "1.23" }
	t.Cleanup(func() { queryUSDCBalanceStr = prevBalance })

	prevDir, _ := os.Getwd()
	work := t.TempDir()
	_ = os.Chdir(work)
	t.Cleanup(func() { _ = os.Chdir(prevDir) })

	w := onboardingCall(t, e, e.tok, "{}")
	if w.Code == http.StatusForbidden {
		t.Fatalf("owner onboarding must pass the owner gate; got 403 (%s)", w.Body.String())
	}
	var resp beginnerOnboardingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("owner onboarding response is not the expected shape: %v (%s)", err, w.Body.String())
	}
	if resp.Address == "" || resp.PrivateKey == "" {
		t.Fatalf("owner onboarding did not return a wallet: %s", w.Body.String())
	}
}
