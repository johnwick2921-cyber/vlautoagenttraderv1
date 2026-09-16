package mcp

import (
	"testing"
	"time"
)

// F1a (LONDON-FORENSICS 2026-08-28) — ApplyMaxTokens overrides the completion
// cap for one call scope and restores it (the shared executor client keeps its
// own cap), and LastFinishReason is readable for truncation-aware diagnostics.
func TestApplyMaxTokensSetAndRestore(t *testing.T) {
	c := &Client{MaxTokens: 32768, Cfg: &Config{}}
	restore := ApplyMaxTokens(c, 65536)
	if c.MaxTokens != 65536 {
		t.Fatalf("MaxTokens = %d, want 65536 after apply", c.MaxTokens)
	}
	restore()
	if c.MaxTokens != 32768 {
		t.Fatalf("MaxTokens = %d, want 32768 restored", c.MaxTokens)
	}
	// A non-*Client implementation is a no-op (never panics).
	if fn := ApplyMaxTokens(fakeAIClient{}, 123); fn != nil {
		fn()
	}
	if got := LastFinishReason(c); got != "" {
		t.Fatalf("LastFinishReason = %q, want empty before any call", got)
	}
}

type fakeAIClient struct{}

func (fakeAIClient) SetAPIKey(apiKey, customURL, customModel string) {}
func (fakeAIClient) SetTimeout(timeout time.Duration)                {}
func (fakeAIClient) ResolvedModel() string                           { return "" }
func (fakeAIClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	return "", nil
}
func (fakeAIClient) CallWithRequest(req *Request) (string, error) { return "", nil }
func (fakeAIClient) CallWithRequestStream(req *Request, onChunk func(string)) (string, error) {
	return "", nil
}
func (fakeAIClient) CallWithRequestFull(req *Request) (*LLMResponse, error) {
	return nil, nil
}
func (fakeAIClient) String() string { return "fake" }
