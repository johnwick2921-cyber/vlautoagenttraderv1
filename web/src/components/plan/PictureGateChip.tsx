// W-EXEC-TRUTH W0 (CTO Q6) → W5 — Picture HTF's plan verdict on the day-plan
// card, READ from the server (picture.route / picture.refusal — the same
// verdicts the entry gate acts on and the 📷 boot line prints), rendered
// verbatim; the card never composes its own.
//   route   → a NEUTRAL chip: how Picture reaches the market ("📷 PICTURE →
//             Day Plan scenario (market_in_zone limit)"). Not a refusal.
//   refusal → the RED chip, only for a real refusal.
// Renders nothing when Picture is off for this trader or the read is absent.
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { PicturePlanGate } from '../../lib/api/plan'

const chipBase = {
  fontSize: 9,
  fontWeight: 700,
  letterSpacing: '.08em',
  borderRadius: 5,
  padding: '2px 6px',
  fontFamily: 'var(--vl-font-ui)',
} as const

export function PictureGateChip({
  picture,
  language = 'en',
}: {
  picture?: PicturePlanGate | null
  language?: Language
}) {
  if (!picture || !picture.enabled) return null
  const refusal = picture.refusal?.trim() ?? ''
  const route = picture.route?.trim() ?? ''
  if (!refusal && !route) return null
  return (
    <>
      {refusal && (
        <span
          data-testid="picture-gate-chip"
          title={`Picture HTF: ${refusal}`}
          style={{
            ...chipBase,
            color: 'var(--vl-short)',
            border: '1px solid rgba(224,108,108,.4)',
            background: 'rgba(224,108,108,.08)',
          }}
        >
          📷 PICTURE {refusal}
        </span>
      )}
      {route && (
        <span
          data-testid="picture-route-chip"
          title={tp('pictureRouteTitle', language, { route })}
          style={{
            ...chipBase,
            color: 'var(--vl-muted)',
            border: '1px solid var(--vl-hair)',
          }}
        >
          {tp('pictureSourceBadge', language)} → {route}
        </span>
      )}
    </>
  )
}
