// Package ninjatrader implements the file-CSV bridge to NinjaTrader 8's
// vltrader.cs NinjaScript strategy. Go writes trade_signals.csv (5 fields)
// which NT8 polls every 2 seconds; NT8 appends to trades_taken.csv (3 fields)
// which Go tails. No sockets, no daemon — atomic file ops only.
//
// Verified against vltrader.cs source — see
// https://github.com/J0shusmc/Claude-Trader-NinjaTrader.
package ninjatrader

import (
	"fmt"
	"strings"
)

// SignalRow is one row of trade_signals.csv that vltrader.cs consumes.
type SignalRow struct {
	DateTime   string // MM/dd/yyyy HH:mm:ss — vltrader.cs DateTime format
	Direction  string // "LONG" or "SHORT"
	EntryPrice float64
	StopLoss   float64
	TakeProfit float64
}

// FillRow is one row of trades_taken.csv that vltrader.cs appends.
type FillRow struct {
	DateTime   string // MM/dd/yyyy HH:mm:ss
	Direction  string // "LONG" or "SHORT"
	EntryPrice float64
}

const signalsHeader = "DateTime,Direction,Entry_Price,Stop_Loss,Take_Profit"
const fillsHeader = "DateTime,Direction,Entry_Price"

// Validate rejects malformed signals before they reach disk.
func (s SignalRow) Validate() error {
	if s.DateTime == "" {
		return fmt.Errorf("ninjatrader signal: empty DateTime")
	}
	dir := strings.ToUpper(s.Direction)
	if dir != "LONG" && dir != "SHORT" {
		return fmt.Errorf("ninjatrader signal: direction must be LONG or SHORT, got %q", s.Direction)
	}
	if s.EntryPrice <= 0 || s.StopLoss <= 0 || s.TakeProfit <= 0 {
		return fmt.Errorf("ninjatrader signal: prices must be positive")
	}
	if dir == "LONG" && s.StopLoss >= s.EntryPrice {
		return fmt.Errorf("ninjatrader signal: LONG stop_loss (%.2f) must be below entry (%.2f)", s.StopLoss, s.EntryPrice)
	}
	if dir == "LONG" && s.TakeProfit <= s.EntryPrice {
		return fmt.Errorf("ninjatrader signal: LONG take_profit (%.2f) must be above entry (%.2f)", s.TakeProfit, s.EntryPrice)
	}
	if dir == "SHORT" && s.StopLoss <= s.EntryPrice {
		return fmt.Errorf("ninjatrader signal: SHORT stop_loss (%.2f) must be above entry (%.2f)", s.StopLoss, s.EntryPrice)
	}
	if dir == "SHORT" && s.TakeProfit >= s.EntryPrice {
		return fmt.Errorf("ninjatrader signal: SHORT take_profit (%.2f) must be below entry (%.2f)", s.TakeProfit, s.EntryPrice)
	}
	return nil
}
