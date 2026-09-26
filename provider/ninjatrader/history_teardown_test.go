package ninjatrader

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

// F11 (port of #117 2f4db4f3) — history delivery must survive subscription
// teardown. Exercises real frame dispatch over loopback while the importer
// tears down and replaces its subscription. The panic the fix removes (send on
// a channel closed between RUnlock and the send) is a timing interleaving:
// this pin proves delivery stays correct under the churn; the interleaving
// itself is a -race reproduction (L16 — the race slot is the CTO's).
func TestHistoryFrameDeliveryConcurrentTeardown(t *testing.T) {
	srv := NewTCPServer(nil)
	srv.SetAddrForTest("127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()
	conn, err := net.Dial("tcp", srv.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	srv.SubscribeBarsHistoryFor("race")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20000; i++ {
			srv.UnsubscribeBarsHistoryFor("race")
			srv.SubscribeBarsHistoryFor("race")
		}
	}()
	for i := 0; i < 1000; i++ {
		if err := WriteFrame(conn, FrameBarsHistoryData, BarsHistoryDataPayload{RequestID: "race", Contract: "MNQ 09-26"}); err != nil {
			t.Error(err)
			break
		}
		if err := WriteFrame(conn, FrameBarsHistoryError, BarsHistoryErrorPayload{RequestID: "race", Contract: "MNQ 09-26", Reason: "fixture"}); err != nil {
			t.Error(err)
			break
		}
	}
	wg.Wait()
	data, _ := srv.SubscribeBarsHistoryFor("sentinel")
	if err := WriteFrame(conn, FrameBarsHistoryData, BarsHistoryDataPayload{RequestID: "sentinel", Contract: "MNQ 09-26"}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-data:
		if got.RequestID != "sentinel" {
			t.Fatal(got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("read loop did not deliver sentinel")
	}
}
