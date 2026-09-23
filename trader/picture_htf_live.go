package trader

import (
	"fmt"
	"sync"
	"time"

	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// PICTURE-HTF LIVE WIRING (2026-09-20) — routes NATIVE LIVE NT8 bar events to
// each trader's two-picture evaluator. Historical replays never arrive here
// (the provider fans out live frames only), so a replay receipt can never
// mint a real opportunity.

// pictureHtfTraders is the process-wide registry of AutoTraders that own an
// evaluator (keyed by trader id — a restarted trader replaces its entry).
var pictureHtfTraders sync.Map // id → *AutoTrader

func init() {
	ntwire.SetLiveBarSink(pictureHtfLiveBars)
}

// pictureHtfLiveBars is the process-wide sink: converts wire bars to klines
// and fans out to every registered trader. Registered traders whose mode is
// off drop the frame in the evaluator (cheap no-op).
func pictureHtfLiveBars(symbol, tf string, bars []ntwire.Bar, receivedAt time.Time) {
	if len(bars) == 0 {
		return
	}
	dur, ok := kernel.TFDurationMs(tf)
	kl := make([]market.Kline, 0, len(bars))
	for _, b := range bars {
		k := market.Kline{OpenTime: b.T, Open: b.O, High: b.H, Low: b.L, Close: b.C}
		if ok {
			k.CloseTime = b.T + dur - 1
		}
		kl = append(kl, k)
	}
	pictureHtfTraders.Range(func(_, v any) bool {
		if at, ok := v.(*AutoTrader); ok {
			at.NotifyLiveBars(symbol, tf, kl, receivedAt)
		}
		return true
	})
}

// registerPictureHtf installs the trader in the live-bar registry.
func (at *AutoTrader) registerPictureHtf() {
	if at == nil || at.id == "" {
		return
	}
	pictureHtfTraders.Store(at.id, at)
}

// pictureHtfResolvedConfig is the trader's Picture knobs with defaults
// applied — the one resolution the evaluator and the plan card both read.
func (at *AutoTrader) pictureHtfResolvedConfig() store.PictureHtfConfig {
	if sc := at.GetStrategyConfig(); sc != nil && sc.DayPlan != nil && sc.DayPlan.PictureHtf != nil {
		return store.PictureHtfResolved(sc.DayPlan.PictureHtf)
	}
	return store.PictureHtfResolved(nil)
}

// pictureHtfEvaluator lazily builds (or rebuilds, when the strategy knobs
// change) the trader's two-picture evaluator. Returns nil when the mode is
// absent/disabled or the trader is not on the NT8 path.
func (at *AutoTrader) pictureHtfEvaluator() *PictureHtfEvaluator {
	if at == nil || at.exchange != "ninjatrader" {
		return nil
	}
	at.pictureHtfMu.Lock()
	defer at.pictureHtfMu.Unlock()
	cfg := at.pictureHtfResolvedConfig()
	sig := fmt.Sprintf("%t|%.5f|%d|%d|%d|%d|%.4f",
		cfg.Enabled, cfg.TickSize, cfg.PivotWindow, cfg.SwingLookback, cfg.EntryWindowSec, cfg.FreshnessSec, cfg.MinRR)
	if at.pictureHtf != nil && at.pictureHtfSig == sig {
		// The broker consumer self-heals here: if its listener channel ever
		// closed (underlying subscription died), the registry entry was
		// deleted and this per-cycle call re-listens.
		at.ensurePictureHtfBrokerConsumer()
		return at.pictureHtf
	}
	at.pictureHtf = NewPictureHtfEvaluator(at, cfg)
	at.pictureHtfSig = sig
	at.ensurePictureHtfBrokerConsumer()
	return at.pictureHtf
}

// NotifyLiveBars routes a native LIVE bar event into the evaluator.
func (at *AutoTrader) NotifyLiveBars(symbol, tf string, bars []market.Kline, receivedAt time.Time) {
	if ev := at.pictureHtfEvaluator(); ev != nil {
		ev.OnBars(symbol, tf, bars, receivedAt)
	}
}

// pictureHtfTickFallback is the wall-clock fallback the run loop calls once
// per cycle: it covers a missed boundary frame (feed stall) with the same
// evaluation, and runs the reconciliation sweep so pending rows recover
// across disconnects/restarts WITHOUT another entry (the sweep only ever
// observes the broker book). The evaluator's freshness gate still applies.
func (at *AutoTrader) pictureHtfTickFallback(now time.Time) {
	if ev := at.pictureHtfEvaluator(); ev != nil {
		ev.Evaluate(at.futuresSymbol(), now)
		pictureHtfReconcilePending(at)
	}
}

// pictureHtfBootLine is the mode's boot line: mode, rule version, SIM status,
// native-data readiness, and the AddOn capability verdict — READ from the
// live far side, never assumed.
func (at *AutoTrader) pictureHtfBootLine() string { return at.pictureHtfBootLineAt(time.Now()) }

// pictureHtfBootLineAt is the 📷 boot line on an injected clock.
func (at *AutoTrader) pictureHtfBootLineAt(now time.Time) string {
	ev := at.pictureHtfEvaluator()
	mode := "off"
	if ev != nil && ev.Enabled() {
		mode = "on"
	}
	sim := "SIM-only"
	native := "native NT8 bars (final+emitted_at)"
	cap := "not proven"
	if pictureHtfCapabilityProven(at) {
		cap = "proven"
	}
	// W-EXEC-TRUTH W0 (CTO Q6): the plan-mode verdict, READ — under strict
	// Picture is refused until W5 makes it a Day Plan scenario, and the line
	// says so in those words.
	planGate := "admitted (plan mode is not strict)"
	if r := at.pictureStrictVisible(now); r != "" {
		planGate = r
	}
	return fmt.Sprintf("picture-htf: mode=%s rule=v1 %s data=%s addon=%s (build=%q, need ≥ %s) plan_gate=%s",
		mode, sim, native, cap, at.farSideBuildID(), ntwire.MinAddonBuildPictureHtf, planGate)
}

// logPictureHtfBootLine prints the boot line at trader start.
func (at *AutoTrader) logPictureHtfBootLine() {
	logger.Info("📷 " + at.pictureHtfBootLine())
}
