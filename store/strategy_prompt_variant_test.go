package store

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestStrategyConfigPromptVariantRoundTrip proves the Phase-2 keystone
// persistence: prompt_variant survives Marshal → Unmarshal at the top level of
// StrategyConfig (the same JSON the FE saves and the DB stores), so the live
// loop can read back exactly what the owner picked. Also proves the back-compat
// case: an unset variant marshals to absent (omitempty) and unmarshals to "".
func TestStrategyConfigPromptVariantRoundTrip(t *testing.T) {
	// (1) A saved variant round-trips top-level.
	cfg := StrategyConfig{StrategyType: "ai_trading", PromptVariant: "aggressive"}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"prompt_variant":"aggressive"`) {
		t.Fatalf("expected top-level prompt_variant in JSON, got: %s", b)
	}
	var back StrategyConfig
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.PromptVariant != "aggressive" {
		t.Fatalf("round-trip PromptVariant = %q, want %q", back.PromptVariant, "aggressive")
	}

	// (2) Back-compat: no variant → omitted in JSON, "" after unmarshal (so the
	// live loop falls back to the venue rule — byte-identical to pre-Phase-2).
	none := StrategyConfig{StrategyType: "ai_trading"}
	nb, _ := json.Marshal(none)
	if strings.Contains(string(nb), "prompt_variant") {
		t.Fatalf("unset variant must be omitted (omitempty), got: %s", nb)
	}

	// (3) An old config JSON with no prompt_variant field unmarshals to "".
	var legacy StrategyConfig
	if err := json.Unmarshal([]byte(`{"strategy_type":"ai_trading","ai_config":{}}`), &legacy); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if legacy.PromptVariant != "" {
		t.Fatalf("legacy config PromptVariant = %q, want empty", legacy.PromptVariant)
	}
}
