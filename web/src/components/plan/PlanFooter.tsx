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
  latestVersion,
  versionCount,
  onSelectVersion,
  titleForVersion,
}: {
  version: number
  dayType?: string
  modelId?: string
  replansLeft?: number
  language: Language
  // ITEM 1 (2026-08-17): the card renders TWO chip rows — the header and this
  // one. Only the header was wired when click-to-open shipped, so tapping the
  // FOOTER chips did nothing, which is exactly what "still not clickable" looks
  // like from the owner's chair. Both rows now share one handler.
  latestVersion?: number
  versionCount?: number
  onSelectVersion?: (version: number) => void
  titleForVersion?: (version: number) => string | undefined
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
        <VersionChips
          version={version}
          latest={latestVersion}
          count={versionCount}
          onSelect={onSelectVersion}
          titleFor={titleForVersion}
        />
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
