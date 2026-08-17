package trader

import (
	"testing"
	"time"

	"nofx/mcp"
)

// P0-latency — the two halves that guarantee a decision can never be ACTED ON
// after its bar has closed: the call timeout (fits inside the bar window) and
// the stale-bar discard (a decision spanning a bar close is dropped, never
// executed). These are the seams the runCycle wiring depends on; the wiring
// itself is exercised by the existing cycle tests.

func TestStaleBarDiscard(t *testing.T) {
	cases := []struct {
		name    string
		decBar  int64
		latest  int64
		haveBar bool
		want    bool
	}{
		{"same bar still latest", 100, 100, true, false},
		{"newer bar closed during call", 100, 200, true, true},
		{"no bars provider down", 100, 0, false, false},
		{"no decision bar captured (crypto)", 0, 200, true, false},
		{"clock skew backwards never discards", 200, 100, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := staleBarDiscard(c.decBar, c.latest, c.haveBar); got != c.want {
				t.Fatalf("staleBarDiscard(%d, %d, %v) = %v, want %v", c.decBar, c.latest, c.haveBar, got, c.want)
			}
		})
	}
}

// fakeDecisionClient records SetTimeout and stubs the rest of mcp.AIClient.
type fakeDecisionClient struct {
	timeout time.Duration
	set     bool
}

func (f *fakeDecisionClient) SetAPIKey(string, string, string) {}
func (f *fakeDecisionClient) SetTimeout(d time.Duration)       { f.timeout = d; f.set = true }
func (f *fakeDecisionClient) ResolvedModel() string            { return "test-model" }
func (f *fakeDecisionClient) CallWithMessages(string, string) (string, error) {
	return "", nil
}
func (f *fakeDecisionClient) CallWithRequest(*mcp.Request) (string, error) { return "", nil }
func (f *fakeDecisionClient) CallWithRequestStream(*mcp.Request, func(string)) (string, error) {
	return "", nil
}
func (f *fakeDecisionClient) CallWithRequestFull(*mcp.Request) (*mcp.LLMResponse, error) {
	return nil, nil
}

func TestApplyDecisionCallTimeout(t *testing.T) {
	// Futures: the decision client is capped so a call fits inside the primary
	// bar window (180s of a 5-minute bar leaves ≥120s for the rest of the cycle).
	nt := &fakeDecisionClient{}
	applyDecisionCallTimeout(nt, "ninjatrader")
	if !nt.set || nt.timeout != decisionCallTimeout {
		t.Fatalf("ninjatrader client must be capped at decisionCallTimeout (%v), got set=%v timeout=%v",
			decisionCallTimeout, nt.set, nt.timeout)
	}

	// Crypto: byte-identical to the pre-change behavior (no cap).
	bn := &fakeDecisionClient{}
	applyDecisionCallTimeout(bn, "binance")
	if bn.set {
		t.Fatalf("crypto client must NOT be capped, got timeout=%v", bn.timeout)
	}

	// nil client is a no-op, never a panic.
	applyDecisionCallTimeout(nil, "ninjatrader")

	// The cap really fits inside the 5-minute primary bar window.
	if decisionCallTimeout >= 5*time.Minute {
		t.Fatalf("decisionCallTimeout %v must be smaller than the 5m bar window", decisionCallTimeout)
	}
}
