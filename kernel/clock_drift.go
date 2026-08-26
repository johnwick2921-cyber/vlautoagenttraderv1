package kernel

import (
	"nofx/logger"
	"nofx/market"
	"nofx/telemetry"
)

// C2 — CLOCK-DRIFT ENTRY GUARD.
//
// The bot timestamps every signal with the LOCAL clock, and the NT8 AddOn rejects
// signals whose timestamp is more than ~60s from its own (Tradovate) clock. So if
// the local system clock drifts more than a minute from real time, entries get
// silently rejected or mis-timed. NTP isn't guaranteed on this box, so we detect
// drift from the only external time reference we already have: the freshest feed
// bar's labeled timestamp.
//
// A bar is labeled at its OPEN, so under a correct clock and a live feed the bar
// that just closed is labeled ~one interval ago; comparing the local clock to the
// bar's approximate CLOSE (open + interval) yields ~0. A large positive drift means
// the clock is AHEAD (or the feed is lagging); a large negative drift means the
// clock is BEHIND (the feed labeled a bar in the future — impossible unless the
// local clock is wrong, which B4's staleness check cannot catch). Either way,
// entries are blocked; exits / open-position management are never touched.

const clockDriftToleranceMs = 60_000 // >60s local-vs-feed skew blocks NEW entries

// clockDriftMs approximates the local-vs-feed clock skew: nowMs minus the freshest
// bar's close (open label + one interval). ~0 under a correct clock + live feed.
func clockDriftMs(nowMs, freshestBarMs, intervalMs int64) int64 {
	return nowMs - (freshestBarMs + intervalMs)
}

// freshestIntradayBar returns the newest 1m (else 5m) bar time and its interval,
// (0,0) if neither is present.
func freshestIntradayBar(md *market.Data) (barMs, intervalMs int64) {
	for _, tf := range []string{"1m", "5m"} {
		if b := newestBarMs(md, tf); b > 0 {
			return b, tfIntervalMs(tf)
		}
	}
	return 0, 0
}

func absI64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// applyClockDriftBlock (C2) detects local-vs-feed clock skew. Since 2026-08-18
// outgoing NT8 signals are stamped with the FEED clock (trader/ninjatrader
// tcp_trader.go feedNowUTC), so a skewed local clock can no longer age a signal
// into NT8's 60s stale rejection. The guard no longer converts entries to wait —
// it logs the skew as a warning so the operator still sees when NTP needs
// attention. Only open_long/open_short are examined; no 1m/5m data → nothing
// to compare (never blocks).
func applyClockDriftBlock(fd *FullDecision, ctx *Context, nowMs int64) {
	if fd == nil || ctx == nil {
		return
	}
	for i := range fd.Decisions {
		d := &fd.Decisions[i]
		if d.Action != "open_long" && d.Action != "open_short" {
			continue
		}
		barMs, ivMs := freshestIntradayBar(ctx.MarketDataMap[d.Symbol])
		if barMs <= 0 || ivMs <= 0 {
			continue // fail-open
		}
		drift := clockDriftMs(nowMs, barMs, ivMs)
		if absI64(drift) <= clockDriftToleranceMs {
			continue
		}
		dir := "AHEAD of"
		if drift < 0 {
			dir = "BEHIND"
		}
		logger.Warnf("⚠️ clock-drift DETECTED (no entry block): local clock is %s the feed by %ds (>%ds) — signals are feed-stamped so entries proceed; fix the host clock / NTP. Exits unaffected.",
			dir, absI64(drift)/1000, clockDriftToleranceMs/1000)
		telemetry.IncClockSkewObserved(ctx.TraderID)
	}
}
