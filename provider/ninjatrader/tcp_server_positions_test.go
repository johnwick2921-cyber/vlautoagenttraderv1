// Open-position read-back receive-path integration test (the account
// switch-back / manual-open fix). Spins a real TCPServer + MockTCPClient, has
// the mock write a `positions` snapshot frame, and asserts PositionsFor
// reflects it — proving the Go receive path (read → decode → per-account cache
// → accessor) without NT8. Also checks the full-snapshot REPLACE semantics: an
// empty snapshot means the account is known-flat. Mirrors
// tcp_server_account_test.go.
package ninjatrader

import (
	"context"
	"testing"
	"time"
)

func TestTCPServer_PositionsReceivePath(t *testing.T) {
	srv := NewTCPServer(nil)
	addr := freeEphemeralAddr(t)
	srv.SetAddrForTest(addr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	client := NewMockTCPClient(addr, 50*time.Millisecond)
	if err := client.Start(ctx); err != nil {
		t.Fatalf("client Start: %v", err)
	}
	defer client.Stop()
	waitForConnected(t, srv, 2*time.Second)

	// Before any frame, PositionsFor reports not-yet-received (caller falls
	// back to the fill-derived cache).
	if _, ok := srv.PositionsFor("Sim101"); ok {
		t.Fatal("PositionsFor ok=true before any positions frame")
	}

	want := PositionsPayload{
		Account: "Sim101",
		Positions: []OpenPosition{
			{Symbol: "MNQ", Side: "long", Quantity: 2, AvgPrice: 30550.25},
		},
	}
	if err := writeFromMock(client, FramePositions, want); err != nil {
		t.Fatalf("write positions: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var got []OpenPosition
	var ok bool
	for time.Now().Before(deadline) {
		if got, ok = srv.PositionsFor("Sim101"); ok {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ok {
		t.Fatal("PositionsFor never became valid after positions frame")
	}
	if len(got) != 1 || got[0].Symbol != "MNQ" || got[0].Side != "long" ||
		got[0].Quantity != 2 || got[0].AvgPrice != 30550.25 {
		t.Errorf("PositionsFor = %+v, want 1x{MNQ long 2 @30550.25}", got)
	}

	// A subsequent EMPTY snapshot (account went flat) must REPLACE the cache —
	// known-flat, not stale. This is the switch-back / close correctness.
	if err := writeFromMock(client, FramePositions, PositionsPayload{Account: "Sim101", Positions: []OpenPosition{}}); err != nil {
		t.Fatalf("write empty positions: %v", err)
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if g, ok := srv.PositionsFor("Sim101"); ok && len(g) == 0 {
			return // flat snapshot applied — pass
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Error("empty positions snapshot did not replace cache to flat")
}
