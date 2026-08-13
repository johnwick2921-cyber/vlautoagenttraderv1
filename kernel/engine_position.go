package kernel

import (
	"fmt"
	"nofx/logger"
	"nofx/market"
	"nofx/telemetry"
)

// futuresMaxNotionalLeverage is the notional sanity ceiling for CME futures
// in the risk gate: max position notional = equity × this. Futures are
// leveraged instruments (a $50k account holds ~$60k MNQ notional on ~$2.2k
// margin), so the crypto equity×ratio cap is wrong. This is a coarse ceiling
// that rejects absurd sizes; precise contract sizing + clamp happens in the
// executor (auto_trader_orders.go).
const futuresMaxNotionalLeverage = 20.0

// ============================================================================
// Decision Validation
// ============================================================================

func validateDecisions(decisions []Decision, accountEquity float64, btcEthLeverage, altcoinLeverage int, btcEthPosRatio, altcoinPosRatio float64, minRiskReward float64, minConfidence int, maxNotionalLev float64, ctx *Context) error {
	for i := range decisions {
		if err := validateDecision(&decisions[i], accountEquity, btcEthLeverage, altcoinLeverage, btcEthPosRatio, altcoinPosRatio, minRiskReward, minConfidence, maxNotionalLev, ctx); err != nil {
			return fmt.Errorf("decision #%d validation failed: %w", i+1, err)
		}
	}
	return nil
}

// validateDecision code-enforces the trade rules on one decision. ctx (nil-safe)
// supplies the F1 entry reference (MarketDataMap[symbol].CurrentPrice) and the
// TraderID for the rr_gate counter; pass nil in unit tests that don't exercise R:R.
func validateDecision(d *Decision, accountEquity float64, btcEthLeverage, altcoinLeverage int, btcEthPosRatio, altcoinPosRatio float64, minRiskReward float64, minConfidence int, maxNotionalLev float64, ctx *Context) error {
	validActions := map[string]bool{
		"open_long":   true,
		"open_short":  true,
		"close_long":  true,
		"close_short": true,
		"hold":        true,
		"wait":        true,
	}

	if !validActions[d.Action] {
		return fmt.Errorf("invalid action: %s", d.Action)
	}

	if d.Action == "open_long" || d.Action == "open_short" {
		maxLeverage := altcoinLeverage
		posRatio := altcoinPosRatio
		maxPositionValue := accountEquity * posRatio
		isFutures := market.IsCMEFuturesSymbol(d.Symbol)
		switch {
		case isFutures:
			// CME futures: a notional sanity ceiling (equity × the configured
			// multiplier; Chunk 3 made it editable, default 20). maxNotionalLev<=0
			// → the cap is DISABLED (master/toggle off): use a huge ceiling so it
			// never binds. The executor does the precise contract sizing + clamp.
			if maxNotionalLev > 0 {
				maxPositionValue = accountEquity * maxNotionalLev
			} else {
				maxPositionValue = accountEquity * 1e9
			}
		case d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT":
			maxLeverage = btcEthLeverage
			posRatio = btcEthPosRatio
			maxPositionValue = accountEquity * posRatio
		}

		if d.Leverage <= 0 {
			return fmt.Errorf("leverage must be greater than 0: %d", d.Leverage)
		}
		if d.Leverage > maxLeverage {
			logger.Infof("⚠️  [Leverage Fallback] %s leverage exceeded (%dx > %dx), auto-adjusting to limit %dx",
				d.Symbol, d.Leverage, maxLeverage, maxLeverage)
			d.Leverage = maxLeverage
		}
		if d.PositionSizeUSD <= 0 {
			return fmt.Errorf("position size must be greater than 0: %.2f", d.PositionSizeUSD)
		}

		const minPositionSizeGeneral = 12.0
		const minPositionSizeBTCETH = 60.0

		if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
			if d.PositionSizeUSD < minPositionSizeBTCETH {
				return fmt.Errorf("%s opening amount too small (%.2f USDT), must be ≥%.2f USDT", d.Symbol, d.PositionSizeUSD, minPositionSizeBTCETH)
			}
		} else {
			if d.PositionSizeUSD < minPositionSizeGeneral {
				return fmt.Errorf("opening amount too small (%.2f USDT), must be ≥%.2f USDT", d.PositionSizeUSD, minPositionSizeGeneral)
			}
		}

		tolerance := maxPositionValue * 0.01
		if d.PositionSizeUSD > maxPositionValue+tolerance {
			switch {
			case isFutures:
				return fmt.Errorf("%s futures notional cannot exceed %.0f USD (%.0fx account equity), actual: %.0f", d.Symbol, maxPositionValue, maxNotionalLev, d.PositionSizeUSD)
			case d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT":
				return fmt.Errorf("BTC/ETH single coin position value cannot exceed %.0f USDT (%.1fx account equity), actual: %.0f", maxPositionValue, posRatio, d.PositionSizeUSD)
			default:
				return fmt.Errorf("altcoin single coin position value cannot exceed %.0f USDT (%.1fx account equity), actual: %.0f", maxPositionValue, posRatio, d.PositionSizeUSD)
			}
		}
		if d.StopLoss <= 0 || d.TakeProfit <= 0 {
			return fmt.Errorf("stop loss and take profit must be greater than 0")
		}

		if d.Action == "open_long" {
			if d.StopLoss >= d.TakeProfit {
				return fmt.Errorf("for long positions, stop loss price must be less than take profit price")
			}
		} else {
			if d.StopLoss <= d.TakeProfit {
				return fmt.Errorf("for short positions, stop loss price must be greater than take profit price")
			}
		}

		// F1 — REAL risk:reward, computed from the ACTUAL entry reference. The old
		// code synthesized the entry 20% into the SL→TP range, which pinned R:R at
		// exactly 4.0 and made min_risk_reward_ratio a TAUTOLOGY (3.0 always passed;
		// >4 could never pass). Entry reference for a MARKET futures/crypto entry is
		// the current-price snapshot at decision time (the prompt's snapshot); an
		// explicit AI limit price (d.Price) wins when supplied. Wrong-side / zero-risk
		// is validateDecision's authority (B2 price-sanity is absolute-distance only),
		// so a non-positive risk is rejected HERE — it does not duplicate B2.
		entryRef := d.Price
		entrySource := "ai_limit_price"
		if entryRef <= 0 {
			entrySource = "current_price_snapshot"
			if ctx != nil {
				if md := ctx.MarketDataMap[d.Symbol]; md != nil {
					entryRef = md.CurrentPrice
				}
			}
		}

		// STRATEGY STUDIO PHASE 1: the R/R floor is the per-strategy
		// min_risk_reward_ratio (CODE ENFORCED), not a hardcoded 3.0. The store
		// clamps a saved value to ≥1; this ≤0 fallback only guards an unset call.
		effRR := minRiskReward
		if effRR <= 0 {
			effRR = 3.0
		}

		if entryRef <= 0 {
			// No entry reference (no market data for the symbol) → cannot honestly
			// evaluate R:R. Fail-open: skip the gate and let the stale-data /
			// price-sanity gates handle a dead feed. Never fabricate an entry.
			logger.Infof("ℹ️ R:R gate SKIPPED for %s %s — no entry reference (no market data); deferring to price-sanity / stale-data gates.", d.Symbol, d.Action)
		} else {
			var risk, reward float64
			if d.Action == "open_long" {
				risk = entryRef - d.StopLoss
				reward = d.TakeProfit - entryRef
			} else {
				risk = d.StopLoss - entryRef
				reward = entryRef - d.TakeProfit
			}
			traderID := ""
			if ctx != nil {
				traderID = ctx.TraderID
			}
			if risk <= 0 {
				// Stop at/through the entry → no risk to reward. This is the
				// wrong-side condition validateDecision owns (B2 doesn't catch it).
				telemetry.IncGateBlock(traderID, "rr_gate")
				logger.Warnf("📐 R:R REJECT %s %s: entry=%.4f (%s) SL=%.4f TP=%.4f → risk=%.4f ≤ 0 (stop at/through entry) — no risk to reward.",
					d.Symbol, d.Action, entryRef, entrySource, d.StopLoss, d.TakeProfit, risk)
				return fmt.Errorf("invalid risk: entry %.4f is at/through stop %.4f for %s (risk=%.4f ≤ 0) — no risk to reward", entryRef, d.StopLoss, d.Action, risk)
			}
			rr := reward / risk
			verdict := "PASS"
			if rr < effRR {
				verdict = "FAIL"
			}
			logger.Infof("📐 R:R eval %s %s: entry=%.4f (%s) SL=%.4f TP=%.4f → risk=%.4f reward=%.4f R:R=%.2f (min %.2f) → %s",
				d.Symbol, d.Action, entryRef, entrySource, d.StopLoss, d.TakeProfit, risk, reward, rr, effRR, verdict)
			if rr < effRR {
				telemetry.IncGateBlock(traderID, "rr_gate")
				return fmt.Errorf("risk/reward ratio too low (%.2f:1), must be ≥%.2f:1 [entry %.4f (%s) risk %.4f reward %.4f] [stop loss: %.4f take profit: %.4f]",
					rr, effRR, entryRef, entrySource, risk, reward, d.StopLoss, d.TakeProfit)
			}
		}

		// STRATEGY STUDIO PHASE 1: min confidence is now CODE ENFORCED — reject a
		// low-confidence open when min_confidence is configured (0 = disabled →
		// back-compat for strategies that never set it). Applies to crypto + futures.
		if minConfidence > 0 && d.Confidence < minConfidence {
			return fmt.Errorf("confidence too low (%d), must be ≥%d to open position", d.Confidence, minConfidence)
		}
	}

	return nil
}
