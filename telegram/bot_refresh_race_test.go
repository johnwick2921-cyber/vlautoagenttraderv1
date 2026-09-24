package telegram

// M3 ha verifier defect 3 — the -race reproduction, WRITTEN FOR THE CTO'S
// RACE SLOT (the M3 load rule held its author to no -race; it has not been
// run under the detector by its author). Run it with:
//
//	GOMAXPROCS=4 nice -n 19 ionice -c 3 go test -race -run TestRaceBotRefreshAgainstInFlightManager ./telegram/
//
// Without -race it SKIPS: it asserts nothing the detector does not.
//
// What it drives: the production botIdentity.refresh on the "main loop"
// (this test's goroutine), re-minting on EVERY call, while answers run on
// per-message goroutines through the manager refresh built — each goroutine
// holding the manager captured before its go statement, the shape runBot has
// (runBot itself needs a live Telegram API; its shape is pinned structurally
// by TestRunBotGoroutinesReadNoBotIdentityField). Each answer calls the
// manager's LLM factory twice (Manager.Run → agent.New → getLLM, then
// Agent.Run → getLLM), so a factory that reads the identity's fields
// (newLLMClient(b.st, b.userID) — the pre-fix refresh) races the next
// refresh's b.userID write and the detector reports it; the fixed factory
// reads locals and the test is clean.

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/auth"
	"nofx/store"
	"nofx/telegram/agent"
)

func TestRaceBotRefreshAgainstInFlightManager(t *testing.T) {
	if !raceDetectorOn {
		t.Skip("proves nothing without the race detector — run: GOMAXPROCS=4 nice -n 19 ionice -c 3 go test -race -run TestRaceBotRefreshAgainstInFlightManager ./telegram/")
	}
	// No model anywhere: the factory returns nil, so Agent.Run stops at its
	// first getLLM() — no network, no LLM, no API call.
	for _, k := range []string{"DEEPSEEK_API_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY"} {
		t.Setenv(k, "")
	}
	dataDir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	st, err := store.New(filepath.Join(dataDir, "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	prev := auth.JWTSecret
	auth.SetJWTSecret(btSecret)
	t.Cleanup(func() { auth.JWTSecret = prev })
	// The credential epoch an hour AHEAD: every token the bot mints is issued
	// at or before it, so EVERY refresh re-mints and rewrites every field of
	// the identity — the rewrite the race needs, on every iteration.
	if err := st.User().Create(&store.User{ID: btOwnerID, Email: btOwnerEmail, PasswordHash: "not-a-real-hash",
		CreatedAt: time.Now().Add(-time.Hour).UTC(), UpdatedAt: time.Now().Add(time.Hour).UTC()}); err != nil {
		t.Fatal(err)
	}

	ident := newBotIdentity(st, 0)
	if !ident.refresh() {
		t.Fatal("refresh with an account on the box = false")
	}
	const messages = 64
	replies := make(chan string, messages)
	var wg sync.WaitGroup
	rebuilt := 0
	for i := 0; i < messages; i++ {
		before := ident.agents
		// runBot's main loop: refresh before every AI call …
		if !ident.refresh() {
			t.Fatal("refresh = false mid-run")
		}
		if ident.agents != before {
			rebuilt++
		}
		// … the capture before the go statement …
		agents := ident.agents
		wg.Add(1)
		// … and the goroutine reads only what it was handed.
		go func(agents *agent.Manager, chatID int64) {
			defer wg.Done()
			replies <- agents.Run(chatID, "status", nil)
		}(agents, int64(1+i%3))
	}
	wg.Wait()
	close(replies)

	// Vacuity guards: the rewrite happened every time, and every answer
	// reached the factory (the no-model reply comes only from a nil getLLM()).
	if rebuilt != messages {
		t.Fatalf("refresh rebuilt the manager on %d of %d calls — want every call (no rewrite, no race to catch)", rebuilt, messages)
	}
	n := 0
	for r := range replies {
		n++
		if !strings.HasPrefix(r, "AI assistant unavailable") {
			t.Fatalf("an answer = %q — want the no-model reply (the factory ran and returned nil)", r)
		}
	}
	if n != messages {
		t.Fatalf("%d answers for %d messages", n, messages)
	}
}
