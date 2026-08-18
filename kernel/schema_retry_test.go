// B2a — schema-strict bounded retry: a malformed/unparseable AI response is
// retried (parse error fed back) up to maxRetries, then the caller skips the cycle.
package kernel

import (
	"errors"
	"strings"
	"testing"
	"time"

	"nofx/mcp"
)

type mockAIClient struct {
	replies []string // sequential responses; last one repeats
	calls   int
}

func (m *mockAIClient) CallWithMessages(_, _ string) (string, error) {
	i := m.calls
	if i >= len(m.replies) {
		i = len(m.replies) - 1
	}
	m.calls++
	return m.replies[i], nil
}
func (m *mockAIClient) SetAPIKey(string, string, string)             {}
func (m *mockAIClient) SetTimeout(time.Duration)                     {}
func (m *mockAIClient) ResolvedModel() string                        { return "mock-model" }
func (m *mockAIClient) CallWithRequest(*mcp.Request) (string, error) { return "", nil }
func (m *mockAIClient) CallWithRequestStream(*mcp.Request, func(string)) (string, error) {
	return "", nil
}
func (m *mockAIClient) CallWithRequestFull(*mcp.Request) (*mcp.LLMResponse, error) { return nil, nil }

func TestCallWithSchemaRetry(t *testing.T) {
	parse := func(s string) (*FullDecision, error) {
		if strings.Contains(s, "BAD") {
			return &FullDecision{}, errors.New("bad schema")
		}
		return &FullDecision{Decisions: []Decision{{Action: "wait"}}}, nil
	}

	// Malformed twice, then good on the 3rd → success, exactly 3 calls.
	m := &mockAIClient{replies: []string{"BAD", "BAD", "GOOD"}}
	d, _, _, perr, cerr := callWithSchemaRetry(m, "sys", "user", parse, 2)
	if cerr != nil || perr != nil {
		t.Fatalf("should succeed by the 3rd attempt: perr=%v cerr=%v", perr, cerr)
	}
	if m.calls != 3 {
		t.Fatalf("expected 3 AI calls, got %d", m.calls)
	}
	if d == nil || len(d.Decisions) != 1 {
		t.Fatal("expected the good decision returned")
	}

	// Always malformed → parseErr after exactly 3 attempts (bounded 2 retries).
	m2 := &mockAIClient{replies: []string{"BAD"}}
	_, _, _, perr2, cerr2 := callWithSchemaRetry(m2, "sys", "user", parse, 2)
	if cerr2 != nil {
		t.Fatalf("no transport error expected: %v", cerr2)
	}
	if perr2 == nil {
		t.Fatal("always-bad must return a parse error (caller then skips the cycle)")
	}
	if m2.calls != 3 {
		t.Fatalf("bounded retry: expected exactly 3 attempts, got %d", m2.calls)
	}

	// Good on the first attempt → exactly 1 call.
	m3 := &mockAIClient{replies: []string{"GOOD"}}
	_, _, _, perr3, _ := callWithSchemaRetry(m3, "sys", "user", parse, 2)
	if perr3 != nil || m3.calls != 1 {
		t.Fatalf("happy path: 1 call, no error; got calls=%d perr=%v", m3.calls, perr3)
	}
}

// errClient is a tiny AIClient whose calls fail from attempt N on.
type errClient struct {
	first string
	calls int
	fail  bool
}

func (e *errClient) SetAPIKey(_, _, _ string)                         {}
func (e *errClient) SetTimeout(_ time.Duration)                       {}
func (e *errClient) ResolvedModel() string                            { return "stub" }
func (e *errClient) CallWithRequest(_ *mcp.Request) (string, error)   { return "", nil }
func (e *errClient) CallWithRequestStream(_ *mcp.Request, _ func(string)) (string, error) {
	return "", nil
}
func (e *errClient) CallWithRequestFull(_ *mcp.Request) (*mcp.LLMResponse, error) {
	return nil, nil
}
func (e *errClient) CallWithMessages(_, _ string) (string, error) {
	if e.fail {
		return "", errors.New("call timeout")
	}
	e.fail = true
	return e.first, nil
}

// TestSchemaRetryPreservesLastOutput — P5.5: when a later retry ERRORS (e.g. the
// 180s cap), the last non-empty model output must survive into the record, so
// the owner can see what was actually lost instead of an empty raw_response.
func TestSchemaRetryPreservesLastOutput(t *testing.T) {
	first := "reasoning prose with no decision json anywhere"
	client := &errClient{first: first}
	alwaysFail := func(resp string) (*FullDecision, error) {
		return nil, errors.New("no decision json")
	}
	dec, raw, _, parseErr, callErr := callWithSchemaRetry(client, "sys", "usr", alwaysFail, 1)
	if dec != nil || callErr == nil || parseErr != nil {
		t.Fatalf("expected call error state, got dec=%v parseErr=%v callErr=%v", dec, parseErr, callErr)
	}
	if raw != first {
		t.Fatalf("last non-empty output lost: got %q want %q", raw, first)
	}
}
