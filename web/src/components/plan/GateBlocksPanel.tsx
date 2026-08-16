// W16/R3 — "why was my entry refused?" made answerable from the UI.
//
// telemetry.IncGateBlock has counted every refusal since B6, and
// GET /api/risk/gate-blocks has served the table the whole time — with ZERO
// frontend consumers. The counters were real and invisible.
//
// This panel is deliberately honest about what the counter IS: an in-memory
// tally for the CURRENT CME session-day that resets at 17:00 CT and on restart.
// It says so, rather than letting an empty table read as "nothing was blocked".
// The per-decision reason lives on the decision row itself (DecisionAudit).

import { useEffect, useState } from 'react'
import { tp } from '../../i18n/plan-translations'
import type { Language } from '../../i18n/translations'
import { planApi } from '../../lib/api/plan'

// Human labels for the gate names telemetry emits. An unknown gate falls back to
// its raw name rather than being hidden — a refusal we cannot label is still a
// refusal the owner must see.
const GATE_LABELS: Record<string, { en: string; icon: string }> = {
  feed_down: { en: 'NT8 feed down', icon: '📡' },
  dead_man: { en: 'Dead-man watchdog', icon: '⏱' },
  frozen: { en: 'Trader frozen', icon: '🧊' },
  boot_integrity: { en: 'Boot integrity', icon: '🔐' },
  consecutive_loss: { en: 'Consecutive-loss halt', icon: '🛑' },
  last_entry: { en: 'Past last-entry time', icon: '🕒' },
  session_gate: { en: 'Outside session window', icon: '🗓' },
  plan_mode: { en: 'Against the plan', icon: '📋' },
  approval: { en: 'Awaiting approval', icon: '✋' },
  clock_drift: { en: 'Clock drift', icon: '⏰' },
  b3_order_dedup: { en: 'Duplicate order dropped', icon: '👯' },
  b3_rate_breaker: { en: 'Order rate breaker', icon: '🚨' },
  level_burned_retouch: { en: 'Burned level re-touched', icon: '🔥' },
  night_transition: { en: 'Night/day transition', icon: '🌙' },
}

export function GateBlocksPanel({
  traderId,
  language,
}: {
  traderId?: string
  language: Language
}) {
  const [rows, setRows] = useState<{ gate: string; count: number }[]>([])
  const [sessionDay, setSessionDay] = useState('')
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    if (!traderId) return
    let alive = true
    const load = async () => {
      const res = await planApi.getGateBlocks()
      if (!alive || !res) return
      const mine = res.by_trader?.[traderId] ?? {}
      // Process-wide gates (B3) are filed under the empty trader id; show them too
      // rather than dropping a real refusal on a technicality.
      const shared = res.by_trader?.[''] ?? {}
      const merged: Record<string, number> = { ...mine }
      for (const [g, n] of Object.entries(shared))
        merged[g] = (merged[g] ?? 0) + n
      setRows(
        Object.entries(merged)
          .map(([gate, count]) => ({ gate, count }))
          .sort((a, b) => b.count - a.count)
      )
      setSessionDay(res.session_day_utc ?? '')
      setLoaded(true)
    }
    void load()
    const t = setInterval(load, 20_000)
    return () => {
      alive = false
      clearInterval(t)
    }
  }, [traderId])

  if (!traderId || !loaded) return null

  return (
    <div
      data-testid="gate-blocks-panel"
      className="p-3 flex flex-col gap-2"
      style={{
        background: 'var(--vl-card)',
        border: '1px solid var(--vl-hair)',
        borderRadius: 'var(--vl-radius-card)',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      <div className="flex items-baseline justify-between">
        <span
          className="text-[10px] uppercase tracking-widest"
          style={{ color: 'var(--vl-faint)' }}
        >
          {tp('gateBlocksHeader', language)}
        </span>
        <span className="text-[9px]" style={{ color: 'var(--vl-faint)' }}>
          {sessionDay ? sessionDay.slice(0, 10) : ''}
        </span>
      </div>

      {rows.length === 0 ? (
        <div
          data-testid="gate-blocks-empty"
          className="text-[11px]"
          style={{ color: 'var(--vl-muted)' }}
        >
          {tp('gateBlocksNone', language)}
        </div>
      ) : (
        <div className="flex flex-col gap-1">
          {rows.map((r) => {
            const meta = GATE_LABELS[r.gate]
            return (
              <div
                key={r.gate}
                data-testid={`gate-block-${r.gate}`}
                className="flex items-center justify-between text-[11px]"
              >
                <span style={{ color: 'var(--vl-ivory)' }}>
                  {meta ? `${meta.icon} ${meta.en}` : r.gate}
                </span>
                <span
                  className="vl-num px-1.5 rounded"
                  style={{
                    color: 'var(--vl-short)',
                    background: 'rgba(224,108,108,0.10)',
                  }}
                >
                  {r.count}
                </span>
              </div>
            )
          })}
        </div>
      )}

      <div
        className="text-[9px] leading-snug"
        style={{ color: 'var(--vl-faint)' }}
      >
        {tp('gateBlocksNote', language)}
      </div>
    </div>
  )
}

export default GateBlocksPanel
