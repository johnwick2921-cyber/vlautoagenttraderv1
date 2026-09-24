package agent

// M3 red-team H1 (red2 #1), ported to a pin at the PRODUCTION call sites end
// to end: api.NewServer + Server.Start (the real router, real middleware, a
// real loopback socket), the bot token the production bot mints
// (GenerateBotToken — telegram/bot.go), and the Telegram agent's ONLY tool
// (newAPICallTool / execute) — exactly what an LLM steered by an inbound
// message can do. The chain PUT /api/user/password → POST /api/login must not
// yield a token for the owner, and the owner's password must survive.

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
	"nofx/manager"
	"nofx/store"
	"nofx/trader"
)

const (
	chainOwnerID    = "cccccccc-1111-2222-3333-444444444444"
	chainOwnerEmail = "owner@example.test"
	chainOwnerPass  = "owner-original-pass"
	chainSecret     = "chain-test-secret-not-the-default-0123456789"
)

func chainFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return p
}

// chainBoot starts the production server on a loopback port over a temp
// installation with one owner row; it returns the port and the store.
func chainBoot(t *testing.T) (int, *store.Store) {
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
	auth.SetJWTSecret(chainSecret)
	t.Cleanup(func() { auth.JWTSecret = prev })

	hash, err := auth.HashPassword(chainOwnerPass)
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour).UTC()
	if err := st.User().Create(&store.User{ID: chainOwnerID, Email: chainOwnerEmail, PasswordHash: hash, CreatedAt: past, UpdatedAt: past}); err != nil {
		t.Fatal(err)
	}
	port := chainFreePort(t)
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
	return port, st
}

func TestAgentToolCannotTakeOverTheOwnersAccount(t *testing.T) {
	port, st := chainBoot(t)
	before, err := st.User().GetByID(chainOwnerID)
	if err != nil {
		t.Fatal(err)
	}
	bot, err := GenerateBotToken(chainOwnerID)
	if err != nil {
		t.Fatal(err)
	}
	tool := newAPICallTool(port, bot)

	// Positive control: the agent's tool and token still work on an ordinary
	// protected route (the fix must not break the Telegram agent).
	if out := tool.execute(&apiRequest{Method: "GET", Path: "/api/my-traders"}); strings.HasPrefix(out, "API error") || strings.HasPrefix(out, "API call failed") {
		t.Fatalf("positive control: the bot token no longer reaches an ordinary protected route: %.200s", out)
	}

	out1 := tool.execute(&apiRequest{Method: "PUT", Path: "/api/user/password", Body: map[string]any{"new_password": "agent-chosen-pass-1"}})
	out2 := tool.execute(&apiRequest{Method: "POST", Path: "/api/login", Body: map[string]any{"email": chainOwnerEmail, "password": "agent-chosen-pass-1"}})
	var login struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal([]byte(out2), &login)
	if login.Token != "" || !strings.HasPrefix(out1, "API error 403") {
		t.Fatalf("the Telegram agent, through its only tool and the bot JWT, took over the owner's account:\n"+
			"  PUT /api/user/password -> %.120s\n  POST /api/login (agent-chosen password) -> token_len=%d\n"+
			"  want: 403 on the password change and no token", strings.TrimSpace(out1), len(login.Token))
	}
	after, err := st.User().GetByID(chainOwnerID)
	if err != nil {
		t.Fatal(err)
	}
	if after.PasswordHash != before.PasswordHash || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatal("the refused agent call still wrote the owner's row")
	}
	if !auth.CheckPassword(chainOwnerPass, after.PasswordHash) {
		t.Fatal("positive control: the owner's own password no longer verifies")
	}
}
