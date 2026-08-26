// P4.3 — the rules block: no-trade windows + the plan-death line, in a
// short-tinted container (this is where the plan STOPS trading).

import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'

export function RulesBlock({
  noTrade,
  deathCondition,
  language,
}: {
  noTrade: string[]
  deathCondition: string
  language: Language
}) {
  const hasNoTrade = noTrade && noTrade.length > 0
  if (!hasNoTrade && !deathCondition) return null
  return (
    <div
      className="flex flex-col gap-1.5 p-2.5"
      style={{
        background: 'rgba(224,108,108,0.06)',
        border: '1px solid rgba(224,108,108,0.22)',
        borderRadius: 'var(--vl-radius-inner)',
      }}
    >
      {hasNoTrade && (
        <div
          className="text-[11px]"
          style={{ fontFamily: 'var(--vl-font-ui)' }}
        >
          <span
            className="uppercase tracking-wide"
            style={{ color: 'var(--vl-short)' }}
          >
            {tp('noTrade', language)}
          </span>
          <span style={{ color: 'var(--vl-muted)' }}>
            {' '}
            · {noTrade.join(' · ')}
          </span>
        </div>
      )}
      {deathCondition && (
        <div
          className="text-[11px]"
          style={{ fontFamily: 'var(--vl-font-ui)' }}
        >
          <span
            className="uppercase tracking-wide"
            style={{ color: 'var(--vl-short)' }}
          >
            {tp('planDies', language)}
          </span>
          <span style={{ color: 'var(--vl-muted)' }}> · {deathCondition}</span>
        </div>
      )}
    </div>
  )
}
