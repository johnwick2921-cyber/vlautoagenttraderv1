// P4.2 — the handover banner. Phases: expired → reading → born →
// read-failed (fail-closed). Announced via aria-live so the session rollover is
// heard, not just seen.

import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { SessionName } from './sessionConfig'

export type HandoverPhase = 'expired' | 'reading' | 'born' | 'read-failed'

interface Props {
  phase: HandoverPhase
  session: SessionName
  language: Language
  onDismiss?: () => void
}

export function HandoverBanner({ phase, session, language, onDismiss }: Props) {
  const sessionLabel = tp(
    session === 'ASIA'
      ? 'sessionAsia'
      : session === 'LONDON'
        ? 'sessionLondon'
        : 'sessionNY',
    language
  )

  const cfg: Record<
    HandoverPhase,
    {
      color: string
      bg: string
      key:
        | 'handoverExpired'
        | 'handoverReading'
        | 'handoverBorn'
        | 'handoverReadFailed'
      pulse?: boolean
    }
  > = {
    expired: {
      color: 'var(--vl-faint)',
      bg: 'var(--vl-card-2)',
      key: 'handoverExpired',
    },
    reading: {
      color: 'var(--vl-gold)',
      bg: 'var(--vl-gold-dim)',
      key: 'handoverReading',
      pulse: true,
    },
    born: {
      color: 'var(--vl-long)',
      bg: 'rgba(63,191,143,0.12)',
      key: 'handoverBorn',
    },
    'read-failed': {
      color: 'var(--vl-short)',
      bg: 'rgba(224,108,108,0.12)',
      key: 'handoverReadFailed',
    },
  }
  const m = cfg[phase]

  return (
    <div
      role="status"
      aria-live="polite"
      className="flex items-center justify-between gap-2 px-3 py-2 text-xs"
      style={{
        color: m.color,
        background: m.bg,
        border: `1px solid ${m.color}`,
        borderRadius: 'var(--vl-radius-inner)',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      <span className="flex items-center gap-2">
        <span
          aria-hidden
          className={m.pulse ? 'vl-pulse' : undefined}
          style={{
            width: 7,
            height: 7,
            borderRadius: 999,
            background: m.color,
            display: 'inline-block',
          }}
        />
        <span className="font-semibold">
          {tp(m.key, language, { session: sessionLabel })}
        </span>
      </span>
      {onDismiss && (
        <button
          onClick={onDismiss}
          aria-label={tp('dismiss', language)}
          className="opacity-70 hover:opacity-100"
          style={{ color: m.color }}
        >
          ✕
        </button>
      )}
    </div>
  )
}
