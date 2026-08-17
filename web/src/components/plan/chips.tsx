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

// ── VersionChips (ITEM 15) ──
//
// These used to be inert <span>s whose labels were FABRICATED from a single
// integer — `Array.from({length: version})` — and collapsed at n>=4 to "v1 … vN",
// which hid v2..v5 behind an ellipsis with no way to reach them. So the component
// had never seen the list of versions, only how many there were: it could not say
// which existed, when they were written, or why any of them died.
//
// Now each chip is a real button over a real record. The row wraps instead of
// collapsing (see .vl-version-row) so no version is unreachable, and the count is
// bounded in practice by replan_cap + 1.
export function VersionChips({
  version,
  latest,
  count,
  onSelect,
  titleFor,
  noTradeVersion,
  noTradeLabel,
}: {
  /** the version currently being VIEWED */
  version: number
  /** the newest stored version (defaults to `version`) */
  latest?: number
  /** how many versions exist (defaults to `latest`) — chips render 1..count */
  count?: number
  /** omit to keep the chips read-only (the pre-ITEM-15 behavior) */
  onSelect?: (version: number) => void
  /** tooltip per version — the death reason, when one is known */
  titleFor?: (version: number) => string | undefined
  /**
   * The version that is the NO-TRADE TERMINAL MARKER, not a real plan.
   * replan_cap=4 legitimately ends at a row called "v6" (v1..v5 real, v6 the
   * marker), and showing that as a plain "v6" is what read as "the cap didn't
   * work". It renders as ⛔ NO-TRADE instead of a version number.
   */
  noTradeVersion?: number
  /** label for the NO-TRADE chip, already localised by the caller */
  noTradeLabel?: string
}) {
  const newest = latest ?? version
  const total = count ?? newest
  if (total <= 0) return null
  return (
    <span className="vl-version-row" role="group" aria-label="plan versions">
      {Array.from({ length: total }, (_, i) => i + 1).map((v) => {
        const isNoTrade = noTradeVersion === v
        return (
          <button
            key={v}
            type="button"
            className="vl-version-chip"
            data-no-trade={isNoTrade ? '1' : undefined}
            aria-current={v === version}
            aria-label={
              isNoTrade
                ? `no-trade marker (version ${v})`
                : `plan version ${v}${v === newest ? ' (current)' : ''}`
            }
            title={titleFor?.(v)}
            disabled={!onSelect}
            onClick={onSelect ? () => onSelect(v) : undefined}
          >
            {isNoTrade ? `⛔ ${noTradeLabel ?? 'NO-TRADE'}` : `v${v}`}
          </button>
        )
      })}
    </span>
  )
}

// ── ConflictChip (P5.3): ⚡ overlapping owner/AI elements with opposing
// instructions. Owner wins on execution; the AI element is ghosted, both kept
// visible + logged. Text+shape (never color alone). ──
export function ConflictChip({ language }: { language: Language }) {
  return (
    <span
      className="inline-flex items-center gap-1 text-[9px] font-bold uppercase tracking-wider"
      style={{
        color: 'var(--vl-short)',
        border: '1px solid rgba(224,108,108,0.4)',
        background: 'rgba(224,108,108,0.08)',
        borderRadius: 'var(--vl-radius-chip)',
        padding: '1px 5px',
        fontFamily: 'var(--vl-font-ui)',
      }}
      title={tp('conflictHint', language)}
    >
      ⚡ {tp('conflict', language)}
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
        | 'lifecycleNoTrade'
    }
  > = {
    active: { color: 'var(--vl-gold)', key: 'lifecycleActive' },
    expired: { color: 'var(--vl-faint)', key: 'lifecycleExpired' },
    died: { color: 'var(--vl-short)', key: 'lifecycleDied' },
    superseded: { color: 'var(--vl-faint)', key: 'lifecycleSuperseded' },
    // 'no_trade' is written by the fail-closed / re-plans-exhausted path
    // (trader/auto_trader_planner.go:227). It was missing from this map, so the
    // `?? map.active` fallback painted a NO-TRADE plan with the GOLD "ACTIVE"
    // chip — the one lifecycle value production actually writes, displayed as its
    // opposite. A plan that is sitting the session out must never look live.
    no_trade: { color: 'var(--vl-short)', key: 'lifecycleNoTrade' },
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
