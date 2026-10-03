package store

import (
	"encoding/json"
	"strings"
	"testing"
)

// Item A (crypto removal): the hyper_* coin-source schema fields are gone, but
// stored rows carrying the legacy keys must still load — unknown JSON fields are
// ignored, never a load failure — and the kept fields survive intact. The codec
// must not re-emit the removed keys.
func TestLegacyStrategyRowWithHyperCoinSourceKeysStillLoads(t *testing.T) {
	blob := []byte(`{"strategy_type":"ai_trading","language":"en","ai_config":{"coin_source":{
		"source_type": "static",
		"static_coins": ["MNQ"],
		"use_hyper_all": true,
		"use_hyper_main": false,
		"hyper_main_limit": 20
	}}}`)
	var cfg StrategyConfig
	if err := json.Unmarshal(blob, &cfg); err != nil {
		t.Fatalf("legacy stored row must load (unknown hyper_* keys ignored), got: %v", err)
	}
	if cfg.CoinSource.SourceType != "static" || len(cfg.CoinSource.StaticCoins) != 1 || cfg.CoinSource.StaticCoins[0] != "MNQ" {
		t.Fatalf("kept coin-source fields must survive the legacy row: %+v", cfg.CoinSource)
	}

	out, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal after legacy load: %v", err)
	}
	for _, key := range []string{"use_hyper_all", "use_hyper_main", "hyper_main_limit"} {
		if strings.Contains(string(out), key) {
			t.Fatalf("the removed hyper_* key must not be re-emitted (%s): %s", key, out)
		}
	}
}

// Plan C1 (crypto removal): every legacy coin source type degrades to "static"
// at load. The degrade switch must cover hyper_all / hyper_main / mixed (and the
// pre-existing ai500 / oi_top / oi_low), otherwise a stored legacy row reaches
// the engine as an unknown source_type instead of static. Mutant guard: dropping
// "mixed" from the case must make this test fail.
func TestLegacyCoinSourceTypesDegradeToStatic(t *testing.T) {
	for _, legacy := range []string{"hyper_all", "hyper_main", "mixed", "ai500", "oi_top", "oi_low"} {
		blob := []byte(`{"strategy_type":"ai_trading","ai_config":{"coin_source":{
			"source_type": "` + legacy + `",
			"static_coins": ["MNQ"]
		}}}`)
		var cfg StrategyConfig
		if err := json.Unmarshal(blob, &cfg); err != nil {
			t.Fatalf("legacy row with source_type %q must load, got: %v", legacy, err)
		}
		cfg.NormalizeProductSchema()
		if cfg.CoinSource.SourceType != "static" {
			t.Fatalf("legacy source_type %q must degrade to static, got %q", legacy, cfg.CoinSource.SourceType)
		}
	}
}
