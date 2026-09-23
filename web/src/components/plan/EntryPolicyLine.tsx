import type { PlanOrderLeg } from '../../lib/api/plan'

// W-EXEC-TRUTH W3 (h) — the plan card's entry line for a market_in_zone leg:
//
//   Entry: around 31,005 (zone 31,000–31,010) · Waiting for price
//
// Everything is READ from the ledger row the plan API serves
// (api/handler_plan_order_truth.go); the card never computes a verdict. A
// legacy or planned_order leg renders NOTHING. A missing number reads "n/a",
// never 0.

export const ENTRY_POLICY_MARKET_IN_ZONE = 'market_in_zone'

const isNum = (v: unknown): v is number =>
  typeof v === 'number' && Number.isFinite(v)

// A price: en-US, up to 2 decimals (31,005.25). 0 or below is no price.
const price = (v: number | null | undefined): string =>
  isNum(v) && v > 0
    ? v.toLocaleString('en-US', { maximumFractionDigits: 2 })
    : 'n/a'

// Slippage is a measurement whose 0 is real; the sign is kept (+ = worse).
const ticks = (v: number): string =>
  `${v.toLocaleString('en-US', { maximumFractionDigits: 2 })} ${Math.abs(v) === 1 ? 'tick' : 'ticks'}`

const ct = (ms: number | undefined): string =>
  isNum(ms) && ms > 0
    ? new Date(ms).toLocaleTimeString('en-US', {
        timeZone: 'America/Chicago',
        hour12: false,
      }) + ' CT'
    : 'n/a'

// The executor's verdict vocabulary (W3 (d)/(e)): the zone codes
// inside | beyond | short_of_zone | unknown, and the pass verdicts
// "waiting: …" / "refused: …". No verdict yet = waiting.
type VerdictKind = 'waiting' | 'refused' | 'other'
function verdictKind(verdict?: string): VerdictKind {
  const v = (verdict ?? '').trim().toLowerCase()
  if (!v) return 'waiting'
  if (v.startsWith('refused') || v.startsWith('blocked')) return 'refused'
  if (
    v.startsWith('waiting') ||
    v.startsWith('short_of_zone') ||
    v.startsWith('unknown')
  )
    return 'waiting'
  return 'other'
}

function refusalReason(verdict: string): string {
  const t = verdict.trim()
  const m = /^(refused|blocked)\s*:?\s*/i.exec(t)
  return (m ? t.slice(m[0].length) : t).trim() || 'UNKNOWN'
}

/** The status half of the line, from the row's state (+ verdict while armed). */
export function entryPolicyStatus(leg: PlanOrderLeg): string {
  switch (leg.state) {
    case 'armed': {
      const kind = verdictKind(leg.verdict)
      if (kind === 'waiting') return 'Waiting for price'
      if (kind === 'refused') return `Blocked: ${refusalReason(leg.verdict!)}`
      // A verdict the card has no word for is shown verbatim, not guessed at.
      return `Armed: ${leg.verdict!.trim()}`
    }
    case 'place_pending':
    case 'working':
      return `Placed at ${price(leg.composed?.entry)}`
    case 'filled': {
      const px =
        isNum(leg.fill_price) && leg.fill_price > 0
          ? price(leg.fill_price)
          : 'UNKNOWN'
      const slip = isNum(leg.fill_slippage_ticks)
        ? ` (${ticks(leg.fill_slippage_ticks)})`
        : ''
      return `Filled ${px}${slip}`
    }
    case 'cancelled':
    case 'rejected':
      return `Blocked: ${leg.reason?.trim() || 'UNKNOWN'}`
    case 'cancel_pending':
      return `Cancel pending${leg.reason ? `: ${leg.reason}` : ''}`
  }
  return `Order ${leg.state || 'UNKNOWN'}${leg.reason ? `: ${leg.reason}` : ''}`
}

/** The whole line, or null for a leg that is not market_in_zone (legacy). */
export function entryPolicyLine(leg: PlanOrderLeg): string | null {
  if (leg.policy !== ENTRY_POLICY_MARKET_IN_ZONE) return null
  return `Entry: around ${price(leg.planned_entry)} (zone ${price(leg.zone_lo)}–${price(leg.zone_hi)}) · ${entryPolicyStatus(leg)}`
}

// The receipt behind the line, for the hover title.
function entryPolicyDetail(leg: PlanOrderLeg): string {
  const parts = [
    `verdict: ${leg.verdict?.trim() || 'none recorded'}`,
    `read ${price(leg.eval_price)} on the ${ct(leg.eval_bar_ms)} bar`,
    `limit ${price(leg.composed?.entry)} placed ${ct(leg.placed_at_ms)}`,
  ]
  if (leg.state === 'filled')
    parts.push(
      `filled ${ct(leg.filled_at_ms)}`,
      'ticks = fill vs the limit sent (+ worse, − better)'
    )
  return parts.join(' · ')
}

export function EntryPolicyLine({ legs }: { legs?: PlanOrderLeg[] }) {
  const lines = (legs ?? []).flatMap((leg) => {
    const text = entryPolicyLine(leg)
    return text ? [{ leg, text }] : []
  })
  if (!lines.length) return null
  const split = (legs?.length ?? 0) >= 2
  return (
    <>
      {lines.map(({ leg, text }) => (
        <span
          key={leg.leg_index}
          data-testid={`entry-policy-line-${leg.leg_index}`}
          className="text-[10px]"
          title={entryPolicyDetail(leg)}
          style={{ color: 'var(--vl-muted)', fontFamily: 'var(--vl-font-ui)' }}
        >
          {split ? `L${leg.leg_index + 1} · ` : ''}
          {text}
        </span>
      ))}
    </>
  )
}
