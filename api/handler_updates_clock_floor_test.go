package api

// W-ONE-BUTTON M3 red-team fold (red-3 #2): the server's "expired" verdict
// is sticky across a clock step-back. The seen-job store keeps a clock floor
// — the latest server clock reading it has recorded, at every consumption
// and at every refusal of a GENUINE (MAC-verified) expired code — and any
// code whose expires_at is at or below it is expired, whatever the clock
// says afterwards. A step-back smaller than the 5-minute window costs
// nothing (fresh codes still expire above the floor); a larger one refuses
// installs until the clock passes the floor again (CTO ruling on F1: a clock
// step-back is a fault; refusing until the clock passes the floor is the
// right fail-closed).

import (
	"bytes"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"nofx/internal/updateauth"
)

type updClock struct {
	mu  sync.Mutex
	cur time.Time
}

func (c *updClock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.cur }
func (c *updClock) set(unix int64) { c.mu.Lock(); c.cur = time.Unix(unix, 0); c.mu.Unlock() }
func newUpdClock(e *updEnv) *updClock {
	c := &updClock{cur: time.Now()}
	e.s.updatesNow = c.now
	return c
}

func TestInstallAnExpiredGrantStaysExpiredAfterAClockStepBack(t *testing.T) {
	for _, consumedAfter := range []bool{true, false} {
		name := map[bool]string{true: "another install consumed after the expiry", false: "no other install"}[consumedAfter]
		t.Run(name, func(t *testing.T) {
			e := newUpdEnv(t)
			clk := newUpdClock(e)
			key := mustKey(t, e.dataDir)
			T := clk.now().Unix()
			bodyA := grantBodyUnder(t, key, updRelease, "sticky-expiry-a001", T+300) // minted at T, pasted too late

			clk.set(T + 400)
			if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
				t.Fatalf("precondition: at T+400 grant A = %d %s, want 403 (expired)", w.Code, w.Body.String())
			}
			if consumedAfter {
				if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "sticky-expiry-b001", T+700)); w.Code != http.StatusUnprocessableEntity {
					t.Fatalf("precondition: fresh grant B = %d, want 422", w.Code)
				}
			}
			clk.set(T + 200) // the clock is stepped back 200 s: A is inside its window again
			if w := e.do("POST", "/api/updates/install", bodyA); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
				t.Errorf("grant A, refused as expired at T+400, after a 200 s step-back = %d %s, want 403 %s", w.Code, w.Body.String(), forbiddenBody)
			}
			// positive control: a step-back smaller than the window does not
			// block a code minted at the stepped-back clock
			if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "sticky-expiry-c001", T+500)); w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("positive control: fresh grant at the stepped-back clock = %d %s, want 422", w.Code, w.Body.String())
			}
		})
	}
}

// A step-back LARGER than the window refuses even freshly minted codes until
// the clock passes the floor again; and a forged expired code (wrong MAC)
// writes nothing — only a key holder moves the floor.
func TestInstallClockFloorBindsFreshCodesAfterALargeStepBackAndForgedCodesWriteNothing(t *testing.T) {
	e := newUpdEnv(t)
	clk := newUpdClock(e)
	key := mustKey(t, e.dataDir)
	T := clk.now().Unix()
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "floor-job-a000001", T+300)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("A = %d", w.Code)
	}
	clk.set(T - 400) // a 400 s step-back: a fresh code expires at T-100, below the floor T
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "floor-job-b000001", T-100)); w.Code != http.StatusForbidden || w.Body.String() != forbiddenBody {
		t.Errorf("fresh code expiring below the clock floor = %d %s, want 403", w.Code, w.Body.String())
	}
	clk.set(T - 250) // the clock catches up: exp T+50 is above the floor
	if w := e.do("POST", "/api/updates/install", grantBodyUnder(t, key, updRelease, "floor-job-c000001", T+50)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("positive control: a code expiring above the floor = %d %s, want 422", w.Code, w.Body.String())
	}
	// a forged expired code: 403 and the store is byte-identical
	before, _ := os.ReadFile(updateauth.SeenPath(e.dataDir))
	clk.set(T + 10_000)
	forged := grantBodyUnder(t, bytes.Repeat([]byte{7, 9}, 16), updRelease, "floor-job-d000001", T+9_000)
	if w := e.do("POST", "/api/updates/install", forged); w.Code != http.StatusForbidden {
		t.Fatalf("forged expired code = %d, want 403", w.Code)
	}
	if after, _ := os.ReadFile(updateauth.SeenPath(e.dataDir)); !bytes.Equal(before, after) {
		t.Error("a forged (wrong-MAC) expired code wrote the seen store")
	}
}
