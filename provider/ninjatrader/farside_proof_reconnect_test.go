package ninjatrader

import (
	"testing"
	"time"
)

// waitFarSideBuild polls FarSideBuildID until it equals want or the deadline
// passes.
func waitFarSideBuild(t *testing.T, s *TCPServer, want string) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.FarSideBuildID() == want {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return s.FarSideBuildID() == want
}

// TestFarSideBuildClearedOnDisconnectAndReprovenOnReconnect is the RED test for
// FIX-P1A: the far-side AddOn build proof must be cleared when the connection
// dies (closeConn) and re-proven only by the NEW connection's own frames. The
// proof gates live safety behavior (stop-slot admission, protective-stop
// placement — trader/ninjatrader/tcp_trader.go:773,1121), so a stale build id
// surviving a reconnect would keep those gates green against a far side that
// no longer exists or was replaced by an older AddOn.
func TestFarSideBuildClearedOnDisconnectAndReprovenOnReconnect(t *testing.T) {
	s := startedServer(t, nil)

	// 1. First connection proves the far side with its hello build id.
	w1 := dialWire(t, s, HelloPayload{BuildID: "2026-09-23-m21"})
	if !waitFarSideBuild(t, s, "2026-09-23-m21") {
		t.Fatalf("build proof never arrived from hello: got %q", s.FarSideBuildID())
	}

	// 2. Disconnect must CLEAR the proof — a stale proof survives a reconnect
	//    today (closeConn only nils the conn).
	_ = w1.conn.Close()
	if !waitFarSideBuild(t, s, "") {
		t.Fatalf("far-side build proof survived the disconnect: %q", s.FarSideBuildID())
	}

	// 3. A new connection re-proves from ITS OWN hello.
	w2 := dialWire(t, s, HelloPayload{BuildID: "2026-09-23-m99"})
	if !waitFarSideBuild(t, s, "2026-09-23-m99") {
		t.Fatalf("new connection's hello did not re-prove the build: got %q", s.FarSideBuildID())
	}

	// 4. A reconnect whose hello carries NO build id must fail closed: the
	//    cleared proof must NOT be resurrected by any stale value.
	_ = w2.conn.Close()
	if !waitFarSideBuild(t, s, "") {
		t.Fatalf("proof survived the second disconnect: %q", s.FarSideBuildID())
	}
	_ = dialWire(t, s, HelloPayload{}) // hello without BuildID
	if waitFarSideBuild(t, s, "2026-09-23-m99") {
		t.Fatalf("buildless hello resurrected a proof: %q", s.FarSideBuildID())
	}
}
