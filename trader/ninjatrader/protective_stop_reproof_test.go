package ninjatrader

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
)

// dialForTest re-dials the single-client slot after a disconnect (the slot
// frees asynchronously when the read loop notices the close).
func dialForTest(t *testing.T, s *ntwire.TCPServer) net.Conn {
	t.Helper()
	var c net.Conn
	var err error
	for i := 0; i < 200; i++ {
		c, err = net.Dial("tcp", s.ListenAddrForTest().String())
		if err == nil {
			return c
		}
		if c != nil {
			_ = c.Close()
			c = nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("re-dial never succeeded: %v", err)
	return nil
}

// TestProtectiveStopRefusedAfterDisconnectThenPlacedAfterReproof is the CTO
// safety-gap proof for FIX-P1A: clearing farSideBuild on disconnect makes
// PlaceProtectiveStop fail closed while the new connection has not proven its
// AddOn build — and the D5 reconciler's NEXT 1-minute pass re-attempts, so the
// position is never left permanently naked once the hello re-proves the build.
//
// Production call sites: TCPTrader.PlaceProtectiveStop over a real loopback
// server; the refusal is the far-side-build guard (tcp_trader.go ~1121), the
// re-proof is the same hello handler the real AddOn uses.
func TestProtectiveStopRefusedAfterDisconnectThenPlacedAfterReproof(t *testing.T) {
	s, _, conn1, _ := stopEntryServer(t)
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	// Prove the far side with the ORIGINAL connection's hello.
	if err := ntwire.WriteFrame(conn1, ntwire.FrameHello, ntwire.HelloPayload{
		ProtocolVersion: ntwire.ProtocolVersion,
		Source:          "vltrader-addon",
		BuildID:         ntwire.MinAddonBuildProtectiveStop,
	}); err != nil {
		t.Fatal(err)
	}
	waitProven := func(want bool) bool {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			proven := ntwire.FarSideProven(s.FarSideBuildID(), ntwire.MinAddonBuildProtectiveStop)
			if proven == want {
				return true
			}
			time.Sleep(5 * time.Millisecond)
		}
		return false
	}
	if !waitProven(true) {
		t.Fatalf("initial hello never proved the build: %q", s.FarSideBuildID())
	}

	// Baseline: while proven, the protective stop goes to the wire (nil error).
	if err := tr.PlaceProtectiveStop("MNQ", "long", 1, 29600, "sig-base", "test"); err != nil {
		t.Fatalf("baseline placement refused while proven: %v", err)
	}

	// DISCONNECT: the proof is retired by closeConn (FIX-P1A).
	_ = conn1.Close()
	deadline := time.Now().Add(2 * time.Second)
	for s.FarSideBuildID() != "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if s.FarSideBuildID() != "" {
		t.Fatalf("proof survived the disconnect: %q", s.FarSideBuildID())
	}

	// A protective stop requested BEFORE the new hello → REFUSED (fail closed).
	err := tr.PlaceProtectiveStop("MNQ", "long", 1, 29580, "sig-gap", "test")
	if err == nil {
		t.Fatal("protective stop placed while the far side was unproven")
	}
	// TRACEABILITY PIN (B-rules, FIX-P1A): build "" after a disconnect is
	// "not proven since reconnect", NOT "build too old". A mutation that folds
	// this back into ErrAddonBuildTooOld — or drops the reason text — fails
	// the two assertions below.
	if !errors.Is(err, ntwire.ErrFarSideNotProven) {
		t.Fatalf("want ErrFarSideNotProven (the proof was retired by the disconnect), got %v", err)
	}
	if errors.Is(err, ntwire.ErrAddonBuildTooOld) {
		t.Fatalf("a link-down refusal must NOT read as addon-build-too-old: %v", err)
	}
	if !strings.Contains(err.Error(), "far side not proven since reconnect") {
		t.Fatalf("refusal must name the true reason: %v", err)
	}
	if !strings.Contains(err.Error(), "guard=far_side_build") {
		t.Fatalf("refusal error does not name the guard: %v", err)
	}

	// New connection + hello re-proves the build.
	conn2 := dialForTest(t, s)
	defer func() { _ = conn2.Close() }()
	if err := ntwire.WriteFrame(conn2, ntwire.FrameHello, ntwire.HelloPayload{
		ProtocolVersion: ntwire.ProtocolVersion,
		Source:          "vltrader-addon",
		BuildID:         ntwire.MinAddonBuildProtectiveStop,
	}); err != nil {
		t.Fatal(err)
	}
	if !waitProven(true) {
		t.Fatalf("reconnect hello never re-proved the build: %q", s.FarSideBuildID())
	}

	// The NEXT protection pass places it — no permanent naked position.
	if err := tr.PlaceProtectiveStop("MNQ", "long", 1, 29580, "sig-reproof", "test"); err != nil {
		t.Fatalf("placement after re-proof was refused: %v", err)
	}
}
