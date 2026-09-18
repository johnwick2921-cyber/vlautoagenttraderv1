// W-ARM-STATE-UI (2026-09-18) — the EXECUTOR's verdict per scenario, rendered
// beside the evaluator's. Sourced ONLY from armed_orders rows (plan.armed,
// served by armedMapFor) and the executor geometry records
// (plan.structural_geometry, served by planStructuralGeometry). No record for
// the displayed plan version → the column renders NOTHING (no dash, no "ok").

import type { PlanArmView, StructuralGeometryView } from '../../lib/api/plan'

export type ExecutorVerdictState =
  | 'armed'
  | 'filled'
  | 'cancelled'
  | 'refused'
  | 'not_attempted'
  // superseded / shadowed / any other terminal ledger state — labelled
  // verbatim as "<state>: <state_reason>", never guessed into a category.
  | 'other'
  // the WRITE SITE disabled the arm (arm.arm_disabled_reason) — the executor
  // never ran for this scenario; sourced ONLY from that field.
  | 'disabled_at_write'

export interface ExecutorLine {
  state: ExecutorVerdictState
  orderId?: number
  reason?: string
  detail?: string
  timeMs?: number
  label: string
}

function idSuffix(id?: number): string {
  return typeof id === 'number' ? ` #${id}` : ''
}

function fmtTime(ms: number): string {
  const d = new Date(ms)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// F2/F4 — the armed_orders ledger row is the ONLY proof of an arm. Once a
// non-UNKNOWN row exists for this version it speaks and the geometry records
// are NOT consulted at all (the FE cannot compare times: planOrderLeg carries
// no created_at on the wire, so a ledger 'armed' over a later geometry refusal
// shows 'armed' — defensible, documented here rather than guessed).
function lineForState(
  state: string,
  rowId: number | undefined,
  reason: string | undefined
): ExecutorLine {
  if (state === 'filled')
    return {
      state: 'filled',
      orderId: rowId,
      reason,
      label: `filled${idSuffix(rowId)}`,
    }
  if (state === 'armed' || state === 'place_pending' || state === 'working')
    return {
      state: 'armed',
      orderId: rowId,
      reason,
      label: `armed${idSuffix(rowId)}`,
    }
  if (state === 'cancelled' || state === 'rejected' || state === 'expired')
    return {
      state: 'cancelled',
      orderId: rowId,
      reason,
      label: `cancelled: ${reason || state}`,
    }
  // superseded, shadowed, or any state this file has never met: show the
  // ledger's own words, never invent a category.
  return {
    state: 'other',
    orderId: rowId,
    reason,
    label: `${state}${reason ? `: ${reason}` : ''}`,
  }
}

/**
 * Pure derivation — exported so the vitest pins exercise the production call
 * site. armed_orders speaks first (the executor's final word for this plan
 * version); state UNKNOWN means the ledger has no row for this version — it is
 * NOT a verdict, so the geometry records speak next.
 */
export function executorLinesFor(
  scenario: string,
  arm: PlanArmView | undefined,
  geometry: StructuralGeometryView[] | null | undefined,
  disabledAtWrite?: string
): ExecutorLine[] {
  // A write-disabled arm was judged BEFORE any executor ran — it speaks first
  // and alone (sourced ONLY from arm.arm_disabled_reason; absent → nothing).
  if (disabledAtWrite) {
    return [
      {
        state: 'disabled_at_write',
        reason: disabledAtWrite,
        label: `disabled at write: ${disabledAtWrite}`,
      },
    ]
  }
  if (arm && arm.state !== 'UNKNOWN') {
    const legs = (arm.legs ?? []).filter((l) => l.state !== 'UNKNOWN')
    const lines =
      arm.state === 'mixed'
        ? legs.map((l) => lineForState(l.state, l.row_id, l.reason))
        : [lineForState(arm.state, legs[0]?.row_id, arm.reason)]
    // F2 — a ledger row exists for this version: it is the verdict. No
    // fall-through to geometry, ever.
    if (lines.length > 0) return lines
  }
  const rows = (geometry ?? []).filter((r) => r.scenario === scenario)
  if (rows.length === 0) return []
  const latest = rows.reduce((a, b) =>
    (b.time_ms ?? 0) > (a.time_ms ?? 0) ? b : a
  )
  // F1 — the store's OWN refusal rule (store/structural_geometry.go:119): any
  // non-empty reason that is not admitted/pending_gates, with quantity 0, is
  // a refusal — entry_gate, one_setup, rr, no_provenance, … No whitelist that
  // ages as the executor grows new refusal classes.
  if (
    latest.quantity === 0 &&
    latest.reason !== '' &&
    latest.reason !== 'admitted' &&
    latest.reason !== 'pending_gates'
  )
    return [
      {
        state: 'refused',
        reason: latest.reason,
        detail: latest.detail,
        timeMs: latest.time_ms,
        label: `refused: ${latest.reason}${latest.detail ? ` (${latest.detail})` : ''}`,
      },
    ]
  if (
    latest.reason === 'pending_gates' ||
    (latest.reason === '' && latest.quantity === 0)
  )
    return [
      {
        state: 'not_attempted',
        reason: latest.reason || 'pending_gates',
        detail: latest.detail,
        timeMs: latest.time_ms,
        label: 'not attempted',
      },
    ]
  // F2 — "admitted" is a GATE verdict, not proof of an arm; the ledger row is
  // the only proof. An admitted record with no ledger row renders nothing.
  return []
}

const lineColor = (line: ExecutorLine) =>
  line.state === 'filled'
    ? 'var(--vl-long)'
    : line.state === 'armed'
      ? 'var(--vl-gold)'
      : line.state === 'refused' ||
          line.state === 'cancelled' ||
          line.state === 'disabled_at_write'
        ? 'var(--vl-short)'
        : 'var(--vl-faint)'

export function ExecutorVerdict({
  scenario,
  arm,
  geometry,
  disabledAtWrite,
}: {
  scenario: string
  arm?: PlanArmView
  geometry?: StructuralGeometryView[] | null
  disabledAtWrite?: string
}) {
  const lines = executorLinesFor(scenario, arm, geometry, disabledAtWrite)
  if (lines.length === 0) return null
  return (
    <span
      data-testid={`executor-verdict-${scenario}`}
      className="text-[9px] font-bold px-1.5 py-0.5 rounded"
      style={{
        border: '1px solid var(--vl-hair)',
        background: 'transparent',
        color: lineColor(lines[0]),
        display: 'inline-flex',
        gap: 4,
        alignItems: 'center',
      }}
    >
      {lines.map((line, i) => (
        <span
          key={i}
          title={[
            line.reason,
            line.detail,
            line.timeMs ? fmtTime(line.timeMs) : '',
          ]
            .filter(Boolean)
            .join(' · ')}
        >
          {line.label}
        </span>
      ))}
    </span>
  )
}
