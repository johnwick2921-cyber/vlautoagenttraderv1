// P4.3 — the plan footer: version chips · day-type · model · re-plans-left.

import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import { VersionChips } from './chips'

export function PlanFooter({
  version,
  dayType,
  modelId,
  replansLeft,
  language,
}: {
  version: number
  dayType?: string
  modelId?: string
  replansLeft?: number
  language: Language
}) {
  return (
    <div
      className="flex flex-wrap items-center justify-between gap-2 pt-2 text-[10px]"
      style={{
        borderTop: '1px solid var(--vl-hair)',
        color: 'var(--vl-faint)',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      <div className="flex items-center gap-3">
        <VersionChips version={version} />
        {dayType && (
          <span>
            {tp('dayType', language)}:{' '}
            <span style={{ color: 'var(--vl-muted)' }}>{dayType}</span>
          </span>
        )}
      </div>
      <div className="flex items-center gap-3">
        {typeof replansLeft === 'number' && (
          <span>{tp('rereadsLeft', language, { n: replansLeft })}</span>
        )}
        {modelId && (
          <span>
            {tp('model', language)}:{' '}
            <span className="vl-num" style={{ color: 'var(--vl-muted)' }}>
              {modelId}
            </span>
          </span>
        )}
      </div>
    </div>
  )
}
