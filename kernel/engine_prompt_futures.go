package kernel

import (
	"fmt"
	"strings"

	"nofx/market"
)

// BuildFuturesDecisionSystemPrompt builds the CME index-futures (MNQ) system
// prompt. Unlike the standalone BuildFuturesSystemPrompt above (which emits an
// incompatible LONG/SHORT/NONE shape), this emits the SAME <reasoning> +
// <decision> JSON-array envelope the live parser (parseFullDecisionResponse)
// already consumes — the 6-action enum (open_long/open_short/close_long/
// close_short/hold/wait) with symbol/entry/stop_loss/take_profit/confidence —
// so futures decisions flow through the existing parse → validate → execute
// pipeline unchanged. Selected via the "futures" variant in BuildSystemPrompt.
//
// Futures framing only: MNQ point value/tick, CME session awareness, absolute
// tick-aligned stops. NO funding rate, NO open interest, NO crypto leverage
// tiers (leverage is fixed at 1 — futures margin is contract-based, handled by
// the broker).
func (e *StrategyEngine) BuildFuturesDecisionSystemPrompt(symbol string, accountEquity float64) string {
	return e.buildFuturesPrompt(symbol, accountEquity, "balanced")
}

// futures sub-mode blocks (Phase 2) — TEXT only; they never change the Hard
// Constraints / output format / Risk Control. "balanced" injects NO block.
const (
	futuresModeAggressive   = "## Mode: Aggressive\n- Favor acting on high-confidence setups earlier; a strong 5m/15m alignment can justify entry without waiting for full multi-timeframe confluence.\n- Still obey EVERY Hard Constraint above (R/R, min confidence, one position, tick-aligned stops).\n\n"
	futuresModeConservative = "## Mode: Conservative\n- Act only on strong multi-timeframe confluence; prefer wait/hold in chop or on conflicting signals.\n- Be more selective; after a loss, wait for a clean fresh setup.\n- Still obey EVERY Hard Constraint above.\n\n"
)

// futuresVariantMode reports whether a prompt variant is a futures variant and
// returns its sub-mode: "futures"/"futures-balanced" → (true,"balanced");
// "futures-aggressive" → (true,"aggressive"); "futures-conservative" →
// (true,"conservative"). Non-futures variants → (false,""). An unknown suffix
// falls back to the no-block balanced default (safe).
func futuresVariantMode(variant string) (bool, string) {
	v := strings.ToLower(strings.TrimSpace(variant))
	if v == "futures" {
		return true, "balanced"
	}
	if strings.HasPrefix(v, "futures-") {
		mode := strings.TrimPrefix(v, "futures-")
		if mode == "" {
			mode = "balanced"
		}
		return true, mode
	}
	return false, ""
}

// buildFuturesPrompt builds the CME futures system prompt for a sub-mode
// ("balanced" | "aggressive" | "conservative"). A balanced/empty/unknown mode
// injects NO "## Mode:" block, so it is byte-identical to the prior fixed
// futures prompt (the golden). The sub-mode is TEXT only.
func (e *StrategyEngine) buildFuturesPrompt(symbol string, accountEquity float64, mode string) string {
	var sb strings.Builder
	rc := e.config.RiskControl
	ps := e.config.PromptSections // the 4 editable prompt boxes (Change 4)
	minConf := rc.MinConfidence
	if minConf <= 0 {
		minConf = 60
	}
	minRR := rc.MinRiskRewardRatio
	if minRR <= 0 {
		minRR = 1.5
	}

	// Instrument identity from the ACTIVE symbol (Phase 3 — was hardwired MNQ).
	// Resolving families (index + treasury) get their real name / point value /
	// tick; parked/unknown symbols default to MNQ so the prompt never emits a
	// blank or wrong instrument.
	inst := describeFutures(symbol)
	sym := inst.Symbol
	category, pointWord, priceAdj := "index-futures", "index point", "index "
	if inst.IsTreasury {
		category, pointWord, priceAdj = "Treasury futures", "point", ""
	}
	tickStr := fmt.Sprintf("%g", inst.TickSize)
	pvDec := fmt.Sprintf("%.2f", inst.PointValue)
	pvInt := fmt.Sprintf("%g", inst.PointValue)

	// NB: we deliberately do NOT prepend the crypto GetSchemaPrompt here — it
	// describes USDT-perp fields and would re-introduce the crypto framing this
	// prompt exists to avoid. The market data in the user prompt is
	// self-describing (current_price + OHLCV timeframe tables).

	// 1. Role (editable via the Role Definition box; FIXED CME role when empty).
	// Honored again — the owner's typed role reaches the futures prompt (full
	// control). The earlier crypto leak is fixed at the DATA layer (the box
	// defaults are neutral + a one-time migration cleaned the old crypto boxes),
	// NOT by ignoring the box. The Instrument identity block below is ALWAYS
	// FIXED (futures-specific, never box-driven).
	if ps.RoleDefinition != "" {
		sb.WriteString(ps.RoleDefinition)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("# You are a professional CME " + category + " trading AI specializing in the " + inst.Desc + " (" + sym + ").\n\n")
	}
	sb.WriteString("## Instrument\n")
	sb.WriteString("- Symbol: " + sym + " (" + inst.Desc + " futures)\n")
	sb.WriteString("- Tick size: " + tickStr + " " + pointWord + "s\n")
	sb.WriteString("- Contract multiplier: $" + pvDec + " per " + pointWord + " (1 point = $" + pvInt + ")\n")
	sb.WriteString("- This is a FUTURES contract, NOT a crypto perpetual: there is NO funding rate and NO crypto-style open interest. Ignore any empty Funding Rate / Open Interest sections in the market data.\n")
	sb.WriteString("- Session: CME futures hours (nearly 23h on weekdays, with a daily maintenance break). Do NOT assume 24/7 trading.\n\n")

	// 2. Hard constraints.
	sb.WriteString("# Hard Constraints (Risk Control)\n\n")
	sb.WriteString("- Trade ONLY the " + sym + " symbol provided in the market data. Do NOT invent other symbols.\n")
	sb.WriteString("- Every open_long / open_short MUST include stop_loss and take_profit as ABSOLUTE " + priceAdj + "prices (e.g. 21500.25), in tick increments (multiples of " + tickStr + "), NOT deltas.\n")
	sb.WriteString("- Stop distance: roughly 1.5-3x ATR; sanity range ~15-50 points from entry.\n")
	sb.WriteString(fmt.Sprintf("- Risk/Reward: reward must be at least %.2fx the risk (take_profit vs stop_loss distance from entry).\n", minRR))
	sb.WriteString(fmt.Sprintf("- Min confidence to open: %d. Below that, use hold or wait.\n", minConf))
	sb.WriteString("- One position at a time. No averaging in / pyramiding.\n")
	sb.WriteString("- leverage: always 1 for futures (margin is contract-based; the broker handles it). Do NOT use crypto leverage tiers.\n")
	sb.WriteString("- position_size_usd: the contract notional you intend (≈ price × $" + pvInt + " × contracts). Keep it conservative (start with 1 contract).\n\n")

	// 2a. Trading posture sub-mode (TEXT only; "balanced"/""/unknown = NO block =
	// byte-identical default). Mirrors the crypto "## Mode:" injection and stays
	// WITHIN the Hard Constraints above — it never changes the risk rules.
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "aggressive":
		sb.WriteString(futuresModeAggressive)
	case "conservative":
		sb.WriteString(futuresModeConservative)
	}

	// 2b. Trading Frequency (editable). Appended ONLY when the box is set, so an
	// empty box leaves the futures prompt byte-identical to the prior fixed text.
	if ps.TradingFrequency != "" {
		sb.WriteString("# Trading Frequency\n")
		sb.WriteString(ps.TradingFrequency)
		sb.WriteString("\n\n")
	}

	// 3. Indicators available.
	sb.WriteString("# Available Data\n")
	sb.WriteString("Multi-timeframe " + sym + " bars (")
	e.writeAvailableIndicators(&sb)
	sb.WriteString(fmt.Sprintf("Use confluence across timeframes. Confidence ≥ %d required to open.\n\n", minConf))

	// 3a. Volume Profile (SVP) — Part B3. Gated on per-strategy svp_enabled AND a
	// non-empty computed line (threaded in from engine_analysis.go, empty on the
	// preview/test paths). OFF or empty writes NOTHING, so the futures golden stays
	// byte-identical. ONE context line + ONE legend line, consistent with the
	// regime/decision framing above (POC as a magnet; balance vs trend).
	if boolOrDefault(rc.SvpEnabled, false) && e.svpContextLine != "" {
		sb.WriteString(e.svpContextLine)
		sb.WriteString("\n")
		sb.WriteString("Legend: POC = the session's highest-volume price (a magnet). Inside the value area (VAL–VAH) = balanced → fade the edges back toward POC. Holding OUTSIDE the value area on volume = trend → join the move, don't fade it.\n\n")
	}

	// 3b. Entry Standards (editable). Appended ONLY when set — empty = unchanged.
	if ps.EntryStandards != "" {
		sb.WriteString("# Entry Standards\n")
		sb.WriteString(ps.EntryStandards)
		sb.WriteString("\n\n")
	}

	// 4. Decision process (editable via the Decision Process box; FIXED steps
	// when empty). Honored again — the owner's typed decision flow reaches the
	// futures prompt. The crypto-default leak is fixed at the DATA layer, not by
	// ignoring the box.
	if ps.DecisionProcess != "" {
		sb.WriteString(ps.DecisionProcess)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("# Decision Process\n")
		sb.WriteString("1. If a position is open: should it be held, or closed (close_long/close_short) for profit/stop?\n")
		sb.WriteString("2. If flat: do the 5m/15m/1h bars + indicators show a high-confidence directional setup?\n")
		sb.WriteString("3. Write your chain of thought, THEN output the structured JSON decision.\n")
		sb.WriteString("4. action=wait (no setup) and action=hold (keep current position) are valid, frequently-correct answers. Do NOT force a trade.\n\n")
	}

	// 5. Output format — MUST match the existing parser exactly.
	sb.WriteString("# Output Format (Strictly Follow)\n\n")
	sb.WriteString("**Must use XML tags <reasoning> and <decision> to separate chain of thought and decision JSON, avoiding parsing errors**\n\n")
	sb.WriteString("<reasoning>\n")
	sb.WriteString("Your chain-of-thought analysis of the " + sym + " bars and indicators.\n")
	sb.WriteString("</reasoning>\n\n")
	sb.WriteString("<decision>\n")
	sb.WriteString("```json\n[\n")
	sb.WriteString("  {\"symbol\": \"" + sym + "\", \"action\": \"open_long\", \"leverage\": 1, \"position_size_usd\": 60000, \"stop_loss\": 21480.00, \"take_profit\": 21560.00, \"confidence\": 80}\n")
	sb.WriteString("]\n```\n")
	sb.WriteString("</decision>\n\n")
	sb.WriteString("When there is no good setup, output a single wait decision:\n")
	sb.WriteString("<decision>\n```json\n[{\"symbol\": \"" + sym + "\", \"action\": \"wait\"}]\n```\n</decision>\n\n")

	// 6. Field description.
	sb.WriteString("## Field Description\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait\n")
	sb.WriteString("- `symbol`: always \"" + sym + "\"\n")
	sb.WriteString(fmt.Sprintf("- `confidence`: 0-100 (open only when ≥ %d)\n", minConf))
	sb.WriteString("- `leverage`: 1 (futures)\n")
	sb.WriteString("- Required when opening: stop_loss, take_profit, confidence (absolute tick-aligned prices)\n")
	sb.WriteString("- **IMPORTANT**: all numeric values must be concrete numbers, NOT formulas (e.g. `21480.00`, not `21500 - 20`).\n")
	sb.WriteString("- The <decision> block MUST be a JSON array, even for a single decision.\n\n")

	if e.config.CustomPrompt != "" {
		sb.WriteString("# Personalized Strategy\n\n")
		sb.WriteString(e.config.CustomPrompt)
		sb.WriteString("\n\nNote: supplements the rules above; cannot violate the risk-control constraints.\n")
	}

	return sb.String()
}

// futuresInstrument is the per-symbol identity the futures decision prompt needs.
type futuresInstrument struct {
	Symbol     string  // bare root, e.g. "MNQ"
	Desc       string  // "Micro E-mini Nasdaq-100"
	PointValue float64 // $/point (market.FuturesPointValue)
	TickSize   float64 // 0.25 (market.FuturesTickSize)
	IsTreasury bool    // CBOT Treasury family (different price/point wording)
}

// futuresPromptDesc holds the prompt wording per RESOLVING-family root (index +
// treasury, Phases 1-2). Energy/metals are parked (Phase 2.5) and intentionally
// absent — describeFutures defaults them (and any unknown) to MNQ.
var futuresPromptDesc = map[string]struct {
	desc       string
	isTreasury bool
}{
	"MNQ": {"Micro E-mini Nasdaq-100", false},
	"NQ":  {"E-mini Nasdaq-100", false},
	"ES":  {"E-mini S&P 500", false},
	"MES": {"Micro E-mini S&P 500", false},
	"RTY": {"E-mini Russell 2000", false},
	"M2K": {"Micro E-mini Russell 2000", false},
	"YM":  {"E-mini Dow", false},
	"MYM": {"Micro E-mini Dow", false},
	"ZB":  {"30-Year U.S. Treasury Bond", true},
	"ZN":  {"10-Year U.S. T-Note", true},
	"ZF":  {"5-Year U.S. T-Note", true},
	"ZT":  {"2-Year U.S. T-Note", true},
}

// describeFutures resolves the instrument identity for the system prompt. A
// resolving family gets its real name / point value / tick; anything else
// (energy/metals parked, or unknown) safely defaults to MNQ so the prompt never
// emits a blank or wrong instrument.
func describeFutures(symbol string) futuresInstrument {
	root := strings.ToUpper(strings.TrimSpace(symbol))
	if i := strings.IndexByte(root, ' '); i > 0 {
		root = root[:i] // "MNQ 06-26" -> "MNQ"
	}
	if i := strings.Index(root, ".C."); i > 0 {
		root = root[:i] // "NQ.C.0" -> "NQ"
	}
	d, ok := futuresPromptDesc[root]
	pv := market.FuturesPointValue(root)
	tick := market.FuturesTickSize(root)
	if !ok || pv <= 0 || tick <= 0 {
		return futuresInstrument{"MNQ", "Micro E-mini Nasdaq-100", 2.0, 0.25, false}
	}
	return futuresInstrument{root, d.desc, pv, tick, d.isTreasury}
}

// FuturesPromptConfig captures the few parameters the system prompt needs
// to describe an index-futures contract to the model.
type FuturesPromptConfig struct {
	Symbol             string  // "NQ" or "MNQ"
	ContractMultiplier float64 // NQ = 20 ($20/point), MNQ = 2 ($2/point)
	TickSize           float64 // 0.25 for both NQ and MNQ
	MinStopPoints      float64 // 15
	MaxStopPoints      float64 // 50
	MinRiskReward      float64 // 1.5
}

// FuturesContext is the per-cycle data shoved into the user prompt.
//
// NOTE: market.ExportCalculateMACD returns only the MACD line (one float64).
// Signal/histogram require extending the indicator API and are deferred.
type FuturesContext struct {
	Symbol       string
	CurrentPrice float64
	// indicator snapshot
	EMA20     float64
	EMA50     float64
	RSI14     float64
	MACD      float64 // MACD line only
	ATR14     float64
	BollUpper float64
	BollLower float64
}

func BuildFuturesSystemPrompt(c FuturesPromptConfig) string {
	var b strings.Builder
	b.WriteString("# You are a professional index-futures trading AI specializing in CME E-mini Nasdaq-100 contracts.\n\n")
	b.WriteString(fmt.Sprintf("## Instrument\n- Symbol: %s\n- Tick size: %.2f points\n- Contract multiplier: $%.2f per point\n\n", c.Symbol, c.TickSize, c.ContractMultiplier))
	b.WriteString("## Hard constraints\n")
	b.WriteString("- Every entry MUST include a stop loss and a take profit, expressed as absolute prices.\n")
	b.WriteString(fmt.Sprintf("- Stop loss distance: minimum %.0f points, maximum %.0f points from entry.\n", c.MinStopPoints, c.MaxStopPoints))
	b.WriteString(fmt.Sprintf("- Minimum risk/reward: %.2f (reward must be at least %.2fx the risk).\n", c.MinRiskReward, c.MinRiskReward))
	b.WriteString("- One position at a time. Do NOT propose averaging in or pyramiding.\n")
	b.WriteString("- Prices must be in tick increments (multiples of " + fmt.Sprintf("%.2f", c.TickSize) + ").\n")
	b.WriteString("- The market session is CME futures hours; do not assume 24/7 trading.\n\n")
	b.WriteString("## Decision output\n")
	b.WriteString("Respond ONLY with JSON of the following exact shape:\n")
	b.WriteString("```json\n")
	b.WriteString(`{"action":"LONG"|"SHORT"|"NONE","entry":0.00,"stop_loss":0.00,"take_profit":0.00,"reasoning":"<one-paragraph explanation>"}`)
	b.WriteString("\n```\n")
	b.WriteString("\n- `action=NONE` is a valid and frequently correct answer. Do not force a trade.\n")
	b.WriteString("- All three price fields are absolute (e.g. 21500.25), not deltas from entry.\n\n")
	b.WriteString("## Trade plan checklist (apply before answering LONG/SHORT)\n")
	b.WriteString("1. Is there a clear directional bias from EMA20 vs EMA50 alignment?\n")
	b.WriteString("2. Does RSI confirm or contradict that bias? (extreme = caution)\n")
	b.WriteString("3. Is MACD positive (for LONG) or negative (for SHORT)?\n")
	b.WriteString("4. Is ATR consistent with your proposed stop distance? Stop should be ~1.5-3x ATR.\n")
	b.WriteString("5. Where is the Bollinger band — overextended (mean revert) or trending (continuation)?\n")
	b.WriteString("6. Risk/reward calculation: (take_profit - entry) / (entry - stop_loss) for LONG. Must exceed " + fmt.Sprintf("%.2f", c.MinRiskReward) + ".\n")
	return b.String()
}

func BuildFuturesUserPrompt(ctx FuturesContext) string {
	var b strings.Builder
	b.WriteString("## Current market\n")
	b.WriteString(fmt.Sprintf("- Symbol: %s\n", ctx.Symbol))
	b.WriteString(fmt.Sprintf("- Current price: %.2f\n\n", ctx.CurrentPrice))
	b.WriteString("## Indicator snapshot (1-minute timeframe)\n")
	b.WriteString(fmt.Sprintf("- EMA20: %.2f (current price %s)\n", ctx.EMA20, sidePos(ctx.CurrentPrice, ctx.EMA20)))
	b.WriteString(fmt.Sprintf("- EMA50: %.2f (current price %s)\n", ctx.EMA50, sidePos(ctx.CurrentPrice, ctx.EMA50)))
	b.WriteString(fmt.Sprintf("- EMA20 vs EMA50: %s\n", emaAlignment(ctx.EMA20, ctx.EMA50)))
	b.WriteString(fmt.Sprintf("- RSI14: %.1f (%s)\n", ctx.RSI14, rsiBucket(ctx.RSI14)))
	b.WriteString(fmt.Sprintf("- MACD: %.2f (line only; signal/histogram require extended indicator API)\n", ctx.MACD))
	b.WriteString(fmt.Sprintf("- ATR14: %.2f points\n", ctx.ATR14))
	b.WriteString(fmt.Sprintf("- Bollinger Bands: upper %.2f, lower %.2f, position: %s\n", ctx.BollUpper, ctx.BollLower, bollPosition(ctx.CurrentPrice, ctx.BollUpper, ctx.BollLower)))
	b.WriteString("\n## Decision\nGive me your trade decision in the JSON format specified by the system prompt.\n")
	return b.String()
}

func sidePos(price, ref float64) string {
	if price > ref {
		return "above"
	}
	if price < ref {
		return "below"
	}
	return "equal"
}

func emaAlignment(ema20, ema50 float64) string {
	if ema20 > ema50 {
		return "bullish (20 > 50)"
	}
	if ema20 < ema50 {
		return "bearish (20 < 50)"
	}
	return "neutral"
}

func rsiBucket(r float64) string {
	switch {
	case r >= 70:
		return "overbought"
	case r <= 30:
		return "oversold"
	default:
		return "neutral"
	}
}

func bollPosition(p, upper, lower float64) string {
	switch {
	case p >= upper:
		return "above upper band (overextended)"
	case p <= lower:
		return "below lower band (overextended)"
	default:
		return "inside bands"
	}
}
