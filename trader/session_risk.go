package trader

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── SESSION RISK LIMITS (2026-09-09, dispatch 104) ───────────────────────────
//
// Two gates that existed but did not guard the path that trades.
//
// THE BREAKER. consecutive_loss_halt has been wired since D1 — on the DECISION
// path only (auto_trader_orders.go). Under plan_mode=strict the decision path
// cannot enter at all ("plan_mode=strict executes plan scenarios on the ARM
// path only", entry_gate.go), so the circuit breaker guarded a door nobody
// walks through. Unlike the daily limit it is NOT gated by the guardrails
// master, so it is the one session limit that can actually bite today.
//
// THE NO-TRADE BAND. Same shape, found the same day: lunch and first-N are
// enforced at auto_trader_orders.go:281 and NOWHERE else — armed_executor.go
// contained zero references to InLunchNoTrade, InFirstNoTradeMinutes or
// sessionEntryBlocked. An arm could be placed at 12:15 CT, inside the band the
// rest of the system believes is closed.
//
// THRESHOLDS ARE LABELLED, NOT MEASURED. The research establishes session risk
// limits as a REQUIREMENT and does not establish the NUMBERS. N and M are [I]
// until the record says otherwise, and they ride the boot line so the owner
// reads what is actually enforced rather than what a doc claims.
//
// MEASURED ON THIS TAPE (n=73 usable era closes, 2026-08-15→2026-09-09):
// the longest losing run is SEVEN (ids 585-591). N=8 therefore NEVER FIRES on
// this tape — it sits one above the observed maximum, the same shape as the
// $450 daily limit that trips 0 of 12 days. M=5 does fire. Both facts are in
// the report; neither is hidden behind a default.

const (
	// breakerHaltDefault — N. vet-06's [I] proposal (8 of the last 10 losing).
	// Never fires on the retained tape; see the note above.
	breakerHaltDefault = 8
	// breakerWarnDefault — M. WARN-first: counts and surfaces, never refuses.
	breakerWarnDefault = 5
)

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return def
}

// breakerHaltN resolves N: the strategy knob when the owner has set it, else
// the [I] default. 0 in the knob means "unset", not "off" — the wave turns the
// breaker ON (dispatch 104 D2). Set BREAKER_HALT_N=0 to disable it entirely.
func breakerHaltN(cfg *store.StrategyConfig) int {
	if cfg != nil && cfg.RiskControl.ConsecutiveLossHalt > 0 {
		return cfg.RiskControl.ConsecutiveLossHalt
	}
	return envInt("BREAKER_HALT_N", breakerHaltDefault)
}

// breakerWarnM resolves M, the WARN-first threshold.
func breakerWarnM() int { return envInt("BREAKER_WARN_M", breakerWarnDefault) }

// sessionRiskVerdict is the whole adjudication as one value, so a caller cannot
// act on half of it.
type sessionRiskVerdict struct {
	Refuse bool
	Reason string
	Class  string // the armRefusalClass key this refusal is counted under
	Warn   bool
	Losses int
	N      int
	M      int
}

// adjudicateSessionRisk is PURE. bandReason is "" when the no-trade band is
// open for business.
//
// ORDER MATTERS AND IS DELIBERATE: the band is a WINDOW (a fact about the
// clock) and the breaker is a STATE (a fact about the record). The window is
// checked first because it is the cheaper, more certain refusal — and because a
// breaker WARN inside the band would tell the owner about a run of losses at a
// moment when nothing could have been entered anyway.
func adjudicateSessionRisk(losses, n, m int, bandReason string) sessionRiskVerdict {
	if bandReason != "" {
		return sessionRiskVerdict{
			Refuse: true, Class: "no_trade_band", Losses: losses, N: n, M: m,
			Reason: "no_trade_band: " + bandReason,
		}
	}
	if n > 0 && losses >= n {
		return sessionRiskVerdict{
			Refuse: true, Class: "consecutive_loss", Losses: losses, N: n, M: m,
			Reason: fmt.Sprintf("consecutive_loss_halt: %d consecutive losing trades this session-day (limit %d) — no new entry on any path until the CME roll", losses, n),
		}
	}
	v := sessionRiskVerdict{Losses: losses, N: n, M: m}
	if m > 0 && losses >= m {
		v.Warn = true
		v.Reason = fmt.Sprintf("consecutive_loss WARN: %d consecutive losing trades this session-day (warn %d, halt %d) — counted, not refused", losses, m, n)
	}
	return v
}

// sessionRiskGateAt is the production entry point: it reads THIS trader's
// session-day loss run and the no-trade band, and adjudicates once. now is the
// caller's clock (A28).
//
// FAIL-OPEN on a read error. A circuit breaker that trips because the database
// hiccuped is a worse failure than the one it prevents — and the arm path's
// other guards are unaffected.
func (at *AutoTrader) sessionRiskGateAt(now time.Time) sessionRiskVerdict {
	n := breakerHaltN(at.config.StrategyConfig)
	m := breakerWarnM()
	bandReason, blocked := at.sessionEntryBlockedAt(now)
	if !blocked {
		bandReason = ""
	}
	losses := 0
	if at.store != nil {
		sinceMs := kernel.CMESessionDayStart(now).UnixMilli()
		if got, err := at.store.Position().CountConsecutiveLossesSince(at.id, sinceMs); err == nil {
			losses = got
		} else {
			at.logWarnf("🛑 session-risk: loss-run query failed (%v) — breaker NOT applied this cycle (fail-open)", err)
			return adjudicateSessionRisk(0, n, m, bandReason)
		}
	}
	return adjudicateSessionRisk(losses, n, m, bandReason)
}

// SessionRiskBootLine — D5. Every field READ from the code that enforces it
// (A11); the labels are part of the line because a number without its evidence
// tier is a number someone will later mistake for a measurement.
func SessionRiskBootLine(cfg *store.StrategyConfig, dailyLimit float64, masterOn, dailyLegOn bool, flatHHMM string) string {
	master, leg := "off", "off"
	if masterOn {
		master = "on"
	}
	if dailyLegOn {
		leg = "on"
	}
	daily := fmt.Sprintf("$%.0f[O]", dailyLimit)
	if !masterOn || !dailyLegOn {
		daily += " DECORATIVE (guardrails master " + master + ", daily_loss_enabled " + leg + " — both must be on)"
	}
	n, m := breakerHaltN(cfg), breakerWarnM()
	// A THRESHOLD THAT CAN NEVER FIRE IS REPORTED, NEVER CLAMPED. If the owner
	// set a halt below the WARN, the WARN is dead — the halt refuses first, so
	// nothing ever reaches M. Silently clamping would hide the setting he chose;
	// he should see what he set and what it costs him.
	warn := fmt.Sprintf("warn=%d[I]", m)
	if n > 0 && m >= n {
		warn = fmt.Sprintf("warn=%d[I] UNREACHABLE (halt=%d fires first)", m, n)
	}
	return fmt.Sprintf(
		"session risk: daily=%s · breaker=%d[I] %s (not master-gated; never fires on the retained tape, max run 7, ids 585-591) · no-trade-band=arm+decision · post-loss counter=on(%dm) · flat@%s=position+arms+pending",
		daily, n, warn, postLossWindowMin(), flatHHMM)
}

// ── D3 — THE POST-LOSS RE-ARM COUNTER ───────────────────────────────────────
//
// COUNTER ONLY, BY DESIGN. It never refuses. K is [I] until the record can
// support a number: the retained era holds only FOUR stop-outs, and ZERO of
// them were re-armed at the same level, so nothing in the tape justifies a
// cool-down that blocks. What the tape justifies is counting.

const postLossWindowDefault = 30 // minutes, [I]

func postLossWindowMin() int { return envInt("POST_LOSS_WINDOW_MIN", postLossWindowDefault) }

// sameMergedLevel reports whether a re-arm targets the level the losing trade
// was entered at. A tick is the tolerance: two prices a tick apart are two
// levels, and anything looser would label unrelated arms.
func sameMergedLevel(armEntryPx, lostEntryPx, tick float64) bool {
	if armEntryPx <= 0 || lostEntryPx <= 0 || tick <= 0 {
		return false
	}
	d := armEntryPx - lostEntryPx
	if d < 0 {
		d = -d
	}
	return d <= tick/2
}

// postLossReArm reports whether this arm is a re-arm at the SAME level within K
// minutes of a resolved LOSING close, and the label to carry if so.
//
// Returns false on any uncertainty. A counter that over-counts is a counter the
// owner learns to discount.
func (at *AutoTrader) postLossReArm(armEntryPx float64, now time.Time) (bool, string) {
	if at.store == nil || armEntryPx <= 0 {
		return false, ""
	}
	k := postLossWindowMin()
	sinceMs := now.Add(-time.Duration(k) * time.Minute).UnixMilli()
	lost, ok := at.store.Position().LastLosingClose(at.id, sinceMs)
	if !ok || lost == nil {
		return false, ""
	}
	if !store.PostLossWindow(lost.ExitTime, now, k) {
		return false, ""
	}
	if !sameMergedLevel(armEntryPx, lost.EntryPrice, market.FuturesTickSize(at.futuresSymbol())) {
		return false, ""
	}
	mins := int(now.Sub(time.UnixMilli(lost.ExitTime)).Minutes())
	return true, fmt.Sprintf("re-arm after loss (%d min after position %d closed at a loss on the same level %.2f)",
		mins, lost.ID, lost.EntryPrice)
}

// SessionRiskBootLineForBoot is the process-level boot line. At boot there is no
// bound strategy in hand, so the guardrail toggles are read from the DEFAULT
// trader's resolved config where one exists and print n/a where it does not —
// never a literal, and never a cheerful default standing in for a value the
// process has not read (A11/A24).
func SessionRiskBootLineForBoot(st *store.Store) string {
	var cfg *store.StrategyConfig
	limit, masterOn, legOn := 0.0, false, false
	resolved := false
	if st != nil {
		if c, l, m, g, ok := bootRiskFacts(st); ok {
			cfg, limit, masterOn, legOn, resolved = c, l, m, g, true
		}
	}
	if !resolved {
		return fmt.Sprintf(
			"session risk: daily=n/a (no bound strategy read at boot) · breaker=%d[I] warn=%d[I] (not master-gated) · no-trade-band=arm+decision · post-loss counter=on(%dm) · flat=position+arms+pending",
			breakerHaltN(nil), breakerWarnM(), postLossWindowMin())
	}
	return SessionRiskBootLine(cfg, limit, masterOn, legOn, "session close")
}

// bootRiskFacts resolves the guardrail toggles at boot from THE STRATEGY BOUND
// TO A TRADER — the same row deskGuardrail reads at runtime
// (at.config.StrategyConfig). One source for the bound row, both readers.
//
// THE DEFECT THIS REPLACES, live on the 2026-09-09 18:14 boot line: this
// scanned EVERY strategy row and returned the first carrying risk-shaped
// fields. The store held nine; it read one ("New Strategy",
// daily_loss_enabled=true, consecutive_loss_halt=2) that is bound to NO trader,
// and printed its knobs as though they governed. The line said
// "daily_loss_enabled on · breaker=2" about a desk where neither was true.
//
// The runtime was never wrong — breakerHaltN reads the trader's own config — so
// only the boot line lied, on the one line whose whole job is to say what is
// enforced. Class 82 shipped in the line beside the deskGuardrail fix written
// for it.
//
// MORE THAN ONE BOUND STRATEGY is reported, not averaged: a single line cannot
// speak for two desks, and picking one silently is how this defect started.
func bootRiskFacts(st *store.Store) (cfg *store.StrategyConfig, limit float64, masterOn, dailyLegOn, ok bool) {
	if st == nil || st.GormDB() == nil {
		return nil, 0, false, false, false
	}
	var ids []string
	if err := st.GormDB().
		Raw(`SELECT DISTINCT strategy_id FROM traders WHERE strategy_id IS NOT NULL AND strategy_id <> ''`).
		Scan(&ids).Error; err != nil || len(ids) != 1 {
		return nil, 0, false, false, false
	}
	var rows []*store.Strategy
	if err := st.GormDB().Where("id = ?", ids[0]).Find(&rows).Error; err != nil || len(rows) != 1 {
		return nil, 0, false, false, false
	}
	c, err := rows[0].ParseConfig()
	if err != nil || c == nil {
		return nil, 0, false, false, false
	}
	rc := c.RiskControl
	return c, rc.DailyLossLimitUSD,
		rc.GuardrailsEnabled != nil && *rc.GuardrailsEnabled,
		rc.DailyLossEnabled != nil && *rc.DailyLossEnabled, true
}
