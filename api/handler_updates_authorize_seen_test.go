package api

// W-ONE-BUTTON M3 red-team fold (red-3 #5): the attended minter never hands
// out a code the server can only refuse. updateauth.Authorize reads the seen
// store (read-only) and refuses to mint when the code it would mint expires
// at or below the store's pruned-through watermark (the server would answer
// 409 "job already used" for a job never used) or its clock floor (403), or
// when the store is unreadable or — once enrolled — missing. Proven against
// the production router: every code Authorize does mint is authorized (422).

import (
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"nofx/internal/updateauth"
)

func TestAuthorizeRefusesToMintACodeTheServerWouldRefuse(t *testing.T) {
	e := newUpdEnv(t)
	clk := newUpdClock(e)
	key := mustKey(t, e.dataDir)
	T := clk.now().Unix()
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "mint-wm-job-a0001", T+300)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("A = %d", w.Code)
	}
	clk.set(T + 1000) // B prunes A: pruned_through = T+300, clock_floor = T+1000
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "mint-wm-job-b0001", T+1300)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("B = %d", w.Code)
	}
	for name, at := range map[string]int64{
		"below the pruned-through watermark (server: 409)": T - 100,
		"below the clock floor (server: 403)":              T + 600,
	} {
		clk.set(at)
		g, err := updateauth.Authorize(e.dataDir, updRelease, time.Unix(at, 0))
		if err == nil {
			w := e.do("POST", "/api/updates/install", grantBody(g))
			t.Errorf("%s: Authorize minted job %s (exp %d) and the server answered %d %s", name, g.JobID, g.ExpiresAt, w.Code, w.Body.String())
			continue
		}
		if !errors.Is(err, updateauth.ErrClockBehindSeenStore) {
			t.Errorf("%s: Authorize = %v, want ErrClockBehindSeenStore", name, err)
		}
	}
	// positive control: once the clock is past the floor the minted code is
	// authorized by the server
	now := T + 1001
	clk.set(now)
	g, err := updateauth.Authorize(e.dataDir, updRelease, time.Unix(now, 0))
	if err != nil {
		t.Fatalf("positive control: Authorize past the floor: %v", err)
	}
	if w := e.do("POST", "/api/updates/install", grantBody(g)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: the minted code = %d %s, want 422", w.Code, w.Body.String())
	}
	// an unreadable store, or a missing one after enrollment: nothing minted
	p := updateauth.SeenPath(e.dataDir)
	if err := os.Rename(p, p+".aside"); err != nil {
		t.Fatal(err)
	}
	if g, err := updateauth.Authorize(e.dataDir, updRelease, time.Unix(now, 0)); err == nil || !errors.Is(err, updateauth.ErrSeenCorrupt) {
		t.Errorf("missing seen store: Authorize = (%v, %v), want ErrSeenCorrupt and no grant", g, err)
	}
	if err := os.WriteFile(p, []byte("junk"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := updateauth.Authorize(e.dataDir, updRelease, time.Unix(now, 0)); !errors.Is(err, updateauth.ErrSeenCorrupt) {
		t.Errorf("corrupt seen store: Authorize = %v, want ErrSeenCorrupt", err)
	}
	if b, _ := os.ReadFile(p); string(b) != "junk" {
		t.Error("Authorize wrote the seen store (it must only read it)")
	}
}
