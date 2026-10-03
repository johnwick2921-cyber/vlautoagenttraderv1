package trader

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	ntwire "vl/provider/ninjatrader"
	ntTrader "vl/trader/ninjatrader"
)

// TestDeskLinkFollowsRealSocket (P-D ruling): the owner-facing desk strip LINK
// must follow the REAL TCP socket — up only while a client is connected, down
// the moment the socket is gone, never a latched feed_status frame. The
// production call site is deskLinkStatus → HealthLinkConnected →
// TCPTrader.IsConnected.
func TestDeskLinkFollowsRealSocket(t *testing.T) {
	srv := ntwire.NewTCPServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.SetAddrForTest("127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("tcp server Start: %v", err)
	}
	defer srv.Stop()

	client, err := net.Dial("tcp", srv.ListenAddrForTest().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()
	waitAddonRegistered(t, srv) // CTO M7: the producer must not race the accept

	at := NewAutoTraderOnBrokerForTest("t-desk", "ninjatrader", ntTrader.NewTCPTrader(srv, "MNQ", "Sim101"), true)
	if link, known := at.deskLinkStatus(); !known || link != "up" {
		t.Fatalf("with a connected socket deskLinkStatus = (%q,%v), want (up,true)", link, known)
	}

	// The socket closes (NT8 gone). The link must read down.
	_ = client.Close()
	waitDesk(t, 2*time.Second, func() bool { return !srv.IsConnected() })
	if link, known := at.deskLinkStatus(); !known || link != "down" {
		t.Fatalf("with the socket gone deskLinkStatus = (%q,%v), want (down,true) — a latched feed_status must not read up", link, known)
	}

	// No NT8 trader at all: UNKNOWN, never fabricated.
	at2 := NewAutoTraderOnBrokerForTest("t-nocrypto", "ninjatrader", nil, true)
	if link, known := at2.deskLinkStatus(); known || link != "UNKNOWN (no NT8 link)" {
		t.Fatalf("with no NT8 trader deskLinkStatus = (%q,%v), want (UNKNOWN (no NT8 link),false)", link, known)
	}
}

func waitDesk(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
