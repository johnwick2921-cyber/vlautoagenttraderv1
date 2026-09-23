// W-EXEC-TRUTH W0 (CTO Q6) — Picture HTF's plan-mode verdict on the day-plan
// card. Under plan_mode=strict Picture is REFUSED until W5 makes it a Day Plan
// scenario, and the refusal must be visible. The text is the server's READ
// (picture.refusal — the same verdict the entry gate refuses on and the 📷
// boot line prints), rendered verbatim; the card never composes its own.
// Renders nothing when Picture is off for this trader or plan mode admits it.
import type { PicturePlanGate } from '../../lib/api/plan'

export function PictureGateChip({
  picture,
}: {
  picture?: PicturePlanGate | null
}) {
  if (!picture || !picture.enabled || !picture.refusal) return null
  return (
    <span
      data-testid="picture-gate-chip"
      title={`Picture HTF: ${picture.refusal}`}
      style={{
        fontSize: 9,
        fontWeight: 700,
        letterSpacing: '.08em',
        color: 'var(--vl-short)',
        border: '1px solid rgba(224,108,108,.4)',
        background: 'rgba(224,108,108,.08)',
        borderRadius: 5,
        padding: '2px 6px',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      📷 PICTURE {picture.refusal}
    </span>
  )
}
