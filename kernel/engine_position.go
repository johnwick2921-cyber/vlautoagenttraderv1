package kernel

import (
	"fmt"
	"nofx/logger"
	"nofx/market"
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

func validateDecisions(decisions []Decision, accountEquity float64, btcEthLeverage, altcoinLeverage int, btcEthPosRatio, altcoinPosRatio float64, minRiskReward float64, minConfidence int, maxNotionalLev float64) error {
	for i := range decisions {
		if err := validateDecision(&decisions[i], accountEquity, btcEthLeverage, altcoinLeverage, btcEthPosRatio, altcoinPosRatio, minRiskReward, minConfidence, maxNotionalLev); err != nil {
			return fmt.Errorf("decision #%d validation failed: %w", i+1, err)
		}
	}
	return nil
}

func validateDecision(d *Decision, accountEquity float64, btcEthLeverage, altcoinLeverage int, btcEthPosRatio, altcoinPosRatio float64, minRiskReward float64, minConfidence int, maxNotionalLev float64) error {
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

		var entryPrice float64
		if d.Action == "open_long" {
			entryPrice = d.StopLoss + (d.TakeProfit-d.StopLoss)*0.2
		} else {
			entryPrice = d.StopLoss - (d.StopLoss-d.TakeProfit)*0.2
		}

		var riskPercent, rewardPercent, riskRewardRatio float64
		if d.Action == "open_long" {
			riskPercent = (entryPrice - d.StopLoss) / entryPrice * 100
			rewardPercent = (d.TakeProfit - entryPrice) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		} else {
			riskPercent = (d.StopLoss - entryPrice) / entryPrice * 100
			rewardPercent = (entryPrice - d.TakeProfit) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		}

		// STRATEGY STUDIO PHASE 1: the R/R floor is now the per-strategy
		// min_risk_reward_ratio (CODE ENFORCED), not a hardcoded 3.0. Unset (≤0)
		// falls back to 3.0, preserving prior behavior. Applies to crypto + futures.
		effRR := minRiskReward
		if effRR <= 0 {
			effRR = 3.0
		}
		if riskRewardRatio < effRR {
			return fmt.Errorf("risk/reward ratio too low (%.2f:1), must be ≥%.1f:1 [risk: %.2f%% reward: %.2f%%] [stop loss: %.2f take profit: %.2f]",
				riskRewardRatio, effRR, riskPercent, rewardPercent, d.StopLoss, d.TakeProfit)
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
