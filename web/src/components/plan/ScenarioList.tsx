import type { ScenarioDeath, ScenarioLevelIdentity } from '../../lib/api/plan'
// P4.3 — scenario rows: StatusDot · id · quality · name/grammar + the target
// (uses) chain. Status is READ-ONLY from the backend (single-authority rule) —
// absent → 'armed' (plan-born). The UI never computes trading state.

import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type {
  PlanArmView,
  PlanScenario,
  ScenarioStatusValue,
} from '../../lib/api/plan'
import { OrderTerms } from './OrderTerms'
import { ScenarioEconomics } from './ScenarioEconomics'
import { FadePermissionChip, type FadeLabelView } from './FadePermissionChip'
import { OneSetupChip, type OneSetupView } from './OneSetupChip'
import { StatusDot, type ScenarioStatus } from './chips'

export function QualityChip({ quality }: { quality: string }) {
  const q = quality || 'B'
  // C7 — A/A+ use the A token (was: A got the B token); B and C get their own.
  const color =
    q === 'A+' || q === 'A'
      ? 'var(--vl-grade-a)'
      : q === 'B'
        ? 'var(--vl-grade-b)'
        : 'var(--vl-grade-c)'
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

export type ConfirmVerdict = {
  outcome?: string
  evaluated_ms?: number
  reference_ms?: number
  reference_source?: string
  bucket?: {
    open_ms: number
    close_ms: number
    minutes: number
    closed: boolean
  }
  rule: string
  ref_price: number
  side: string
  met: boolean
  detail: string
  legs?: Array<{
    met: boolean
    rule: string
    ref_price: number
    side: string
    detail: string
  }>
}

// ConfirmChip (guide-export 2026-08-27) — the machine confirm verdict chip
// (CONFIRM MET / confirm not met). Advisory; never a gate. F2 (waterfall-class
// wave): two-leg scenarios render the leg states — a partial never reads MET.
// E5/E6 (entry-mechanics 2026-08-30): the new rule names render as-is.
function confirmRuleLabel(rule: string): string {
  switch (rule) {
    case '1x5m_close':
      return '1×5m'
    case '2x5m_close':
      return '2×5m'
    case '1m_mss':
      return '1m-MSS'
    case 'time_hold':
      return 'time-hold'
    default:
      return rule
  }
}

export function ConfirmChip({ id, c }: { id: string; c: ConfirmVerdict }) {
  const legs = c.legs && c.legs.length > 0
  const label = confirmRuleLabel(c.rule)
  const outcome = c.outcome || 'UNKNOWN'
  return (
    <span
      data-testid={`confirm-chip-${id}`}
      className="text-[9px] font-bold px-1.5 py-0.5 rounded"
      title={`${label} ${c.side} ${c.ref_price} — ${c.detail} (machine-computed, advisory)`}
      style={
        c.met && outcome === 'MET'
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
      Recorded {label} {outcome}
      {legs &&
        ` (${c.legs!.map((l, i) => `${i + 1}/${c.legs!.length} ${l.met ? 'MET' : 'NOT MET'}`).join(' · ')})`}
      <span className="block font-normal">
        {c.detail || 'Bucket evidence unavailable'}
        {!c.outcome && ' · legacy record: bucket evidence unavailable'}
      </span>
      <span className="block font-normal">
        ≈ activation is a separate estimate; order authorization is separate.
      </span>
    </span>
  )
}

export type FvgState = {
  id: string
  fvg_lo: number
  fvg_hi: number
  ce: number
  entry_mode: string
  state: string
  touch_number: number
  met: boolean
}

// FvgStateChip (guide-export 2026-08-27) — the fvg_entry gap-band live state
// (IN-ZONE / ABOVE / BELOW / FILLED_INVALID). Advisory; never a gate.
export function FvgStateChip({ f }: { f: FvgState }) {
  const tone =
    f.state === 'FILLED_INVALID'
      ? 'var(--vl-short)'
      : f.state === 'IN_ZONE'
        ? 'var(--vl-long)'
        : 'var(--vl-muted)'
  return (
    <span
      data-testid={`fvg-chip-${f.id}`}
      className="text-[9px] font-bold px-1.5 py-0.5 rounded"
      title={`fvg_entry gap ${f.fvg_lo}–${f.fvg_hi} (CE ${f.ce}, mode ${f.entry_mode}) — ${f.state}${f.touch_number > 0 ? ` · touch #${f.touch_number}` : ''} (machine-computed, advisory)`}
      style={{
        color: tone,
        border: '1px solid var(--vl-hair)',
      }}
    >
      {f.state}
      {f.touch_number > 0 ? ` · #${f.touch_number}` : ''}
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
  // C7 — a direction that isn't long/short renders neutral, not short-red.
  const dirColor =
    dir === 'long'
      ? 'var(--vl-long)'
      : dir === 'short'
        ? 'var(--vl-short)'
        : 'var(--vl-muted)'
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
      {/* Structural invalidation; the economics block renders the path. */}
      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px]">
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

export type ArmLegState = { state: string; leg_index?: number; kind?: string }

export function ArmedChip({
  arm,
}: {
  arm?: { state: string; reason?: string; legs?: ArmLegState[] }
}) {
  if (!arm) return null
  const legGlyphs = (s: string) =>
    s === 'working'
      ? '📌'
      : s === 'filled'
        ? '⚡'
        : s === 'cancelled' || s === 'rejected'
          ? '✕'
          : '⏳'
  const splitLegs = arm.legs && arm.legs.length >= 2 ? arm.legs : null
  const legLine = splitLegs
    ? ` · ${splitLegs.map((l) => `L${(l.leg_index ?? 0) + 1}${legGlyphs(l.state)}`).join(' ')}`
    : ''
  switch (arm.state) {
    case 'armed':
    case 'place_pending':
      return (
        <span
          data-testid={`armed-chip`}
          className="text-[9px] font-bold px-1.5 py-0.5 rounded"
          style={{
            color: 'var(--vl-gold)',
            border: '1px solid var(--vl-gold-line)',
          }}
        >
          ⏳{' '}
          {arm.state === 'place_pending'
            ? 'placement pending'
            : 'order authorized'}
          {legLine}
        </span>
      )
    case 'working':
      return (
        <span
          data-testid={`armed-chip`}
          className="text-[9px] font-bold px-1.5 py-0.5 rounded vl-pulse"
          style={{
            color: 'var(--vl-gold)',
            border: '1px solid var(--vl-gold-line)',
          }}
        >
          📌 working{legLine}
        </span>
      )
    case 'filled':
      return (
        <span
          data-testid={`armed-chip`}
          className="text-[9px] font-bold px-1.5 py-0.5 rounded"
          style={{
            color: 'var(--vl-long)',
            border: '1px solid rgba(63,191,143,0.35)',
          }}
        >
          ⚡ filled{legLine}
        </span>
      )
    case 'cancelled':
    case 'rejected':
      return (
        <span
          data-testid={`armed-chip`}
          className="text-[9px] font-bold px-1.5 py-0.5 rounded"
          title={arm.reason ?? ''}
          style={{
            color: 'var(--vl-short)',
            border: '1px solid rgba(224,108,108,0.4)',
          }}
        >
          ✕ {arm.state}
          {legLine}
          {arm.reason ? ` · ${arm.reason}` : ''}
        </span>
      )
  }
  return null
}

export function ScenarioList({
  scenarios,
  statusMap,
  deaths,
  identities,
  meta,
  fvgStates,
  armedStates,
  fadeLabels,
  oneSetup,
  language,
}: {
  scenarios: PlanScenario[]
  statusMap?: Record<string, ScenarioStatusValue>
  deaths?: Record<string, ScenarioDeath>
  identities?: Record<string, ScenarioLevelIdentity>
  /** Wave 2 armed orders — per-scenario arm state (⏳/📌/⚡/✕+reason). */
  armedStates?: Record<string, PlanArmView>
  /** W2 FADE PERMISSION — per-scenario label; absent renders "not evaluated". */
  fadeLabels?: Record<string, FadeLabelView>
  /** ONE SETUP — the seam's recorded verdict per scenario; absent renders "not evaluated". */
  oneSetup?: OneSetupView
  /** A1/A4/C1 (fail-register wave): verdict basis, unevaluable ids, confirm verdicts */
  meta?: {
    basis?: Record<string, string>
    unevaluable?: string[]
    confirm?: Record<string, ConfirmVerdict>
  }
  /** FVG ENTRY MODEL (2026-08-26) — per-scenario gap-band live states (advisory). */
  fvgStates?: Array<{
    id: string
    fvg_lo: number
    fvg_hi: number
    ce: number
    entry_mode: string
    state: string
    touch_number: number
    met: boolean
  }>
  language: Language
}) {
  return (
    <div role="table" aria-label={tp('scenarios', language)}>
      <span
        className="text-[10px] uppercase tracking-widest"
        style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
      >
        {tp('scenarios', language)}
        <span title={tp('scenarioOrderSeparation', language)}>
          {' '}
          · {tp('scenarioActivation', language)}
        </span>
      </span>
      <p className="text-[10px]" style={{ color: 'var(--vl-muted)' }}>
        {tp('scenarioOrderSeparation', language)}
      </p>
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
              {deaths?.[s.id] && (
                <div
                  data-testid={`scenario-death-${s.id}`}
                  className="text-[10px]"
                  title={deaths[s.id].condition}
                >
                  Recorded {deaths[s.id].cause} · v{deaths[s.id].version} ·
                  anchor {deaths[s.id].anchor.toFixed(2)} · first observed{' '}
                  {deaths[s.id].observed_at} · {deaths[s.id].basis} verdict
                  history
                </div>
              )}
              {!deaths?.[s.id] && stored === 'invalidated' && (
                <div
                  data-testid={`scenario-death-unknown-${s.id}`}
                  className="text-[10px]"
                >
                  Recorded invalidation time UNKNOWN for this version and anchor
                </div>
              )}
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
                    <ConfirmChip id={s.id} c={meta.confirm[s.id]} />
                  )}
                  {fvgStates?.find((f) => f.id === s.id) && (
                    <FvgStateChip f={fvgStates.find((x) => x.id === s.id)!} />
                  )}
                  {/* Wave 2 armed orders — the arm state chip (⏳/📌/⚡/✕). */}
                  <ArmedChip arm={armedStates?.[s.id]} />
                  {/* W2 fade permission — a label, never a gate. */}
                  <FadePermissionChip id={s.id} v={fadeLabels?.[s.id]} />
                  {/* ONE SETUP — what the arm seam decided, never a re-evaluation. */}
                  <OneSetupChip id={s.id} os={oneSetup} />
                </div>
              )}
              <ScenarioIdentity
                identity={identities?.[s.id]}
                authoredId={s.level_id}
              />
              <ScenarioEconomics scenario={s} />
              <OrderTerms legs={armedStates?.[s.id]?.legs} />
            </div>
          )
        })}
      </div>
    </div>
  )
}

export function ScenarioIdentity({
  identity,
  authoredId,
}: {
  identity?: ScenarioLevelIdentity
  authoredId?: string | null
}) {
  const id = identity?.level_id ?? authoredId ?? null
  const level = identity?.level
  return (
    <div
      data-testid="scenario-level-identity"
      style={{ fontSize: 11, overflowWrap: 'anywhere' }}
    >
      Level ID: {id ?? 'NULL'}
      {level ? (
        <>
          {' '}
          · {level.names?.join(' · ') || level.label} @ {level.price.toFixed(2)}{' '}
          · {level.tf ?? 'UNKNOWN'} · formation close{' '}
          {level.formed_close_ms ?? 'UNKNOWN'}
          {identity?.disagreed && (
            <>
              {' '}
              · evaluator anchor{' '}
              {identity.evaluator_anchor?.toFixed(2) ?? 'UNKNOWN'} differs;
              decision unchanged
            </>
          )}
        </>
      ) : (
        <>
          {' '}
          ·{' '}
          {id
            ? 'Unresolved [WARN]'
            : 'Unnamed / legacy [WARN] — no identity inferred'}
        </>
      )}
    </div>
  )
}
