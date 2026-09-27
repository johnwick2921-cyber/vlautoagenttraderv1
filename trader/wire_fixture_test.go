package trader

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
)

// ── W-EXEC-TRUTH W0b (CTO M7) — a wire fixture waits for the server to
// REGISTER the dialed client before any producer runs ────────────────────────
//
// net.Dial returns when the kernel completes the TCP handshake; the server
// registers the client later, on its accept loop (TCPServer.acceptLoop sets
// s.conn under connMu). Every immediate send (close_position, cancel_order,
// move_stop) reads s.conn and fails with "no NT client connected" before
// then. A fixture that dials and returns therefore races its own producer:
// CI's 2-core runner lost the race in TestReconcileStillFlattensAnUnexplainedShort
// (job 107165325139 — the flatten went out before "client connected", 3 s
// later the test failed), while every fast box won it.

// waitAddonRegistered blocks, bounded, until s has registered its client —
// the same state every send reads. Fail-closed: a fixture that cannot prove
// the client is registered stops the test instead of racing it.
func waitAddonRegistered(t testing.TB, s *ntwire.TCPServer) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !s.IsConnected() {
		if time.Now().After(deadline) {
			t.Fatalf("fixture: the server never registered the dialed AddOn client (5s)")
		}
		time.Sleep(time.Millisecond)
	}
}

// wireFixtureDialAllowlist names the dials that intentionally do NOT wait:
// each drives no producer after the dial.
var wireFixtureDialAllowlist = map[string]string{
	"maintenance_drop_test.go:noSignalOnReconnect": "reads with a deadline to assert that NOTHING arrives on a fresh client; no producer runs after the dial",
	"maintenance_drop_test.go:dialRaw":             "returns a bare conn to tests that assert on the server's reaction to the dial itself",
}

// The source guard: every net.Dial in this package's tests is followed,
// within a few lines, by waitAddonRegistered — or is allowlisted with a
// reason. A new fixture that dials and returns fails here, not on CI.
func TestWireFixturesWaitForTheServerToRegisterTheClient(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	funcRe := regexp.MustCompile(`^func (?:\([^)]*\) )?(\w+)\(`)
	checked := 0
	for _, f := range files {
		if f == "wire_fixture_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(b), "\n")
		fn := ""
		for i, l := range lines {
			if m := funcRe.FindStringSubmatch(l); m != nil {
				fn = m[1]
			}
			if !strings.Contains(l, "net.Dial(") {
				continue
			}
			checked++
			if _, ok := wireFixtureDialAllowlist[f+":"+fn]; ok {
				continue
			}
			found := false
			for j := i + 1; j < len(lines) && j <= i+8; j++ {
				if strings.Contains(lines[j], "waitAddonRegistered(") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s:%d (%s): net.Dial is not followed by waitAddonRegistered — the producer races the server's accept (CTO M7)", f, i+1, fn)
			}
		}
	}
	if checked < 10 {
		t.Fatalf("the guard found only %d dials — it is going vacuous", checked)
	}
}
