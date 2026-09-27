import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { PlanComposedOf } from '../../lib/api/plan'

// W-EXEC-TRUTH W5 — the plan card says when a plan is not (only) the AI's.
//
// MachinePlanBanner: the served row is a MACHINE plan — the no-plan door: a
// Picture HTF scenario was recorded before any AI plan existed for the session
// (machine_plan: true, READ from the server). The first AI plan supersedes it.
//
// ComposedOfLine: what composed plan_final, from the fold's own record
// (composed_of: the user overlay versions applied, and each machine scenario
// with the overlay that carried it):
//   composed of: base + overlays o1,o3 · machine P1 (o4)
// Absent composed_of → nothing (a server that does not record it is not a
// plan with nothing folded). A missing number reads "n/a", never 0.

const isNum = (v: unknown): v is number =>
  typeof v === 'number' && Number.isFinite(v)

export function MachinePlanBanner({
  machinePlan,
  language,
}: {
  machinePlan?: boolean
  language: Language
}) {
  if (machinePlan !== true) return null
  return (
    <div
      data-testid="machine-plan-banner"
      className="flex items-center gap-2 px-3 py-2 text-[12px]"
      style={{
        background: 'var(--vl-gold-dim)',
        border: '1px solid var(--vl-gold-line)',
        borderRadius: 'var(--vl-radius-chip)',
        color: 'var(--vl-ivory)',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      <span aria-hidden>📷</span>
      <span>{tp('machinePlanBanner', language)}</span>
    </div>
  )
}

/** The composed-of text, or null when the server did not record it. */
export function composedOfText(
  composedOf: PlanComposedOf | null | undefined,
  language: Language = 'en'
): string | null {
  if (!composedOf) return null
  let out = `${tp('composedOf', language)}: ${tp('composedOfBase', language)}`
  const user = (composedOf.user_overlays ?? []).map((v) =>
    isNum(v) ? `o${v}` : 'n/a'
  )
  if (user.length > 0)
    out += ` + ${tp('composedOfOverlays', language)} ${user.join(',')}`
  const machine = (composedOf.machine ?? []).map(
    (m) =>
      `${m?.scenario_id?.trim() || 'n/a'} (${isNum(m?.overlay_version) ? `o${m.overlay_version}` : 'n/a'})`
  )
  if (machine.length > 0)
    out += ` · ${tp('composedOfMachine', language)} ${machine.join(', ')}`
  return out
}

export function ComposedOfLine({
  composedOf,
  language,
}: {
  composedOf?: PlanComposedOf | null
  language: Language
}) {
  const text = composedOfText(composedOf, language)
  if (!text) return null
  const refs = (composedOf?.machine ?? [])
    .map((m) => `${m?.scenario_id || 'n/a'} ← ${m?.ref || 'n/a'}`)
    .join(' · ')
  return (
    <div
      data-testid="plan-composed-of"
      className="text-[10px]"
      title={refs || undefined}
      style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
    >
      {text}
    </div>
  )
}
