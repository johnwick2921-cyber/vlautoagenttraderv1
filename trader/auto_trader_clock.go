package trader

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// P2 — THE CLOCK. Bar-close cadence + the skip-while-open gate. Both are GATED on
// day_plan being enabled for a futures trader, so the default (crypto / plan-off)
// keeps the scan-timer loop and same-side-refusal behavior byte-identical —
// additive + dormant until the owner arms day_plan at ★ RESTART 1.

// dayPlanEnabled is the shared activation for every P2 clock behavior: a futures
// (NinjaTrader) trader whose strategy has day_plan enabled. Default false → the
// whole clock is dormant and crypto/plan-off behavior is byte-identical.
func (at *AutoTrader) dayPlanEnabled() bool {
	if at.exchange != "ninjatrader" || at.config.StrategyConfig == nil {
		return false
	}
	dp := at.config.StrategyConfig.DayPlan
	return dp != nil && dp.PlanEnabled
}

// barCloseCadenceActive reports whether this trader should fire cycles on
// primary-TF bar closes instead of the scan timer.
// P10 note: since the owner ruling (2026-08-19) this only ARMS the cadence
// machinery; the MODE below decides whether the bar-close gate actually gates.
func (at *AutoTrader) barCloseCadenceActive() bool { return at.dayPlanEnabled() }

// Cadence modes (P10 — OWNER RULING 2026-08-19): the Studio scan interval is
// the ACTUAL decision cadence. "interval" (default) runs a full cycle every
// scheduler tick on the LATEST bar state including the forming primary-TF bar;
// "bar_close" is the legacy day-plan P2 gate (one cycle per closed primary
// bar), selectable per-trader in Studio, stored in traders.cadence_mode.
const (
	CadenceInterval = "interval"
	CadenceBarClose = "bar_close"
)

// cadenceMode resolves the trader's mode: explicit "bar_close" keeps the
// legacy gate; everything else (empty, "interval", garbage) is interval — the
// P10 default. Garbage never invents the stricter gate silently.
func (at *AutoTrader) cadenceMode() string {
	if at.config.CadenceMode == CadenceBarClose {
		return CadenceBarClose
	}
	return CadenceInterval
}

// skipNoNewData (P10.4) is the ONLY allowed interval-mode skip besides the
// existing gates: when a tick finds a byte-identical world — same newest
// primary-TF bar (open/high/low/close/volume, forming or not), and FLAT — the
// cycle is skipped with a logged reason instead of burning a paid AI call on
// an identical snapshot. Any bar mutation (a forming bar's close/volume moves
// on every real tick) or an open position runs the cycle.
func (at *AutoTrader) skipNoNewData(now time.Time) bool {
	if market.FuturesBarsProvider == nil {
		return false // no bar state to compare — never invent a skip
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), at.primaryTimeframe(), 1)
	if len(bars) == 0 {
		return false
	}
	b := bars[len(bars)-1]
	sig := fmt.Sprintf("%d|%.4f|%.4f|%.4f|%.4f|%.4f", b.OpenTime, b.Open, b.High, b.Low, b.Close, b.Volume)
	if sig != at.lastCycleBarSig {
		at.lastCycleBarSig = sig
		return false
	}
	// Identical bar state — only skip when flat (in-position cycles are the
	// dashboard heartbeat and must keep running; PR #49 contract).
	if at.store != nil {
		if positions, err := at.store.Position().GetOpenPositions(at.id); err == nil && len(positions) > 0 {
			return false
		}
	}
	at.logInfof("⏭ cycle_skip=no_new_data — newest %s bar unchanged since the last cycle and flat; not burning an AI call on an identical snapshot. (%s)",
		at.primaryTimeframe(), kernel.FormatCT(now))
	return true
}

// skipWhileOpen (P2.2) reports whether the AI decision cycle should be skipped
// because the strategy is already holding a position — calmer and cheaper than
// per-decision same-side refusal. GATED on day_plan → dormant by default. The
// held trade is still managed independently: the NT8 OCO bracket, auto-breakeven,
// and close-sync/reconcile all run outside the decision cycle, so exits and flips
// (bracket/target/stop) are unaffected.
func (at *AutoTrader) skipWhileOpen() (bool, string) {
	if !at.dayPlanEnabled() || at.store == nil {
		return false, ""
	}
	positions, err := at.store.Position().GetOpenPositions(at.id)
	if err != nil || len(positions) == 0 {
		return false, ""
	}
	// P0 hotfix (2026-08-19) — the gate never trusts a stale local row against
	// live NT8 truth: bracket exits land in the store via the reconcile "sync"
	// path (≤~80s), and during that window this row is a phantom. On desync
	// (broker flat, store open): CRITICAL + no skip; the reconciler keeps
	// closing authority. See position_desync.go. POSITION_RECONCILE default on.
	if at.skipGateDesync(positions) {
		return false, ""
	}
	p := positions[0]
	return true, fmt.Sprintf("%d open (%s %s)", len(positions), p.Symbol, p.Side)
}

// primaryTimeframe is the strategy's primary bar interval (default 5m).
func (at *AutoTrader) primaryTimeframe() string {
	if at.config.StrategyConfig == nil {
		return "5m"
	}
	tf := at.config.StrategyConfig.Indicators.Klines.PrimaryTimeframe
	if tf == "" {
		return "5m"
	}
	return tf
}

func (at *AutoTrader) futuresSymbol() string {
	s := at.config.NinjaTraderSymbol
	if s == "" {
		return "MNQ"
	}
	return s
}

// sessionChainDate is the plan-chain identity date for a session: the CT date
// of the session INSTANCE the moment belongs to (P0-B), falling back to the
// midnight-roll planner date when the session has no parseable window.
func sessionChainDate(sess *kernel.SessionDef, now time.Time) string {
	if d, ok := kernel.PlanChainTradeDate(sess, now); ok {
		return d
	}
	return plannerTradeDateCT(now)
}

// latestClosedPrimaryBarMs returns the CloseTime (ms) of the most recent CLOSED
// primary-TF bar, ok=false when none is available (provider down / warming).
func (at *AutoTrader) latestClosedPrimaryBarMs() (int64, bool) {
	if market.FuturesBarsProvider == nil {
		return 0, false
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), at.primaryTimeframe(), 5)
	nowMs := time.Now().UnixMilli()
	var latest int64
	for i := range bars {
		if bars[i].CloseTime < nowMs && bars[i].CloseTime > latest {
			latest = bars[i].CloseTime
		}
	}
	if latest == 0 {
		return 0, false
	}
	return latest, true
}

// barCloseGate decides whether a tick should run a decision cycle under bar-close
// cadence, and returns the updated last-close watermark. When cadence is not
// active it ALWAYS runs (scan-timer behavior unchanged). When active it runs only
// once per NEW primary-TF bar close — never mid-bar, and a restart resumes on the
// next close.
func barCloseGate(active bool, lastCloseMs, latestClosedMs int64, haveBar bool) (run bool, newLastMs int64) {
	if !active {
		return true, lastCloseMs
	}
	if !haveBar || latestClosedMs <= lastCloseMs {
		return false, lastCloseMs
	}
	return true, latestClosedMs
}

// ---- P2.3 — day-trader clock: last_entry + eod_flat ------------------------

// hhmmToMin parses "HH:MM" (24h) into minutes-since-midnight; ok=false on bad input.
func hhmmToMin(s string) (int, bool) {
	var h, m int
	n, err := fmt.Sscanf(strings.TrimSpace(s), "%d:%d", &h, &m)
	if err != nil || n != 2 || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// ctMinutesNow returns now's minutes-since-midnight in America/Chicago.
func ctMinutesNow(now time.Time) int {
	loc := kernel.CTLocation()
	ct := now.In(loc)
	return ct.Hour()*60 + ct.Minute()
}

// timeReachedCT reports whether now's CT wall-clock is at/after hhmm. Correct for
// all-day-after-threshold gates (last-entry, eod-flat) — NOT for the planner read,
// which must fire only inside its session's window (see inSessionReadWindow).
func timeReachedCT(now time.Time, hhmm string) bool {
	target, ok := hhmmToMin(hhmm)
	if !ok {
		return false
	}
	return ctMinutesNow(now) >= target
}

// inSessionReadWindow (W1) reports whether now (CT) is inside a session's read
// window [ReadCT, WindowEndCT) — wrap-aware for midnight-spanning sessions (ASIA
// 16:55→02:00). This is the correct "should the planner read THIS session now"
// gate: it fires at the session's read time and stays valid through the session,
// but NEVER during another session's hours. This kills the spurious Sunday-17:00
// NY read (17:00 is past NY's 15:00 window end, so NY never fires there), so
// Monday's plan is built at the real 08:25 read, not from stale Sunday-evening data.
func inSessionReadWindow(now time.Time, readCT, windowEndCT string) bool {
	read, ok1 := hhmmToMin(readCT)
	end, ok2 := hhmmToMin(windowEndCT)
	if !ok1 || !ok2 {
		return false
	}
	n := ctMinutesNow(now)
	if end > read {
		return n >= read && n < end // same-day window (NY 08:25→15:00)
	}
	return n >= read || n < end // wraps midnight (ASIA 16:55→02:00)
}

// inDailyRollWindow (W1) reports whether now (CT) is in the trade-date roll-up
// window [15:00, 16:00) — after the RTH close, BEFORE the 16:00 CME maintenance
// break/17:00 roll. The current trade_date + P&L window are still the CLOSING
// day's here (they roll at 17:00), so the daily digest summarizes the RIGHT day;
// and it is reachable Mon–FRI (the old >=16:00 trigger fell inside the break, so
// it fired at 17:00+ with the NEW day's empty window, and Friday's never fired).
func inDailyRollWindow(now time.Time) bool {
	n := ctMinutesNow(now)
	return n >= 15*60 && n < 16*60
}

// effectiveEODFlatCT returns the flat time, pulled IN by a registered half-day
// early-close for the current CME session-day (holiday/half-day awareness via the
// P0 registry, which the P1.8 calendar populates). Empty HalfDays → configFlat.
// P4 note (2026-08-19): the live flatten now resolves through halfDayCutoffMin
// (early_close − offset); this zero-offset form is kept for its tests/API.
func effectiveEODFlatCT(reg kernel.SessionRegistry, sessionDayKey, configFlat string) string {
	if early, ok := reg.HalfDays[sessionDayKey]; ok && strings.TrimSpace(early) != "" {
		em, ok1 := hhmmToMin(early)
		cm, ok2 := hhmmToMin(configFlat)
		if ok1 && (!ok2 || em < cm) {
			return early // half-day closes earlier → pull the flat in
		}
	}
	return configFlat
}

// lastEntryCT — the LEGACY day-scoped cutoff. UNREACHABLE since the P2
// session-scope redesign (2026-08-18): entryBlockedByLastEntry resolves the
// cutoff per session via sessionCutoffCT below. Kept only so an old
// dp.LastEntryCT config value is visible to a reader; nothing evaluates it.
// See sessionCutoffCT + entryBlockedByLastEntry for the live path.
func (at *AutoTrader) lastEntryCT() string {
	if dp := at.config.StrategyConfig.DayPlan; dp != nil && strings.TrimSpace(dp.LastEntryCT) != "" {
		return dp.LastEntryCT
	}
	return "13:00" // 14:00 ET (NY-only era)
}

// sessionCutoffCT resolves "session end − offsetMin" as CT wall-clock minutes
// and as an "HH:MM" string, wrap-aware for midnight-spanning sessions (ASIA
// 17:00→02:00, offset 15 → 01:45). ok=false on malformed registry times.
func sessionCutoffCT(sess *kernel.SessionDef, offsetMin int) (cutoffMin int, hhmm string, ok bool) {
	endMin, okE := hhmmToMin(sess.WindowEndCT)
	if !okE {
		return 0, "", false
	}
	cutoffMin = ((endMin-offsetMin)%1440 + 1440) % 1440
	return cutoffMin, fmt.Sprintf("%02d:%02d", cutoffMin/60, cutoffMin%60), true
}

// pastSessionCutoff reports whether now (CT) is INSIDE the session window and
// at/after the session-relative cutoff. Comparison is done in minutes-since-
// session-start so a midnight wrap cannot invert it. Outside the window it is
// always false — the session-open gate, not last-entry, owns that refusal.
func pastSessionCutoff(now time.Time, sess *kernel.SessionDef, cutoffMin int) bool {
	startMin, okS := hhmmToMin(sess.WindowStartCT)
	endMin, okE := hhmmToMin(sess.WindowEndCT)
	if !okS || !okE {
		return false
	}
	sessLen := ((endMin-startMin)%1440 + 1440) % 1440
	nowOff := ((ctMinutesNow(now)-startMin)%1440 + 1440) % 1440
	cutOff := ((cutoffMin-startMin)%1440 + 1440) % 1440
	return nowOff < sessLen && nowOff >= cutOff
}

// eodFlatCT — LEGACY day-scoped flat time. UNREACHABLE since the session-scope
// redesign (2026-08-18); enforceEODFlatAt resolves per session. Kept for
// visibility of an old dp.EODFlatCT config value only.
func (at *AutoTrader) eodFlatCT() string {
	if dp := at.config.StrategyConfig.DayPlan; dp != nil && strings.TrimSpace(dp.EODFlatCT) != "" {
		return dp.EODFlatCT
	}
	return "14:45" // 15:45 ET
}

// entryBlockedByLastEntry (P2.3, session-scoped 2026-08-18) reports (reason,
// blocked): whether NEW entries are blocked because the ACTIVE session's
// last-entry cutoff (session end − last_entry_offset_min, America/Chicago,
// DST-correct via kernel.CTLocation) has passed. Gated on day_plan.
//
// THE BUG THIS REPLACES (incident B, zero-trade cause): the old body was
// timeReachedCT(now, "13:00") — a DAY-scoped comparison that stayed true from
// 13:00 CT to midnight, so the single NY-era cutoff refused every Asia-evening
// entry (live refusal observed at ~21:00 CT). Defect classes 2 (day-scope where
// session-scope required) and 3 (the 13:00 literal shadowing per-session
// config). The old lastEntryCT path is now unreachable from any gate.
//
// Outside every session window this gate never fires — the session gate owns
// that refusal — and the message names the session + resolved time.
func (at *AutoTrader) entryBlockedByLastEntry() (string, bool) {
	return at.entryBlockedByLastEntryAt(time.Now())
}

// entryBlockedByLastEntryAt is the injectable-clock body, so the T1–T4 table
// tests can pin real CT instants (including a DST-transition date).
func (at *AutoTrader) entryBlockedByLastEntryAt(now time.Time) (string, bool) {
	if !at.dayPlanEnabled() {
		return "", false
	}
	sess, ok := at.sessionRegistry(now).ActiveSession(now)
	if !ok {
		return "", false // no session → the session-open gate is the refusal
	}
	offset := at.config.StrategyConfig.DayPlan.LastEntryOffsetFor(sess.Name)
	cutoffMin, hhmm, okC := sessionCutoffCT(sess, offset)
	if !okC {
		return "", false // malformed registry times — never invent a block
	}
	// P4 (ledger-close 2026-08-19) — a registered half-day pulls the LAST-ENTRY
	// cutoff in too: the cutoff resolves against early_close_CT − offset when
	// that is earlier (min semantics, same raw-minute convention as the flat
	// pull-in). Sessions that end before the early close are unaffected.
	if adj, adjHHMM, okH := halfDayCutoffMin(at.sessionRegistry(now), kernel.CMESessionDayKey(now), offset); okH && adj < cutoffMin {
		cutoffMin, hhmm = adj, adjHHMM
	}
	if pastSessionCutoff(now, sess, cutoffMin) {
		return fmt.Sprintf("past last-entry %s CT (%s)", hhmm, sess.Name), true
	}
	return "", false
}

// halfDayCutoffMin resolves "early_close_CT − offset" for the session-day's
// registered half-day (P4). ok=false when no half-day / unparseable value.
func halfDayCutoffMin(reg kernel.SessionRegistry, sessionDayKey string, offsetMin int) (int, string, bool) {
	early, ok := reg.HalfDays[sessionDayKey]
	if !ok || strings.TrimSpace(early) == "" {
		return 0, "", false
	}
	em, okE := hhmmToMin(early)
	if !okE {
		return 0, "", false // fail-safe: garbage half-day never invents a cutoff
	}
	adj := ((em-offsetMin)%1440 + 1440) % 1440
	return adj, fmt.Sprintf("%02d:%02d", adj/60, adj%60), true
}

// enforceEODFlat (P2.3, session-scoped 2026-08-18) force-flattens any open
// position at/after the ACTIVE session's flat time (session end −
// eod_flat_offset_min, half-day pull-in preserved) by routing DIRECTLY through
// the trader close path — bypassing hold-lock naturally (RECON #10). Returns
// true when it acted (the caller then skips the rest of the cycle). Gated on
// day_plan.
//
// THE TWIN BUG (class 7 of the last-entry fix): the old body was
// timeReachedCT(now, "14:45") — day-scoped, true 14:45 CT → midnight — which
// never mattered while the 13:00 last-entry bug guaranteed no Asia positions,
// but the moment Asia entries flow it would flatten each one ON SIGHT at, say,
// 21:00 CT. Fixing last-entry without this would have been worse than fixing
// neither.
//
// Scope: INSIDE the active session, flatten past its resolved flat time.
// BETWEEN sessions (the 14:45→17:00 CT gap, weekends) any open position is
// flattened too — that is by definition past the previous session's flat, and
// it preserves the old rule's one virtue: nothing rides through the close.
func (at *AutoTrader) enforceEODFlat() bool {
	return at.enforceEODFlatAt(time.Now())
}

func (at *AutoTrader) enforceEODFlatAt(now time.Time) bool {
	if !at.dayPlanEnabled() || at.store == nil || at.trader == nil {
		return false
	}
	reg := at.sessionRegistry(now)
	flat := "" // resolved wall-clock, for the log only
	if sess, ok := reg.ActiveSession(now); ok {
		offset := at.config.StrategyConfig.DayPlan.EODFlatOffsetFor(sess.Name)
		flatMin, hhmm, okC := sessionCutoffCT(sess, offset)
		if !okC {
			return false // malformed registry times — never invent a flatten
		}
		// Half-day early close pulls the flat IN. P4 (ledger-close 2026-08-19):
		// the flat now resolves against early_close_CT − eod_flat_offset (the
		// dispatch 4.4 contract) — with the default offset 0 this is byte-
		// identical to the original effectiveEODFlatCT pull-in.
		if adj, adjHHMM, okH := halfDayCutoffMin(reg, kernel.CMESessionDayKey(now), offset); okH && adj < flatMin {
			flatMin, hhmm = adj, adjHHMM
		}
		if !pastSessionCutoff(now, sess, flatMin) {
			return false
		}
		flat = fmt.Sprintf("%s CT (%s)", hhmm, sess.Name)
	} else {
		// No active session: past every session's flat by definition.
		flat = "no active session"
	}
	positions, err := at.store.Position().GetOpenPositions(at.id)
	if err != nil || len(positions) == 0 {
		return false
	}
	at.logWarnf("🕒 EOD-FLAT (%s): session close — flattening %d open position(s) via the trader close path.", flat, len(positions))
	for _, p := range positions {
		var e error
		if strings.EqualFold(p.Side, "LONG") {
			_, e = at.trader.CloseLong(p.Symbol, 0) // 0 = close all
		} else {
			_, e = at.trader.CloseShort(p.Symbol, 0)
		}
		if e != nil {
			at.logErrorf("🕒 EOD-FLAT: close %s %s failed: %v", p.Symbol, p.Side, e)
			continue
		}
		// Cancel the resting OCO bracket after the flatten (belt-and-suspenders;
		// the OCO also auto-cancels its other leg on the close fill).
		if err := at.trader.CancelStopOrders(p.Symbol); err != nil {
			at.logWarnf("🕒 EOD-FLAT: cancel bracket %s failed (non-fatal): %v", p.Symbol, err)
		}
	}
	return true
}

// recordExcursionForClosedSymbol (P2.4) processes the just-closed trade for the
// AI-emitted-close path. W5 — the SAME analytics also fire from the loop poll for
// real exits (NT8 OCO / EOD-flat / manual), so no exit path is missed.
func (at *AutoTrader) recordExcursionForClosedSymbol(symbol string) {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	closed, err := at.store.Position().GetClosedPositions(at.id, 1)
	if err != nil || len(closed) == 0 {
		return
	}
	if p := closed[0]; p.Symbol == symbol {
		at.recordClosedTradeAnalytics(p)
	}
}

// recordClosedTradeAnalytics (W5) computes MAE/MFE + the A–F adherence grade +
// the matched-random reaction verdict for ONE closed trade. Idempotent (a graded
// row is skipped), so the AI-close call and the loop poll never double-process.
// Gated on day_plan → dormant for crypto / plan-off.
func (at *AutoTrader) recordClosedTradeAnalytics(p *store.TraderPosition) {
	if p == nil || p.AdherenceGrade != "" { // already processed
		return
	}
	if !at.dayPlanEnabled() || at.store == nil || market.FuturesBarsProvider == nil {
		return
	}
	if p.EntryPrice <= 0 || p.EntryTime <= 0 {
		return
	}
	exitMs := p.ExitTime
	if exitMs <= 0 {
		exitMs = time.Now().UnixMilli()
	}
	bars := market.FuturesBarsProvider(at.futuresSymbol(), "1m", kernel.AISVPBarCount)
	ex := kernel.ComputeExcursion(p.EntryPrice, p.Side, bars, p.EntryTime, exitMs)
	if err := at.store.Position().UpdateExcursion(p.ID, ex.MAE, ex.MFE); err != nil {
		at.logWarnf("📐 excursion update failed for %s: %v", p.Symbol, err)
		return
	}
	// P5.5 — ADHERENCE GRADE (A–F), separate from P&L.
	inKZ, inNoTrade := kernel.SessionWindowFacts(at.sessionRegistry(time.Now()), time.UnixMilli(p.EntryTime))
	grade, _ := kernel.GradeAdherence(kernel.AdherenceInput{
		Cited:      p.CitedScenarioID != "",
		Matched:    p.PlanMatched,
		OffPlan:    p.CitedScenarioID == "",
		InNoTrade:  inNoTrade,
		InKillzone: inKZ,
	})
	if err := at.store.Position().SetAdherence(p.ID, grade); err != nil {
		at.logWarnf("🎓 adherence grade update failed for %s: %v", p.Symbol, err)
		return
	}
	at.logInfof("📐🎓 %s close: MAE %.2f / MFE %.2f pts · adherence %s (%s) cited=%q matched=%v",
		p.Symbol, ex.MAE, ex.MFE, grade, kernel.AdherenceLabel(grade), p.CitedScenarioID, p.PlanMatched)

	// W6 — P0 close+P&L alert.
	at.emitAlert("P0", "close", fmt.Sprintf("close:%d", p.ID),
		fmt.Sprintf("Closed %s — P&L %.2f", p.Symbol, p.RealizedPnL),
		fmt.Sprintf("adherence %s (%s)", grade, kernel.AdherenceLabel(grade)))

	// P5.6 — matched-random reaction verdict for the traded level type.
	at.recordMatchedRandomForClose(p, ex)
}

// maybeRecordClosedTradeAnalytics (W5) is the LOOP POLL that catches trades closed
// on the REAL exit paths (NT8 OCO SL/TP, EOD-flat, manual) — none of which run the
// AI decision cycle. It grades every ungraded close since the analytics epoch
// (set on first run so pre-existing history is ignored). Idempotent via the grade
// flag; gated on day_plan.
func (at *AutoTrader) maybeRecordClosedTradeAnalytics() {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	const key = "dayplan_analytics_since"
	sinceStr, _ := at.store.GetSystemConfig(key)
	if sinceStr == "" {
		// First run: stamp the epoch + skip all pre-existing closes (not day-plan trades).
		_ = at.store.SetSystemConfig(key, strconv.FormatInt(time.Now().UnixMilli(), 10))
		return
	}
	sinceMs, err := strconv.ParseInt(sinceStr, 10, 64)
	if err != nil {
		return
	}
	rows, err := at.store.Position().GetUngradedClosedPositions(at.id, sinceMs, 20)
	if err != nil {
		return
	}
	for _, p := range rows {
		at.recordClosedTradeAnalytics(p)
	}
}

// tickOnce runs one loop iteration: a grid cycle, or (for AI strategies) a
// decision cycle gated by bar-close cadence.
func (at *AutoTrader) tickOnce(isGrid bool) {
	if isGrid {
		if err := at.RunGridCycle(); err != nil {
			at.logErrorf("❌ Grid execution failed: %v", err)
		}
		return
	}
	active := at.barCloseCadenceActive()
	// P10 (owner ruling 2026-08-19): the bar-close gate is a MODE now, not a
	// hard gate. mode=bar_close keeps the legacy behavior byte-identical (E14);
	// mode=interval (default) runs a full cycle every tick on the latest bar
	// state — forming bar included — subject only to the no-new-data dedup.
	if active && at.cadenceMode() == CadenceBarClose {
		latest, have := at.latestClosedPrimaryBarMs()
		run, newLast := barCloseGate(true, at.lastBarCloseMs, latest, have)
		at.lastBarCloseMs = newLast
		if !run {
			return // bar-close mode: no new primary-TF bar closed → idle this tick
		}
	}
	if active && at.cadenceMode() == CadenceInterval && at.skipNoNewData(time.Now()) {
		return
	}
	if err := at.runCycle(); err != nil {
		at.logErrorf("❌ Execution failed: %v", err)
	}
}
