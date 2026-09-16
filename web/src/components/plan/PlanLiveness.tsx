import type { ScenarioLiveness } from '../../lib/api/plan'

export function PlanLiveness({ value }: { value?: ScenarioLiveness }) {
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
    </div>
  )
}
