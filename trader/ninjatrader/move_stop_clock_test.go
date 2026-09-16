// THE LAST BAR-CLOCK ON AN OUTGOING COMMAND — move_stop (2026-09-07).
//
// The entry paths were corrected by the placement-truth wave and are pinned by
// TestEntryCommandClockAndReceivedRejection. move_stop was the one it did not
// reach: MoveStopPayload.Timestamp was still stamped from feedNowUTC, the last
// BAR's close, while the AddOn ages a received timestamp against
// DateTime.UtcNow (VLTraderTCPClient.cs:814).
//
// It is INERT today — HandleMoveStop reads only signal_id and new_stop_loss —
// so this pin guards a landmine rather than a live fault. The landmine is
// specific: a freshness guard on move_stop, the obvious hardening after this
// wave, would inherit the bug fully formed, and auto-breakeven would then fail
// during feed gaps, which is exactly when a runner most needs its stop moved.
package ninjatrader

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// The owner's pin, applied to the move_stop path: a command composed while the
// bar cache is 30 minutes stale must still carry a wall-clock stamp the AddOn
// would measure in milliseconds.
// moveStopServer is a loopback fixture whose reader keeps move_stop frames.
// stopEntryServer's reader discards every non-signal frame, so it cannot be
// reused here — a fixture that silently drops the frame under test would make
// this pass vacuously.
func moveStopServer(t *testing.T) (*ntwire.TCPServer, *store.Store, net.Conn, chan ntwire.MoveStopPayload) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "ms.db"))
	if err != nil {
		t.Fatal(err)
	}
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop(); cancel(); _ = st.Close() })

	conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	moves := make(chan ntwire.MoveStopPayload, 4)
	go func() {
		for {
			env, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			if env.Type != ntwire.FrameMoveStop {
				continue
			}
			var p ntwire.MoveStopPayload
			if json.Unmarshal(env.Payload, &p) == nil {
				moves <- p
			}
		}
	}()
	return s, st, conn, moves
}

// The owner's pin, applied to the move_stop path: a command composed while the
// bar cache is 30 minutes stale must still carry a wall-clock stamp the AddOn
// would measure in milliseconds.
func TestMoveStopStampIsWallClockNotBarClock(t *testing.T) {
	s, st, conn, moves := moveStopServer(t)

	if err := ntwire.WriteFrame(conn, ntwire.FrameHeartbeat, ntwire.HeartbeatPayload{BuildID: ntwire.MinAddonBuildStopSlot}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for s.FarSideBuildID() == "" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	tr := NewTCPTrader(s, "MNQ", "Sim101")
	tr.StartCloseSync("clock-trader", "fixture", "ninjatrader", st)

	// THE FIXTURE: a bar cache frozen 31 minutes ago — the 2026-09-07 shape,
	// where a 32.4-minute gap made every stamp read 1824.5s old to NT8.
	bar := ntwire.Bar{T: time.Now().Add(-31 * time.Minute).UnixMilli(), O: 29600, H: 29602, L: 29598, C: 29601, V: 10}
	s.BarCache().SeedHistorical("MNQ", "1m", []ntwire.Bar{bar})
	if time.Since(tr.feedNowUTC("MNQ")) < 29*time.Minute {
		t.Fatal("fixture lacks an old market clock — the test would pass vacuously")
	}

	tr.rememberEntryOrderID("MNQ", "long", "sig-movestop")

	before := time.Now().UTC().Add(-time.Second)
	if err := tr.MoveStopToBreakeven("long", 29590); err != nil {
		t.Fatalf("move stop: %v", err)
	}

	select {
	case p := <-moves:
		got, err := time.Parse(time.RFC3339, p.Timestamp)
		if err != nil {
			t.Fatalf("timestamp %q is not RFC3339: %v", p.Timestamp, err)
		}
		age := time.Since(got)
		if age > 30*time.Second {
			t.Errorf("move_stop stamp is %s old with a 31-minute-stale bar cache — it is on the BAR clock, not the wall clock.\n  stamp %s\n  now   %s\nNT8 ages this against DateTime.UtcNow and would reject it the moment a freshness guard exists.",
				age.Round(time.Second), got.Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
		}
		if got.Before(before) {
			t.Errorf("stamp %s predates the call (%s) — it cannot be a creation time", got, before)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no move_stop frame arrived")
	}
}

// The two stamps in this file must agree on what "timestamp" means. Before this
// wave MoveStopPayload used the bar clock while PlaceProtectiveStopPayload two
// hundred lines below used the wall clock — one file saying two different
// things, which is how the next reader learns the wrong convention.
func TestOutboundCommandStampsAgreeOnTheirClock(t *testing.T) {
	raw, err := os.ReadFile("tcp_trader.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	src := string(raw)
	// Any outgoing payload field stamped from the bar clock, in either gofmt
	// alignment, fails this.
	if strings.Contains(src, "Timestamp:   t.feedNowUTC(") ||
		strings.Contains(src, "Timestamp:  t.feedNowUTC(") ||
		strings.Contains(src, "Timestamp: t.feedNowUTC(") {
		t.Errorf("an outgoing command is still stamped from the BAR clock — every command timestamp is a creation time and must be the wall clock")
	}
}
