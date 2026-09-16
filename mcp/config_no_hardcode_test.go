package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

// P0 2026-08-19 — guard against the silent-default disease (max_tokens=2000
// hardcoded). Every AI parameter must flow from Cfg into the wire body, and
// nothing may be hardcoded at assembly time.
func TestBuildMCPRequestBodyReflectsConfig(t *testing.T) {
	client := NewClient(func(c *Config) {
		c.Provider = ProviderCustom
		c.Temperature = 0.9
		c.TopP = 0.8
		c.MaxTokens = 12345
	}).(*Client)
	body := client.BuildMCPRequestBody("sys", "user")
	if got := body["temperature"]; got != 0.9 {
		t.Fatalf("temperature = %v, want 0.9 (cfg must drive the wire)", got)
	}
	if got := body["max_tokens"]; got != 12345 {
		t.Fatalf("max_tokens = %v, want 12345 (cfg must drive the wire)", got)
	}
	if got := body["top_p"]; got != 0.8 {
		t.Fatalf("top_p = %v, want 0.8 (cfg must drive the wire)", got)
	}
	if body["model"] != client.Model {
		t.Fatalf("model = %v", body["model"])
	}
}

func TestBuildMCPRequestBodyOmitsTopPWhenUnset(t *testing.T) {
	client := NewClient(func(c *Config) {
		c.Temperature = 0.5
		c.MaxTokens = 8000
		c.TopP = 0
	}).(*Client)
	body := client.BuildMCPRequestBody("sys", "user")
	if _, ok := body["top_p"]; ok {
		t.Fatalf("top_p must be OMITTED when unset (0), got %v", body["top_p"])
	}
}

// TestBuildMCPRequestBodyDefaultsComeFromConfig asserts the default path is
// env/config-driven, not a literal. If someone reintroduces a hardcoded
// temperature or max_tokens literal at assembly time, this fails.
func TestBuildMCPRequestBodyDefaultsComeFromConfig(t *testing.T) {
	client := NewClient().(*Client)
	body := client.BuildMCPRequestBody("sys", "user")
	if got := body["temperature"]; got != DefaultConfig().Temperature {
		t.Fatalf("default temperature %v != DefaultConfig().Temperature %v", got, DefaultConfig().Temperature)
	}
	if got := body["max_tokens"]; got != DefaultConfig().MaxTokens {
		t.Fatalf("default max_tokens %v != DefaultConfig().MaxTokens %v", got, DefaultConfig().MaxTokens)
	}
}

// TestEffectiveAIParamsSnapshotExposesHiddenDefaults — the startup log depends
// on every knob reporting whether the operator explicitly set it.
func TestEffectiveAIParamsSnapshotExposesHiddenDefaults(t *testing.T) {
	snap := EffectiveAIParamsSnapshot("deepseek-v4-pro")
	if snap.Model == "" || snap.MaxTokens <= 0 || snap.TimeoutSeconds <= 0 {
		t.Fatalf("snapshot incomplete: %+v", snap)
	}
	// UNSET env (this test runs without AI_* set) must be flagged as unset,
	// so main.go can WARN. Verify by round-tripping into JSON.
	b, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "Set\":false") && !strings.Contains(s, `Set":false`) {
		t.Fatalf("expected at least one unset-default flag in %s", s)
	}
}
