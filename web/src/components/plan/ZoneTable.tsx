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

// T4 (2026-08-26) — live touch chip per level row. Telemetry only.
const TOUCH_CHIPS: Record<string, { glyph: string; color: string }> = {
  approaching: { glyph: '○', color: 'var(--vl-muted)' },
  touching: { glyph: '◐', color: 'var(--vl-gold)' },
  rejected: { glyph: '✕', color: '#34d399' },
  accepted: { glyph: '▲', color: '#f87171' },
}

// TouchChip (guide-export 2026-08-27) — the real touch-state glyph chip.
export function TouchChip({ state }: { state: string }) {
  const c = TOUCH_CHIPS[state]
  if (!c) return null
  return (
    <span
      aria-label={`touch ${state}`}
      title={`touch ${state}`}
      className="text-[10px] leading-none"
      style={{ color: c.color, fontFamily: 'var(--vl-font-ui)' }}
    >
      {c.glyph}
    </span>
  )
}

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
  const touchChip = fact.touch_state ? TOUCH_CHIPS[fact.touch_state] : undefined
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
      {/* Machine grade (8.4): the deterministic detector-side grade stamped at
          plan write. Shown beside the model's when it differs — a model A next
          to a machine C is visible at a glance. */}
      {fact.machine_grade && fact.machine_grade !== fact.grade && (
        <span
          className="inline-flex items-center gap-0.5 text-[9px] font-mono"
          style={{ color: 'var(--vl-faint)' }}
          title="machine grade — detector-side (type × freshness × confluence × HTF)"
        >
          m:{fact.machine_grade}
        </span>
      )}
      {/* W3 (2026-09-09) — the merged map, matched to this level by price.
          ONE wrapper on purpose: the row grid declares five tracks at :73, so
          these ride in a single cell instead of overflowing it. Every badge is
          absent-renders-nothing — an unmatched level looks exactly as before. */}
      <div className="flex items-center gap-1">
        {fact.names && fact.names.length > 1 && (
          <span
            className="inline-flex items-center text-[9px] font-mono"
            style={{ color: 'var(--vl-faint)' }}
            title={`one price, ${fact.names.length} references — merged: ${fact.names.join(' · ')}`}
            data-testid="level-merged-names"
          >
            {fact.names.join(' · ')}
          </span>
        )}
        {fact.map_role && (
          <span
            className="inline-flex items-center text-[9px] font-mono"
            style={{ color: 'var(--vl-faint)' }}
            title={
              fact.entry_candidate === false && fact.not_entry_reason
                ? `not an entry — ${fact.not_entry_reason}`
                : 'what this reference is FOR in this read'
            }
            data-testid="level-map-role"
          >
            {fact.map_role}
          </span>
        )}
        {fact.projection && (
          <span
            className="inline-flex items-center text-[9px] font-mono"
            style={{ color: 'var(--vl-faint)' }}
            title={`projection — ${fact.projection_method ?? 'method not recorded'} · never an entry`}
            data-testid="level-projection"
          >
            projection
          </span>
        )}
        {typeof fact.distance_atr === 'number' && (
          <span
            className="inline-flex items-center text-[9px] font-mono"
            style={{ color: 'var(--vl-faint)' }}
            title="distance from price in ATR5m"
            data-testid="level-distance-atr"
          >
            {Math.abs(fact.distance_atr).toFixed(1)}·ATR
          </span>
        )}
      </div>
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
        {/* T4 (2026-08-26) — live touch chip: ○ approaching · ◐ touching · ✕
            rejected · ▲ accepted. Telemetry only — zero order authority. */}
        {touchChip && (
          <span
            title={`touch: ${fact.touch_state}`}
            className="text-[11px]"
            style={{ color: touchChip.color }}
          >
            {touchChip.glyph}
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
