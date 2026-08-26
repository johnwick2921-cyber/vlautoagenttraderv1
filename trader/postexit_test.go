package trader

import (
	"os"
	"testing"
	"time"

	"nofx/discipline"
)

// 4.5 — env gates: default ON, explicit off honored; delay default + override.
func TestPostExitEnvGates(t *testing.T) {
	os.Unsetenv("POST_EXIT_RESCAN")
	if !postExitRescanEnabled() {
		t.Fatal("POST_EXIT_RESCAN must default ON")
	}
	os.Setenv("POST_EXIT_RESCAN", "off")
	if postExitRescanEnabled() {
		t.Fatal("explicit off must disable")
	}
	os.Unsetenv("POST_EXIT_RESCAN")
	os.Unsetenv("POST_EXIT_DELAY_MS")
	if postExitDelayMs() != defaultPostExitDelayMs {
		t.Fatalf("delay default = %d, want %d", postExitDelayMs(), defaultPostExitDelayMs)
	}
	os.Setenv("POST_EXIT_DELAY_MS", "0")
	defer os.Unsetenv("POST_EXIT_DELAY_MS")
	if postExitDelayMs() != 0 {
		t.Fatal("delay override not honored")
	}
}

// 4.2/4.5 — dedup per position id: TP/SL/partial-fill double-reports collapse to
// exactly ONE kick; the kick carries the post_exit trigger.
func TestPostExitDedupFiresOnce(t *testing.T) {
	os.Setenv("POST_EXIT_DELAY_MS", "0")
	defer os.Unsetenv("POST_EXIT_DELAY_MS")
	at := &AutoTrader{exchange: "ninjatrader", kickCh: make(chan string, 4)}

	at.notifyPositionClosed(524) // e.g. the TP close report
	at.notifyPositionClosed(524) // …and the reconcile double-report
	at.notifyPositionClosed(524)

	got := 0
	deadline := time.After(2 * time.Second)
	for {
		select {
		case r := <-at.kickCh:
			if r != "post_exit" {
				t.Fatalf("kick reason = %q, want post_exit", r)
			}
			got++
		case <-deadline:
			if got != 1 {
				t.Fatalf("post-exit kicks = %d, want exactly 1 (dedup per position id)", got)
			}
			return
		}
	}
}

// A DIFFERENT position id fires its own rescan (dedup is per id, not global).
func TestPostExitNewPositionFiresAgain(t *testing.T) {
	os.Setenv("POST_EXIT_DELAY_MS", "0")
	defer os.Unsetenv("POST_EXIT_DELAY_MS")
	at := &AutoTrader{exchange: "ninjatrader", kickCh: make(chan string, 4)}
	at.notifyPositionClosed(1)
	at.notifyPositionClosed(2)
	got := 0
	deadline := time.After(2 * time.Second)
	for got < 2 {
		select {
		case <-at.kickCh:
			got++
		case <-deadline:
			t.Fatalf("got %d kicks, want 2 (one per closed position)", got)
		}
	}
}

// 4.5 — OFF switch restores today's behavior exactly: zero kicks. Crypto: never.
func TestPostExitOffAndCryptoNoKick(t *testing.T) {
	os.Setenv("POST_EXIT_RESCAN", "off")
	os.Setenv("POST_EXIT_DELAY_MS", "0")
	defer os.Unsetenv("POST_EXIT_RESCAN")
	defer os.Unsetenv("POST_EXIT_DELAY_MS")
	at := &AutoTrader{exchange: "ninjatrader", kickCh: make(chan string, 4)}
	at.notifyPositionClosed(7)
	os.Unsetenv("POST_EXIT_RESCAN")
	crypto := &AutoTrader{exchange: "binance", kickCh: make(chan string, 4)}
	crypto.notifyPositionClosed(8)
	select {
	case r := <-at.kickCh:
		t.Fatalf("OFF switch leaked a kick (%q)", r)
	case r := <-crypto.kickCh:
		t.Fatalf("crypto trader leaked a kick (%q)", r)
	case <-time.After(300 * time.Millisecond):
	}
}

// 4.4 — B7 precedence: the post-exit cycle RUNS, but with a cooldown armed a
// same-direction re-entry is refused by the (pre-existing, LIVE) kernel gate.
// This pins the arming→blocking chain the rescan must flow through.
func TestPostExitB7CooldownKeepsVeto(t *testing.T) {
	now := time.Now().UnixMilli()
	discipline.NoteStopLossExit("t-b7", "MNQ", "SHORT", 29650.0, now)
	// Immediately after the SL exit, price near the stop, 20-min cooldown:
	_, reason, blocked := discipline.ReentryBlocked("t-b7", "MNQ", "SHORT", 20, 40.0, 29655.0, now+5_000)
	if !blocked {
		t.Fatal("B7 must veto the same-direction re-entry inside the cooldown — post-exit rescans are not privileged")
	}
	if reason == "" {
		t.Fatal("the refusal must carry a B7-named reason for the record")
	}
	// After the timer elapses the veto lifts (rescan entries become possible).
	_, _, blocked = discipline.ReentryBlocked("t-b7", "MNQ", "SHORT", 20, 40.0, 29655.0, now+21*60_000)
	if blocked {
		t.Fatal("cooldown must expire on the timer")
	}
}
