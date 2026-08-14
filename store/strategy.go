package store

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"nofx/config"

	"gorm.io/gorm"
)

// SupportedTimeframes is the SINGLE SOURCE OF TRUTH for the Strategy Studio
// timeframe selector: every interval the engine can genuinely serve end-to-end.
// For NT8 futures these are exactly the timeframes the AddOn auto-subscribes and
// streams into the BarCache (provider/ninjatrader defaultAutoBarsTimeframes —
// kept in lockstep by TestDefaultAutoBarsTimeframes_MatchesSupported); the live
// BarCache serves all of them per-series (there is NO Go-side aggregation, so an
// interval outside this set — e.g. 2m — would yield empty futures klines). For
// crypto, CoinAnk serves the same standard intervals. The frontend fetches this
// list via GET /api/strategies/timeframes instead of hardcoding its own copy.
var SupportedTimeframes = []string{
	"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w",
}

// SoftWarnTimeframesAbove is the advisory threshold: selecting more than this
// many timeframes is allowed but warns (bigger AI prompt → slower, costlier
// decisions). It is NEVER a hard block (that was the old artificial max-4 cap).
const SoftWarnTimeframesAbove = 6

// MaxTimeframes caps SelectedTimeframes at the full supported capability. It was
// a hard 4 — an artificial limit that silently truncated configs on save; the AI
// prompt (formatKlineTimeframes) and the per-timeframe analysis loop both handle
// N timeframes. The clamp now only guards against absurd input beyond the real
// capability, never the legitimate 5-14 range.
var MaxTimeframes = len(SupportedTimeframes)

// Hard limits to prevent token explosion in AI requests
const (
	MaxCandidateCoins = 10
	MaxPositions      = 3
	MinKlineCount     = 10
	MaxKlineCount     = 30
	MinLeverage       = 1
	MaxBTCETHLeverage = 20
	MaxAltLeverage    = 20
	MinPositionRatio  = 0.5
	MaxPositionRatio  = 10.0
	MinRiskReward     = 1.0
	MaxRiskReward     = 10.0
	MinMarginUsage    = 0.1
	MaxMarginUsage    = 1.0
	MinPositionSize   = 10.0
	MaxPositionSize   = 1000.0
	MinConfidence     = 50
	MaxConfidence     = 100
)

// ClampLimits enforces product-level limits on strategy config to prevent token overflow.
// MaxIndicatorPeriod is the largest allowed indicator period. Beyond this a series
// can never warm up within the fetch/cache depth (2500 cap), so it would silently
// render an EMPTY series — reject at save instead (F11c).
const MaxIndicatorPeriod = 500

// ValidateIndicatorPeriods (F11c) rejects out-of-range indicator periods at save:
// each configured EMA/RSI/ATR/BOLL period must be > 0 and ≤ MaxIndicatorPeriod. An
// absurd value like 9999 is a hard error here, never a silently-empty series.
// Empty period lists are fine (they fall back to the legacy fixed periods).
func (c *StrategyConfig) ValidateIndicatorPeriods() error {
	check := func(name string, ps []int) error {
		for _, p := range ps {
			if p <= 0 || p > MaxIndicatorPeriod {
				return fmt.Errorf("%s period %d is out of range — each indicator period must be between 1 and %d", name, p, MaxIndicatorPeriod)
			}
		}
		return nil
	}
	for _, f := range []struct {
		name string
		ps   []int
	}{
		{"EMA", c.Indicators.EMAPeriods},
		{"RSI", c.Indicators.RSIPeriods},
		{"ATR", c.Indicators.ATRPeriods},
		{"BOLL", c.Indicators.BOLLPeriods},
	} {
		if err := check(f.name, f.ps); err != nil {
			return err
		}
	}
	return nil
}

func (c *StrategyConfig) ClampLimits() {
	c.NormalizeProductSchema()

	// Clamp coin source limits
	if c.CoinSource.AI500Limit > MaxCandidateCoins {
		c.CoinSource.AI500Limit = MaxCandidateCoins
	}
	if c.CoinSource.OITopLimit > MaxCandidateCoins {
		c.CoinSource.OITopLimit = MaxCandidateCoins
	}
	if c.CoinSource.OILowLimit > MaxCandidateCoins {
		c.CoinSource.OILowLimit = MaxCandidateCoins
	}

	// Clamp static coins
	if len(c.CoinSource.StaticCoins) > MaxCandidateCoins {
		c.CoinSource.StaticCoins = c.CoinSource.StaticCoins[:MaxCandidateCoins]
	}

	// Clamp kline count
	if c.Indicators.Klines.PrimaryCount < MinKlineCount {
		c.Indicators.Klines.PrimaryCount = MinKlineCount
	}
	if c.Indicators.Klines.PrimaryCount > MaxKlineCount {
		c.Indicators.Klines.PrimaryCount = MaxKlineCount
	}
	if c.Indicators.Klines.LongerCount > MaxKlineCount {
		c.Indicators.Klines.LongerCount = MaxKlineCount
	}

	// Clamp timeframes
	if len(c.Indicators.Klines.SelectedTimeframes) > MaxTimeframes {
		c.Indicators.Klines.SelectedTimeframes = c.Indicators.Klines.SelectedTimeframes[:MaxTimeframes]
	}

	// Clamp max positions
	if c.RiskControl.MaxPositions < 1 {
		c.RiskControl.MaxPositions = 1
	}
	if c.RiskControl.MaxPositions > MaxPositions {
		c.RiskControl.MaxPositions = MaxPositions
	}

	// Clamp leverage limits to the same bounds as the manual config UI.
	if c.RiskControl.BTCETHMaxLeverage < MinLeverage {
		c.RiskControl.BTCETHMaxLeverage = MinLeverage
	}
	if c.RiskControl.BTCETHMaxLeverage > MaxBTCETHLeverage {
		c.RiskControl.BTCETHMaxLeverage = MaxBTCETHLeverage
	}
	if c.RiskControl.AltcoinMaxLeverage < MinLeverage {
		c.RiskControl.AltcoinMaxLeverage = MinLeverage
	}
	if c.RiskControl.AltcoinMaxLeverage > MaxAltLeverage {
		c.RiskControl.AltcoinMaxLeverage = MaxAltLeverage
	}

	// Clamp position value ratio limits.
	if c.RiskControl.BTCETHMaxPositionValueRatio < MinPositionRatio {
		c.RiskControl.BTCETHMaxPositionValueRatio = MinPositionRatio
	}
	if c.RiskControl.BTCETHMaxPositionValueRatio > MaxPositionRatio {
		c.RiskControl.BTCETHMaxPositionValueRatio = MaxPositionRatio
	}
	if c.RiskControl.AltcoinMaxPositionValueRatio < MinPositionRatio {
		c.RiskControl.AltcoinMaxPositionValueRatio = MinPositionRatio
	}
	if c.RiskControl.AltcoinMaxPositionValueRatio > MaxPositionRatio {
		c.RiskControl.AltcoinMaxPositionValueRatio = MaxPositionRatio
	}

	// Clamp risk parameters and entry requirements.
	if c.RiskControl.MinRiskRewardRatio < MinRiskReward {
		c.RiskControl.MinRiskRewardRatio = MinRiskReward
	}
	if c.RiskControl.MinRiskRewardRatio > MaxRiskReward {
		c.RiskControl.MinRiskRewardRatio = MaxRiskReward
	}
	if c.RiskControl.MaxMarginUsage < MinMarginUsage {
		c.RiskControl.MaxMarginUsage = MinMarginUsage
	}
	if c.RiskControl.MaxMarginUsage > MaxMarginUsage {
		c.RiskControl.MaxMarginUsage = MaxMarginUsage
	}
	if c.RiskControl.MinPositionSize < MinPositionSize {
		c.RiskControl.MinPositionSize = MinPositionSize
	}
	if c.RiskControl.MinPositionSize > MaxPositionSize {
		c.RiskControl.MinPositionSize = MaxPositionSize
	}
	if c.RiskControl.MinConfidence < MinConfidence {
		c.RiskControl.MinConfidence = MinConfidence
	}
	if c.RiskControl.MinConfidence > MaxConfidence {
		c.RiskControl.MinConfidence = MaxConfidence
	}
}

// NormalizeProductSchema keeps saved strategy JSON aligned with the product
// editor schema. LLMs may emit user-facing labels such as "AI500"; persistence
// must use the exact frontend/backend enum values.
func (c *StrategyConfig) NormalizeProductSchema() {
	c.StrategyType = normalizeStrategyType(c.StrategyType)
	c.CoinSource.SourceType = normalizeCoinSourceType(c.CoinSource.SourceType)
	if c.CoinSource.SourceType == "" {
		c.CoinSource.SourceType = inferCoinSourceType(c.CoinSource)
	}

	switch c.CoinSource.SourceType {
	case "ai500":
		c.CoinSource.UseAI500 = true
		c.CoinSource.UseOITop = false
		c.CoinSource.UseOILow = false
		if c.CoinSource.AI500Limit <= 0 {
			c.CoinSource.AI500Limit = 3
		}
	case "oi_top":
		c.CoinSource.UseAI500 = false
		c.CoinSource.UseOITop = true
		c.CoinSource.UseOILow = false
		if c.CoinSource.OITopLimit <= 0 {
			c.CoinSource.OITopLimit = 3
		}
	case "oi_low":
		c.CoinSource.UseAI500 = false
		c.CoinSource.UseOITop = false
		c.CoinSource.UseOILow = true
		if c.CoinSource.OILowLimit <= 0 {
			c.CoinSource.OILowLimit = 3
		}
	case "static":
		c.CoinSource.UseAI500 = false
		c.CoinSource.UseOITop = false
		c.CoinSource.UseOILow = false
	default:
		c.CoinSource.SourceType = "ai500"
		c.CoinSource.UseAI500 = true
		if c.CoinSource.AI500Limit <= 0 {
			c.CoinSource.AI500Limit = 3
		}
	}

	c.CoinSource.StaticCoins = normalizeSymbols(c.CoinSource.StaticCoins)
	c.CoinSource.ExcludedCoins = normalizeSymbols(c.CoinSource.ExcludedCoins)
	c.Indicators.Klines.PrimaryTimeframe = normalizeTimeframe(c.Indicators.Klines.PrimaryTimeframe)
	c.Indicators.Klines.LongerTimeframe = normalizeTimeframe(c.Indicators.Klines.LongerTimeframe)
	c.Indicators.Klines.SelectedTimeframes = normalizeTimeframes(c.Indicators.Klines.SelectedTimeframes)
	if len(c.Indicators.Klines.SelectedTimeframes) > 0 {
		c.Indicators.Klines.EnableMultiTimeframe = true
	}
}

func normalizeStrategyType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "grid", "grid_strategy", "grid-trading", "grid trading", "grid_trading", "网格", "网格策略", "网格交易":
		return "grid_trading"
	case "", "ai", "ai_strategy", "ai-trading", "ai trading", "ai_trading", "ai策略", "ai 策略", "ai交易策略", "ai智能策略":
		return "ai_trading"
	default:
		return value
	}
}

func normalizeCoinSourceType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	compact := strings.NewReplacer(" ", "", "_", "", "-", "", "数据源", "", "选币", "", "币种", "").Replace(value)
	switch {
	case compact == "":
		return ""
	case strings.Contains(compact, "ai500"):
		return "ai500"
	case strings.Contains(compact, "oitop") || strings.Contains(value, "oi top") || strings.Contains(value, "持仓量最高") || strings.Contains(value, "持仓量靠前"):
		return "oi_top"
	case strings.Contains(compact, "oilow") || strings.Contains(value, "oi low") || strings.Contains(value, "持仓量最低") || strings.Contains(value, "持仓量较低"):
		return "oi_low"
	case strings.Contains(value, "static") || strings.Contains(value, "固定") || strings.Contains(value, "静态"):
		return "static"
	default:
		return value
	}
}

func inferCoinSourceType(source CoinSourceConfig) string {
	switch {
	case len(source.StaticCoins) > 0:
		return "static"
	case source.UseAI500:
		return "ai500"
	case source.UseOITop:
		return "oi_top"
	case source.UseOILow:
		return "oi_low"
	default:
		return "ai500"
	}
}

func normalizeSymbols(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range splitLooseStringList(values) {
		value = strings.TrimSpace(value)
		// Task 12 / Cluster D — CME futures symbols keep their case
		// (NQ.c.0 not NQ.C.0). Uppercasing breaks the Databento
		// continuous symbology convention. Crypto path unchanged.
		if isCMEFuturesSymbol(value) {
			value = strings.Trim(value, "，,;； ")
		} else {
			value = strings.ToUpper(value)
			value = strings.Trim(value, "，,;； ")
		}
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// isCMEFuturesSymbol mirrors market.IsCMEFuturesSymbol. Duplicated here
// because store/ deliberately does not depend on market/. Keep the two
// in lockstep — see market/futures_symbol.go for the canonical version
// + documentation.
func isCMEFuturesSymbol(symbol string) bool {
	s := strings.TrimSpace(symbol)
	if s == "" {
		return false
	}
	if strings.Contains(s, ".c.") {
		return true
	}
	upper := strings.ToUpper(s)
	if _, ok := cmeFuturesRootsStore[upper]; ok {
		return true
	}
	for root := range cmeFuturesRootsStore {
		if strings.HasPrefix(upper, root+".") {
			return true
		}
		if strings.HasPrefix(upper, root) && len(upper) == len(root)+2 {
			tail := upper[len(root):]
			if isContractMonthStore(root, tail[0]) && tail[1] >= '0' && tail[1] <= '9' {
				return true
			}
		}
	}
	return false
}

// cmeFuturesRootsStore mirrors market.cmeFuturesRoots. See note on
// isCMEFuturesSymbol above.
var cmeFuturesRootsStore = map[string]struct{}{
	"NQ":  {},
	"MNQ": {},
	"ES":  {},
	"MES": {},
	"RTY": {},
	"M2K": {},
	"YM":  {},
	"MYM": {},
	"CL":  {},
	"MCL": {},
	"NG":  {},
	"GC":  {},
	"MGC": {},
	"SI":  {},
	"ZB":  {},
	"ZN":  {},
	"ZF":  {},
	"ZT":  {},
}

func isQuarterlyMonthStore(b byte) bool {
	switch b {
	case 'H', 'M', 'U', 'Z':
		return true
	}
	return false
}

// futuresMonthCodesStore mirrors market.futuresMonthCodes — per-root month codes
// so contract-code recognition is family-correct: index/Treasury stay quarterly
// (NQF6 not matched) while energy lists all 12 (NGF6 matched). Keep in sync with
// cmeFuturesRootsStore (every root needs an entry).
var futuresMonthCodesStore = map[string]string{
	"NQ": "HMUZ", "MNQ": "HMUZ", "ES": "HMUZ", "MES": "HMUZ",
	"RTY": "HMUZ", "M2K": "HMUZ", "YM": "HMUZ", "MYM": "HMUZ",
	"ZB": "HMUZ", "ZN": "HMUZ", "ZF": "HMUZ", "ZT": "HMUZ",
	"CL": "FGHJKMNQUVXZ", "MCL": "FGHJKMNQUVXZ", "NG": "FGHJKMNQUVXZ",
	"GC": "GJMQVZ", "MGC": "GJMQVZ", "SI": "FHKNUZ",
}

// isContractMonthStore mirrors market.isContractMonth.
func isContractMonthStore(root string, b byte) bool {
	return strings.IndexByte(futuresMonthCodesStore[root], b) >= 0
}

func normalizeTimeframes(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range splitLooseStringList(values) {
		tf := normalizeTimeframe(value)
		if tf == "" || seen[tf] {
			continue
		}
		seen[tf] = true
		out = append(out, tf)
	}
	return out
}

func splitLooseStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	joined := strings.TrimSpace(strings.Join(values, ","))
	if strings.HasPrefix(joined, "[") && strings.HasSuffix(joined, "]") {
		var parsed []string
		if err := json.Unmarshal([]byte(joined), &parsed); err == nil {
			return parsed
		}
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			var parsed []string
			if err := json.Unmarshal([]byte(value), &parsed); err == nil {
				parts = append(parts, parsed...)
				continue
			}
		}
		value = strings.Trim(value, "[]")
		for _, part := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n'
		}) {
			part = strings.Trim(strings.TrimSpace(part), "\"'")
			if part != "" {
				parts = append(parts, part)
			}
		}
	}
	return parts
}

func normalizeTimeframe(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Trim(value, "\"'，,。 ")
	if value == "" {
		return ""
	}
	aliases := map[string]string{
		"1分钟":  "1m",
		"3分钟":  "3m",
		"5分钟":  "5m",
		"15分钟": "15m",
		"30分钟": "30m",
		"1小时":  "1h",
		"2小时":  "2h",
		"4小时":  "4h",
		"6小时":  "6h",
		"8小时":  "8h",
		"12小时": "12h",
		"1天":   "1d",
		"3天":   "3d",
		"1周":   "1w",
	}
	if alias, ok := aliases[value]; ok {
		return alias
	}
	allowed := map[string]bool{
		"1m": true, "3m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "2h": true, "4h": true, "6h": true, "8h": true, "12h": true,
		"1d": true, "3d": true, "1w": true,
	}
	if !allowed[value] {
		return ""
	}
	return value
}

// MergeStrategyConfig applies a partial JSON-style patch onto a full strategy config.
// Nested objects are merged recursively so omitted fields keep their previous values.
func MergeStrategyConfig(base StrategyConfig, patch map[string]any) (StrategyConfig, error) {
	baseJSON, err := json.Marshal(base)
	if err != nil {
		return StrategyConfig{}, err
	}

	var mergedMap map[string]any
	if err := json.Unmarshal(baseJSON, &mergedMap); err != nil {
		return StrategyConfig{}, err
	}

	normalizeStrategyConfigPatch(patch)
	if fmt.Sprint(patch["strategy_type"]) == "grid_trading" {
		ensureDefaultGridConfigMap(mergedMap)
	}
	mergeJSONMaps(mergedMap, patch)

	mergedJSON, err := json.Marshal(mergedMap)
	if err != nil {
		return StrategyConfig{}, err
	}

	var merged StrategyConfig
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		return StrategyConfig{}, err
	}
	return merged, nil
}

// PreserveAIConfigOnTypeSwitch (F11b) guards against the grid-switch ai_config
// DESTROYER: when a strategy's TYPE changes, the AI-config bundle (coin source,
// indicators, risk control, prompt sections, custom prompt) must NOT be silently
// destroyed. Unless the caller explicitly confirmed the loss, `merged` gets its AI
// bundle restored from `base`, so switching ai→grid (and back) never loses it. With
// confirmed==true (the FE sent it after showing a dialog naming what's lost), the
// caller's merged config stands.
func PreserveAIConfigOnTypeSwitch(base, merged StrategyConfig, confirmed bool) StrategyConfig {
	if confirmed || base.StrategyType == merged.StrategyType {
		return merged
	}
	merged.CoinSource = base.CoinSource
	merged.Indicators = base.Indicators
	merged.RiskControl = base.RiskControl
	merged.PromptSections = base.PromptSections
	merged.CustomPrompt = base.CustomPrompt
	return merged
}

func DefaultGridStrategyConfig() GridStrategyConfig {
	return GridStrategyConfig{
		Symbol:                "BTCUSDT",
		GridCount:             10,
		TotalInvestment:       1000,
		Leverage:              5,
		UpperPrice:            0,
		LowerPrice:            0,
		UseATRBounds:          true,
		ATRMultiplier:         2.0,
		Distribution:          "gaussian",
		MaxDrawdownPct:        15,
		StopLossPct:           5,
		DailyLossLimitPct:     10,
		UseMakerOnly:          true,
		EnableDirectionAdjust: false,
		DirectionBiasRatio:    0.7,
	}
}

func ensureDefaultGridConfigMap(config map[string]any) {
	if config == nil {
		return
	}
	if existing, ok := config["grid_config"].(map[string]any); ok && len(existing) > 0 {
		return
	}
	defaultGrid := DefaultGridStrategyConfig()
	raw, err := json.Marshal(defaultGrid)
	if err != nil {
		return
	}
	var gridMap map[string]any
	if err := json.Unmarshal(raw, &gridMap); err != nil {
		return
	}
	config["grid_config"] = gridMap
}

func normalizeStrategyConfigPatch(patch map[string]any) {
	if patch == nil {
		return
	}

	if gridConfig, hasGrid := patch["grid_config"]; hasGrid && gridConfig != nil {
		if _, hasType := patch["strategy_type"]; !hasType {
			patch["strategy_type"] = "grid_trading"
		}
	}

	aiKeys := []string{"coin_source", "indicators", "risk_control", "prompt_sections", "custom_prompt"}
	for _, key := range aiKeys {
		value, ok := patch[key]
		if !ok {
			continue
		}
		aiConfig, _ := patch["ai_config"].(map[string]any)
		if aiConfig == nil {
			aiConfig = map[string]any{}
			patch["ai_config"] = aiConfig
		}
		aiConfig[key] = value
		delete(patch, key)
	}

	if fmt.Sprint(patch["strategy_type"]) == "grid_trading" {
		delete(patch, "ai_config")
	}

	if _, hasType := patch["strategy_type"]; hasType {
		return
	}
	if gridConfig, hasGrid := patch["grid_config"]; hasGrid && gridConfig != nil {
		patch["strategy_type"] = "grid_trading"
	}
}

func mergeJSONMaps(dst, src map[string]any) {
	for key, srcVal := range src {
		srcMap, srcIsMap := srcVal.(map[string]any)
		dstMap, dstIsMap := dst[key].(map[string]any)
		if srcIsMap && dstIsMap {
			mergeJSONMaps(dstMap, srcMap)
			continue
		}
		dst[key] = srcVal
	}
}

func StrategyClampWarnings(before, after StrategyConfig, lang string) []string {
	if lang != "zh" {
		lang = "en"
	}
	warnings := make([]string, 0, 8)
	appendInt := func(labelZH, labelEN string, from, to int) {
		if from == to {
			return
		}
		if lang == "zh" {
			warnings = append(warnings, fmt.Sprintf("%s 已从 %d 调整为 %d", labelZH, from, to))
			return
		}
		warnings = append(warnings, fmt.Sprintf("%s adjusted from %d to %d", labelEN, from, to))
	}
	appendFloat := func(labelZH, labelEN string, from, to float64) {
		if from == to {
			return
		}
		if lang == "zh" {
			warnings = append(warnings, fmt.Sprintf("%s 已从 %.2f 调整为 %.2f", labelZH, from, to))
			return
		}
		warnings = append(warnings, fmt.Sprintf("%s adjusted from %.2f to %.2f", labelEN, from, to))
	}

	appendInt("最大持仓数", "max_positions", before.RiskControl.MaxPositions, after.RiskControl.MaxPositions)
	appendInt("BTC/ETH 最大杠杆", "btc_eth_max_leverage", before.RiskControl.BTCETHMaxLeverage, after.RiskControl.BTCETHMaxLeverage)
	appendInt("山寨币最大杠杆", "altcoin_max_leverage", before.RiskControl.AltcoinMaxLeverage, after.RiskControl.AltcoinMaxLeverage)
	appendFloat("BTC/ETH 最大仓位价值倍数", "btc_eth_max_position_value_ratio", before.RiskControl.BTCETHMaxPositionValueRatio, after.RiskControl.BTCETHMaxPositionValueRatio)
	appendFloat("山寨币最大仓位价值倍数", "altcoin_max_position_value_ratio", before.RiskControl.AltcoinMaxPositionValueRatio, after.RiskControl.AltcoinMaxPositionValueRatio)
	appendFloat("最小盈亏比", "min_risk_reward_ratio", before.RiskControl.MinRiskRewardRatio, after.RiskControl.MinRiskRewardRatio)
	appendFloat("最大保证金使用率", "max_margin_usage", before.RiskControl.MaxMarginUsage, after.RiskControl.MaxMarginUsage)
	appendFloat("最小开仓金额", "min_position_size", before.RiskControl.MinPositionSize, after.RiskControl.MinPositionSize)
	appendInt("最低置信度", "min_confidence", before.RiskControl.MinConfidence, after.RiskControl.MinConfidence)
	return warnings
}

// StrategyStore strategy storage
type StrategyStore struct {
	db *gorm.DB
}

// Strategy strategy configuration
type Strategy struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	UserID        string    `gorm:"column:user_id;not null;default:'';index" json:"user_id"`
	Name          string    `gorm:"not null" json:"name"`
	Description   string    `gorm:"default:''" json:"description"`
	IsActive      bool      `gorm:"column:is_active;default:false;index" json:"is_active"`
	IsDefault     bool      `gorm:"column:is_default;default:false" json:"is_default"`
	IsPublic      bool      `gorm:"column:is_public;default:false;index" json:"is_public"`    // whether visible in strategy market
	ConfigVisible bool      `gorm:"column:config_visible;default:true" json:"config_visible"` // whether config details are visible
	Config        string    `gorm:"not null;default:'{}'" json:"config"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Strategy) TableName() string { return "strategies" }

// StrategyConfig strategy configuration details (JSON structure)
type StrategyConfig struct {
	// Strategy type: "ai_trading" (default) or "grid_trading"
	StrategyType string `json:"strategy_type,omitempty"`

	// language setting: "zh" for Chinese, "en" for English
	// This determines the language used for data formatting and prompt generation
	Language string `json:"language,omitempty"`

	// PromptVariant selects the live AI prompt mode (balanced / aggressive /
	// conservative / scalping / futures). Persisted per-strategy; when EMPTY the
	// live loop falls back to the venue rule (ninjatrader→futures, else balanced)
	// so existing strategies (no variant saved) are byte-identical. Prompt-layer
	// only — it does NOT touch any risk gate or the live-account block.
	PromptVariant string `json:"prompt_variant,omitempty"`

	// AI trading configuration fields are kept on the Go struct for engine
	// compatibility, but JSON persistence nests them under ai_config.
	CoinSource     CoinSourceConfig     `json:"-"`
	Indicators     IndicatorConfig      `json:"-"`
	CustomPrompt   string               `json:"-"`
	RiskControl    RiskControlConfig    `json:"-"`
	PromptSections PromptSectionsConfig `json:"-"`

	// Grid trading configuration (only used when StrategyType == "grid_trading")
	GridConfig *GridStrategyConfig `json:"grid_config,omitempty"`

	// Publish settings are shared by AI and grid strategies. The database still
	// stores the authoritative booleans on Strategy, but config JSON may carry
	// this object for agent/frontend schema consistency.
	PublishConfig *PublishStrategyConfig `json:"publish_config,omitempty"`

	// DayPlan is the optional per-strategy Day Plan settings block (P0.1). ROOT
	// placement (sibling of strategy_type) so a grid switch — which drops
	// ai_config — never drops it (RECON #1). Additive + defaults-off: a nil
	// pointer emits no day_plan key, keeping every existing strategy
	// byte-identical.
	DayPlan *DayPlanConfig `json:"day_plan,omitempty"`
}

// AIStrategyConfig contains fields only used by AI trading strategies.
type AIStrategyConfig struct {
	CoinSource     CoinSourceConfig     `json:"coin_source"`
	Indicators     IndicatorConfig      `json:"indicators"`
	CustomPrompt   string               `json:"custom_prompt,omitempty"`
	RiskControl    RiskControlConfig    `json:"risk_control"`
	PromptSections PromptSectionsConfig `json:"prompt_sections,omitempty"`
}

// PublishStrategyConfig contains settings shared by all strategy types.
type PublishStrategyConfig struct {
	IsPublic      bool `json:"is_public"`
	ConfigVisible bool `json:"config_visible"`
}

// MarshalJSON writes the product-facing strategy schema:
// strategy_type + grid_config or ai_config + shared publish_config.
func (c StrategyConfig) MarshalJSON() ([]byte, error) {
	strategyType := strings.TrimSpace(c.StrategyType)
	if strategyType == "" {
		strategyType = "ai_trading"
	}

	out := struct {
		StrategyType  string                 `json:"strategy_type"`
		Language      string                 `json:"language,omitempty"`
		PromptVariant string                 `json:"prompt_variant,omitempty"`
		AIConfig      *AIStrategyConfig      `json:"ai_config,omitempty"`
		GridConfig    *GridStrategyConfig    `json:"grid_config,omitempty"`
		PublishConfig *PublishStrategyConfig `json:"publish_config,omitempty"`
		DayPlan       *DayPlanConfig         `json:"day_plan,omitempty"`
	}{
		StrategyType:  strategyType,
		Language:      c.Language,
		PromptVariant: strings.TrimSpace(c.PromptVariant),
		PublishConfig: c.PublishConfig,
		DayPlan:       c.DayPlan,
	}

	if strategyType == "grid_trading" {
		out.GridConfig = c.GridConfig
	} else {
		out.AIConfig = &AIStrategyConfig{
			CoinSource:     c.CoinSource,
			Indicators:     c.Indicators,
			CustomPrompt:   c.CustomPrompt,
			RiskControl:    c.RiskControl,
			PromptSections: c.PromptSections,
		}
	}

	return json.Marshal(out)
}

// UnmarshalJSON accepts both the new nested schema and old flat configs. Old
// top-level AI fields are normalized into the Go compatibility fields.
func (c *StrategyConfig) UnmarshalJSON(data []byte) error {
	type rawStrategyConfig struct {
		StrategyType  string                 `json:"strategy_type"`
		Language      string                 `json:"language"`
		PromptVariant string                 `json:"prompt_variant"`
		AIConfig      *AIStrategyConfig      `json:"ai_config"`
		GridConfig    *GridStrategyConfig    `json:"grid_config"`
		PublishConfig *PublishStrategyConfig `json:"publish_config"`
		DayPlan       *DayPlanConfig         `json:"day_plan"`

		CoinSource     *CoinSourceConfig     `json:"coin_source"`
		Indicators     *IndicatorConfig      `json:"indicators"`
		CustomPrompt   *string               `json:"custom_prompt"`
		RiskControl    *RiskControlConfig    `json:"risk_control"`
		PromptSections *PromptSectionsConfig `json:"prompt_sections"`
	}

	var raw rawStrategyConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.StrategyType = raw.StrategyType
	c.Language = raw.Language
	c.PromptVariant = strings.TrimSpace(raw.PromptVariant)
	c.GridConfig = raw.GridConfig
	c.PublishConfig = raw.PublishConfig
	c.DayPlan = raw.DayPlan

	if raw.AIConfig != nil {
		c.CoinSource = raw.AIConfig.CoinSource
		c.Indicators = raw.AIConfig.Indicators
		c.CustomPrompt = raw.AIConfig.CustomPrompt
		c.RiskControl = raw.AIConfig.RiskControl
		c.PromptSections = raw.AIConfig.PromptSections
	} else {
		if raw.CoinSource != nil {
			c.CoinSource = *raw.CoinSource
		}
		if raw.Indicators != nil {
			c.Indicators = *raw.Indicators
		}
		if raw.CustomPrompt != nil {
			c.CustomPrompt = *raw.CustomPrompt
		}
		if raw.RiskControl != nil {
			c.RiskControl = *raw.RiskControl
		}
		if raw.PromptSections != nil {
			c.PromptSections = *raw.PromptSections
		}
	}

	if strings.TrimSpace(c.StrategyType) == "" && c.GridConfig != nil {
		c.StrategyType = "grid_trading"
	}
	return nil
}

// DayPlanConfig is the per-strategy Day Plan settings block (spec PAGE 2 field
// list). Additive + defaults-off: a nil *DayPlanConfig (absent day_plan) leaves
// an existing strategy byte-identical, and PlanEnabled=false is the master
// switch even when the block is present. Lives at ROOT of StrategyConfig.
type DayPlanConfig struct {
	// PlanEnabled is the master switch (default false = off).
	PlanEnabled bool `json:"plan_enabled"`
	// PlannerModel is the reasoner binding from the multi-key registry; empty
	// falls back to the strategy's primary model (RECON #9).
	PlannerModel string `json:"planner_model,omitempty"`
	// PlanMode: advisory (default) | direction | strict. Promotion by evidence.
	PlanMode string `json:"plan_mode,omitempty"`
	// PlannerTimeframes are the structure-summary TFs (default D,4h,1h,15m).
	PlannerTimeframes []string `json:"planner_timeframes,omitempty"`
	// ProximityFilterATR: day-trade lock, 0.5–3.0 (default 1.5).
	ProximityFilterATR float64 `json:"proximity_filter_atr,omitempty"`
	// MaxLevels: level table cap, 3–12 (default 8).
	MaxLevels int `json:"max_levels,omitempty"`
	// ScenarioCap: scenarios cap, 1–5 (default 3).
	ScenarioCap int `json:"scenario_cap,omitempty"`
	// AcceptanceRule: 2x5m (default) | 15m-close.
	AcceptanceRule string `json:"acceptance_rule,omitempty"`
	// ReplanCap: re-reads per session, 0–4 (default 2).
	ReplanCap int `json:"replan_cap,omitempty"`
	// SessionsEnabled: subset of NY | ASIA | LONDON (default [NY]); each other
	// session earns enablement via replay + NY match-rate evidence.
	SessionsEnabled []string `json:"sessions_enabled,omitempty"`
	// ApprovalRequired: OFF (default) = fully automatic.
	ApprovalRequired bool `json:"approval_required"`
	// EveningDigest: 17:30 evening digest (default true).
	EveningDigest bool `json:"evening_digest"`
	// Sessions holds minimal per-session overrides; absent/nil fields inherit
	// from the strategy-level values above (⚪ inherit / 🔸 override).
	Sessions []DayPlanSessionOverride `json:"sessions,omitempty"`
}

// DayPlanSessionOverride is a minimal per-session override. Every field is a
// pointer: nil means "inherit from the strategy-level DayPlanConfig", a set
// value overrides only that field for the named session.
type DayPlanSessionOverride struct {
	Session        string  `json:"session"` // NY | ASIA | LONDON
	Enable         *bool   `json:"enable,omitempty"`
	ReplanCap      *int    `json:"replan_cap,omitempty"`
	PlanMode       *string `json:"plan_mode,omitempty"`
	AcceptanceRule *string `json:"acceptance_rule,omitempty"`
	MinGrade       *string `json:"min_grade,omitempty"` // A | B | C
	MaxTrades      *int    `json:"max_trades,omitempty"`
}

// DefaultDayPlanConfig returns the spec default block (plan OFF). It is NOT
// injected into GetDefaultStrategyConfig — creating a strategy leaves day_plan
// absent (byte-identical) until the owner opts in; the frontend/creation flow
// seeds this when the block is first turned on.
func DefaultDayPlanConfig() *DayPlanConfig {
	return &DayPlanConfig{
		PlanEnabled:        false,
		PlanMode:           "advisory",
		PlannerTimeframes:  []string{"D", "4h", "1h", "15m"},
		ProximityFilterATR: 1.5,
		MaxLevels:          8,
		ScenarioCap:        3,
		AcceptanceRule:     "2x5m",
		ReplanCap:          2,
		SessionsEnabled:    []string{"NY"},
		ApprovalRequired:   false,
		EveningDigest:      true,
	}
}

// GridStrategyConfig grid trading specific configuration
type GridStrategyConfig struct {
	// Trading pair (e.g., "BTCUSDT")
	Symbol string `json:"symbol"`
	// Number of grid levels (5-50)
	GridCount int `json:"grid_count"`
	// Total investment in USDT
	TotalInvestment float64 `json:"total_investment"`
	// Leverage (1-20)
	Leverage int `json:"leverage"`
	// Upper price boundary (0 = auto-calculate from ATR)
	UpperPrice float64 `json:"upper_price"`
	// Lower price boundary (0 = auto-calculate from ATR)
	LowerPrice float64 `json:"lower_price"`
	// Use ATR to auto-calculate bounds
	UseATRBounds bool `json:"use_atr_bounds"`
	// ATR multiplier for bound calculation (default 2.0)
	ATRMultiplier float64 `json:"atr_multiplier"`
	// Position distribution: "uniform" | "gaussian" | "pyramid"
	Distribution string `json:"distribution"`
	// Maximum drawdown percentage before emergency exit
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
	// Stop loss percentage per position
	StopLossPct float64 `json:"stop_loss_pct"`
	// Daily loss limit percentage
	DailyLossLimitPct float64 `json:"daily_loss_limit_pct"`
	// Use maker-only orders for lower fees
	UseMakerOnly bool `json:"use_maker_only"`
	// Enable automatic grid direction adjustment based on box breakouts
	EnableDirectionAdjust bool `json:"enable_direction_adjust"`
	// Direction bias ratio for long_bias/short_bias modes (default 0.7 = 70%/30%)
	DirectionBiasRatio float64 `json:"direction_bias_ratio"`
}

// PromptSectionsConfig editable sections of System Prompt
type PromptSectionsConfig struct {
	// role definition (title + description)
	RoleDefinition string `json:"role_definition,omitempty"`
	// trading frequency awareness
	TradingFrequency string `json:"trading_frequency,omitempty"`
	// entry standards
	EntryStandards string `json:"entry_standards,omitempty"`
	// decision process
	DecisionProcess string `json:"decision_process,omitempty"`
}

// CoinSourceConfig coin source configuration
type CoinSourceConfig struct {
	// source type shown in the product editor: "static" | "ai500" | "oi_top" | "oi_low"
	SourceType string `json:"source_type"`
	// static coin list (used when source_type = "static")
	StaticCoins []string `json:"static_coins,omitempty"`
	// excluded coins list (filtered out from all sources)
	ExcludedCoins []string `json:"excluded_coins,omitempty"`
	// whether to use AI500 coin pool
	UseAI500 bool `json:"use_ai500"`
	// AI500 coin pool maximum count
	AI500Limit int `json:"ai500_limit,omitempty"`
	// whether to use OI Top (OI increase ranking, suitable for long positions)
	UseOITop bool `json:"use_oi_top"`
	// OI Top maximum count
	OITopLimit int `json:"oi_top_limit,omitempty"`
	// whether to use OI Low (OI decrease ranking, suitable for short positions)
	UseOILow bool `json:"use_oi_low"`
	// OI Low maximum count
	OILowLimit int `json:"oi_low_limit,omitempty"`
	// whether to use Hyperliquid All coins (all available perp pairs)
	UseHyperAll bool `json:"use_hyper_all"`
	// whether to use Hyperliquid Main coins (top N by 24h volume)
	UseHyperMain bool `json:"use_hyper_main"`
	// Hyperliquid Main maximum count (default 20)
	HyperMainLimit int `json:"hyper_main_limit,omitempty"`
	// Note: API URLs are now built automatically using NofxOSAPIKey from IndicatorConfig
}

// IndicatorConfig indicator configuration
type IndicatorConfig struct {
	// K-line configuration
	Klines KlineConfig `json:"klines"`
	// raw kline data (OHLCV) - always enabled, required for AI analysis
	EnableRawKlines bool `json:"enable_raw_klines"`
	// technical indicator switches
	EnableEMA         bool `json:"enable_ema"`
	EnableMACD        bool `json:"enable_macd"`
	EnableRSI         bool `json:"enable_rsi"`
	EnableATR         bool `json:"enable_atr"`
	EnableBOLL        bool `json:"enable_boll"` // Bollinger Bands
	EnableVolume      bool `json:"enable_volume"`
	EnableOI          bool `json:"enable_oi"`           // open interest
	EnableFundingRate bool `json:"enable_funding_rate"` // funding rate
	EnableSVP         bool `json:"enable_svp"`          // session volume profile → futures AI prompt line (POC/VAH/VAL); default OFF
	// EMA period configuration
	EMAPeriods []int `json:"ema_periods,omitempty"` // default [20, 50]
	// RSI period configuration
	RSIPeriods []int `json:"rsi_periods,omitempty"` // default [7, 14]
	// ATR period configuration
	ATRPeriods []int `json:"atr_periods,omitempty"` // default [14]
	// BOLL period configuration (period, standard deviation multiplier is fixed at 2)
	BOLLPeriods []int `json:"boll_periods,omitempty"` // default [20] - can select multiple timeframes
	// external data sources
	ExternalDataSources []ExternalDataSource `json:"external_data_sources,omitempty"`

	// ========== NofxOS Unified API Configuration ==========
	// Unified API Key for all NofxOS data sources
	NofxOSAPIKey string `json:"nofxos_api_key,omitempty"`

	// quantitative data sources (capital flow, position changes, price changes)
	EnableQuantData    bool `json:"enable_quant_data"`    // whether to enable quantitative data
	EnableQuantOI      bool `json:"enable_quant_oi"`      // whether to show OI data
	EnableQuantNetflow bool `json:"enable_quant_netflow"` // whether to show Netflow data

	// OI ranking data (market-wide open interest increase/decrease rankings)
	EnableOIRanking   bool   `json:"enable_oi_ranking"`             // whether to enable OI ranking data
	OIRankingDuration string `json:"oi_ranking_duration,omitempty"` // duration: 1h, 4h, 24h
	OIRankingLimit    int    `json:"oi_ranking_limit,omitempty"`    // number of entries (default 10)

	// NetFlow ranking data (market-wide fund flow rankings - institution/personal)
	EnableNetFlowRanking   bool   `json:"enable_netflow_ranking"`             // whether to enable NetFlow ranking data
	NetFlowRankingDuration string `json:"netflow_ranking_duration,omitempty"` // duration: 1h, 4h, 24h
	NetFlowRankingLimit    int    `json:"netflow_ranking_limit,omitempty"`    // number of entries (default 10)

	// Price ranking data (market-wide gainers/losers)
	EnablePriceRanking   bool   `json:"enable_price_ranking"`             // whether to enable price ranking data
	PriceRankingDuration string `json:"price_ranking_duration,omitempty"` // durations: "1h" or "1h,4h,24h"
	PriceRankingLimit    int    `json:"price_ranking_limit,omitempty"`    // number of entries per ranking (default 10)
}

// KlineConfig K-line configuration
type KlineConfig struct {
	// primary timeframe: "1m", "3m", "5m", "15m", "1h", "4h"
	PrimaryTimeframe string `json:"primary_timeframe"`
	// primary timeframe K-line count
	PrimaryCount int `json:"primary_count"`
	// longer timeframe
	LongerTimeframe string `json:"longer_timeframe,omitempty"`
	// longer timeframe K-line count
	LongerCount int `json:"longer_count,omitempty"`
	// whether to enable multi-timeframe analysis
	EnableMultiTimeframe bool `json:"enable_multi_timeframe"`
	// selected timeframe list (new: supports multi-timeframe selection)
	SelectedTimeframes []string `json:"selected_timeframes,omitempty"`
}

// ExternalDataSource external data source configuration
type ExternalDataSource struct {
	Name        string            `json:"name"`   // data source name
	Type        string            `json:"type"`   // type: "api" | "webhook"
	URL         string            `json:"url"`    // API URL
	Method      string            `json:"method"` // HTTP method
	Headers     map[string]string `json:"headers,omitempty"`
	DataPath    string            `json:"data_path,omitempty"`    // JSON data path
	RefreshSecs int               `json:"refresh_secs,omitempty"` // refresh interval (seconds)
}

// RiskControlConfig risk control configuration
type RiskControlConfig struct {
	// Max number of coins held simultaneously (CODE ENFORCED)
	MaxPositions int `json:"max_positions"`

	// BTC/ETH exchange leverage for opening positions (AI guided)
	BTCETHMaxLeverage int `json:"btc_eth_max_leverage"`
	// Altcoin exchange leverage for opening positions (AI guided)
	AltcoinMaxLeverage int `json:"altcoin_max_leverage"`

	// BTC/ETH single position max value = equity × this ratio (CODE ENFORCED, default: 5)
	BTCETHMaxPositionValueRatio float64 `json:"btc_eth_max_position_value_ratio"`
	// Altcoin single position max value = equity × this ratio (CODE ENFORCED, default: 1)
	AltcoinMaxPositionValueRatio float64 `json:"altcoin_max_position_value_ratio"`

	// Max margin utilization (e.g. 0.9 = 90%) (CODE ENFORCED)
	MaxMarginUsage float64 `json:"max_margin_usage"`
	// Min position size in USDT (CODE ENFORCED)
	MinPositionSize float64 `json:"min_position_size"`

	// Min take_profit / stop_loss ratio (AI guided)
	MinRiskRewardRatio float64 `json:"min_risk_reward_ratio"`
	// Min AI confidence to open position (AI guided)
	MinConfidence int `json:"min_confidence"`

	// === Strategy Studio Phase 1 — prop-firm daily guardrails (per-strategy;
	// env = fallback for the value; the per-guardrail toggle governs enforcement). ===

	// Master switch: nil/true = guardrails active; false = ALL guardrails bypassed.
	GuardrailsEnabled *bool `json:"guardrails_enabled,omitempty"`

	// Daily realized-LOSS limit (USD, positive): loss ≥ this halts new entries for
	// the CME session-day. Enabled defaults ON (nil) to preserve the existing live
	// env daily-loss gate; the value falls back to RISK_MAX_DAILY_LOSS_USD when 0.
	DailyLossLimitUSD float64 `json:"daily_loss_limit_usd,omitempty"`
	DailyLossEnabled  *bool   `json:"daily_loss_enabled,omitempty"`

	// Daily realized-PROFIT target (USD, positive): profit ≥ this stops new entries
	// for the session-day. New guardrail → defaults OFF (nil).
	DailyProfitTargetUSD float64 `json:"daily_profit_target_usd,omitempty"`
	DailyProfitEnabled   *bool   `json:"daily_profit_enabled,omitempty"`

	// Max ENTRIES per CME session-day. New guardrail → defaults OFF (nil).
	MaxDailyTrades        int   `json:"max_daily_trades,omitempty"`
	MaxDailyTradesEnabled *bool `json:"max_daily_trades_enabled,omitempty"`

	// D1 — CONSECUTIVE-LOSS halt: after this many consecutive LOSING closed trades
	// in the CME session-day, block NEW entries until the next session (open-pos
	// management via SL/TP is unaffected). 0 = OFF. Resets on a winning/break-even
	// close or a new session. New guardrail → default 0 (off). NOT gated by the
	// guardrails master switch — it is a per-strategy circuit breaker.
	ConsecutiveLossHalt int `json:"consecutive_loss_halt,omitempty"`

	// B7 — RE-ENTRY COOLDOWN: after a STOP-LOSS exit, block a SAME-DIRECTION
	// re-entry on that symbol for this many minutes OR until price moves ≥ 1×ATR15
	// away from the stop — whichever comes first. Per-trader; opposite direction and
	// other symbols are never blocked. FE default 20; 0 = OFF (existing strategies
	// stay off until set). Not gated by the guardrails master switch.
	ReentryCooldownMinutes int `json:"reentry_cooldown_minutes,omitempty"`

	// Chunk 3 — max CONTRACTS per futures order (clamp). Unset → the 10-contract
	// default (the prior hidden const maxFuturesContracts). Toggle default ON.
	MaxContractsPerOrder int   `json:"max_contracts_per_order,omitempty"`
	MaxContractsEnabled  *bool `json:"max_contracts_enabled,omitempty"`

	// Chunk 3 — futures NOTIONAL ceiling multiplier: max position notional =
	// equity × this. Unset → 20 (the prior hidden const futuresMaxNotionalLeverage),
	// now VISIBLE + EDITABLE. Toggle default ON (safety backstop).
	MaxNotionalLeverage float64 `json:"max_notional_leverage,omitempty"`
	NotionalCapEnabled  *bool   `json:"notional_cap_enabled,omitempty"`

	// Chunk 4 — time/news BLACKOUT window (daily, HH:MM in America/Chicago). When
	// enabled, the bot makes no new decisions inside [start,end] CT (NT8-side SL/TP
	// still protect open positions). New guardrail → toggle defaults OFF.
	BlackoutEnabled *bool  `json:"blackout_enabled,omitempty"`
	BlackoutStartCT string `json:"blackout_start_ct,omitempty"`
	BlackoutEndCT   string `json:"blackout_end_ct,omitempty"`

	// Chunk 5 — CONSISTENCY rule: no single CME session-day's realized profit may
	// exceed this % of all-time total realized profit. Only triggers once there is
	// prior-day profit (a fresh/single-day account never self-locks). New guardrail
	// → toggle defaults OFF.
	ConsistencyMaxDayPct float64 `json:"consistency_max_day_pct,omitempty"`
	ConsistencyEnabled   *bool   `json:"consistency_enabled,omitempty"`

	// Hold-lock ("Hold discipline"): when ON, once a position is OPEN the bot
	// SUPPRESSES an AI-initiated close — the trade rides to the stop/target the
	// AI set (a real OCO bracket resting at the exchange), instead of the AI
	// bailing on noise minutes in. Only AI decisions are gated; Emergency Flat
	// and the drawdown monitor bypass it. New feature → default OFF (nil/false).
	HoldDisciplineEnabled *bool `json:"hold_discipline,omitempty"`

	// Auto-breakeven (NT8 futures): once an open position is BreakevenTriggerPoints
	// in profit, move its stop to entry (breakeven) so a winner can't turn into a
	// loss. The move is sent to the AddOn (move_stop frame), which modifies the
	// resting bracket in place. New feature → default OFF; trigger defaults to 50.
	BreakevenEnabled       *bool   `json:"breakeven_enabled,omitempty"`
	BreakevenTriggerPoints float64 `json:"breakeven_trigger_points,omitempty"`
}

// NewStrategyStore creates a new StrategyStore
func NewStrategyStore(db *gorm.DB) *StrategyStore {
	return &StrategyStore{db: db}
}

func (s *StrategyStore) initTables() error {
	// AutoMigrate will add missing columns without dropping existing data
	return s.db.AutoMigrate(&Strategy{})
}

func (s *StrategyStore) initDefaultData() error {
	// No longer pre-populate strategies - create on demand when user configures
	return nil
}

// defaultCoinSource returns the seed coin source for a fresh strategy.
// Futures mode (TRADING_MODE=futures) seeds the single NT8 instrument as a
// static coin so a reseeded DB does not revert to the dead ai500 pool (the
// reseed-durable counterpart to the runtime N11 flip). Crypto mode keeps the
// ai500 default.
func defaultCoinSource() CoinSourceConfig {
	if cfg := config.Get(); cfg != nil && cfg.TradingMode == "futures" {
		return CoinSourceConfig{
			SourceType:  "static",
			StaticCoins: []string{"MNQ"},
			AI500Limit:  3,
			OITopLimit:  3,
			OILowLimit:  3,
		}
	}
	return CoinSourceConfig{
		SourceType: "ai500",
		UseAI500:   true,
		AI500Limit: 3,
		UseOITop:   false,
		OITopLimit: 3,
		UseOILow:   false,
		OILowLimit: 3,
	}
}

// GetDefaultStrategyConfig returns the default strategy configuration for the given language
func GetDefaultStrategyConfig(lang string) StrategyConfig {
	// Normalize language to "zh" or "en"
	normalizedLang := "en"
	if lang == "zh" {
		normalizedLang = "zh"
	}

	config := StrategyConfig{
		Language:   normalizedLang,
		CoinSource: defaultCoinSource(),
		Indicators: IndicatorConfig{
			Klines: KlineConfig{
				PrimaryTimeframe:     "5m",
				PrimaryCount:         20,
				LongerTimeframe:      "4h",
				LongerCount:          10,
				EnableMultiTimeframe: true,
				SelectedTimeframes:   []string{"5m", "15m", "1h"},
			},
			EnableRawKlines:   true, // Required - raw OHLCV data for AI analysis
			EnableEMA:         false,
			EnableMACD:        false,
			EnableRSI:         false,
			EnableATR:         false,
			EnableBOLL:        false,
			EnableVolume:      true,
			EnableOI:          true,
			EnableFundingRate: true,
			EMAPeriods:        []int{20, 50},
			RSIPeriods:        []int{7, 14},
			ATRPeriods:        []int{14},
			BOLLPeriods:       []int{20},
			// NofxOS unified API key
			NofxOSAPIKey: "cm_568c67eae410d912c54c",
			// Quant data
			EnableQuantData:    true,
			EnableQuantOI:      true,
			EnableQuantNetflow: true,
			// OI ranking data
			EnableOIRanking:   true,
			OIRankingDuration: "1h",
			OIRankingLimit:    10,
			// NetFlow ranking data
			EnableNetFlowRanking:   true,
			NetFlowRankingDuration: "1h",
			NetFlowRankingLimit:    10,
			// Price ranking data
			EnablePriceRanking:   true,
			PriceRankingDuration: "1h,4h,24h",
			PriceRankingLimit:    10,
		},
		RiskControl: RiskControlConfig{
			MaxPositions:                 3,   // Max 3 coins simultaneously (CODE ENFORCED)
			BTCETHMaxLeverage:            5,   // BTC/ETH exchange leverage (AI guided)
			AltcoinMaxLeverage:           5,   // Altcoin exchange leverage (AI guided)
			BTCETHMaxPositionValueRatio:  5.0, // BTC/ETH: max position = 5x equity (CODE ENFORCED)
			AltcoinMaxPositionValueRatio: 1.0, // Altcoin: max position = 1x equity (CODE ENFORCED)
			MaxMarginUsage:               0.9, // Max 90% margin usage (CODE ENFORCED)
			MinPositionSize:              12,  // Min 12 USDT per position (CODE ENFORCED)
			MinRiskRewardRatio:           3.0, // Min 3:1 profit/loss ratio (AI guided)
			MinConfidence:                75,  // Min 75% confidence (AI guided)
		},
	}

	// Role + Decision are intentionally NOT seeded: an empty box lets each market's
	// prompt builder supply the correct default — the instrument-aware CME role on
	// futures (engine_prompt_futures.go), the crypto role on crypto
	// (engine_prompt.go) — instead of one shared (market-wrong) seed. Frequency +
	// Entry stay seeded (market-neutral). Saved user boxes are untouched.
	if lang == "zh" {
		config.PromptSections = PromptSectionsConfig{
			TradingFrequency: `# ⏱️ 交易频率意识

- 优秀交易员：每天2-4笔 ≈ 每小时0.1-0.2笔
- 每小时超过2笔 = 过度交易
- 单笔持仓时间 ≥ 30-60分钟
如果你发现自己每个周期都在交易 → 标准太低；如果持仓不到30分钟就平仓 → 太冲动。`,
			EntryStandards: `# 🎯 入场标准（严格）

只在多个信号共振时入场。自由使用任何有效的分析方法，避免单一指标、信号矛盾、横盘震荡、或平仓后立即重新开仓等低质量行为。`,
		}
	} else {
		config.PromptSections = PromptSectionsConfig{
			TradingFrequency: `# ⏱️ Trading Frequency Awareness

- Excellent trader: 2-4 trades per day ≈ 0.1-0.2 trades per hour
- >2 trades per hour = overtrading
- Single position holding time ≥ 30-60 minutes
If you find yourself trading every cycle → standards are too low; if closing positions in <30 minutes → too impulsive.`,
			EntryStandards: `# 🎯 Entry Standards (Strict)

Only enter positions when multiple signals resonate. Freely use any effective analysis methods, avoid low-quality behaviors such as single indicators, contradictory signals, sideways oscillation, or immediately restarting after closing positions.`,
		}
	}

	// CME futures (NT8): tune the indicator DEFAULTS for a new futures strategy —
	// disable the crypto-only NofxOS/ranking feeds and enable the technical
	// indicators the futures prompt leans on. Defaults-only (new-strategy
	// template); existing saved strategies are never mutated. See helper.
	if isFuturesMode() {
		applyFuturesIndicatorDefaults(&config.Indicators)
	}

	return config
}

// applyFuturesIndicatorDefaults tunes the indicator defaults for a NEW
// CME-futures strategy (called only when isFuturesMode()):
//
//  1. Disable the crypto-only NofxOS / market-wide ranking feeds — they return
//     no data for an index-futures instrument and just burn the dead claw402
//     path (HTTP 402/404) ~4 min/cycle (plan §5310: "NQ strategies just leave
//     them disabled").
//  2. Enable the computed technical indicators the futures prompt actually leans
//     on — ATR (stop sizing), EMA (trend), RSI (momentum) — which otherwise
//     default OFF, leaving the futures AI with raw bars + volume only. Periods
//     are already seeded ([20,50] / [7,14] / [14]).
//
// Indicators are prompt-data and NEVER gate a trade (the gate reads only Risk
// Control), so this is a defaults-only, prompt-input change. It runs only inside
// GetDefaultStrategyConfig (the new-strategy template) — existing saved
// strategies are never touched. MACD/BOLL are deliberately left off.
func applyFuturesIndicatorDefaults(ind *IndicatorConfig) {
	// (1) crypto-only feeds OFF on futures.
	ind.EnableQuantData = false
	ind.EnableQuantOI = false
	ind.EnableQuantNetflow = false
	ind.EnableOIRanking = false
	ind.EnableNetFlowRanking = false
	ind.EnablePriceRanking = false
	// Open Interest is the Binance crypto-perp feed too — it returns zeros for
	// MNQ and the futures prompt already says to ignore it (no real futures OI
	// is wired; the NT8 bridge carries OHLCV only). Off by default so a new
	// futures strategy doesn't list/value an empty OI section.
	ind.EnableOI = false
	// (2) computed technical indicators ON for futures.
	ind.EnableATR = true
	ind.EnableEMA = true
	ind.EnableRSI = true
}

// isFuturesMode reports whether the bot is running in CME-futures mode. Lives
// in its own function because GetDefaultStrategyConfig shadows the `config`
// package name with a local StrategyConfig variable.
func isFuturesMode() bool {
	c := config.Get()
	return c != nil && c.TradingMode == "futures"
}

// Create create a strategy
func (s *StrategyStore) Create(strategy *Strategy) error {
	return s.db.Create(strategy).Error
}

// Update update a strategy
func (s *StrategyStore) Update(strategy *Strategy) error {
	return s.db.Model(&Strategy{}).
		Where("id = ? AND user_id = ?", strategy.ID, strategy.UserID).
		Updates(map[string]interface{}{
			"name":           strategy.Name,
			"description":    strategy.Description,
			"config":         strategy.Config,
			"is_public":      strategy.IsPublic,
			"config_visible": strategy.ConfigVisible,
			"updated_at":     time.Now().UTC(),
		}).Error
}

// Delete delete a strategy
func (s *StrategyStore) Delete(userID, id string) error {
	// do not allow deleting system default strategy
	var st Strategy
	if err := s.db.Where("id = ?", id).First(&st).Error; err == nil {
		if st.IsDefault {
			return fmt.Errorf("cannot delete system default strategy")
		}
		if st.IsActive {
			return fmt.Errorf("cannot delete active strategy")
		}
	}

	// Check if any trader references this strategy
	var count int64
	if err := s.db.Model(&Trader{}).
		Where("user_id = ? AND strategy_id = ?", userID, id).
		Count(&count).Error; err == nil && count > 0 {
		return fmt.Errorf("cannot delete strategy in use by %d trader(s) - reassign those traders first", count)
	}

	return s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&Strategy{}).Error
}

// List get user's strategy list
func (s *StrategyStore) List(userID string) ([]*Strategy, error) {
	var strategies []*Strategy
	err := s.db.Where("user_id = ? OR is_default = ?", userID, true).
		Order("is_default DESC, created_at DESC").
		Find(&strategies).Error
	if err != nil {
		return nil, err
	}
	return strategies, nil
}

// ListPublic get all public strategies for the strategy market
func (s *StrategyStore) ListPublic() ([]*Strategy, error) {
	var strategies []*Strategy
	err := s.db.Where("is_public = ?", true).
		Order("created_at DESC").
		Find(&strategies).Error
	if err != nil {
		return nil, err
	}
	return strategies, nil
}

// Get get a single strategy
func (s *StrategyStore) Get(userID, id string) (*Strategy, error) {
	var st Strategy
	err := s.db.Where("id = ? AND (user_id = ? OR is_default = ?)", id, userID, true).
		First(&st).Error
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// GetActive get user's currently active strategy
func (s *StrategyStore) GetActive(userID string) (*Strategy, error) {
	var st Strategy
	err := s.db.Where("user_id = ? AND is_active = ?", userID, true).First(&st).Error
	if err == gorm.ErrRecordNotFound {
		// no active strategy, return system default strategy
		return s.GetDefault()
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// GetDefault get system default strategy
func (s *StrategyStore) GetDefault() (*Strategy, error) {
	var st Strategy
	err := s.db.Where("is_default = ?", true).First(&st).Error
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// SetActive set active strategy (will first deactivate other strategies)
func (s *StrategyStore) SetActive(userID, strategyID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// first deactivate all strategies for the user
		if err := tx.Model(&Strategy{}).Where("user_id = ?", userID).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// activate specified strategy
		return tx.Model(&Strategy{}).
			Where("id = ? AND (user_id = ? OR is_default = ?)", strategyID, userID, true).
			Update("is_active", true).Error
	})
}

// Duplicate duplicate a strategy (used to create custom strategy based on default strategy)
func (s *StrategyStore) Duplicate(userID, sourceID, newID, newName string) error {
	// get source strategy
	source, err := s.Get(userID, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get source strategy: %w", err)
	}

	// create new strategy
	newStrategy := &Strategy{
		ID:          newID,
		UserID:      userID,
		Name:        newName,
		Description: "Created based on [" + source.Name + "]",
		IsActive:    false,
		IsDefault:   false,
		Config:      source.Config,
	}

	return s.Create(newStrategy)
}

// ParseConfig parse strategy configuration JSON
func (s *Strategy) ParseConfig() (*StrategyConfig, error) {
	var config StrategyConfig
	if err := json.Unmarshal([]byte(s.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse strategy configuration: %w", err)
	}
	config.applyMissingDefaults()
	return &config, nil
}

// applyMissingDefaults backfills GetDefaultStrategyConfig values for config
// blocks that were persisted empty, so a strategy saved without a coin_source
// or without a klines block never silently falls back to a blank coin source
// (kernel/engine.go default case "unknown coin source type") or to the crypto
// "3m" primary timeframe (kernel/engine_analysis.go), which NT8 never subscribes
// (it auto-subscribes 5m/15m/1h). Only genuinely-empty blocks are filled;
// explicitly-set values are preserved. This is the durable twin of the
// per-strategy DB fixes for coin_source and klines.
func (c *StrategyConfig) applyMissingDefaults() {
	def := GetDefaultStrategyConfig(c.Language)

	// Coin source: a blank source_type with no static coins and no source flags
	// means the block was never set. Default the TYPE to "static" — matching the
	// engine-level guard in GetCandidateCoins (commit abda753d) so the two layers
	// agree — rather than the crypto ai500 default, which would be wrong for a
	// futures trader. An empty static list then degrades to the upstream
	// "no candidates" path instead of the unknown-type hard error.
	if c.CoinSource.SourceType == "" && len(c.CoinSource.StaticCoins) == 0 &&
		!c.CoinSource.UseAI500 && !c.CoinSource.UseOITop && !c.CoinSource.UseOILow &&
		!c.CoinSource.UseHyperAll && !c.CoinSource.UseHyperMain {
		c.CoinSource.SourceType = "static"
	}

	// Klines: no selected timeframes and no primary timeframe means the block was
	// never set -> adopt the default timeframe set (5m/15m/1h, primary 5m), which
	// matches the NT8 auto-subscribed set and avoids the engine "3m" fallback.
	if len(c.Indicators.Klines.SelectedTimeframes) == 0 && c.Indicators.Klines.PrimaryTimeframe == "" {
		c.Indicators.Klines.SelectedTimeframes = def.Indicators.Klines.SelectedTimeframes
		c.Indicators.Klines.PrimaryTimeframe = def.Indicators.Klines.PrimaryTimeframe
		c.Indicators.Klines.LongerTimeframe = def.Indicators.Klines.LongerTimeframe
		c.Indicators.Klines.EnableMultiTimeframe = true
		if c.Indicators.Klines.PrimaryCount == 0 {
			c.Indicators.Klines.PrimaryCount = def.Indicators.Klines.PrimaryCount
		}
		if c.Indicators.Klines.LongerCount == 0 {
			c.Indicators.Klines.LongerCount = def.Indicators.Klines.LongerCount
		}
		// raw OHLCV is required for AI analysis
		c.Indicators.EnableRawKlines = true
	}
}

// SetConfig set strategy configuration
func (s *Strategy) SetConfig(config *StrategyConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to serialize strategy configuration: %w", err)
	}
	s.Config = string(data)
	return nil
}

// ============================================================================
// Token Estimation
// ============================================================================

// TokenEstimate holds the result of token estimation
type TokenEstimate struct {
	Total       int            `json:"total"`
	Breakdown   TokenBreakdown `json:"breakdown"`
	ModelLimits []ModelLimit   `json:"model_limits"`
	Suggestions []string       `json:"suggestions"`
}

// TokenBreakdown shows estimated tokens per component
type TokenBreakdown struct {
	SystemPrompt  int `json:"system_prompt"`
	MarketData    int `json:"market_data"`
	RankingData   int `json:"ranking_data"`
	QuantData     int `json:"quant_data"`
	FixedOverhead int `json:"fixed_overhead"`
}

// ModelLimit shows token usage against a specific model's context limit
type ModelLimit struct {
	Name         string `json:"name"`
	ContextLimit int    `json:"context_limit"`
	UsagePct     int    `json:"usage_pct"`
	Level        string `json:"level"` // "ok" | "warning" | "danger"
}

// Context window sizes (tokens) for each model family
const (
	contextLimitDeepSeek = 131_072   // 128K
	contextLimitOpenAI   = 128_000   // 128K
	contextLimitClaude   = 200_000   // 200K
	contextLimitQwen     = 131_072   // 128K
	contextLimitGemini   = 1_000_000 // 1M
	contextLimitGrok     = 131_072   // 128K
	contextLimitKimi     = 131_072   // 128K
	contextLimitMinimax  = 1_000_000 // 1M
)

// ModelContextLimits maps provider names to their context window sizes (in tokens)
var ModelContextLimits = map[string]int{
	"deepseek": contextLimitDeepSeek,
	"openai":   contextLimitOpenAI,
	"claude":   contextLimitClaude,
	"qwen":     contextLimitQwen,
	"gemini":   contextLimitGemini,
	"grok":     contextLimitGrok,
	"kimi":     contextLimitKimi,
	"minimax":  contextLimitMinimax,
}

// GetContextLimit returns the context limit for a given provider
func GetContextLimit(provider string) int {
	if limit, ok := ModelContextLimits[provider]; ok {
		return limit
	}
	return contextLimitDeepSeek // safe default
}

// GetContextLimitForClient returns context limit for a provider+model pair.
// For claw402, the underlying model is inferred from the model name prefix.
func GetContextLimitForClient(provider, model string) int {
	if provider == "claw402" {
		switch {
		case strings.HasPrefix(model, "claude"):
			return ModelContextLimits["claude"]
		case strings.HasPrefix(model, "gpt"), strings.HasPrefix(model, "o1"), strings.HasPrefix(model, "o3"):
			return ModelContextLimits["openai"]
		case strings.HasPrefix(model, "gemini"):
			return ModelContextLimits["gemini"]
		case strings.HasPrefix(model, "grok"):
			return ModelContextLimits["grok"]
		case strings.HasPrefix(model, "kimi"):
			return ModelContextLimits["kimi"]
		case strings.HasPrefix(model, "qwen"):
			return ModelContextLimits["qwen"]
		case strings.HasPrefix(model, "minimax"):
			return ModelContextLimits["minimax"]
		case strings.HasPrefix(model, "deepseek"):
			return ModelContextLimits["deepseek"]
		default:
			return ModelContextLimits["deepseek"]
		}
	}
	return GetContextLimit(provider)
}

// EstimateTokens estimates the total token count for a strategy configuration.
// This is a pure computation based on config fields — no network calls.
func (c *StrategyConfig) EstimateTokens() TokenEstimate {
	breakdown := TokenBreakdown{}

	// --- System Prompt ---
	// Base system prompt: schema + role + rules + output format
	baseChars := 4000 // English default
	if c.Language == "zh" {
		baseChars = 3000
	}
	// Add prompt sections
	baseChars += len(c.PromptSections.RoleDefinition)
	baseChars += len(c.PromptSections.TradingFrequency)
	baseChars += len(c.PromptSections.EntryStandards)
	baseChars += len(c.PromptSections.DecisionProcess)
	baseChars += len(c.CustomPrompt)

	if c.Language == "zh" {
		breakdown.SystemPrompt = baseChars / 2 // CJK: ~2 chars per token
	} else {
		breakdown.SystemPrompt = baseChars / 4 // English: ~4 chars per token
	}

	// --- Fixed Overhead ---
	// Time, BTC price, account info, section headers
	breakdown.FixedOverhead = 800 / 4 // ~200 tokens

	// --- Market Data ---
	numCoins := c.getEffectiveCoinCount()
	numTimeframes := c.getEffectiveTimeframeCount()
	klineCount := c.Indicators.Klines.PrimaryCount
	if klineCount <= 0 {
		klineCount = 20
	}

	// Per coin per timeframe: kline OHLCV rows
	charsPerCoinTF := klineCount * 80 // each OHLCV line ~80 chars

	// Add enabled indicator overhead per timeframe
	indicatorCharsPerLine := 0
	if c.Indicators.EnableEMA {
		indicatorCharsPerLine += 20 // EMA values appended
	}
	if c.Indicators.EnableMACD {
		indicatorCharsPerLine += 30
	}
	if c.Indicators.EnableRSI {
		indicatorCharsPerLine += 15
	}
	if c.Indicators.EnableATR {
		indicatorCharsPerLine += 15
	}
	if c.Indicators.EnableBOLL {
		indicatorCharsPerLine += 25
	}
	if c.Indicators.EnableVolume {
		indicatorCharsPerLine += 10
	}
	charsPerCoinTF += klineCount * indicatorCharsPerLine

	totalMarketChars := numCoins * numTimeframes * charsPerCoinTF

	// OI + Funding per coin
	if c.Indicators.EnableOI || c.Indicators.EnableFundingRate {
		totalMarketChars += numCoins * 100
	}

	breakdown.MarketData = totalMarketChars / 4 // numeric data: ~4 chars per token

	// --- Quant Data ---
	if c.Indicators.EnableQuantData {
		quantCharsPerCoin := 0
		if c.Indicators.EnableQuantOI {
			quantCharsPerCoin += 300
		}
		if c.Indicators.EnableQuantNetflow {
			quantCharsPerCoin += 300
		}
		breakdown.QuantData = (numCoins * quantCharsPerCoin) / 4
	}

	// --- Ranking Data ---
	rankingChars := 0
	if c.Indicators.EnableOIRanking {
		limit := c.Indicators.OIRankingLimit
		if limit <= 0 {
			limit = 10
		}
		rankingChars += limit * 60
	}
	if c.Indicators.EnableNetFlowRanking {
		limit := c.Indicators.NetFlowRankingLimit
		if limit <= 0 {
			limit = 10
		}
		rankingChars += limit * 80
	}
	if c.Indicators.EnablePriceRanking {
		limit := c.Indicators.PriceRankingLimit
		if limit <= 0 {
			limit = 10
		}
		// Count durations (comma-separated)
		numDurations := 1
		if c.Indicators.PriceRankingDuration != "" {
			numDurations = len(strings.Split(c.Indicators.PriceRankingDuration, ","))
		}
		rankingChars += limit * numDurations * 40
	}
	breakdown.RankingData = rankingChars / 4

	// --- Total with 15% safety margin ---
	subtotal := breakdown.SystemPrompt + breakdown.MarketData + breakdown.RankingData + breakdown.QuantData + breakdown.FixedOverhead
	total := subtotal * 115 / 100

	// --- Model limits ---
	modelLimits := make([]ModelLimit, 0, len(ModelContextLimits))
	for name, limit := range ModelContextLimits {
		pct := total * 100 / limit
		level := "ok"
		if pct >= 100 {
			level = "danger"
		} else if pct >= 80 {
			level = "warning"
		}
		modelLimits = append(modelLimits, ModelLimit{
			Name:         name,
			ContextLimit: limit,
			UsagePct:     pct,
			Level:        level,
		})
	}

	// Sort by usage_pct desc, then name asc for deterministic order
	sort.Slice(modelLimits, func(i, j int) bool {
		if modelLimits[i].UsagePct != modelLimits[j].UsagePct {
			return modelLimits[i].UsagePct > modelLimits[j].UsagePct
		}
		return modelLimits[i].Name < modelLimits[j].Name
	})

	// --- Suggestions ---
	var suggestions []string
	// Find the strictest model (smallest context)
	minLimit := 0
	for _, limit := range ModelContextLimits {
		if minLimit == 0 || limit < minLimit {
			minLimit = limit
		}
	}
	if minLimit > 0 && total > minLimit {
		if numTimeframes > 1 {
			savedPerTF := (numCoins * klineCount * (80 + indicatorCharsPerLine)) / 4 * 115 / 100
			suggestions = append(suggestions, fmt.Sprintf("Reduce 1 timeframe to save ~%d tokens", savedPerTF))
		}
		if numCoins > 1 {
			savedPerCoin := (numTimeframes * klineCount * (80 + indicatorCharsPerLine)) / 4 * 115 / 100
			suggestions = append(suggestions, fmt.Sprintf("Reduce 1 coin to save ~%d tokens", savedPerCoin))
		}
		if klineCount > 15 {
			suggestions = append(suggestions, "Reduce K-line count to 15 to save tokens")
		}
	}

	return TokenEstimate{
		Total:       total,
		Breakdown:   breakdown,
		ModelLimits: modelLimits,
		Suggestions: suggestions,
	}
}

// getEffectiveCoinCount returns the estimated number of coins that will be analyzed
func (c *StrategyConfig) getEffectiveCoinCount() int {
	count := 0
	switch c.CoinSource.SourceType {
	case "static":
		count = len(c.CoinSource.StaticCoins)
	case "ai500":
		count = c.CoinSource.AI500Limit
	case "oi_top":
		count = c.CoinSource.OITopLimit
	case "oi_low":
		count = c.CoinSource.OILowLimit
	default:
		count = c.CoinSource.AI500Limit
	}
	if count <= 0 {
		count = 3
	}
	return count
}

// getEffectiveTimeframeCount returns the number of timeframes that will be used
func (c *StrategyConfig) getEffectiveTimeframeCount() int {
	if len(c.Indicators.Klines.SelectedTimeframes) > 0 {
		return len(c.Indicators.Klines.SelectedTimeframes)
	}
	count := 1
	if c.Indicators.Klines.LongerTimeframe != "" {
		count++
	}
	return count
}
