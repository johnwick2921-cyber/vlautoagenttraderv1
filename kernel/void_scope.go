package kernel

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"nofx/market"
)

// ── VOID SCOPE (2026-09-02) — ONE RESOLVED VIEW OF THE TAPE ───────────────────
//
// The prompt's VOID list and the write-site validator answer the SAME question
// ("has a close come back across this level?") and before this file they asked
// it of different tape:
//
//	render   auto_trader_planner.go:2304 — sinceMs = CMESessionDayStart · 12,000 bars
//	validate breakdown_continue.go:248   — sinceMs = 0 (whole slice)    ·  2,000 bars
//
// So a level broken and reclaimed BEFORE the 17:00 CT boundary was void to the
// validator and invisible to the prompt. That is precisely ONL 29141.25 on the
// 2026-09-02 20:58 CT read: eight seated levels listed VOID, ONL absent, and the
// read rejected on ONL. An overnight low is broken overnight, by definition.
//
// The class-45 parity fixture passed 40/40 across 20 tapes because it fed BOTH
// sides the same sinceMs. It pinned the two FUNCTIONS and never the CALL SITES.
//
// RULE: neither caller chooses a window or a bar slice. The resolver decides
// once, and the VALIDATOR'S scope wins — the prompt must list what the validator
// will reject, so the prompt reads the validator's tape, not its own.
type VoidScope struct {
	Bars     []market.Kline
	SinceMs  int64 // CME session-day start (clamped to the tape's own start)
	Interval string
	BarCount int
	Source   string // "provider" | "given" — how Bars were obtained
	// Horizon (BARS HORIZON 2026-09-09) is what the served slice ACTUALLY
	// covers: span, oldest age, and open-market intervals missing inside it.
	// BarCount is what was ASKED for; len(Bars) is what came back; neither says
	// whether the tape between them was continuous. On 2026-09-09 13:18 CT this
	// read was served 2000 of 2000 across a 3055-minute window with 696 missing
	// open-market minutes — Horizon.GapCount is the only field that says so.
	Horizon BarHorizon
}

// VoidScopeBarCount is the resolved bar depth both sides read. Default
// AISVPBarCount (2,000) — the VALIDATOR's historical depth, deliberately not
// the render path's 12,000: matching the validator is the whole point, and a
// deeper slice would list levels the validator cannot see.
func VoidScopeBarCount() int {
	if v := os.Getenv("VOID_SCOPE_BARS"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			return n
		}
	}
	return AISVPBarCount
}

// VoidScopeInterval is the bar interval both sides read.
func VoidScopeInterval() string { return AISVPBarInterval }

// VoidScopeSinceMs is the window start BOTH sides use: the CME session day.
//
// OWNER RULING 2026-09-02. The first cut of this wave made both sides read the
// WHOLE slice, because that was the validator's historical behaviour. Measured
// on the real tape that voids nearly every ranked level — 20 entries across 12
// levels, both sides of almost each — and a list that says "author no waterfall
// play anywhere" is noise by construction, which would push the model off
// legitimate plays. The deleted render-side comment was RIGHT ("a level broken
// and reclaimed days ago is not today's news"); its error was being applied to
// ONE caller. So the VALIDATOR narrows to the session day as well. That is a
// validator-rule change in the permitted direction — strictly FEWER rejects —
// and the planner_read_facts rows measure its effect.
//
// Clamped to the tape's own start so a short slice never yields an empty window.
func VoidScopeSinceMs(bars []market.Kline, now time.Time) int64 {
	start := CMESessionDayStart(now).UnixMilli()
	if len(bars) > 0 && bars[0].OpenTime > start {
		return bars[0].OpenTime
	}
	return start
}

// ResolveVoidScope fetches the ONE tape both sides read. Nil provider or no
// bars → an empty scope, which renders no VOID list and voids nothing: the
// honest degradation, never a fabricated verdict.
func ResolveVoidScope(symbol string, now time.Time) VoidScope {
	sc := VoidScope{Interval: VoidScopeInterval(), BarCount: VoidScopeBarCount(), Source: "provider"}
	if market.FuturesBarsProvider != nil {
		sc.Bars = market.FuturesBarsProvider(symbol, sc.Interval, sc.BarCount)
	}
	sc.SinceMs = VoidScopeSinceMs(sc.Bars, now)
	// The SAME `now` the caller passed — no second clock (A28), so the recorded
	// age is exact rather than approximately-now.
	sc.Horizon = HorizonOf(sc.Bars, sc.Interval, sc.BarCount, now)
	sc.Horizon.Who = "void"
	return sc
}

// VoidScopeOf wraps an already-fetched slice in the resolved window. Used by
// fixtures and by any caller that already holds the tape — the WINDOW still
// comes from the resolver, never from the caller.
func VoidScopeOf(bars []market.Kline, now time.Time) VoidScope {
	sc := VoidScope{Bars: bars, SinceMs: VoidScopeSinceMs(bars, now), Interval: VoidScopeInterval(), BarCount: VoidScopeBarCount(), Source: "given"}
	sc.Horizon = HorizonOf(sc.Bars, sc.Interval, sc.BarCount, now)
	sc.Horizon.Who = "void"
	return sc
}

// VoidScopeBootLine reports the resolved scope, every field READ from its
// resolver — never a literal.
func VoidScopeBootLine() string {
	return fmt.Sprintf("void scope: session-day window · %s×%d · one resolver for prompt AND validator (parity) · horizon: calendar-aware gap scan, cap=%d steps (a weekend is not a gap)",
		VoidScopeInterval(), VoidScopeBarCount(), barHorizonScanCap)
}

// ResolveVoidScopeOf wraps an already-held tape in the resolved window. Same
// contract as VoidScopeOf; named for symmetry with ResolveVoidScope at call
// sites that already have bars.
func ResolveVoidScopeOf(bars []market.Kline, now time.Time) VoidScope {
	return VoidScopeOf(bars, now)
}
