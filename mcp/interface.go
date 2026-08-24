package mcp

import (
	"time"
)

// ClientEmbedder is implemented by provider types that embed *Client,
// allowing generic extraction of the underlying base client (e.g. for cloning).
type ClientEmbedder interface {
	BaseClient() *Client
}

// AIClient public AI client interface (for external use)
type AIClient interface {
	SetAPIKey(apiKey string, customURL string, customModel string)
	SetTimeout(timeout time.Duration)
	// ResolvedModel returns the EXACT model string this client will call (never a
	// provider alias). Used for model-id pinning on the day plan (§125).
	ResolvedModel() string
	CallWithMessages(systemPrompt, userPrompt string) (string, error)
	CallWithRequest(req *Request) (string, error)
	// CallWithRequestStream streams the LLM response via SSE.
	// onChunk is called with the full accumulated text so far (not raw deltas).
	// Returns the complete final text when done.
	CallWithRequestStream(req *Request, onChunk func(string)) (string, error)
	// CallWithRequestFull returns both text content and tool calls.
	// Use this when the request includes Tools — the LLM may respond with
	// either a plain text reply (LLMResponse.Content) or tool invocations
	// (LLMResponse.ToolCalls), but not both.
	CallWithRequestFull(req *Request) (*LLMResponse, error)
}

// ThinkingTuner lets a call site override the env-default DeepSeek thinking
// knobs with per-model values (4.5 API auto max). Empty strings keep the
// env-derived default (empty also means "omit from the request"). Only the
// base DeepSeek client implements it; other providers are unaffected.
type ThinkingTuner interface {
	SetThinking(mode, effort string)
}

// ApplyThinking pushes per-model DeepSeek thinking knobs onto any client that
// implements ThinkingTuner (a no-op for every other provider). Empty values
// keep the env-derived defaults — call it unconditionally with the model row's
// values, never with guessed strings.
func ApplyThinking(client AIClient, mode, effort string) {
	if client == nil {
		return
	}
	if tuner, ok := client.(ThinkingTuner); ok {
		tuner.SetThinking(mode, effort)
	}
}
