package trader

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// Phase 2 — DISCARD-BURN (final-bundle 2026-08-19).
//
// Baseline (#51 E17): 18.5% of paid AI calls died as supersession discards —
// the call spanned a primary-bar close and the whole decision was thrown away.
// Two additive rails cut that burn without weakening the staleness guarantee:
//
//   2.1 DODGE — if a cycle would START within (recent avg call × 1.2) of the
//       decision-TF's next close, defer the start to close+1s so the call runs
//       entirely inside one bar. Env STALE_DODGE (default on).
//   2.2 RE-EVALUATE — a superseded decision that contains ONLY entries is not
//       auto-discarded: its gates re-run against the fresh CLOSED bar; it
//       executes iff (a) the fresh bar never touched the decision's stop and
//       (b) |price drift since snapshot| < STALE_REEVAL_DRIFT_ATR × ATR(14,5m).
//       Superseded wait/hold-only sets are a free, quiet discard. Sets carrying
//       close_* keep the legacy conservative discard unchanged.
//
// All knobs are env/computed — no literals on the decision path beyond the
// spec-fixed constants named here.

const (
	staleDodgeSafetyFactor = 1.2 // spec-fixed: dodge window = avg_call_ms × 1.2
	aiCallRingSize         = 20  // spec-fixed: average over the last 20 calls
	reevalATRPeriod        = 14  // spec-fixed: ATR(14, 5m) for the drift bound
	reevalATRTimeframe     = "5m"
	defaultReevalDriftATR  = 0.25 // spec-fixed default; env STALE_REEVAL_DRIFT_ATR overrides
)

// staleDodgeEnabled — env STALE_DODGE, default ON; only an explicit off/0/false disables.
func staleDodgeEnabled() bool {
	switch strings.ToLower(os.Getenv("STALE_DODGE")) {
	case "off", "0", "false", "disabled":
		return false
	}
	return true
}

// reevalDriftATRMult — env STALE_REEVAL_DRIFT_ATR, default 0.25.
func reevalDriftATRMult() float64 {
	if v := os.Getenv("STALE_REEVAL_DRIFT_ATR"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return defaultReevalDriftATR
}

// ---- AI-call duration ring (written and read only on the run-loop goroutine) ----

func (at *AutoTrader) recordAICallMs(ms int64) {
	if ms <= 0 {
		return
	}
	at.aiCallMs[at.aiCallIdx%aiCallRingSize] = ms
	at.aiCallIdx++
	if at.aiCallN < aiCallRingSize {
		at.aiCallN++
	}
}

// avgAICallMs returns 0 until at least one call is recorded (no dodge without history).
func (at *AutoTrader) avgAICallMs() int64 {
	if at.aiCallN == 0 {
		return 0
	}
	var sum int64
	for i := 0; i < at.aiCallN; i++ {
		sum += at.aiCallMs[i]
	}
	return sum / int64(at.aiCallN)
}

// ---- 2.1 DODGE ----

// tfMs maps a timeframe label to its period in ms (period math, not a tunable).
func tfMs(tf string) int64 {
	switch tf {
	case "1m":
		return 60_000
	case "3m":
		return 180_000
	case "5m":
		return 300_000
	case "15m":
		return 900_000
	case "30m":
		return 1_800_000
	case "1h":
		return 3_600_000
	}
	return 0
}

// staleDodgeDefer is the pure dodge decision: given now, the decision-TF's next
// close, and the recent average call duration, return how long to defer (to
// close+1s) and whether to defer at all. avg==0 (no history) never defers.
func staleDodgeDefer(nowMs, nextCloseMs, avgCallMs int64) (deferMs int64, dodge bool) {
	if avgCallMs <= 0 || nextCloseMs <= nowMs {
		return 0, false
	}
	window := int64(float64(avgCallMs) * staleDodgeSafetyFactor)
	timeToClose := nextCloseMs - nowMs
	if timeToClose >= window {
		return 0, false
	}
	return timeToClose + 1000, true
}

// staleDodgeCheck resolves the trader's next primary-TF close and applies the
// pure decision. Epoch math only — bar periods are UTC-aligned, no tz on this path.
func (at *AutoTrader) staleDodgeCheck(now time.Time) (deferMs int64, dodge bool) {
	iv := tfMs(at.primaryTimeframe())
	if iv <= 0 {
		return 0, false
	}
	nowMs := now.UnixMilli()
	nextClose := (nowMs/iv)*iv + iv
	return staleDodgeDefer(nowMs, nextClose, at.avgAICallMs())
}

// scheduleKick arms a one-shot deferred cycle on the run loop's kick channel.
// At most one kick may be pending at a time (CAS guard); returns whether armed.
func (at *AutoTrader) scheduleKick(d time.Duration, reason string) bool {
	if at.kickCh == nil || !at.kickPending.CompareAndSwap(false, true) {
		return false
	}
	time.AfterFunc(d, func() {
		select {
		case at.kickCh <- reason:
		default:
			at.kickPending.Store(false) // loop gone/stopped — disarm
		}
	})
	return true
}

// ---- 2.2 RE-EVALUATE ----

// classifyDecisions counts entry and close actions in a decision set.
func classifyDecisions(ds []kernel.Decision) (entries, closes int) {
	for i := range ds {
		switch ds[i].Action {
		case "open_long", "open_short":
			entries++
		case "close_long", "close_short":
			closes++
		}
	}
	return
}

type reevalVerdict struct {
	pass   bool
	reason string
}

// reevalSupersededEntry re-runs the mechanical gates for ONE superseded entry
// against the fresh closed bar. Pure — testable without a provider.
func reevalSupersededEntry(action string, stopLoss, snapPrice, freshHigh, freshLow, freshClose, atr, driftMult float64) reevalVerdict {
	if stopLoss <= 0 {
		return reevalVerdict{false, "no_stated_stop"}
	}
	if atr <= 0 {
		return reevalVerdict{false, "atr_unavailable"}
	}
	switch action {
	case "open_long":
		if freshLow <= stopLoss {
			return reevalVerdict{false, fmt.Sprintf("sl_breached_in_fresh_bar (low %.2f <= sl %.2f)", freshLow, stopLoss)}
		}
	case "open_short":
		if freshHigh >= stopLoss {
			return reevalVerdict{false, fmt.Sprintf("sl_breached_in_fresh_bar (high %.2f >= sl %.2f)", freshHigh, stopLoss)}
		}
	default:
		return reevalVerdict{false, "not_an_entry"}
	}
	if snapPrice > 0 {
		drift := freshClose - snapPrice
		if drift < 0 {
			drift = -drift
		}
		if limit := driftMult * atr; drift >= limit {
			return reevalVerdict{false, fmt.Sprintf("drift_too_big (|%.2f| >= %.2f = %.2f x ATR %.2f)", drift, limit, driftMult, atr)}
		}
	}
	return reevalVerdict{true, ""}
}

// reevalSupersededEntries applies the per-entry re-eval to every entry in the
// superseded set, feeding it fresh bars from the live provider. ALL entries must
// pass; the first refusal refuses the set (conservative).
func (at *AutoTrader) reevalSupersededEntries(ds []kernel.Decision, ctx *kernel.Context) reevalVerdict {
	if market.FuturesBarsProvider == nil {
		return reevalVerdict{false, "no_bar_provider"}
	}
	driftMult := reevalDriftATRMult()
	nowMs := time.Now().UnixMilli()
	for i := range ds {
		d := &ds[i]
		if d.Action != "open_long" && d.Action != "open_short" {
			continue
		}
		// Fresh CLOSED primary bar (the one that superseded the decision).
		var fresh *market.Kline
		pBars := market.FuturesBarsProvider(d.Symbol, at.primaryTimeframe(), 3)
		for j := range pBars {
			if pBars[j].CloseTime <= nowMs && (fresh == nil || pBars[j].CloseTime > fresh.CloseTime) {
				fresh = &pBars[j]
			}
		}
		if fresh == nil {
			return reevalVerdict{false, "no_fresh_closed_bar"}
		}
		// ATR(14, 5m) computed from live bars — needs period+1 bars.
		aBars := market.FuturesBarsProvider(d.Symbol, reevalATRTimeframe, reevalATRPeriod+1)
		atr := market.ExportCalculateATR(aBars, reevalATRPeriod)
		snapPrice := 0.0
		if ctx != nil && ctx.MarketDataMap != nil {
			if md := ctx.MarketDataMap[d.Symbol]; md != nil {
				snapPrice = md.CurrentPrice
			}
		}
		if v := reevalSupersededEntry(d.Action, d.StopLoss, snapPrice, fresh.High, fresh.Low, fresh.Close, atr, driftMult); !v.pass {
			return v
		}
	}
	return reevalVerdict{true, ""}
}

// 2.3 status honesty note: guardrail_skip rows deliberately carry Success=false
// (the ghost-record fix — see stampGuardrailSkip). The "#20161 Failed" dishonesty
// is a DISPLAY conflation, fixed frontend-side: DecisionCard renders
// execution_status=guardrail_skip as a neutral ℹ️ skip, reserving the red badge
// for real failures (and ❌ for verdict_hint=feed|clock). DB semantics unchanged.

// noteKick arms the one-shot bypass flags for a consumed kick (run-loop
// goroutine only). U2 (watcher-eyes hotfix): a post_exit kick used to enter
// tickOnce UPSTREAM of the cadence gates — bar_close mode dropped the promised
// rescan unconditionally, interval mode's no-new-data dedup could eat it (flat
// + unchanged sig IS the post-close state in quiet tape), and the dodge could
// re-defer it. The rescan now bypasses all three, exactly once.
func (at *AutoTrader) noteKick(reason string) {
	switch reason {
	case "stale_dodge":
		at.skipDodgeOnce = true // the dodged cycle runs now; no re-dodge at the boundary
	case "post_exit":
		at.skipCadenceOnce = true
		at.skipDodgeOnce = true
	}
}
