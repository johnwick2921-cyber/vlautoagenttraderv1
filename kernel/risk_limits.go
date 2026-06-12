package kernel

// Plan 3 Task 21 — Risk limits + force-flat kill switch.
//
// Hard server-side guard rails enforced BEFORE expensive prompt assembly.
// The AI prompt itself only contains soft hints ("don't risk more than 1%"),
// which the model may ignore. These limits are non-negotiable.
//
// Design goals:
//   - Plain primitives in CheckPreTrade so the function is trivially testable
//     without constructing a full *Context / *Decision (the plan's illustrative
//     types do not exactly match the real ones).
//   - Decoupled from concrete broker packages (ninjatrader, binance, etc.) so
//     that the engine can call ForceFlat via a tiny interface.
//   - Loaded from env once at startup via LoadRiskLimitsFromConfig().

import (
	"fmt"
	"nofx/config"
	"nofx/logger"
	"sync"
	"time"
)

// RiskLimits caps the risk a strategy can take before the engine intervenes.
// Zero values disable the corresponding check (so partial config is safe).
type RiskLimits struct {
	MaxDailyLossUSD      float64 // hard stop for the trading day (USD, positive number)
	MaxConcurrentTrades  int     // open position cap (count)
	MaxNotionalUSD       float64 // sum of (entry price * quantity) cap across open positions
	MaxContractsPerOrder int     // single-order contract size cap
}

// RiskLimitDecision is the qualitative outcome of a pre-trade check.
// Callers may use the typed err for logging and the decision for action.
type RiskLimitDecision int

const (
	RiskAllow      RiskLimitDecision = iota // no constraints hit
	RiskBlockEntry                          // block this entry, hold existing positions
	RiskForceFlat                           // close everything (daily-loss tripped)
)

// CheckPreTrade evaluates the proposed trade against all enabled limits.
// Inputs are plain primitives so this can be exercised in a unit test
// without a full engine.Context.
//
//   totalPnL          — account.TotalPnL (negative = losing)
//   openPositions     — len(ctx.Positions)
//   requestedNotional — abs(decision.PositionSizeUSD) for the proposed entry
//   existingNotional  — sum of MarkPrice*Quantity over existing positions
//
// Semantics:
//   - PnL strictly LESS THAN -MaxDailyLossUSD trips the daily-loss limit.
//     PnL == -MaxDailyLossUSD does NOT trip (boundary is permissive, matches
//     the plan's `< -r.MaxDailyLossUSD` source). PnL == limit slightly past
//     trips immediately.
//   - openPositions >= cap trips (cap-equals-trip).
//   - existing + requested > cap trips (strict exceed).
//   - When MaxDailyLossUSD == 0 the daily-loss check is disabled; same for
//     MaxConcurrentTrades and MaxNotionalUSD (so partial config is safe).
func (r *RiskLimits) CheckPreTrade(totalPnL float64, openPositions int, requestedNotional float64, existingNotional float64) error {
	if r == nil {
		return nil
	}
	if r.MaxDailyLossUSD > 0 && totalPnL < -r.MaxDailyLossUSD {
		return fmt.Errorf("risk: daily loss limit hit (pnl=%.2f, limit=-%.2f)", totalPnL, r.MaxDailyLossUSD)
	}
	if r.MaxConcurrentTrades > 0 && openPositions >= r.MaxConcurrentTrades {
		return fmt.Errorf("risk: concurrent trade cap reached (open=%d, cap=%d)", openPositions, r.MaxConcurrentTrades)
	}
	if r.MaxNotionalUSD > 0 {
		total := existingNotional + requestedNotional
		if total > r.MaxNotionalUSD {
			return fmt.Errorf("risk: notional cap exceeded (existing=%.2f + requested=%.2f > cap=%.2f)", existingNotional, requestedNotional, r.MaxNotionalUSD)
		}
	}
	return nil
}

// Classify returns the qualitative decision class without re-running checks.
// daily-loss trip → ForceFlat (close everything).
// other trips     → BlockEntry (hold existing, refuse new).
// nil err         → Allow.
func (r *RiskLimits) Classify(totalPnL float64, openPositions int, requestedNotional float64, existingNotional float64) (RiskLimitDecision, error) {
	if r == nil {
		return RiskAllow, nil
	}
	if r.MaxDailyLossUSD > 0 && totalPnL < -r.MaxDailyLossUSD {
		return RiskForceFlat, fmt.Errorf("risk: daily loss limit hit (pnl=%.2f, limit=-%.2f)", totalPnL, r.MaxDailyLossUSD)
	}
	if err := r.CheckPreTrade(totalPnL, openPositions, requestedNotional, existingNotional); err != nil {
		return RiskBlockEntry, err
	}
	return RiskAllow, nil
}

// LoadRiskLimitsFromConfig pulls the four limits from config.Get().
// Called once at startup or each cycle (cheap — just struct copy).
func LoadRiskLimitsFromConfig() RiskLimits {
	c := config.Get()
	return RiskLimits{
		MaxDailyLossUSD:      c.RiskMaxDailyLossUSD,
		MaxConcurrentTrades:  c.RiskMaxConcurrentTrades,
		MaxNotionalUSD:       c.RiskMaxNotionalUSD,
		MaxContractsPerOrder: c.RiskMaxContractsPerOrder,
	}
}

// ============================================================================
// Daily PnL reset
// ============================================================================
//
// The MaxDailyLossUSD limit is evaluated against ctx.Account.TotalPnL, which
// is a session-cumulative figure tracked by the trader. A true "daily" reset
// requires either:
//   (a) the trader to reset its TotalPnL at the CME session boundary
//       (Sunday 18:00 ET); or
//   (b) the engine to remember the start-of-day equity and compute
//       (current_equity - day_open_equity) on each cycle.
//
// For Plan 3 v1 we ship a SIMPLE day-boundary tracker here so the API
// endpoint and tests have something to call. It DOES NOT yet hook into
// CME session boundaries (deferred to a follow-up alongside T22 drift).
//
// Manual reset path: ResetDailyPnL() — called by the operator via the
// (future) force-flat API endpoint after they have wired in a fresh
// session.
//
// Automatic check-and-reset path: MaybeResetDaily(now) — called at the
// top of each decision cycle. If the wall-clock date has changed since
// last reset, the tracker zeroes itself. This is a coarse approximation
// of session boundaries (UTC date rollover, not 18:00 ET) and is
// adequate for SIM-mode paper trading; live trading should switch to
// CME-aware reset by checking IsCMEOpen + transition detection.

var (
	dailyResetMu       sync.Mutex
	lastDailyResetDate string // YYYY-MM-DD in UTC
)

// ResetDailyPnL marks "now" as the new day-start. Cheap, idempotent.
// Operators call this from the force-flat API endpoint or at the
// CME Sunday 18:00 ET open.
func ResetDailyPnL() {
	dailyResetMu.Lock()
	defer dailyResetMu.Unlock()
	lastDailyResetDate = CMESessionDayKey(time.Now())
	logger.Infof("Plan 3 T21: daily window manually reset to CME session-day %s", lastDailyResetDate)
}

// MaybeResetDaily checks for a CME session-day rollover (17:00 CT boundary)
// since the last reset and resets the daily window when it has. Returns true if
// a reset fired. Strategy Studio P1 fix: the reset key is derived from the
// PASSED `now` (via CMESessionDayKey), not time.Now() — so the window is
// deterministic + testable and rolls on the real session boundary, not UTC
// midnight. (Previously ResetDailyPnL stamped time.Now() while the check used
// the passed `now`, so they never agreed.)
func MaybeResetDaily(now time.Time) bool {
	today := CMESessionDayKey(now)
	dailyResetMu.Lock()
	defer dailyResetMu.Unlock()
	if lastDailyResetDate == "" || lastDailyResetDate != today {
		lastDailyResetDate = today
		logger.Infof("Plan 3 T21 / Strategy Studio: daily window reset to CME session-day %s", today)
		return true
	}
	return false
}

// ============================================================================
// Strategy Studio Phase 1 — per-strategy daily guardrails (evolves the gate).
// ============================================================================

// DailyGuardrails carries the RESOLVED per-strategy daily guardrail config (the
// caller resolves per-strategy value → env fallback + the master/per-guardrail
// toggles) plus the live daily-window state. Pure primitives so it is trivially
// unit-testable, like CheckPreTrade.
type DailyGuardrails struct {
	MasterEnabled bool // master switch; false → ALL guardrails bypassed (caller logs)

	DailyRealizedPnL float64 // realized P&L on the CME session-day (negative = net loss)
	TradesToday      int     // entries taken on the CME session-day

	DailyLossEnabled      bool
	DailyLossLimitUSD     float64 // positive USD; loss ≥ this trips (force-flat class)
	DailyProfitEnabled    bool
	DailyProfitTargetUSD  float64 // positive USD; profit ≥ this trips (block-entry)
	MaxDailyTradesEnabled bool
	MaxDailyTrades        int
}

// Check evaluates the ENABLED daily guardrails. Master OFF → always allow (the
// caller logs the bypass). Each guardrail is evaluated only when its toggle is
// ON AND its value is configured (>0); a tripped guardrail returns a non-nil err
// with a clear reason. Daily-loss → ForceFlat; profit-target / max-trades →
// BlockEntry. NB: this controls ONLY the prop-firm guardrails — it can never
// affect the SIM-only / live-account block (that is enforced separately in the
// broker layer and is untoggleable).
func (g DailyGuardrails) Check() (RiskLimitDecision, error) {
	if !g.MasterEnabled {
		return RiskAllow, nil
	}
	if g.DailyLossEnabled && g.DailyLossLimitUSD > 0 && g.DailyRealizedPnL <= -g.DailyLossLimitUSD {
		return RiskForceFlat, fmt.Errorf("daily loss limit hit (realized today=%.2f, limit=-%.2f)", g.DailyRealizedPnL, g.DailyLossLimitUSD)
	}
	if g.DailyProfitEnabled && g.DailyProfitTargetUSD > 0 && g.DailyRealizedPnL >= g.DailyProfitTargetUSD {
		return RiskBlockEntry, fmt.Errorf("daily profit target reached (realized today=%.2f, target=%.2f)", g.DailyRealizedPnL, g.DailyProfitTargetUSD)
	}
	if g.MaxDailyTradesEnabled && g.MaxDailyTrades > 0 && g.TradesToday >= g.MaxDailyTrades {
		return RiskBlockEntry, fmt.Errorf("max daily trades reached (today=%d, max=%d)", g.TradesToday, g.MaxDailyTrades)
	}
	return RiskAllow, nil
}

// boolOrDefault resolves a three-state *bool toggle: nil → the default value.
// Used so a guardrail's per-strategy "enabled" can be on/off/unset; unset
// inherits the guardrail's safe default (daily-loss ON, new guardrails OFF).
func boolOrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// firstPositive returns the first > 0 value — used for per-strategy value with
// an env fallback: firstPositive(perStrategyValue, envValue).
func firstPositive(vals ...float64) float64 {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	return 0
}

// ResolveMaxContracts (Chunk 3) returns the effective max-contracts clamp for a
// futures order, respecting the master switch + the per-guardrail toggle. Returns
// 0 = NO clamp (disabled by master/toggle). Per-strategy value overrides; def is
// the fallback (the prior 10-contract default).
func ResolveMaxContracts(masterEnabled, contractsEnabled *bool, perStrategy, def int) int {
	if !boolOrDefault(masterEnabled, true) || !boolOrDefault(contractsEnabled, true) {
		return 0
	}
	if perStrategy > 0 {
		return perStrategy
	}
	return def
}

// ResolveNotionalLeverage (Chunk 3) returns the effective futures notional-ceiling
// multiplier (max notional = equity × this), respecting the master + toggle.
// Returns 0 = NO cap (disabled by master/toggle). Per-strategy value overrides;
// def is the fallback (the prior equity×20 const).
func ResolveNotionalLeverage(masterEnabled, notionalEnabled *bool, perStrategy, def float64) float64 {
	if !boolOrDefault(masterEnabled, true) || !boolOrDefault(notionalEnabled, true) {
		return 0
	}
	if perStrategy > 0 {
		return perStrategy
	}
	return def
}

// ConsistencyBreached (Chunk 5) reports whether today's realized profit exceeds
// the configured share (pct%) of all-time total realized profit — the prop-firm
// consistency rule ("no single day > X% of total"). SEMANTICS (documented): it
// triggers ONLY once there is prior-day profit (totalProfit − todayProfit > 0),
// so a fresh/single-day account never self-locks on its first profitable day; a
// non-profitable day (todayProfit ≤ 0), a non-positive total, or pct ≤ 0 never
// breach. When breached, the caller blocks new entries for the session-day so
// today's profit stops growing past the allowed share.
func ConsistencyBreached(todayProfit, totalProfit, pct float64) bool {
	if pct <= 0 || todayProfit <= 0 || totalProfit <= 0 {
		return false
	}
	if totalProfit-todayProfit <= 0 {
		return false // first profitable day — nothing to be consistent against yet
	}
	return todayProfit >= (pct/100.0)*totalProfit
}
