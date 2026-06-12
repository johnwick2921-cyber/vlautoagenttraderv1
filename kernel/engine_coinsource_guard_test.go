package kernel

import (
	"testing"

	"nofx/store"
)

// TestGetCandidateCoins_EmptySourceTypeDefaultsToStatic verifies the additive
// guard in GetCandidateCoins: a strategy whose stored config omitted
// ai_config.coin_source (deserializing to an empty SourceType) must NOT hit the
// "unknown coin source type" error that previously made the trader loop skip
// every cycle. An empty SourceType is treated as "static" so StaticCoins are
// still used. This is the regression test for the 2026-05-30 coin-source fix.
func TestGetCandidateCoins_EmptySourceTypeDefaultsToStatic(t *testing.T) {
	e := &StrategyEngine{config: &store.StrategyConfig{
		CoinSource: store.CoinSourceConfig{
			SourceType:  "", // the bug: created without a coin_source
			StaticCoins: []string{"MNQ"},
		},
	}}

	got, err := e.GetCandidateCoins()
	if err != nil {
		t.Fatalf("empty SourceType must not error (got %v)", err)
	}
	if len(got) != 1 || got[0].Symbol != "MNQ" {
		t.Fatalf("expected [MNQ] from static fallback, got %+v", got)
	}
}

// TestGetCandidateCoins_EmptySourceTypeNoStaticCoins verifies the degenerate
// case: empty SourceType AND no StaticCoins degrades to an empty candidate list
// (handled upstream as "no candidates, skip cycle") rather than a hard error.
func TestGetCandidateCoins_EmptySourceTypeNoStaticCoins(t *testing.T) {
	e := &StrategyEngine{config: &store.StrategyConfig{
		CoinSource: store.CoinSourceConfig{SourceType: ""},
	}}

	got, err := e.GetCandidateCoins()
	if err != nil {
		t.Fatalf("empty SourceType + no static coins must not error (got %v)", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty candidate list, got %+v", got)
	}
}
