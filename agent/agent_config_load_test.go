package agent

import (
	"encoding/json"
	"testing"
)

// TestStoredConfigIgnoresRemovedSentinelKeys proves that an agent config JSON
// persisted BEFORE the crypto removal wave still loads after the Sentinel
// removal: unknown keys (watch_symbols, enable_sentinel) must be ignored by
// encoding/json and the surviving fields must decode correctly.
func TestStoredConfigIgnoresRemovedSentinelKeys(t *testing.T) {
	raw := []byte(`{
		"language": "en",
		"watch_symbols": ["BTCUSDT", "ETHUSDT", "SOLUSDT"],
		"enable_sentinel": true,
		"enable_briefs": true,
		"enable_news": false,
		"allow_trade_execution": false,
		"brief_times": [8, 20]
	}`)
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("stored config with removed sentinel keys must still load: %v", err)
	}
	if cfg.Language != "en" {
		t.Errorf("language = %q, want en", cfg.Language)
	}
	if !cfg.EnableBriefs {
		t.Errorf("enable_briefs = false, want true")
	}
	if cfg.EnableNews {
		t.Errorf("enable_news = true, want false")
	}
	if cfg.AllowTradeExecution {
		t.Errorf("allow_trade_execution = true, want false")
	}
	if len(cfg.BriefTimes) != 2 {
		t.Errorf("brief_times = %v, want [8 20]", cfg.BriefTimes)
	}
}
