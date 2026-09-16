// CLASS 50 (refs-only wave, 2026-09-02) — the WEEKLY chip on the day-plan card.
// One state: refs only. The Sunday read lists weekly-class price facts
// (PWH/PWL/IPDA/NWOG) and NO directional call (calibration 2026-09-02: the
// bias was anti-predictive). The chip renders "WEEKLY refs — PWH x · PWL y";
// grey "none" when no doc exists. Tooltip = the doc's ≤3-line narrative.
// Pure view — the weekly doc never gates the session plan.
import type { PlanWeekly } from '../../lib/api/plan'

export function WeeklyChip({ weekly }: { weekly?: PlanWeekly | null }) {
  if (!weekly) {
    return (
      <span
        data-testid="weekly-chip"
        title="WEEKLY: none — no Sunday weekly read stored for this week (plans render the none form; nothing else changes)."
        style={{
          fontSize: 9,
          fontWeight: 700,
          letterSpacing: '.08em',
          color: 'var(--vl-faint)',
          border: '1px solid var(--vl-hair)',
          borderRadius: 5,
          padding: '2px 6px',
          fontFamily: 'var(--vl-font-ui)',
        }}
      >
        WEEKLY none
      </span>
    )
  }
  const hasRefs = (weekly.pwh ?? 0) > 0 && (weekly.pwl ?? 0) > 0
  const label = hasRefs
    ? `WEEKLY refs — PWH ${weekly.pwh?.toFixed(2)} · PWL ${weekly.pwl?.toFixed(2)}`
    : weekly.thin_history
      ? 'WEEKLY refs (thin)'
      : 'WEEKLY refs'
  const tooltip = [
    weekly.thin_history
      ? 'WEEKLY: refs only (thin history)'
      : 'WEEKLY: refs only — no directional call (class 50)',
    ...(weekly.narrative ? weekly.narrative.split('\n').slice(0, 3) : []),
  ].join('\n')

  return (
    <span
      data-testid="weekly-chip"
      title={tooltip}
      style={{
        fontSize: 9,
        fontWeight: 700,
        letterSpacing: '.08em',
        color: 'var(--vl-muted)',
        border: '1px solid var(--vl-hair)',
        borderRadius: 5,
        padding: '2px 6px',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      {label}
    </span>
  )
}
