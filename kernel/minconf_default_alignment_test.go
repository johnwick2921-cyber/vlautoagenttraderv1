package kernel

import (
	"strings"
	"testing"

	"nofx/store"
)

// 6.1 (final-bundle 2026-08-19) — ONE min-confidence default, and the active
// trader's behavior is provably unchanged by the alignment.

// A strategy that explicitly stores 60 (the live MNQ strategy) produces the
// IDENTICAL gate threshold and prompt line before/after the constant change.
func TestStored60UnchangedByAlignment(t *testing.T) {
	cfg := &store.StrategyConfig{}
	cfg.RiskControl.MinConfidence = 60
	cfg.ClampLimits()
	if cfg.RiskControl.MinConfidence != 60 {
		t.Fatalf("stored 60 must survive ClampLimits untouched (got %d)", cfg.RiskControl.MinConfidence)
	}
	e := &StrategyEngine{config: cfg}
	prompt := e.buildFuturesPrompt("MNQ", 50_000, "")
	if !strings.Contains(prompt, "Min confidence to open: 60") {
		t.Error("futures prompt must state the stored 60 verbatim")
	}
}

// An UNSET strategy now gets ONE number everywhere: the clamp default and the
// futures-prompt default are the same constant (60) — the told-60/judged-65
// divergence (PR #54) is structurally impossible.
func TestUnsetStrategyOneDefaultEverywhere(t *testing.T) {
	if store.SafeDefaultMinConfidence != 60 {
		t.Fatalf("owner ruling: the shared default is 60 (got %d)", store.SafeDefaultMinConfidence)
	}
	cfg := &store.StrategyConfig{}
	cfg.ClampLimits() // unset (0) → the shared default
	if cfg.RiskControl.MinConfidence != store.SafeDefaultMinConfidence {
		t.Fatalf("clamp default = %d, want the shared constant %d", cfg.RiskControl.MinConfidence, store.SafeDefaultMinConfidence)
	}
	e := &StrategyEngine{config: &store.StrategyConfig{}} // prompt sees raw 0 → its default
	prompt := e.buildFuturesPrompt("MNQ", 50_000, "")
	if !strings.Contains(prompt, "Min confidence to open: 60") {
		t.Error("futures prompt default must be the same shared 60")
	}
}
