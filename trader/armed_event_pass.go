package trader

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
)

// ── W-EXEC-TRUTH W3 D14 — the live-bar armed pass ───────────────────────────
//
// D6: the executor's price was the forming 1m close sampled once per 2-minute
// scan, so a market_in_zone limit could be placed up to two minutes after the
// confirming close. The live-bar sink now KICKS a per-trader event pass:
//
//   - only while the ACTIVE plan doc has an enabled market_in_zone arm
//     (zoneArmActive, cached by every pass) — a legacy plan never wakes it;
//   - on a FINAL 1m bar for the trader's own instrument root, or when the
//     live price changes a cached zone verdict (a forming bar entering the
//     zone need not wait for its close);
//   - non-blocking (the sink's worker is shared by every trader and must never
//     wait on a pass that can block ~2 s on a synchronous cancel);
//   - at most one pass per second, coalesced to the TRAILING edge: a kick that
//     lands inside the gap is absorbed and the pass after the gap reads the
//     freshest tape, so a boundary frame is never lost to the limiter.
//
// The pass itself is maybeManageArmedOrdersAt — the same pass the scan runs,
// serialized with it by armedPassMu. The 2-minute scan stays the fallback.

// armedPassEnterForTest is a TEST SEAM ONLY (nil in production,
// TestArmedPassEnterSeamIsNilInProduction): called when a pass has taken
// armedPassMu, its returned func when the pass ends — so a test can observe
// whether two passes of one trader were ever inside at the same time.
var armedPassEnterForTest func(traderID string) (exit func())

func armedPassEntered(traderID string) func() {
	if h := armedPassEnterForTest; h != nil {
		return h(traderID)
	}
	return func() {}
}

// armedEventMinGap bounds the event pass to ≤ 1 per second per trader.
const armedEventMinGap = time.Second

// armedEventLoop is one trader's event-pass goroutine.
type armedEventLoop struct {
	kick   chan struct{} // cap 1: a pending kick coalesces every later one
	stop   chan struct{}
	done   chan struct{}
	once   sync.Once
	passes atomic.Int64 // passes that actually ran (the tests read it)
}

func (l *armedEventLoop) poke() {
	select {
	case l.kick <- struct{}{}:
	default: // one already pending — coalesced
	}
}

func (l *armedEventLoop) close() {
	l.once.Do(func() { close(l.stop) })
	select {
	case <-l.done:
	case <-time.After(5 * time.Second):
		logger.Warnf("armed event pass: stop did not finish within 5s (a pass is still running; it re-checks running before placing)")
	}
}

// startArmedEventLoop starts the trader's event loop (Run). NinjaTrader only.
func (at *AutoTrader) startArmedEventLoop() {
	if at == nil || at.exchange != "ninjatrader" {
		return
	}
	l := &armedEventLoop{kick: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	if old := at.armedEvent.Swap(l); old != nil {
		old.close()
	}
	go at.runArmedEventLoop(l)
}

// stopArmedEventLoop stops it (Stop): no event pass starts after this returns.
func (at *AutoTrader) stopArmedEventLoop() {
	if at == nil {
		return
	}
	if l := at.armedEvent.Swap(nil); l != nil {
		l.close()
	}
}

func (at *AutoTrader) runArmedEventLoop(l *armedEventLoop) {
	defer close(l.done)
	var last time.Time
	for {
		select {
		case <-l.stop:
			return
		case <-l.kick:
		}
		if !last.IsZero() {
			if wait := armedEventMinGap - time.Since(last); wait > 0 {
				t := time.NewTimer(wait)
				select {
				case <-l.stop:
					t.Stop()
					return
				case <-t.C:
				}
			}
		}
		// Trailing edge: whatever arrived during the gap is served by THIS pass,
		// which reads the tape as it is now.
		select {
		case <-l.kick:
		default:
		}
		last = time.Now()
		ran := false
		if at.armedEventNowForTest != nil {
			ran = at.armedEventPassAt(at.armedEventNowForTest())
		} else {
			ran = at.armedEventPass()
		}
		if ran {
			l.passes.Add(1)
		}
	}
}

// armedEventPass is the event loop's wall-clock entry (clock-seams.list).
func (at *AutoTrader) armedEventPass() bool {
	return at.armedEventPassAt(time.Now())
}

// armedEventPassAt runs one event-driven armed pass at now. runCycle's own
// early exits do not stand in front of this goroutine, so it checks them
// itself: running, Day Plan on, an NT8 account chosen, the CME open, and a
// zone arm on the active plan. The snapshot is built from the SAME 1m cache
// the scan reads and is never nil (a nil snapshot skips the HTF veto).
// Returns whether a pass ran.
func (at *AutoTrader) armedEventPassAt(now time.Time) bool {
	if at == nil || at.store == nil || at.exchange != "ninjatrader" || !at.runningNow() || !at.dayPlanEnabled() {
		return false
	}
	if !at.zoneArmActive.Load() {
		return false
	}
	if closed, _ := kernel.CMEClosedReason(now); closed {
		return false
	}
	if tr, err := at.store.Trader().GetByID(at.id); err != nil || tr == nil || strings.TrimSpace(tr.Account) == "" {
		return false
	}
	var bars []market.Kline
	if market.FuturesBarsProvider != nil {
		bars = market.FuturesBarsProvider(at.futuresSymbol(), kernel.AISVPBarInterval, kernel.AISVPBarCount)
	}
	at.maybeManageArmedOrdersAt(kernel.StructureSnapshot(bars, now.UnixMilli()), now)
	return true
}

// noteZoneArmActive caches whether the active plan doc has an enabled
// market_in_zone arm (read by the sink without a DB or plan read).
func (at *AutoTrader) noteZoneArmActive(plan *kernel.ActivePlan) {
	on := false
	if plan != nil {
		for _, sc := range plan.Doc.Scenarios {
			if scenarioHasZoneArm(sc) {
				on = true
				break
			}
		}
	}
	at.zoneArmActive.Store(on)
}

// noteLiveBarsForArmedPass is the sink's half: it decides whether this frame
// is a trigger and kicks the loop. Never blocks, never reads the DB.
func (at *AutoTrader) noteLiveBarsForArmedPass(symbol, tf string, bars []ntwire.Bar) {
	if at == nil || len(bars) == 0 || tf != "1m" || !at.zoneArmActive.Load() {
		return
	}
	l := at.armedEvent.Load()
	if l == nil {
		return
	}
	root := instrumentRoot(symbol)
	if root == "" || root != instrumentRoot(at.futuresSymbol()) {
		return // another instrument's frame
	}
	final := false
	for _, b := range bars {
		if b.Final {
			final = true
			break
		}
	}
	if final || at.zoneVerdictChanged(bars[len(bars)-1].C) {
		l.poke()
	}
}

// zoneVerdictChanged reports whether the live price reads a different zone
// verdict than the last pass cached for any armed policy row. The frame is
// live, so staleness is not judged here — the pass judges it.
func (at *AutoTrader) zoneVerdictChanged(price float64) bool {
	ws, _ := at.zoneWatch.Load().([]zoneWatch)
	for _, w := range ws {
		if zonePriceVerdict(price, w.lo, w.hi, w.side) != w.verdict {
			return true
		}
	}
	return false
}
