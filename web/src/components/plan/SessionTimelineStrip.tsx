// P4.2 — the 24h session strip: asia/london/ny bands (session hues) + now-marker
// + killzone ticks + a DST-warning variant. Purely visual (tokens only); the
// backend owns the real active-session state (passed in as `activeSession`).

import { useEffect, useState } from 'react'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import {
  SESSION_BANDS,
  DAY_MINUTES,
  nowCTMinutes,
  toSegments,
  londonDSTWarning,
  type SessionName,
} from './sessionConfig'

interface Props {
  activeSession?: SessionName | null
  language: Language
  now?: Date // injectable for tests; defaults to a live clock
}

const pct = (min: number) => `${(min / DAY_MINUTES) * 100}%`

export function SessionTimelineStrip({ activeSession, language, now }: Props) {
  // Live clock (30s tick) unless a fixed `now` is injected (tests).
  const [tick, setTick] = useState(0)
  useEffect(() => {
    if (now) return
    const id = setInterval(() => setTick((t) => t + 1), 30_000)
    return () => clearInterval(id)
  }, [now])
  void tick

  const ref = now ?? new Date()
  const nowMin = nowCTMinutes(ref)
  const dst = londonDSTWarning(ref)

  const sessionLabel = (name: SessionName) =>
    tp(
      name === 'ASIA'
        ? 'sessionAsia'
        : name === 'LONDON'
          ? 'sessionLondon'
          : 'sessionNY',
      language
    )

  return (
    <div
      role="img"
      aria-label={`Session timeline. ${activeSession ? `Active: ${sessionLabel(activeSession)}.` : ''} ${tp('now', language)} marker.`}
      className="w-full"
    >
      <div
        className="relative w-full overflow-hidden"
        style={{
          height: 34,
          borderRadius: 'var(--vl-radius-inner)',
          background: 'var(--vl-card-2)',
          border: '1px solid var(--vl-hair)',
        }}
      >
        {/* session bands */}
        {SESSION_BANDS.map((band) =>
          toSegments(band.startMin, band.endMin).map(([s, e], i) => {
            const isActive = activeSession === band.name
            return (
              <div
                key={`${band.name}-${i}`}
                className="absolute top-0 bottom-0"
                style={{
                  left: pct(s),
                  width: pct(e - s),
                  background: band.hue,
                  opacity: isActive ? 0.42 : band.enabled ? 0.2 : 0.1,
                  borderRight: isActive
                    ? '1px solid var(--vl-gold-line)'
                    : undefined,
                }}
                title={sessionLabel(band.name)}
              />
            )
          })
        )}

        {/* killzone ticks (thin gold fills inside sessions) */}
        {SESSION_BANDS.flatMap((band) =>
          band.killzones.flatMap((kz) =>
            toSegments(kz.startMin, kz.endMin).map(([s, e], i) => (
              <div
                key={`${kz.name}-${i}`}
                className="absolute"
                style={{
                  left: pct(s),
                  width: pct(e - s),
                  top: 0,
                  height: 5,
                  background: 'var(--vl-killzone-fill)',
                }}
                title={`${tp('killzone', language)}: ${band.name}`}
              />
            ))
          )
        )}

        {/* now marker */}
        <div
          className="absolute top-0 bottom-0"
          aria-hidden
          style={{
            left: pct(nowMin),
            width: 2,
            background: 'var(--vl-ivory)',
            boxShadow: '0 0 4px var(--vl-ivory)',
          }}
        />
      </div>

      {/* legend + DST warning */}
      <div
        className="mt-1 flex items-center justify-between text-[10px]"
        style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
      >
        <div className="flex items-center gap-3">
          {SESSION_BANDS.map((band) => (
            <span key={band.name} className="inline-flex items-center gap-1">
              <span
                aria-hidden
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: 2,
                  background: band.hue,
                  opacity: activeSession === band.name ? 1 : 0.6,
                }}
              />
              <span
                style={{
                  color:
                    activeSession === band.name
                      ? 'var(--vl-ivory)'
                      : 'var(--vl-faint)',
                }}
              >
                {sessionLabel(band.name)}
              </span>
            </span>
          ))}
        </div>
        {dst && (
          <span
            style={{ color: 'var(--vl-short)' }}
            title={tp('dstWarning', language)}
          >
            ⚠ {tp('dstWarning', language)}
          </span>
        )}
      </div>
    </div>
  )
}
