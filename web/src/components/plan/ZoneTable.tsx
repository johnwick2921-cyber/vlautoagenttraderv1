// P4.3 — the level table (one of the three renderers of the SAME level array).
// Columns: price · provenance · grade · fresh · instruction · distance, plus
// 👤 (owner) and 📝 (note) markers when the owner-overlay fields are present.
// A "near" level (|distance| < threshold) turns its distance chip gold. A
// "consumed" level dims 50% but stays legible (audit trail — never deleted).

import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { PlanLevelFact } from '../../lib/api/plan'
import { GradeChip, ProvenanceChip, FreshDot } from './chips'
import { levelFresh, levelNear, fmtDistance, fmtPrice } from './levelState'

function ZoneRow({
  fact,
  language,
}: {
  fact: PlanLevelFact
  language: Language
}) {
  const fresh = levelFresh(fact)
  const near = levelNear(fact)
  const consumed = fresh === 'consumed'
  const isOwner = fact.origin === 'OWNER'
  const priceWords = `${fmtPrice(fact.price)}` // announced value

  return (
    <div
      role="row"
      aria-label={`${fact.label} ${priceWords}, grade ${fact.grade}, ${fresh}, ${fact.instruction}, ${fmtDistance(fact.distance)} points`}
      className="grid items-center gap-2 py-1.5"
      style={{
        gridTemplateColumns: 'minmax(64px,auto) auto auto 1fr auto',
        opacity: consumed ? 0.5 : 1,
        borderBottom: '1px solid var(--vl-hair)',
      }}
    >
      {/* price + owner/note markers */}
      <div className="flex items-center gap-1">
        <span
          className="vl-num text-[13px]"
          style={{ color: 'var(--vl-ivory)' }}
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
}: {
  facts: PlanLevelFact[]
  language: Language
}) {
  return (
    <div role="table" aria-label={tp('keyLevels', language)}>
      <div className="flex items-center justify-between mb-1">
        <span
          className="text-[10px] uppercase tracking-widest"
          style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
        >
          {tp('keyLevels', language)}
        </span>
        <span
          className="text-[10px] uppercase"
          style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
        >
          {tp('colDistance', language)}
        </span>
      </div>
      {facts.length === 0 ? (
        <div
          className="py-3 text-[12px]"
          style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
        >
          {tp('noLevels', language)}
        </div>
      ) : (
        facts.map((f, i) => (
          <ZoneRow
            key={`${f.label}-${f.price}-${i}`}
            fact={f}
            language={language}
          />
        ))
      )}
    </div>
  )
}
