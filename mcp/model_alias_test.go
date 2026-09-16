package mcp

import "testing"

// W2 — model-id pinning: a provider alias resolves to the EXACT model string.
func TestDefaultModelForAlias(t *testing.T) {
	if got := DefaultModelForAlias("deepseek"); got != DefaultDeepSeekModel {
		t.Fatalf("deepseek → %q want %q", got, DefaultDeepSeekModel)
	}
	if got := DefaultModelForAlias("DeepSeek "); got != DefaultDeepSeekModel {
		t.Fatalf("alias resolution must be case/space-insensitive: %q", got)
	}
	if got := DefaultModelForAlias("qwen"); got != DefaultQwenModel {
		t.Fatalf("qwen → %q", got)
	}
	// an exact id is not an alias → "" (leave it as-is)
	if got := DefaultModelForAlias("deepseek-v4-pro"); got != "" {
		t.Fatalf("exact id must not map: got %q", got)
	}
}

func TestIsProviderAlias(t *testing.T) {
	if !IsProviderAlias("deepseek") || !IsProviderAlias("qwen") {
		t.Fatal("bare provider aliases must be detected")
	}
	if IsProviderAlias("deepseek-v4-pro") || IsProviderAlias("") {
		t.Fatal("exact ids (and empty) are not aliases")
	}
}

// A live *Client exposes its exact model via ResolvedModel().
func TestClientResolvedModel(t *testing.T) {
	c := NewAIClientByProvider("deepseek")
	if c == nil {
		t.Skip("deepseek client not constructible in this env")
	}
	if rm := c.ResolvedModel(); IsProviderAlias(rm) || rm == "" {
		t.Fatalf("ResolvedModel must be an exact string, got %q", rm)
	}
}
