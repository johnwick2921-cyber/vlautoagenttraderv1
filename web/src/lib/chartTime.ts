// THE ONE chart-timezone site (chart-timestamp dispatch, 2026-08-19).
//
// lightweight-charts does NO timezone conversion — axis and crosshair labels
// render the raw epoch in UTC. The owner contract pins EVERY viewer to Houston
// (America/Chicago), and NT8's own chart labels in that zone, so every chart
// surface must format through THESE helpers. No component may do its own
// timezone math on bar times; the conversion is Intl-rule-based (DST-safe: a
// January bar renders CST −6, an August bar CDT −5 — never a fixed constant).
//
// Extracted from AdvancedChart (which already carried the correct formatters)
// so PlanMiniChart — which used to hand the lib raw epochs and therefore
// showed UTC labels, +5h vs NT8 — shares the identical single implementation.
// (TradingViewChart is the one exception: the external widget takes
// timezone:'America/Chicago' natively.)

import { TickMarkType, type Time } from 'lightweight-charts'

export const CHART_TZ = 'America/Chicago'

function timeToMs(time: Time): number {
  if (typeof time === 'number') return time * 1000
  if (typeof time === 'string') return new Date(time).getTime() // "YYYY-MM-DD"
  return Date.UTC(time.year, time.month - 1, time.day) // BusinessDay
}

/** X-axis tick labels in CT, respecting the granularity the lib asks for. */
export function ctTickMarkFormatter(
  time: Time,
  tickMarkType: TickMarkType
): string {
  const d = new Date(timeToMs(time))
  switch (tickMarkType) {
    case TickMarkType.Year:
      return d.toLocaleString('zh-CN', { year: 'numeric', timeZone: CHART_TZ })
    case TickMarkType.Month:
      return d.toLocaleString('zh-CN', { month: 'short', timeZone: CHART_TZ })
    case TickMarkType.DayOfMonth:
      return d.toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        timeZone: CHART_TZ,
      })
    default: // Time / TimeWithSeconds — intraday ticks
      return d.toLocaleString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
        timeZone: CHART_TZ,
      })
  }
}

/** Crosshair time label — same zone as the axis, so the two always agree. */
export function ctCrosshairTimeFormatter(time: number): string {
  return new Date(time * 1000).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: CHART_TZ,
  })
}

/** Dev-only diagnosis (dispatch 2.3): raw epoch, UTC render, CT render of the
 *  newest bar — the next mismatch becomes a one-glance read. */
export function logBarDebug(tag: string, epochMs: number | undefined): void {
  if (!import.meta.env.DEV || !epochMs) return
  const utc = new Date(epochMs).toISOString()
  const ct = new Intl.DateTimeFormat('en-US', {
    timeZone: CHART_TZ,
    dateStyle: 'short',
    timeStyle: 'medium',
    hour12: false,
  }).format(epochMs)

  console.debug(`[chartTime] ${tag} raw=${epochMs} utc=${utc} ct=${ct} CT`)
}
