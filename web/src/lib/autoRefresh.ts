// Shared auto-refresh layer (2026-06-10) — every display surface updates
// itself; no more manual F5.
//
// THREE pieces, matching the app's existing architecture instead of
// replacing it:
//
// 1. Named interval constants — every polling surface reads its cadence from
//    here (one tuning point).
// 2. usePollResume — replaces the permanent "pollOff" latch in the dashboard
//    route wrappers. The old latch set refreshInterval to 0 after 2 failed
//    retries and could ONLY be cleared by onSuccess — which never fires once
//    polling is off (revalidateOnFocus is false), so any transient backend
//    blip (every bot restart!) froze the surface until a full F5 or a trader
//    switch. THAT was the "owner F5s constantly" root cause. The replacement
//    keeps the loud failed-state UI but auto-resumes polling after
//    REFRESH_BACKOFF_RESUME_MS.
// 3. useAutoRefresh — for components that poll with raw setInterval (or used
//    to fetch once on mount): skip-if-in-flight (no request pileup), pause
//    while document.hidden (Page Visibility API) with an immediate refresh on
//    return, exponential error backoff (unreachable backend is not hammered),
//    cleanup on unmount.
//
// SWR-based surfaces deliberately STAY on SWR: its refreshInterval already
// dedupes concurrent requests and refreshWhenHidden defaults to false (polls
// pause on hidden tabs). The constants below feed those configs too.
//
// NO-CLOBBER RULE: editors/forms (Strategy Studio editors, Risk Control
// fields, Exchange config, Create Trader modal) are NEVER wired to this
// layer — a background refetch must not overwrite uncommitted user input.

import { useEffect, useRef, useState } from 'react'

// ───────────────────────── interval constants (ms) ─────────────────────────

/** Trader list (Traders page). */
export const REFRESH_TRADERS_MS = 5_000
/** Trader list on the dashboard (drives trader switcher). */
export const REFRESH_TRADERS_DASH_MS = 10_000
/** Exchange configs (rarely change). */
export const REFRESH_EXCHANGES_MS = 60_000
/** Trader status (running state, cycle info). */
export const REFRESH_STATUS_MS = 5_000
/** Account balance / equity snapshot. */
export const REFRESH_ACCOUNT_MS = 5_000
/** Open positions. */
export const REFRESH_POSITIONS_MS = 5_000
/** Recent Decisions list. */
export const REFRESH_DECISIONS_MS = 5_000
/** Trading statistics (win rate etc.). */
export const REFRESH_STATS_MS = 30_000
/** Kline chart bars (matches the NT8 bar cadence). */
export const REFRESH_CHART_MS = 5_000
/** Open-order price lines on the chart (exchange API — keep gentle). */
export const REFRESH_OPEN_ORDERS_MS = 60_000
/** Equity curve / P&L history. */
export const REFRESH_EQUITY_MS = 10_000
/** Trade/position HISTORY + logs/activity feeds (long lists — gentle). */
export const REFRESH_HISTORY_MS = 10_000
/** Market ticker strip. */
export const REFRESH_TICKER_MS = 15_000
/** How long a failed surface backs off before polling resumes on its own. */
export const REFRESH_BACKOFF_RESUME_MS = 30_000

// ───────────────────────── usePollResume ─────────────────────────

/**
 * Drop-in replacement for the dashboard's permanent pollOff latch: `latch()`
 * marks the surface failed (UI shows the failed state, polling pauses), and a
 * timer AUTO-RESUMES polling after `resumeMs` — so a transient backend outage
 * (bot restart/deploy) self-heals instead of freezing the page until F5.
 * `clear()` (call from onSuccess) cancels the failed state immediately.
 */
export function usePollResume(resumeMs: number = REFRESH_BACKOFF_RESUME_MS) {
  const [off, setOff] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const cancelTimer = () => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }

  const latch = () => {
    setOff(true)
    cancelTimer()
    timerRef.current = setTimeout(() => {
      timerRef.current = null
      setOff(false) // resume polling; a real outage just latches again
    }, resumeMs)
  }

  const clear = () => {
    cancelTimer()
    setOff(false)
  }

  // Cleanup on unmount.
  useEffect(() => cancelTimer, [])

  return { off, latch, clear }
}

// ───────────────────────── useAutoRefresh ─────────────────────────

/**
 * Polls `fn` every `intervalMs` with the safety rails the raw setInterval
 * pollers lacked:
 *  - skip-if-in-flight: a tick never starts while the previous call runs
 *  - pause on hidden tab; one immediate refresh when the tab returns
 *  - exponential error backoff (×2 per consecutive failure, capped ×8)
 *  - cleanup on unmount
 *
 * `fn` may be async; rejections count as errors for backoff. The latest `fn`
 * is always used (ref-captured) so callers don't need useCallback gymnastics.
 * Pass `enabled: false` to suspend entirely (e.g. while an editor is dirty).
 */
export function useAutoRefresh(
  fn: () => void | Promise<void>,
  intervalMs: number,
  opts: { enabled?: boolean; runOnMount?: boolean } = {}
) {
  const { enabled = true, runOnMount = false } = opts
  const fnRef = useRef(fn)
  fnRef.current = fn

  useEffect(() => {
    if (!enabled || intervalMs <= 0) return

    let disposed = false
    let inFlight = false
    let errorStreak = 0
    let timer: ReturnType<typeof setTimeout> | null = null

    const tick = async (fromVisibility = false) => {
      if (disposed || inFlight) return
      if (document.hidden && !fromVisibility) {
        schedule() // stay quiet while hidden; visibilitychange wakes us
        return
      }
      inFlight = true
      try {
        await fnRef.current()
        errorStreak = 0
      } catch {
        errorStreak = Math.min(errorStreak + 1, 3) // cap backoff at ×8
      } finally {
        inFlight = false
        if (!disposed) schedule()
      }
    }

    const schedule = () => {
      if (timer !== null) clearTimeout(timer)
      const delay = intervalMs * 2 ** errorStreak
      timer = setTimeout(() => void tick(), delay)
    }

    const onVisibility = () => {
      // Tab came back — refresh immediately instead of waiting a full tick.
      if (!document.hidden) void tick(true)
    }
    document.addEventListener('visibilitychange', onVisibility)

    if (runOnMount) void tick()
    else schedule()

    return () => {
      disposed = true
      if (timer !== null) clearTimeout(timer)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [enabled, intervalMs])
}
