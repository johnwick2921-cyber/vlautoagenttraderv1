// P4.3 — scenario rows: StatusDot · id · quality · name/grammar + the target
// (uses) chain. Status is READ-ONLY from the backend (single-authority rule) —
// absent → 'armed' (plan-born). The UI never computes trading state.

import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { PlanScenario, ScenarioStatusValue } from '../../lib/api/plan'
import { StatusDot, type ScenarioStatus } from './chips'
import { fmtPrice } from './levelState'

function QualityChip({ quality }: { quality: string }) {
  const q = quality || 'B'
  const color =
    q === 'A+'
      ? 'var(--vl-gold)'
      : q === 'A'
        ? 'var(--vl-grade-b)'
        : 'var(--vl-faint)'
  return (
    <span
      className="text-[10px] font-bold"
      style={{
        color,
        border: `1px solid ${color}`,
        borderRadius: 'var(--vl-radius-chip)',
        padding: '0 4px',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      {q}
    </span>
  )
}

function ScenarioRow({
  scenario,
  status,
  language,
}: {
  scenario: PlanScenario
  status: ScenarioStatus
  language: Language
}) {
  const dir = (scenario.direction || '').toLowerCase()
  const dirColor = dir === 'long' ? 'var(--vl-long)' : 'var(--vl-short)'
  return (
    <div
      role="row"
      className="py-2"
      style={{ borderBottom: '1px solid var(--vl-hair)' }}
    >
      <div className="flex items-center gap-2">
        <StatusDot status={status} language={language} />
        <span
          className="text-[11px] font-bold"
          style={{ color: 'var(--vl-gold)', fontFamily: 'var(--vl-font-data)' }}
        >
          {scenario.id}
        </span>
        <span title="quality is INFORMATIONAL (D3 ruling) — the planner's own read; no gate, sizing, or filter consumes it">
          <QualityChip quality={scenario.quality} />
        </span>
        {scenario.consumed && (
          <span
            className="text-[10px] uppercase"
            style={{ color: 'var(--vl-gold)', fontFamily: 'var(--vl-font-ui)' }}
          >
            level consumed
          </span>
        )}
        <span
          className="text-[11px] uppercase"
          style={{ color: dirColor, fontFamily: 'var(--vl-font-ui)' }}
        >
          {dir}
        </span>
      </div>
      {/* grammar line: trigger + condition */}
      <div
        className="mt-1 text-[11px]"
        style={{ color: 'var(--vl-muted)', fontFamily: 'var(--vl-font-ui)' }}
      >
        {scenario.trigger}
        {scenario.condition ? ` · ${scenario.condition}` : ''}
      </div>
      {/* uses-chain (targets) + invalidation */}
      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px]">
        {scenario.target_chain && scenario.target_chain.length > 0 && (
          <span
            style={{
              color: 'var(--vl-faint)',
              fontFamily: 'var(--vl-font-ui)',
            }}
          >
            {tp('targets', language)}:{' '}
            <span className="vl-num" style={{ color: 'var(--vl-long)' }}>
              {scenario.target_chain.map((t) => fmtPrice(t)).join(' → ')}
            </span>
          </span>
        )}
        {scenario.invalid && scenario.invalid !== 'n/a' && (
          <span
            style={{
              color: 'var(--vl-faint)',
              fontFamily: 'var(--vl-font-ui)',
            }}
          >
            {tp('invalidates', language)}:{' '}
            <span className="vl-num" style={{ color: 'var(--vl-short)' }}>
              {scenario.invalid}
            </span>
          </span>
        )}
      </div>
    </div>
  )
}

export function ScenarioList({
  scenarios,
  statusMap,
  meta,
  language,
}: {
  scenarios: PlanScenario[]
  statusMap?: Record<string, ScenarioStatusValue>
  /** A1/A4/C1 (fail-register wave): verdict basis, unevaluable ids, confirm verdicts */
  meta?: {
    basis?: Record<string, string>
    unevaluable?: string[]
    confirm?: Record<
      string,
      {
        rule: string
        ref_price: number
        side: string
        met: boolean
        detail: string
      }
    >
  }
  language: Language
}) {
  return (
    <div role="table" aria-label={tp('scenarios', language)}>
      <span
        className="text-[10px] uppercase tracking-widest"
        style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
      >
        {tp('scenarios', language)}
        <span title="Scenario dots and confirm chips are ADVISORY — they inform the card and the AI's prompt, they never hard-gate an entry (a hard scenario-state gate would recreate the suppression class; AI judgment + plan discipline already gate).">
          {' '}
          (advisory)
        </span>
      </span>
      <div className="mt-1">
        {scenarios.map((s) => {
          const stored = statusMap?.[s.id] as ScenarioStatus | undefined
          const unevaluable =
            !stored || meta?.unevaluable?.includes(s.id) === true
          const heuristic = meta?.basis?.[s.id] === 'heuristic'
          return (
            <div
              key={s.id}
              title={
                unevaluable
                  ? 'unevaluable — no price in the trigger/invalid text snaps to a plan level (±2pts); status is unknown, not "armed"'
                  : heuristic
                    ? 'anchor heuristic (AI-judged prose) — NOT the machine death/flip evaluation'
                    : 'machine verdict (shares the plan-death evaluation)'
              }
              style={heuristic ? { opacity: 0.75 } : undefined}
            >
              {unevaluable ? (
                <div
                  className="flex items-center gap-1.5 text-[11px] py-0.5"
                  style={{ color: 'var(--vl-faint)' }}
                  data-testid={`scenario-unevaluable-${s.id}`}
                >
                  <span>?</span>
                  <span className="font-bold">{s.id}</span>
                  <span className="truncate">{s.trigger}</span>
                </div>
              ) : (
                <div className="flex items-center gap-1.5">
                  <ScenarioRow
                    scenario={s}
                    status={stored as ScenarioStatus}
                    language={language}
                  />
                  {meta?.confirm?.[s.id] && (
                    <span
                      data-testid={`confirm-chip-${s.id}`}
                      className="text-[9px] font-bold px-1.5 py-0.5 rounded"
                      title={`${meta.confirm[s.id].rule} ${meta.confirm[s.id].side} ${meta.confirm[s.id].ref_price} — ${meta.confirm[s.id].detail} (machine-computed, advisory)`}
                      style={
                        meta.confirm[s.id].met
                          ? {
                              color: 'var(--vl-long)',
                              border: '1px solid rgba(63,191,143,0.35)',
                            }
                          : {
                              color: 'var(--vl-muted)',
                              border: '1px solid var(--vl-hair)',
                            }
                      }
                    >
                      {meta.confirm[s.id].met
                        ? 'CONFIRM MET'
                        : 'confirm not met'}
                    </span>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
