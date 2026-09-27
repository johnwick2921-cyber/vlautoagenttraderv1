import type { AuthoredInvalidation, ScenarioLiveness } from '../../lib/api/plan'

// W-EXEC-TRUTH W2 A1/A2 — one line, READ from the row's born check: the policy
// the write site enforced, its read → publish clocks (CT) and how many 5m
// groups it judged. Nothing recorded → n/a, never an inferred policy.
function ctClock(ms: number | null): string {
  if (ms == null || !Number.isFinite(ms)) return 'n/a'
  return (
    new Date(ms).toLocaleTimeString('en-US', {
      timeZone: 'America/Chicago',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }) + ' CT'
  )
}

function authoredInvalidationLine(a?: AuthoredInvalidation): string {
  if (!a || !a.recorded || !a.policy) {
    return 'invalidation: n/a (no born check recorded on this plan row)'
  }
  const n = a.groups?.length ?? 0
  return `invalidation: ${a.policy} · read ${ctClock(a.read_clock_ms)} → publish ${ctClock(a.publish_clock_ms)} · ${n} 5m group${n === 1 ? '' : 's'} judged`
}

export function PlanLiveness({
  value,
  authored,
}: {
  value?: ScenarioLiveness
  authored?: AuthoredInvalidation
}) {
  const known = value?.tradeable != null
  const exhausted = known && value.total > 0 && value.tradeable === 0
  return (
    <div
      data-testid="plan-liveness"
      role={exhausted ? 'status' : undefined}
      style={{ color: exhausted ? '#F6465D' : 'var(--vl-muted)', fontSize: 12 }}
    >
      {known
        ? `tradeable ${value.tradeable}/${value.total}`
        : 'tradeable UNKNOWN'}
      {exhausted && ' · EXHAUSTED — warning only; no exhaustion wake'}
      <span className="block text-[10px]">
        {value?.reason ?? 'Waiting for this version’s scenario evaluation.'}
        {known && value.observed_at && ` · observed ${value.observed_at}`}
      </span>
      <span
        data-testid="plan-authored-invalidation"
        className="block text-[10px]"
      >
        {authoredInvalidationLine(authored)}
      </span>
    </div>
  )
}
