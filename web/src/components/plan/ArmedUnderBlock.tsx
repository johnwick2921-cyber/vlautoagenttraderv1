// INVALIDATION-WIRED F2 (owner ruling 2026-09-03) — a position's provenance,
// never the live plan's.
//
// On 2026-09-03 this card rendered NY v3 S1 long, written 09:15, while the
// account held a position armed under v2 S1 short — filled 09:03, stopped
// 09:20 for −$140. Both scenarios are called "S1". The owner read the one on
// screen and was holding the other.
//
// So when a position is open and it was entered under a DIFFERENT plan version
// than the one displayed, this block states both, armed-under first. It renders
// nothing when flat, and nothing when the versions agree — there is nothing to
// disambiguate then.

import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'

export type OpenPositionProvenance = {
  symbol?: string
  side?: string
  entry_price?: number
  quantity?: number
  cited_scenario_id?: string
  armed_under_version?: number | null
  armed_under_note?: string
  live_plan_version?: number
  version_differs?: boolean
}

export function ArmedUnderBlock({
  position,
  language,
}: {
  position?: OpenPositionProvenance | null
  language: Language
}) {
  if (!position || !position.version_differs) return null

  const side = (position.side ?? '').toLowerCase()
  const scenario = position.cited_scenario_id || '—'
  // A null version means the column was never stamped (rows written before the
  // attribution wave booted). Say that, never "v0".
  const armedUnder =
    typeof position.armed_under_version === 'number'
      ? `v${position.armed_under_version}`
      : position.armed_under_note || 'version not recorded'

  return (
    <div
      className="flex flex-col gap-1 p-2.5"
      style={{
        background: 'rgba(224,180,108,0.07)',
        border: '1px solid rgba(224,180,108,0.28)',
        borderRadius: 'var(--vl-radius-inner)',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      <div className="text-[11px]">
        <span
          className="uppercase tracking-wide"
          style={{ color: 'var(--vl-warn, #e0b46c)' }}
        >
          {tp('armedUnder', language)}
        </span>
        <span style={{ color: 'var(--vl-text)' }}>
          {' '}
          · {armedUnder} {scenario} {side}
          {typeof position.entry_price === 'number' && position.entry_price > 0
            ? ` @ ${position.entry_price.toFixed(2)}`
            : ''}
          {typeof position.quantity === 'number' && position.quantity > 0
            ? ` × ${position.quantity}`
            : ''}
        </span>
      </div>
      <div className="text-[11px]" style={{ color: 'var(--vl-muted)' }}>
        <span className="uppercase tracking-wide">
          {tp('planNow', language)}
        </span>
        <span>
          {' '}
          · v{position.live_plan_version ?? '—'} — the rows below are THIS plan,
          not the position above
        </span>
      </div>
    </div>
  )
}
