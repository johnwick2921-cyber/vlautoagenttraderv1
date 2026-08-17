// ITEM 3 (2026-08-17) — the owner's manual escape hatch.
//
// The planner only reads on its own schedule and on death. When a plan is wrong
// but not dead, there was no way to ask for a fresh one short of waiting. This
// button spends ONE re-read from the same session budget the automatic path uses
// — so it can never talk the bot past its own limits — and says so BEFORE
// spending, because it costs a real API call.
//
// The server re-checks eligibility on the POST, so a stale gate here cannot buy
// an extra read.

import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import { api } from '../../lib/api'
import type { RereadGate } from '../../lib/api/plan'

export function RereadButton({
  traderId,
  language,
  onDone,
}: {
  traderId?: string
  language: Language
  /** re-fetch so the new version appears */
  onDone?: () => void
}) {
  const [gate, setGate] = useState<RereadGate | null>(null)
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (!traderId) return
    let stop = false
    const load = async () => {
      const g = await api.getRereadGate(traderId)
      if (!stop) setGate(g)
    }
    void load()
    const id = setInterval(load, 30_000)
    return () => {
      stop = true
      clearInterval(id)
    }
  }, [traderId])

  if (!traderId || !gate) return null

  const disabled = !gate.allowed || busy

  const run = async () => {
    setBusy(true)
    const res = await api.forceReread(traderId)
    setBusy(false)
    setConfirming(false)
    if (res.ok) {
      toast.success(tp('rereadDone', language))
      // The planner call is asynchronous on the server; give it a beat before
      // re-fetching so the new version is likely to be there.
      setTimeout(() => onDone?.(), 1500)
      setGate(await api.getRereadGate(traderId))
    } else {
      toast.error(res.error || tp('errorFailClosed', language))
      setGate(res.gate ?? (await api.getRereadGate(traderId)))
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
      <button
        type="button"
        data-testid="reread-button"
        disabled={disabled}
        title={gate.allowed ? undefined : gate.reason}
        onClick={() => setConfirming(true)}
        className="text-[11px] px-2 py-1"
        style={{
          color: disabled ? 'var(--vl-faint)' : 'var(--vl-gold)',
          border: `1px solid ${disabled ? 'var(--vl-hair)' : 'var(--vl-gold-line)'}`,
          borderRadius: 'var(--vl-radius-chip)',
          background: 'transparent',
          cursor: disabled ? 'not-allowed' : 'pointer',
          minHeight: 32,
          fontFamily: 'var(--vl-font-ui)',
        }}
      >
        {busy ? (
          <>
            <span className="vl-spin" aria-hidden>
              ◌
            </span>{' '}
            {tp('rereadRunning', language)}
          </>
        ) : (
          <>⟳ {tp('rereadTitle', language)}</>
        )}
      </button>

      {/* The refusal is never silent: a disabled control says why. */}
      {!gate.allowed && gate.reason && (
        <span
          data-testid="reread-reason"
          className="text-[10px] text-right max-w-[220px]"
          style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
        >
          {gate.reason}
        </span>
      )}

      {/* Confirm BEFORE spending — it costs an API call and one re-read. */}
      {confirming && gate.allowed && (
        <div
          data-testid="reread-confirm"
          className="flex flex-col gap-1.5 px-2.5 py-2 mt-1"
          style={{
            background: 'var(--vl-card-2)',
            border: '1px solid var(--vl-gold-line)',
            borderRadius: 'var(--vl-radius-inner)',
            fontFamily: 'var(--vl-font-ui)',
            maxWidth: 260,
          }}
        >
          <span className="text-[11px]" style={{ color: 'var(--vl-ivory)' }}>
            {tp('rereadConfirm', language, { left: String(gate.replans_left) })}
          </span>
          <span className="text-[10px]" style={{ color: 'var(--vl-muted)' }}>
            {tp('rereadConfirmHint', language)}
          </span>
          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={() => setConfirming(false)}
              className="text-[11px] px-2 py-1"
              style={{
                color: 'var(--vl-muted)',
                background: 'transparent',
                border: 0,
                cursor: 'pointer',
              }}
            >
              {tp('rereadCancel', language)}
            </button>
            <button
              type="button"
              data-testid="reread-go"
              onClick={run}
              className="text-[11px] px-2 py-1"
              style={{
                color: 'var(--vl-ink)',
                background: 'var(--vl-gold)',
                border: 0,
                borderRadius: 'var(--vl-radius-chip)',
                cursor: 'pointer',
              }}
            >
              {tp('rereadGo', language)}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
