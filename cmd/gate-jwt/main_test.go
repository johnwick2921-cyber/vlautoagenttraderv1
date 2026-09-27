package main

// M3 (CTO 1790237128086, 03:05): the gate-jwt token — the one
// deploy/cutover.sh reads GET /api/cutover-gate with — is a MACHINE token
// (scope gate-jwt). It must keep passing auth on /api/cutover-gate, and it
// must be refused 403 on every machine-denied surface: /api/updates*,
// /api/user/password, /api/reset-account, /api/reset-password and
// /api/telegram* — and /api/logout (PR #200 F4a), whose gate-jwt refusal is
// pinned in api/machine_logout_test.go rather than here: this test's
// owner-token control on the same request would blacklist the owner token
// part-way through. Pinned with the token the tool itself mints
// (mintGateToken, what main() runs) against the PRODUCTION server
// (api.NewServer + Server.Start on a loopback socket), each refusal beside a
// positive control: the owner's own login token on the same request gets
// past authentication, so a 403 here is the machine-token rule and nothing
// else.

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"nofx/api"
	"nofx/auth"
	"nofx/internal/updateauth"
	"nofx/manager"
	"nofx/store"
	"nofx/trader"
)

const (
	gjOwnerID    = "cccccccc-1111-2222-3333-444444444444"
	gjOwnerEmail = "owner@example.test"
	gjOwnerPass  = "owner-gate-jwt-pass"
	gjSecret     = "gate-jwt-test-secret-not-the-default-0123456789" // ≥ 32 bytes: /api/updates refuses a shorter one
	gjForbidden  = `{"error":"forbidden"}`
)

func gjBoot(t *testing.T) (string, *store.Store) {
	t.Helper()
	dataDir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	st, err := store.New(filepath.Join(dataDir, "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	prevDir := trader.MaintenanceDataDir()
	trader.SetMaintenanceDataDir(dataDir)
	t.Cleanup(func() { trader.SetMaintenanceDataDir(prevDir) })
	prev := auth.JWTSecret
	auth.SetJWTSecret(gjSecret)
	t.Cleanup(func() { auth.JWTSecret = prev })
	hash, err := auth.HashPassword(gjOwnerPass)
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour).UTC()
	if err := st.User().Create(&store.User{ID: gjOwnerID, Email: gjOwnerEmail, PasswordHash: hash, CreatedAt: past, UpdatedAt: past}); err != nil {
		t.Fatal(err)
	}
	// Enrolled for updates, so the owner's control reaches the /updates
	// handlers and a 403 there is the machine-token rule, not "not enrolled".
	if err := updateauth.Enroll(dataDir, gjOwnerID, gjOwnerEmail, hash, time.Now(), false); err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	s := api.NewServer(manager.NewTraderManager(), st, nil, "127.0.0.1", port)
	go func() { _ = s.Start() }()
	t.Cleanup(func() { _ = s.Shutdown() })
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, err := http.Get(base + "/api/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("production server never came up: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	return base, st
}

func gjCall(t *testing.T, base, method, path, tok, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, base+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.HasPrefix(path, "/api/updates") {
		req.Header.Set(api.UpdateHeader, "1") // every OTHER factor of the updates gate is valid
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(b))
}

func TestGateJWTTokenPassesTheCutoverGateAndIsDeniedTheMachineDeniedRoutes(t *testing.T) {
	base, st := gjBoot(t)

	gate, err := mintGateToken(st, gjOwnerEmail)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := auth.ValidateJWT(gate)
	if err != nil {
		t.Fatal(err)
	}
	if cl.Scope != auth.ScopeGateJWT || cl.UserID != gjOwnerID || cl.Email != gjOwnerEmail || !cl.IsMachine() {
		t.Fatalf("gate-jwt mints scope=%q user=%q email-match=%v machine=%v — want its own gate-jwt scope on the owner's id and email", cl.Scope, cl.UserID, cl.Email == gjOwnerEmail, cl.IsMachine())
	}
	code, body := gjCall(t, base, "POST", "/api/login", "", `{"email":"`+gjOwnerEmail+`","password":"`+gjOwnerPass+`"}`)
	if code != http.StatusOK {
		t.Fatalf("owner login = %d", code)
	}
	var login struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal([]byte(body), &login)
	owner := login.Token
	if owner == "" {
		t.Fatal("owner login returned no token")
	}

	// cutover.sh's read: the gate-jwt token passes auth and reaches the
	// handler — exactly what the owner's token gets (no trader here, so the
	// handler's own "Invalid trader ID").
	gCode, gBody := gjCall(t, base, "GET", "/api/cutover-gate", gate, "")
	oCode, oBody := gjCall(t, base, "GET", "/api/cutover-gate", owner, "")
	if gCode == http.StatusUnauthorized || gCode == http.StatusForbidden || gCode != oCode || gBody != oBody || !strings.Contains(gBody, "Invalid trader ID") {
		t.Fatalf("gate-jwt token on GET /api/cutover-gate = %d %s — want the handler's own answer, the same as the owner's (%d %s), never 401/403", gCode, gBody, oCode, oBody)
	}

	// Every machine-denied surface: the gate-jwt token 403 {"error":"forbidden"};
	// the owner's token on the same request is past authentication.
	for _, p := range []struct{ method, path, body string }{
		{"GET", "/api/updates", ""},
		{"POST", "/api/updates/check", "{}"},
		{"POST", "/api/updates/install", "{}"},
		{"GET", "/api/updates/jobs/0123456789abcdef", ""},
		{"GET", "/api/updates/jobs/0123456789abcdef/receipt", ""},
		{"PUT", "/api/user/password", "{}"},
		{"POST", "/api/reset-account", "{}"},
		{"POST", "/api/reset-password", `{"email":"` + gjOwnerEmail + `","new_password":"machine-chosen-pass-9"}`},
		{"GET", "/api/telegram", ""},
		{"POST", "/api/telegram", "{}"},
		{"POST", "/api/telegram/model", "{}"},
		{"DELETE", "/api/telegram/binding", ""},
	} {
		gCode, gBody := gjCall(t, base, p.method, p.path, gate, p.body)
		if gCode != http.StatusForbidden || gBody != gjForbidden {
			t.Errorf("gate-jwt token: %s %s = %d %s — want 403 %s (machine tokens are denied there)", p.method, p.path, gCode, gBody, gjForbidden)
		}
		oCode, oBody := gjCall(t, base, p.method, p.path, owner, p.body)
		if oCode == http.StatusUnauthorized || oBody == gjForbidden {
			t.Errorf("positive control: the owner's token on %s %s = %d %s — the same request must get past authentication", p.method, p.path, oCode, oBody)
		}
		t.Logf("%s %s: gate-jwt %d · owner %d", p.method, p.path, gCode, oCode)
	}
	if n, _ := st.User().Count(); n != 1 {
		t.Fatalf("users = %d after the probes — want the owner's row untouched", n)
	}
}
