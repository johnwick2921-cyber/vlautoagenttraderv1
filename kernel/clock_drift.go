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

// applyClockDriftBlock (C2) neutralizes an open decision to `wait` when the local
// clock is skewed from the freshest feed timestamp by more than the tolerance in
// EITHER direction. Only open_long/open_short are affected. No 1m/5m data → fail-
// open (can't assess → never block).
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
		logger.Warnf("🚨 clock-drift ENTRY BLOCK: %s %s → WAIT — local clock is %s the feed by %ds (>%ds); signals would be mis-timed / NT8-rejected. Check NTP / system time. Exits unaffected.",
			d.Symbol, d.Action, dir, absI64(drift)/1000, clockDriftToleranceMs/1000)
		d.Action = "wait"
		telemetry.IncGateBlock(ctx.TraderID, "clock_drift")
	}
}
