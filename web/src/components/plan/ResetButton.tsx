// P6 (2026-08-17) — THE OWNER RESET, the escape hatch ABOVE the re-read.
//
// ⟳ Re-read spends ONE budget item on the SAME chain. Reset ABANDONS the chain
// (history and death reasons preserved — plans are append-only), restores the
// whole re-plan budget, clears the NO-TRADE state, and writes a fresh plan with
// trigger_reason "owner reset". Open positions and their brackets are never
// touched. Confirm-before-spend; disabled with a stated reason when the session
// is disabled / night / market closed / no plan yet. The server re-checks
// eligibility on the POST.

import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import { api } from '../../lib/api'
import type { ResetGate } from '../../lib/api/plan'

export function ResetButton({
  traderId,
  language,
  onDone,
}: {
  traderId?: string
  language: Language
  /** re-fetch so the new plan appears */
  onDone?: () => void
}) {
  const [gate, setGate] = useState<ResetGate | null>(null)
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (!traderId) return
    let stop = false
    const load = async () => {
      const g = await api.getResetGate(traderId)
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
    const res = await api.forceReset(traderId)
    setBusy(false)
    setConfirming(false)
    if (res.ok) {
      toast.success(tp('resetDone', language))
      if (res.note) {
        toast.warning(res.note)
      }
      setTimeout(() => onDone?.(), 1500)
      setGate(await api.getResetGate(traderId))
    } else {
      toast.error(res.error || tp('errorFailClosed', language))
      setGate(res.gate ?? (await api.getResetGate(traderId)))
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
      <button
        type="button"
        data-testid="reset-button"
        disabled={disabled}
        title={gate.allowed ? undefined : gate.reason}
        onClick={() => setConfirming(true)}
        className="text-[11px] px-2 py-1"
        style={{
          color: disabled ? 'var(--vl-faint)' : 'var(--vl-short)',
          border: `1px solid ${disabled ? 'var(--vl-hair)' : 'rgba(224,108,108,0.4)'}`,
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
            {tp('resetRunning', language)}
          </>
        ) : (
          <>↺ {tp('resetTitle', language)}</>
        )}
      </button>

      {/* The refusal is never silent: a disabled control says why. */}
      {!gate.allowed && gate.reason && (
        <span
          data-testid="reset-reason"
          className="text-[10px] text-right max-w-[220px]"
          style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
        >
          {gate.reason}
        </span>
      )}

      {/* Confirm BEFORE the reset — it abandons the current chain. */}
      {confirming && gate.allowed && (
        <div
          data-testid="reset-confirm"
          className="flex flex-col gap-1.5 px-2.5 py-2 mt-1"
          style={{
            background: 'var(--vl-card-2)',
            border: '1px solid rgba(224,108,108,0.4)',
            borderRadius: 'var(--vl-radius-inner)',
            fontFamily: 'var(--vl-font-ui)',
            maxWidth: 280,
          }}
        >
          <span className="text-[11px]" style={{ color: 'var(--vl-ivory)' }}>
            {tp('resetConfirm', language)}
          </span>
          <span className="text-[10px]" style={{ color: 'var(--vl-muted)' }}>
            {tp('resetConfirmHint', language)}
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
              data-testid="reset-go"
              onClick={run}
              className="text-[11px] px-2 py-1"
              style={{
                color: 'var(--vl-ink)',
                background: 'var(--vl-short)',
                border: 0,
                borderRadius: 'var(--vl-radius-chip)',
                cursor: 'pointer',
              }}
            >
              {tp('resetGo', language)}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
