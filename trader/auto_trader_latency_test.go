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
	// Futures: the decision client is aligned with the ONE config-driven AI
	// timeout (env AI_HTTP_TIMEOUT_SECONDS → AI_TIMEOUT_SECONDS → 300s). The old
	// behavior pinned here was a hardcoded 180s that killed DeepSeek reads
	// mid-body once max_tokens was raised and reasoning calls ran 150s+.
	nt := &fakeDecisionClient{}
	applyDecisionCallTimeout(nt, "ninjatrader")
	if !nt.set || nt.timeout != mcp.ResolvedAITimeout() {
		t.Fatalf("ninjatrader client must carry the config-driven timeout (%v), got set=%v timeout=%v",
			mcp.ResolvedAITimeout(), nt.set, nt.timeout)
	}

	// The env var actually drives it — owner config wins over the default.
	t.Setenv("AI_HTTP_TIMEOUT_SECONDS", "451")
	nt2 := &fakeDecisionClient{}
	applyDecisionCallTimeout(nt2, "ninjatrader")
	if nt2.timeout != 451*time.Second {
		t.Fatalf("AI_HTTP_TIMEOUT_SECONDS=451 must reach the decision client, got %v", nt2.timeout)
	}

	// The legacy name still works when the canonical one is unset.
	t.Setenv("AI_HTTP_TIMEOUT_SECONDS", "")
	t.Setenv("AI_TIMEOUT_SECONDS", "452")
	nt3 := &fakeDecisionClient{}
	applyDecisionCallTimeout(nt3, "ninjatrader")
	if nt3.timeout != 452*time.Second {
		t.Fatalf("legacy AI_TIMEOUT_SECONDS must be honored, got %v", nt3.timeout)
	}

	// Crypto: byte-identical to the pre-change behavior (no cap).
	bn := &fakeDecisionClient{}
	applyDecisionCallTimeout(bn, "binance")
	if bn.set {
		t.Fatalf("crypto client must NOT be capped, got timeout=%v", bn.timeout)
	}

	// nil client is a no-op, never a panic.
	applyDecisionCallTimeout(nil, "ninjatrader")
}
