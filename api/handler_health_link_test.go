package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vl/manager"
	ntwire "vl/provider/ninjatrader"
	"vl/trader"
	ntTrader "vl/trader/ninjatrader"
)

// TestHealthLinkFollowsRealSocket is the P-D stale-link-latch RED test, at the
// production call site (handleHealth → manager roster → AutoTrader surface →
// TCPTrader → TCPServer socket). Tonight's defect: a latched edge-triggered
// feed_status frame ("Connected") kept /api/health nt8[<id>].link = "up" for
// hours after the :36974 socket was gone — the link field never consulted the
// real socket. It must read "down" once the socket is closed, while the
// last-known feed_status stays visible (honest, not fabricated).
func TestHealthLinkFollowsRealSocket(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
	if err := ntwire.WriteFrame(client, ntwire.FrameFeedStatus, ntwire.FeedStatusPayload{
		PriceStatus: "Connected",
		Time:        time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("write feed_status: %v", err)
	}
	waitFor(t, 2*time.Second, func() bool { return srv.FeedStatus() == "Connected" })

	// NT8 goes away: the socket closes. The edge-triggered feed_status stays
	// latched at "Connected" — the latch this test exists to fence.
	_ = client.Close()
	waitFor(t, 2*time.Second, func() bool { return !srv.IsConnected() })

	tcpT := ntTrader.NewTCPTrader(srv, "MNQ", "Sim101")
	at := trader.NewAutoTraderOnBrokerForTest("t-nt8", "ninjatrader", tcpT, true)
	tm := manager.NewTraderManager()
	tm.SetTraderForTest("t-nt8", at)

	s := &Server{traderManager: tm}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/health", nil)
	s.handleHealth(c)

	if w.Code != http.StatusOK {
		t.Fatalf("health = %d, want 200", w.Code)
	}
	var body struct {
		NT8 map[string]struct {
			Link       string `json:"link"`
			FeedStatus string `json:"feed_status"`
		} `json:"nt8"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("health body not JSON: %v", err)
	}
	info, ok := body.NT8["t-nt8"]
	if !ok {
		t.Fatalf("nt8[t-nt8] missing from health payload: %+v", body.NT8)
	}
	if info.Link != "down" {
		t.Fatalf("nt8[t-nt8].link must read down with the socket closed (P-D latch), got %q", info.Link)
	}
	if info.FeedStatus != "Connected" {
		t.Fatalf("last-known feed_status must stay visible, got %q", info.FeedStatus)
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
