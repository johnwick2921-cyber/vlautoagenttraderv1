package mcp

import (
	"testing"
)

// TestDeepSeekThinkingDefaults pins the 2026-08-22 default: DeepSeek requests
// carry thinking {type: enabled} + reasoning_effort max (the docs' true maximum)
// by default, and the keys are provider-scoped (OpenAI untouched) + overridable
// via env (empty = omit).
func TestDeepSeekThinkingDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ThinkingMode != "enabled" {
		t.Fatalf("default thinking mode = %q, want enabled", cfg.ThinkingMode)
	}
	if cfg.ReasoningEffort != "max" {
		t.Fatalf("default reasoning effort = %q, want max", cfg.ReasoningEffort)
	}

	ds := &Client{Provider: ProviderDeepSeek, Cfg: cfg, Model: "deepseek-v4-pro"}
	body := ds.BuildRequestBodyFromRequest(&Request{
		Model:    "deepseek-v4-pro",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	think, ok := body["thinking"].(map[string]any)
	if !ok || think["type"] != "enabled" {
		t.Fatalf("deepseek body missing thinking {type: enabled}: %+v", body["thinking"])
	}
	if body["reasoning_effort"] != "max" {
		t.Fatalf("deepseek body reasoning_effort = %v, want max", body["reasoning_effort"])
	}

	// Provider-scoped: OpenAI gets neither.
	oa := &Client{Provider: ProviderOpenAI, Cfg: cfg, Model: "gpt-x"}
	oaBody := oa.BuildRequestBodyFromRequest(&Request{
		Model:    "gpt-x",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if _, exists := oaBody["thinking"]; exists {
		t.Fatalf("openai body must not carry thinking")
	}
	if _, exists := oaBody["reasoning_effort"]; exists {
		t.Fatalf("openai body must not carry reasoning_effort")
	}

	// Empty strings omit the keys entirely (provider default in force).
	off := &Client{Provider: ProviderDeepSeek, Cfg: &Config{}, Model: "deepseek-v4-pro"}
	offBody := off.BuildRequestBodyFromRequest(&Request{
		Model:    "deepseek-v4-pro",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if _, exists := offBody["thinking"]; exists {
		t.Fatalf("empty cfg must omit thinking")
	}
	if _, exists := offBody["reasoning_effort"]; exists {
		t.Fatalf("empty cfg must omit reasoning_effort")
	}
}
