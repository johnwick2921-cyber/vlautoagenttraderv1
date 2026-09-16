package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// W11 — AIConfigFingerprint is a stable, deterministic hash of ONLY the indicator
// config that reaches the futures prompt: the toggles, the configured periods, and
// the kline timeframes. It intentionally EXCLUDES the crypto ranking/quant fields
// and the NofxOSAPIKey secret — both to keep a secret out of a (possibly-logged)
// hash and because they never affect the futures prompt.
//
// Determinism: json.Marshal emits struct fields in declaration order; ,omitempty
// makes nil and empty slices identical; slice ORDER is preserved (it is
// semantically meaningful — periods/timeframes render verbatim). Logged on every
// plan row so a replay can prove which indicator config produced a plan.
func (c IndicatorConfig) AIConfigFingerprint() string {
	proj := struct {
		EMA          bool     `json:"enable_ema"`
		MACD         bool     `json:"enable_macd"`
		RSI          bool     `json:"enable_rsi"`
		ATR          bool     `json:"enable_atr"`
		BOLL         bool     `json:"enable_boll"`
		Vol          bool     `json:"enable_volume"`
		OI           bool     `json:"enable_oi"`
		Fund         bool     `json:"enable_funding_rate"`
		SVP          bool     `json:"enable_svp"`
		Raw          bool     `json:"enable_raw_klines"`
		EMAP         []int    `json:"ema_periods,omitempty"`
		RSIP         []int    `json:"rsi_periods,omitempty"`
		ATRP         []int    `json:"atr_periods,omitempty"`
		BOLLP        []int    `json:"boll_periods,omitempty"`
		PrimaryTF    string   `json:"primary_timeframe"`
		PrimaryCount int      `json:"primary_count"`
		LongerTF     string   `json:"longer_timeframe,omitempty"`
		LongerCount  int      `json:"longer_count,omitempty"`
		MultiTF      bool     `json:"enable_multi_timeframe"`
		SelectedTF   []string `json:"selected_timeframes,omitempty"`
	}{
		EMA: c.EnableEMA, MACD: c.EnableMACD, RSI: c.EnableRSI, ATR: c.EnableATR, BOLL: c.EnableBOLL,
		Vol: c.EnableVolume, OI: c.EnableOI, Fund: c.EnableFundingRate, SVP: c.EnableSVP, Raw: c.EnableRawKlines,
		EMAP: c.EMAPeriods, RSIP: c.RSIPeriods, ATRP: c.ATRPeriods, BOLLP: c.BOLLPeriods,
		PrimaryTF: c.Klines.PrimaryTimeframe, PrimaryCount: c.Klines.PrimaryCount,
		LongerTF: c.Klines.LongerTimeframe, LongerCount: c.Klines.LongerCount,
		MultiTF: c.Klines.EnableMultiTimeframe, SelectedTF: c.Klines.SelectedTimeframes,
	}
	b, _ := json.Marshal(proj)
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum[:8]) // 16-hex-char fingerprint
}
