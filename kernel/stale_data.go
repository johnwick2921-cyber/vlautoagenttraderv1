package kernel

import (
	"fmt"

	"nofx/logger"
	"nofx/market"
	"nofx/telemetry"
)

// B4 — stale-data ENTRY block. A NEW entry requires the freshest intraday bar to
// be recent (≤ 2× its interval old); a frozen / stale feed → refuse the entry. Only
// open_long/open_short are affected — open-position management and exits are never
// blocked (a stale feed must not strand an open position).

const staleBarMaxMultiple = 2 // freshest bar must be ≤ this × its interval old

// tfIntervalMs returns the bar interval in ms for an intraday timeframe, 0 unknown.
func tfIntervalMs(tf string) int64 {
	switch tf {
	case "1m":
		return 60_000
	case "5m":
		return 300_000
	}
	return 0
}

// newestBarMs returns the timestamp (ms) of the newest kline for tf, 0 if none.
func newestBarMs(md *market.Data, tf string) int64 {
	if md == nil || md.TimeframeData == nil {
		return 0
	}
	td := md.TimeframeData[tf]
	if td == nil || len(td.Klines) == 0 {
		return 0
	}
	return td.Klines[len(td.Klines)-1].Time
}

// barIsStale reports whether a bar of the given interval is older than the allowed
// multiple at nowMs. Unknown inputs (<=0) → false (can't assess → fail-open).
func barIsStale(newestBarMs, intervalMs, nowMs int64) bool {
	if newestBarMs <= 0 || intervalMs <= 0 {
		return false
	}
	return nowMs-newestBarMs > int64(staleBarMaxMultiple)*intervalMs
}

// staleEntryFeed reports whether the freshest intraday bar (1m preferred, else 5m)
// is stale enough to block NEW entries. No 1m/5m data → fail-open (not stale).
func staleEntryFeed(md *market.Data, nowMs int64) (tf string, ageMs, limitMs int64, stale bool) {
	for _, cand := range []string{"1m", "5m"} {
		bMs := newestBarMs(md, cand)
		if bMs <= 0 {
			continue
		}
		iv := tfIntervalMs(cand)
		limit := int64(staleBarMaxMultiple) * iv
		return cand, nowMs - bMs, limit, barIsStale(bMs, iv, nowMs)
	}
	return "", 0, 0, false
}

// applyStaleDataBlock (B4) neutralizes an open decision to `wait` when the freshest
// 1m/5m bar is stale (feed frozen). Only open_long/open_short are affected — exits
// and open-position management are never blocked.
func applyStaleDataBlock(fd *FullDecision, ctx *Context, nowMs int64) {
	if fd == nil || ctx == nil {
		return
	}
	for i := range fd.Decisions {
		d := &fd.Decisions[i]
		if d.Action != "open_long" && d.Action != "open_short" {
			continue
		}
		md := ctx.MarketDataMap[d.Symbol]
		if tf, age, limit, stale := staleEntryFeed(md, nowMs); stale {
			logger.Warnf("⛔ stale-data ENTRY BLOCK: %s %s → WAIT — freshest %s bar is %dms old (>%dms = 2×interval); feed likely frozen. Exits/position-management unaffected.",
				d.Symbol, d.Action, tf, age, limit)
			reason := fmt.Sprintf("freshest %s bar %dms old (>%dms)", tf, age, limit)
			d.RefusalReason = "stale-data: " + reason
			d.Action = "wait"
			telemetry.IncGateBlock(ctx.TraderID, "stale_data")
			telemetry.RecordError(ctx.TraderID, "entry_refused", "stale-data: "+reason, telemetry.CostTradeLost)
		}
	}
}
