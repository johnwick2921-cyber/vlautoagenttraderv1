// W13 — PLAN RE-ALIGNMENT ON OWNER EDIT (owner decision 2026-08-16).
//
// After an overlay save the planner re-examines the WHOLE plan. This renders the
// three owner-facing states, in place, under the levels table:
//   reviewing  → a quiet inline line ("⟳ planner reviewing your change…")
//   no-change  → a small chip that auto-fades ("✓ planner: no plan change needed")
//   proposal   → the patch card (the SAME PatchPreview Ask-Planner uses) with the
//                target version in the header + [✅ Apply] / [Keep as-is]
// Apply routes through the existing /plan/ask/apply — one mutation door. Keep
// leaves the plan untouched and logs the choice by simply never applying.

import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import { api } from '../../lib/api'
import { guardedCall } from '../../lib/api/guarded'
import type { RealignResponse } from '../../lib/api/plan'
import { PatchPreview } from './AskPlannerPanel'

export type RealignState =
  | { phase: 'idle' }
  | { phase: 'reviewing' }
  | { phase: 'no-change' }
  | { phase: 'capped' }
  | { phase: 'failed' }
  | { phase: 'proposal'; res: RealignResponse }

interface Props {
  state: RealignState
  traderId?: string
  symbol: string
  language: Language
  onDismiss: () => void
  onApplied: () => void // parent re-fetches → version chip bumps + rows flash
}

export function RealignPanel({
  state,
  traderId,
  symbol,
  language,
  onDismiss,
  onApplied,
}: Props) {
  const [busy, setBusy] = useState(false)

  // the no-change chip auto-fades (owner-specified: quiet, non-blocking)
  useEffect(() => {
    if (
      state.phase !== 'no-change' &&
      state.phase !== 'capped' &&
      state.phase !== 'failed'
    )
      return
    const id = setTimeout(onDismiss, state.phase === 'no-change' ? 4200 : 6000)
    return () => clearTimeout(id)
  }, [state.phase, onDismiss])

  if (state.phase === 'idle') return null

  if (state.phase === 'reviewing') {
    return (
      <div
        data-testid="realign-reviewing"
        className="px-3 py-1.5 text-[11px] flex items-center gap-2"
        style={{ color: 'var(--vl-muted)' }}
      >
        <span className="vl-spin" aria-hidden>
          ⟳
        </span>
        {tp('realignReviewing', language)}
      </div>
    )
  }

  if (
    state.phase === 'no-change' ||
    state.phase === 'capped' ||
    state.phase === 'failed'
  ) {
    const label =
      state.phase === 'no-change'
        ? `✓ ${tp('realignNoChange', language)}`
        : state.phase === 'capped'
          ? tp('realignCapped', language)
          : tp('realignFailed', language)
    const color =
      state.phase === 'no-change' ? 'var(--vl-long)' : 'var(--vl-faint)'
    return (
      <div
        data-testid={`realign-${state.phase}`}
        className="mx-3 my-1.5 px-2.5 py-1 text-[10.5px] rounded-md inline-block"
        style={{
          color,
          border: `1px solid ${color === 'var(--vl-long)' ? 'rgba(63,191,143,.25)' : 'var(--vl-hair)'}`,
        }}
      >
        {label}
      </div>
    )
  }

  // ── proposal ────────────────────────────────────────────────────────────────
  const { res } = state
  const patch = res.reply?.patch || ''
  const apply = async () => {
    if (!traderId || !res.qa_id || busy) return
    setBusy(true)
    const g = await guardedCall(() =>
      api.applyAsk(traderId, res.qa_id!, symbol)
    )
    setBusy(false)
    if (!g.ok || !g.value.ok) {
      toast.error(tp('saveFailed', language), {
        description: g.ok ? g.value.error : g.error,
      })
      return // panel recovers; apply stays available
    }
    toast.success(tp('overlayApplied', language))
    onApplied()
    onDismiss()
  }

  return (
    <div
      data-testid="realign-proposal"
      className="mx-3 my-2 rounded-[10px] overflow-hidden"
      style={{
        border: '1px solid var(--vl-gold-line)',
        background: 'var(--vl-card-2)',
      }}
    >
      <div
        className="px-3 py-1.5 text-[8.5px] font-bold tracking-[.12em] flex items-center gap-2"
        style={{ color: 'var(--vl-gold)', background: 'var(--vl-gold-dim)' }}
      >
        {tp('realignProposal', language).toUpperCase()}
        {res.would_become && (
          <span
            style={{
              color: 'var(--vl-muted)',
              fontWeight: 500,
              letterSpacing: '.04em',
            }}
          >
            · {tp('realignWouldBecome', language)}{' '}
            <span className="vl-num">{res.would_become}</span>
          </span>
        )}
      </div>

      {res.reply?.evidence && (
        <div
          className="px-3 pt-2 text-[11px] leading-[1.5]"
          style={{ color: 'var(--vl-ivory)', opacity: 0.85 }}
        >
          {res.reply.evidence}
        </div>
      )}
      {res.reply?.summary && (
        <div
          className="px-3 pt-1.5 text-[11px] leading-[1.5]"
          style={{ color: 'var(--vl-muted)' }}
        >
          {res.reply.summary}
        </div>
      )}

      <div className="pt-2">
        <PatchPreview patch={patch} />
      </div>

      <div className="flex gap-2 px-3 pb-3">
        <button
          onClick={apply}
          disabled={busy}
          className="flex-1 text-[11px] font-semibold py-2 rounded-lg"
          style={{
            background: 'var(--vl-gold)',
            color: 'var(--vl-ink)',
            opacity: busy ? 0.6 : 1,
          }}
        >
          ✅ {tp('applyMerge', language)}
        </button>
        <button
          onClick={onDismiss}
          className="flex-1 text-[11px] font-semibold py-2 rounded-lg"
          style={{
            border: '1px solid var(--vl-hair)',
            color: 'var(--vl-muted)',
          }}
        >
          {tp('keepAsIs', language)}
        </button>
      </div>
    </div>
  )
}

// RealignButton — the manual fallback affordance. Always available (it is also the
// escape hatch once the auto cap is spent); a manual call bypasses cap + debounce.
export function RealignButton({
  language,
  disabled,
  onClick,
}: {
  language: Language
  disabled?: boolean
  onClick: () => void
}) {
  return (
    <button
      data-testid="realign-button"
      onClick={onClick}
      disabled={disabled}
      className="text-[10px] px-2 py-1 rounded-md"
      style={{
        border: '1px solid var(--vl-hair)',
        color: disabled ? 'var(--vl-faint)' : 'var(--vl-muted)',
        opacity: disabled ? 0.5 : 1,
      }}
    >
      ⟳ {tp('realignButton', language)}
    </button>
  )
}
