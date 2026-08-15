// P4 — shared Plan Card primitives (tokens only, no raw hex). Every status is
// conveyed by text + shape, never color alone (design-system a11y rule).

import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'

// ── GradeChip: A=gold · B=grey · C=faint ──
export function GradeChip({ grade }: { grade: string }) {
  const g = (grade || '').toUpperCase()
  const color =
    g === 'A'
      ? 'var(--vl-grade-a)'
      : g === 'B'
        ? 'var(--vl-grade-b)'
        : 'var(--vl-grade-c)'
  return (
    <span
      className="inline-flex items-center justify-center text-[10px] font-bold"
      style={{
        color,
        border: `1px solid ${color}`,
        borderRadius: 'var(--vl-radius-chip)',
        minWidth: 16,
        height: 16,
        padding: '0 3px',
        fontFamily: 'var(--vl-font-ui)',
      }}
      title={`Grade ${g || '—'}`}
    >
      {g || '—'}
    </span>
  )
}

// ── ProvenanceChip: the source label (PDH, ONH, nPOC·Tue, RN, EQH…) ──
export function ProvenanceChip({ label }: { label: string }) {
  return (
    <span
      className="inline-flex items-center text-[11px]"
      style={{
        color: 'var(--vl-muted)',
        background: 'var(--vl-card-2)',
        border: '1px solid var(--vl-hair)',
        borderRadius: 'var(--vl-radius-chip)',
        padding: '1px 6px',
        fontFamily: 'var(--vl-font-data)',
      }}
    >
      {label || '—'}
    </span>
  )
}

export type Fresh = 'fresh' | 'tested' | 'consumed'

// ── FreshDot: fresh (long) · tested (gold) · consumed (faint) ──
export function FreshDot({
  fresh,
  language,
}: {
  fresh: Fresh
  language: Language
}) {
  const map: Record<
    Fresh,
    { color: string; key: 'freshFresh' | 'freshTested' | 'freshConsumed' }
  > = {
    fresh: { color: 'var(--vl-long)', key: 'freshFresh' },
    tested: { color: 'var(--vl-gold)', key: 'freshTested' },
    consumed: { color: 'var(--vl-faint)', key: 'freshConsumed' },
  }
  const m = map[fresh]
  return (
    <span
      className="inline-flex items-center gap-1 text-[11px]"
      style={{ color: 'var(--vl-muted)', fontFamily: 'var(--vl-font-ui)' }}
    >
      <span
        aria-hidden
        style={{
          width: 6,
          height: 6,
          borderRadius: 999,
          background: m.color,
          display: 'inline-block',
        }}
      />
      {tp(m.key, language)}
    </span>
  )
}

export type ScenarioStatus =
  | 'armed'
  | 'waiting'
  | 'triggered'
  | 'invalidated'
  | 'expired'

// ── StatusDot: ○ armed · ◌ waiting · ● triggered(pulse) · ✕ invalidated · ○ expired(dim) ──
export function StatusDot({
  status,
  language,
}: {
  status: ScenarioStatus
  language: Language
}) {
  const map: Record<
    ScenarioStatus,
    {
      glyph: string
      color: string
      pulse?: boolean
      key:
        | 'statusArmed'
        | 'statusWaiting'
        | 'statusTriggered'
        | 'statusInvalidated'
        | 'statusExpired'
    }
  > = {
    armed: { glyph: '○', color: 'var(--vl-status-armed)', key: 'statusArmed' },
    waiting: { glyph: '◌', color: 'var(--vl-muted)', key: 'statusWaiting' },
    triggered: {
      glyph: '●',
      color: 'var(--vl-status-triggered)',
      pulse: true,
      key: 'statusTriggered',
    },
    invalidated: {
      glyph: '✕',
      color: 'var(--vl-status-invalid)',
      key: 'statusInvalidated',
    },
    expired: { glyph: '○', color: 'var(--vl-faint)', key: 'statusExpired' },
  }
  const m = map[status]
  return (
    <span
      className="inline-flex items-center gap-1.5"
      style={{ color: m.color, fontFamily: 'var(--vl-font-ui)' }}
      title={tp(m.key, language)}
    >
      <span
        aria-hidden
        className={m.pulse ? 'vl-pulse' : undefined}
        style={{ fontSize: 12, lineHeight: 1 }}
      >
        {m.glyph}
      </span>
      <span className="text-[11px] uppercase tracking-wide">
        {tp(m.key, language)}
      </span>
    </span>
  )
}

// ── VersionChip: v1 · v2 … collapses at 4 (design-system Open Q #2) ──
export function VersionChips({ version }: { version: number }) {
  const chip = (label: string, active: boolean) => (
    <span
      key={label}
      className="text-[10px]"
      style={{
        color: active ? 'var(--vl-gold)' : 'var(--vl-faint)',
        border: `1px solid ${active ? 'var(--vl-gold-line)' : 'var(--vl-hair)'}`,
        borderRadius: 'var(--vl-radius-chip)',
        padding: '0 5px',
        fontFamily: 'var(--vl-font-data)',
      }}
    >
      {label}
    </span>
  )
  if (version <= 0) return null
  if (version >= 4) {
    return (
      <span className="inline-flex items-center gap-1">
        {chip('v1', false)}
        <span className="text-[10px]" style={{ color: 'var(--vl-faint)' }}>
          …
        </span>
        {chip(`v${version}`, true)}
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-1">
      {Array.from({ length: version }, (_, i) =>
        chip(`v${i + 1}`, i + 1 === version)
      )}
    </span>
  )
}

// ── LifecycleChip: active(gold) · expired/died/superseded(faint/short) ──
export function LifecycleChip({
  lifecycle,
  language,
}: {
  lifecycle: string
  language: Language
}) {
  const map: Record<
    string,
    {
      color: string
      key:
        | 'lifecycleActive'
        | 'lifecycleExpired'
        | 'lifecycleDied'
        | 'lifecycleSuperseded'
    }
  > = {
    active: { color: 'var(--vl-gold)', key: 'lifecycleActive' },
    expired: { color: 'var(--vl-faint)', key: 'lifecycleExpired' },
    died: { color: 'var(--vl-short)', key: 'lifecycleDied' },
    superseded: { color: 'var(--vl-faint)', key: 'lifecycleSuperseded' },
  }
  const m = map[lifecycle] ?? map.active
  return (
    <span
      className="text-[10px] font-bold uppercase tracking-wider"
      style={{
        color: m.color,
        border: `1px solid ${m.color}`,
        borderRadius: 'var(--vl-radius-chip)',
        padding: '1px 6px',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      {tp(m.key, language)}
    </span>
  )
}
