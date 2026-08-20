package store

import (
	"fmt"
	"math"
	"strings"
)

// TraderStats trading statistics metrics
type TraderStats struct {
	TotalTrades    int     `json:"total_trades"`
	WinTrades      int     `json:"win_trades"`
	LossTrades     int     `json:"loss_trades"`
	WinRate        float64 `json:"win_rate"`
	ProfitFactor   float64 `json:"profit_factor"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	TotalPnL       float64 `json:"total_pnl"`
	TotalFee       float64 `json:"total_fee"`
	AvgWin         float64 `json:"avg_win"`
	AvgLoss        float64 `json:"avg_loss"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
}

// GetPositionStats gets position statistics
func (s *PositionStore) GetPositionStats(traderID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	type result struct {
		Total    int
		Wins     int
		TotalPnL float64
		TotalFee float64
	}
	var r result

	err := s.db.Model(&TraderPosition{}).
		Select("COUNT(*) as total, SUM(CASE WHEN COALESCE(pnl_corrected, realized_pnl) > 0 THEN 1 ELSE 0 END) as wins, COALESCE(SUM(COALESCE(pnl_corrected, realized_pnl)), 0) as total_pnl, COALESCE(SUM(fee), 0) as total_fee").
		Where("trader_id = ? AND status = ?", traderID, "CLOSED").
		Scan(&r).Error
	if err != nil {
		return nil, err
	}

	stats["total_trades"] = r.Total
	stats["win_trades"] = r.Wins
	stats["total_pnl"] = r.TotalPnL
	stats["total_fee"] = r.TotalFee
	if r.Total > 0 {
		stats["win_rate"] = float64(r.Wins) / float64(r.Total) * 100
	} else {
		stats["win_rate"] = 0.0
	}

	return stats, nil
}

// CountConsecutiveLossesSince returns the number of consecutive LOSING closed
// trades at the TAIL of this trader's closed-trade history since sinceMs (the CME
// session-day start, Unix ms UTC). A winning or break-even close (realized_pnl >= 0)
// ends the streak; reconcile_flat orphans (unknown P&L) are excluded. Used by the
// D1 consecutive-loss halt. Scoped by trader_id (each trader's closes are recorded
// under its own id).
func (s *PositionStore) CountConsecutiveLossesSince(traderID string, sinceMs int64) (int, error) {
	var rows []TraderPosition
	err := s.db.
		Where("trader_id = ? AND status = ? AND close_reason <> ? AND exit_time >= ?",
			traderID, "CLOSED", CloseReasonReconcileFlat, sinceMs).
		Order("exit_time DESC").
		Find(&rows).Error
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range rows {
		if r.EffectivePnL() < 0 {
			n++
		} else {
			break // a win / break-even ends the losing streak
		}
	}
	return n, nil
}

// GetSessionDayActivity returns the realized P&L (CLOSED trades, excluding
// reconcile-flat orphans whose P&L is UNKNOWN) and the entry count (positions
// OPENED) since sinceMs (Unix ms UTC), optionally scoped to one NT account.
// Used by the Strategy Studio daily guardrails: daily loss/profit on realized
// P&L, max-daily-trades on entries — both measured from the CME session-day
// start. Variadic account mirrors GetClosedPositions/GetFullStats.
func (s *PositionStore) GetSessionDayActivity(traderID string, sinceMs int64, account ...string) (realizedPnL float64, entries int, err error) {
	acct := ""
	if len(account) > 0 {
		acct = account[0]
	}

	var pnl struct{ Total float64 }
	pq := s.db.Model(&TraderPosition{}).
		Select("COALESCE(SUM(COALESCE(pnl_corrected, realized_pnl)), 0) as total") /* P0 2026-08-20: corrections win */.
		Where("trader_id = ? AND status = ? AND close_reason <> ? AND exit_time >= ?",
			traderID, "CLOSED", CloseReasonReconcileFlat, sinceMs)
	if acct != "" {
		pq = pq.Where("account = ?", acct)
	}
	if err = pq.Scan(&pnl).Error; err != nil {
		return 0, 0, err
	}

	var cnt int64
	cq := s.db.Model(&TraderPosition{}).
		Where("trader_id = ? AND entry_time >= ?", traderID, sinceMs)
	if acct != "" {
		cq = cq.Where("account = ?", acct)
	}
	if err = cq.Count(&cnt).Error; err != nil {
		return 0, 0, err
	}

	return pnl.Total, int(cnt), nil
}

// GetFullStats gets complete trading statistics, optionally scoped to one
// account (mirrors GetClosedPositions): account=="" → trader-global (crypto +
// legacy); account!="" → only that NT account's closed trades, excluding
// pre-migration rows (account=''). Variadic so existing callers are unchanged.
func (s *PositionStore) GetFullStats(traderID string, account ...string) (*TraderStats, error) {
	stats := &TraderStats{}
	acct := ""
	if len(account) > 0 {
		acct = account[0]
	}

	// Exclude reconcile-flat orphan closes: their realized P&L is UNKNOWN (exit
	// fill never captured), not a real $0 — counting them would skew win-rate /
	// total P&L. They still appear in the position LIST (rendered "—").
	var count int64
	cq := s.db.Model(&TraderPosition{}).Where("trader_id = ? AND status = ? AND close_reason <> ?", traderID, "CLOSED", CloseReasonReconcileFlat)
	if acct != "" {
		cq = cq.Where("account = ?", acct)
	}
	if err := cq.Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return stats, nil
	}

	var positions []TraderPosition
	pq := s.db.Where("trader_id = ? AND status = ? AND close_reason <> ?", traderID, "CLOSED", CloseReasonReconcileFlat)
	if acct != "" {
		pq = pq.Where("account = ?", acct)
	}
	err := pq.Order("exit_time ASC").
		Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query position statistics: %w", err)
	}

	var pnls []float64
	var totalWin, totalLoss float64

	for _, pos := range positions {
		stats.TotalTrades++
		stats.TotalPnL += pos.EffectivePnL()
		stats.TotalFee += pos.Fee
		pnls = append(pnls, pos.EffectivePnL())

		if pos.EffectivePnL() > 0 {
			stats.WinTrades++
			totalWin += pos.EffectivePnL()
		} else if pos.EffectivePnL() < 0 {
			stats.LossTrades++
			totalLoss += -pos.EffectivePnL()
		}
	}

	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinTrades) / float64(stats.TotalTrades) * 100
	}
	if totalLoss > 0 {
		stats.ProfitFactor = totalWin / totalLoss
	}
	if stats.WinTrades > 0 {
		stats.AvgWin = totalWin / float64(stats.WinTrades)
	}
	if stats.LossTrades > 0 {
		stats.AvgLoss = totalLoss / float64(stats.LossTrades)
	}
	if len(pnls) > 1 {
		stats.SharpeRatio = calculateSharpeRatioFromPnls(pnls)
	}
	if len(pnls) > 0 {
		stats.MaxDrawdownPct = calculateMaxDrawdownFromPnls(pnls)
	}

	return stats, nil
}

// RecentTrade recent trade record
type RecentTrade struct {
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	EntryPrice   float64 `json:"entry_price"`
	ExitPrice    float64 `json:"exit_price"`
	RealizedPnL  float64 `json:"realized_pnl"`
	PnLPct       float64 `json:"pnl_pct"`
	EntryTime    int64   `json:"entry_time"`
	ExitTime     int64   `json:"exit_time"`
	HoldDuration string  `json:"hold_duration"`
}

// GetRecentTrades gets recent closed trades, optionally scoped to one account
// (mirrors GetClosedPositions): account=="" → trader-global (crypto + legacy);
// account!="" → only that NT account's trades, excluding pre-migration rows
// (account=''). Variadic so existing callers stay trader-global unchanged.
func (s *PositionStore) GetRecentTrades(traderID string, limit int, account ...string) ([]RecentTrade, error) {
	var positions []TraderPosition
	q := s.db.Where("trader_id = ? AND status = ?", traderID, "CLOSED")
	if len(account) > 0 && account[0] != "" {
		q = q.Where("account = ?", account[0])
	}
	err := q.Order("exit_time DESC").
		Limit(limit).
		Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query recent trades: %w", err)
	}

	var trades []RecentTrade
	for _, pos := range positions {
		t := RecentTrade{
			Symbol:      pos.Symbol,
			Side:        strings.ToLower(pos.Side),
			EntryPrice:  pos.EntryPrice,
			ExitPrice:   pos.ExitPrice,
			RealizedPnL: pos.EffectivePnL(),
			EntryTime:   pos.EntryTime / 1000, // Convert ms to seconds for API compatibility
		}

		if pos.ExitTime > 0 {
			t.ExitTime = pos.ExitTime / 1000 // Convert ms to seconds
			durationMs := pos.ExitTime - pos.EntryTime
			t.HoldDuration = formatDurationMs(durationMs)
		}

		if pos.EntryPrice > 0 {
			if t.Side == "long" {
				t.PnLPct = (pos.ExitPrice - pos.EntryPrice) / pos.EntryPrice * 100 * float64(pos.Leverage)
			} else {
				t.PnLPct = (pos.EntryPrice - pos.ExitPrice) / pos.EntryPrice * 100 * float64(pos.Leverage)
			}
		}

		trades = append(trades, t)
	}

	return trades, nil
}

// calculateSharpeRatioFromPnls calculates Sharpe ratio
func calculateSharpeRatioFromPnls(pnls []float64) float64 {
	if len(pnls) < 2 {
		return 0
	}

	var sum float64
	for _, pnl := range pnls {
		sum += pnl
	}
	mean := sum / float64(len(pnls))

	var variance float64
	for _, pnl := range pnls {
		variance += (pnl - mean) * (pnl - mean)
	}
	stdDev := math.Sqrt(variance / float64(len(pnls)-1))

	if stdDev == 0 {
		return 0
	}

	return mean / stdDev
}

// calculateMaxDrawdownFromPnls calculates maximum drawdown
func calculateMaxDrawdownFromPnls(pnls []float64) float64 {
	if len(pnls) == 0 {
		return 0
	}

	const startingEquity = 10000.0
	equity := startingEquity
	peak := startingEquity
	var maxDD float64

	for _, pnl := range pnls {
		equity += pnl
		if equity > peak {
			peak = equity
		}
		if peak > 0 {
			dd := (peak - equity) / peak * 100
			if dd > maxDD {
				maxDD = dd
			}
		}
	}

	return maxDD
}

// SymbolStats per-symbol trading statistics
type SymbolStats struct {
	Symbol      string  `json:"symbol"`
	TotalTrades int     `json:"total_trades"`
	WinTrades   int     `json:"win_trades"`
	WinRate     float64 `json:"win_rate"`
	TotalPnL    float64 `json:"total_pnl"`
	AvgPnL      float64 `json:"avg_pnl"`
	AvgHoldMins float64 `json:"avg_hold_mins"`
}

// GetSymbolStats gets per-symbol trading statistics
func (s *PositionStore) GetSymbolStats(traderID string, limit int) ([]SymbolStats, error) {
	var positions []TraderPosition
	// Exclude reconcile-flat orphan closes (unknown P&L — see CloseReasonReconcileFlat).
	err := s.db.Where("trader_id = ? AND status = ? AND close_reason <> ?", traderID, "CLOSED", CloseReasonReconcileFlat).Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query symbol stats: %w", err)
	}

	// Group by symbol
	symbolMap := make(map[string]*SymbolStats)
	symbolHoldMins := make(map[string][]float64)

	for _, pos := range positions {
		if _, ok := symbolMap[pos.Symbol]; !ok {
			symbolMap[pos.Symbol] = &SymbolStats{Symbol: pos.Symbol}
			symbolHoldMins[pos.Symbol] = []float64{}
		}
		s := symbolMap[pos.Symbol]
		s.TotalTrades++
		s.TotalPnL += pos.EffectivePnL()
		if pos.EffectivePnL() > 0 {
			s.WinTrades++
		}

		if pos.ExitTime > 0 {
			holdMins := float64(pos.ExitTime-pos.EntryTime) / 60000.0 // ms to minutes
			symbolHoldMins[pos.Symbol] = append(symbolHoldMins[pos.Symbol], holdMins)
		}
	}

	var stats []SymbolStats
	for symbol, s := range symbolMap {
		if s.TotalTrades > 0 {
			s.WinRate = float64(s.WinTrades) / float64(s.TotalTrades) * 100
			s.AvgPnL = s.TotalPnL / float64(s.TotalTrades)
		}
		if len(symbolHoldMins[symbol]) > 0 {
			var totalMins float64
			for _, m := range symbolHoldMins[symbol] {
				totalMins += m
			}
			s.AvgHoldMins = totalMins / float64(len(symbolHoldMins[symbol]))
		}
		stats = append(stats, *s)
	}

	// Sort by TotalPnL descending and limit
	for i := 0; i < len(stats)-1; i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].TotalPnL > stats[i].TotalPnL {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	if limit > 0 && len(stats) > limit {
		stats = stats[:limit]
	}

	return stats, nil
}

// HoldingTimeStats holding duration analysis
type HoldingTimeStats struct {
	Range      string  `json:"range"`
	TradeCount int     `json:"trade_count"`
	WinRate    float64 `json:"win_rate"`
	AvgPnL     float64 `json:"avg_pnl"`
}

// GetHoldingTimeStats analyzes performance by holding duration
func (s *PositionStore) GetHoldingTimeStats(traderID string) ([]HoldingTimeStats, error) {
	var positions []TraderPosition
	err := s.db.Where("trader_id = ? AND status = ? AND exit_time > 0", traderID, "CLOSED").Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query holding time stats: %w", err)
	}

	rangeStats := map[string]*struct {
		count   int
		wins    int
		totalPnL float64
	}{
		"<1h":   {},
		"1-4h":  {},
		"4-24h": {},
		">24h":  {},
	}

	for _, pos := range positions {
		if pos.ExitTime == 0 {
			continue
		}
		holdHours := float64(pos.ExitTime-pos.EntryTime) / 3600000.0 // ms to hours

		var rangeKey string
		switch {
		case holdHours < 1:
			rangeKey = "<1h"
		case holdHours < 4:
			rangeKey = "1-4h"
		case holdHours < 24:
			rangeKey = "4-24h"
		default:
			rangeKey = ">24h"
		}

		r := rangeStats[rangeKey]
		r.count++
		r.totalPnL += pos.EffectivePnL()
		if pos.EffectivePnL() > 0 {
			r.wins++
		}
	}

	var stats []HoldingTimeStats
	for _, rangeKey := range []string{"<1h", "1-4h", "4-24h", ">24h"} {
		r := rangeStats[rangeKey]
		if r.count > 0 {
			stats = append(stats, HoldingTimeStats{
				Range:      rangeKey,
				TradeCount: r.count,
				WinRate:    float64(r.wins) / float64(r.count) * 100,
				AvgPnL:     r.totalPnL / float64(r.count),
			})
		}
	}

	return stats, nil
}

// DirectionStats long/short performance comparison
type DirectionStats struct {
	Side       string  `json:"side"`
	TradeCount int     `json:"trade_count"`
	WinRate    float64 `json:"win_rate"`
	TotalPnL   float64 `json:"total_pnl"`
	AvgPnL     float64 `json:"avg_pnl"`
}

// GetDirectionStats analyzes long vs short performance
func (s *PositionStore) GetDirectionStats(traderID string) ([]DirectionStats, error) {
	var positions []TraderPosition
	// Exclude reconcile-flat orphan closes (unknown P&L — see CloseReasonReconcileFlat).
	err := s.db.Where("trader_id = ? AND status = ? AND close_reason <> ?", traderID, "CLOSED", CloseReasonReconcileFlat).Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query direction stats: %w", err)
	}

	sideStats := make(map[string]*DirectionStats)
	for _, pos := range positions {
		if _, ok := sideStats[pos.Side]; !ok {
			sideStats[pos.Side] = &DirectionStats{Side: pos.Side}
		}
		s := sideStats[pos.Side]
		s.TradeCount++
		s.TotalPnL += pos.EffectivePnL()
		if pos.EffectivePnL() > 0 {
			s.WinRate++
		}
	}

	var stats []DirectionStats
	for _, s := range sideStats {
		if s.TradeCount > 0 {
			s.AvgPnL = s.TotalPnL / float64(s.TradeCount)
			s.WinRate = s.WinRate / float64(s.TradeCount) * 100
		}
		stats = append(stats, *s)
	}

	return stats, nil
}
