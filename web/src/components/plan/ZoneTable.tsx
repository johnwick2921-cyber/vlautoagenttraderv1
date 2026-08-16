// P4.3 / P5.2-3 — the level table (one of the three renderers of the SAME level
// array). Columns: price · provenance · grade · fresh · instruction · distance,
// plus 👤 (owner) and 📝 (note) markers. A near level turns its distance gold; a
// consumed level dims 50% (audit trail). P5: tapping a row opens the edit sheet;
// an owner/AI conflict ghosts the AI row + shows the ⚡ conflict chip (owner wins).

import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { PlanLevelFact } from '../../lib/api/plan'
import { GradeChip, ProvenanceChip, FreshDot, ConflictChip } from './chips'
import {
  levelFresh,
  levelNear,
  fmtDistance,
  fmtPrice,
  detectConflicts,
} from './levelState'

function ZoneRow({
  fact,
  index,
  language,
  onEdit,
  ghosted,
  flagged,
}: {
  fact: PlanLevelFact
  index: number
  language: Language
  onEdit?: (fact: PlanLevelFact, index: number) => void
  ghosted?: boolean
  flagged?: boolean
}) {
  const fresh = levelFresh(fact)
  const near = levelNear(fact)
  const consumed = fresh === 'consumed'
  const isOwner = fact.origin === 'OWNER'
  const priceWords = `${fmtPrice(fact.price)}` // announced value
  const editable = !!onEdit

  return (
    <div
      role="row"
      aria-label={`${fact.label} ${priceWords}, grade ${fact.grade}, ${fresh}, ${fact.instruction}, ${fmtDistance(fact.distance)} points`}
      onClick={editable ? () => onEdit!(fact, index) : undefined}
      className="grid items-center gap-2 py-1.5"
      style={{
        gridTemplateColumns: 'minmax(64px,auto) auto auto 1fr auto',
        opacity: ghosted ? 0.35 : consumed ? 0.5 : 1,
        borderBottom: '1px solid var(--vl-hair)',
        cursor: editable ? 'pointer' : undefined,
      }}
    >
      {/* price + owner/note markers */}
      <div className="flex items-center gap-1">
        <span
          className="vl-num text-[13px]"
          style={{
            color: 'var(--vl-ivory)',
            textDecoration: ghosted ? 'line-through' : undefined,
          }}
        >
          {fmtPrice(fact.price)}
        </span>
        {isOwner && (
          <span
            title={tp('ownerLevel', language)}
            aria-label={tp('ownerLevel', language)}
            style={{ fontSize: 11 }}
          >
            👤
          </span>
        )}
        {fact.note && (
          <span
            title={fact.note}
            aria-label={tp('hasNote', language)}
            style={{ fontSize: 11 }}
          >
            📝
          </span>
        )}
        {flagged && <ConflictChip language={language} />}
      </div>

      <ProvenanceChip label={fact.label} />
      <GradeChip grade={fact.grade} />
      <FreshDot fresh={fresh} language={language} />

      <div className="flex items-center justify-end gap-2">
        {fact.scenario_id && (
          <span
            className="text-[10px]"
            style={{
              color: 'var(--vl-gold)',
              fontFamily: 'var(--vl-font-data)',
            }}
          >
            {fact.scenario_id}
          </span>
        )}
        <span
          className="text-[11px] truncate max-w-[120px]"
          style={{ color: 'var(--vl-muted)', fontFamily: 'var(--vl-font-ui)' }}
          title={fact.instruction}
        >
          {fact.instruction}
        </span>
        <span
          className="vl-num text-[12px]"
          style={{
            color: near ? 'var(--vl-gold)' : 'var(--vl-faint)',
            fontWeight: near ? 700 : 400,
          }}
        >
          {fmtDistance(fact.distance)}
        </span>
      </div>
    </div>
  )
}

export function ZoneTable({
  facts,
  language,
  onEdit,
  onAdd,
  onBulkAdd,
  flashKey = 0,
  emptyReason,
}: {
  facts: PlanLevelFact[]
  language: Language
  onEdit?: (fact: PlanLevelFact, index: number) => void
  onAdd?: () => void
  onBulkAdd?: () => void
  /** W13 — bump to flash the rows gold once after an applied re-align. */
  flashKey?: number
  /**
   * FAIL LOUD (P0 2026-08-17): why this plan has no levels. A bare "No levels in
   * this plan" is indistinguishable from a rendering bug — it read as one for a
   * whole session while the planner was in fact killing every version on arrival.
   * The card ALWAYS supplies this; an unexplained emptiness says so explicitly.
   */
  emptyReason?: string
}) {
  const { ghosted, flagged } = detectConflicts(facts)
  return (
    <div role="table" aria-label={tp('keyLevels', language)}>
      <div className="flex items-center justify-between mb-1">
        <span
          className="text-[10px] uppercase tracking-widest"
          style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
        >
          {tp('keyLevels', language)}
        </span>
        <div className="flex items-center gap-2">
          {onAdd && (
            <button
              onClick={onAdd}
              className="text-[10px]"
              style={{
                color: 'var(--vl-gold)',
                fontFamily: 'var(--vl-font-ui)',
              }}
            >
              ＋ {tp('addLevel', language)}
            </button>
          )}
          {onBulkAdd && (
            <button
              onClick={onBulkAdd}
              className="text-[10px]"
              style={{
                color: 'var(--vl-muted)',
                fontFamily: 'var(--vl-font-ui)',
              }}
            >
              {tp('bulkAdd', language)}
            </button>
          )}
          <span
            className="text-[10px] uppercase"
            style={{
              color: 'var(--vl-faint)',
              fontFamily: 'var(--vl-font-ui)',
            }}
          >
            {tp('colDistance', language)}
          </span>
        </div>
      </div>
      {facts.length === 0 ? (
        <div
          data-testid="zone-table-empty"
          className="flex flex-col gap-1 py-3"
          style={{ fontFamily: 'var(--vl-font-ui)' }}
        >
          <span className="text-[12px]" style={{ color: 'var(--vl-faint)' }}>
            {tp('noLevels', language)}
          </span>
          <span
            data-testid="zone-table-empty-reason"
            className="text-[11px]"
            style={{ color: 'var(--vl-short)' }}
          >
            {tp('noLevelsWhy', language)}:{' '}
            {emptyReason || tp('noLevelsUnknown', language)}
          </span>
        </div>
      ) : (
        facts.map((f, i) => (
          <ZoneRow
            key={`${f.label}-${f.price}-${i}-${flashKey}`}
            data-flash={flashKey > 0 ? '1' : undefined}
            fact={f}
            index={i}
            language={language}
            onEdit={onEdit}
            ghosted={ghosted.has(i)}
            flagged={flagged.has(i)}
          />
        ))
      )}
    </div>
  )
}
