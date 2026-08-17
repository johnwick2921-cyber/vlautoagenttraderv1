// P5.4 (FE) — Ask-Planner: bottom-sheet over the dashboard on mobile, right-side
// panel on desktop (≥1200px, portaled out of the narrow grid column). Renders the
// thread + the structured reply form (EVIDENCE → YOUR POINT[chip] → VERDICT[chip]
// + patch preview) exactly per askplanner-mockup. A patch applies ONLY on an
// explicit tap; bare disagreements never carry one (enforced server-side too).

import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { toast } from 'sonner'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import { api } from '../../lib/api'
import type { PlanQAMessage, PatchOp } from '../../lib/api/plan'

interface Props {
  open: boolean
  traderId: string
  symbol: string
  planId?: string
  planVersion?: number
  language: Language
  onClose: () => void
  onApplied: () => void // parent re-fetches so the card version-bumps
  /**
   * ITEM 2 — set when the thread is NOT about a live plan. The backend already
   * strips any patch outside an ACTIVE context (so Apply is structurally absent,
   * not merely disabled); this is the human-readable half of that contract.
   */
  contextLabel?: string
}

function useIsWide(threshold = 1200) {
  const [wide, setWide] = useState(() =>
    typeof window !== 'undefined' ? window.innerWidth >= threshold : false
  )
  useEffect(() => {
    const onResize = () => setWide(window.innerWidth >= threshold)
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [threshold])
  return wide
}

export function PatchPreview({ patch }: { patch: string }) {
  let ops: PatchOp[] = []
  try {
    ops = JSON.parse(patch)
  } catch {
    return null
  }
  if (!ops.length) return null
  return (
    <div
      style={{
        margin: '0 0 8px',
        border: '1px solid var(--vl-gold-line)',
        borderRadius: 9,
        overflow: 'hidden',
      }}
    >
      {ops.map((op, i) => {
        const add = op.op === 'add' || op.op === 'replace'
        return (
          <div
            key={i}
            className="flex gap-2 px-2.5 py-1 text-[10.5px]"
            style={{
              color: add ? 'var(--vl-long)' : 'var(--vl-short)',
              borderTop: i ? '1px solid var(--vl-hair)' : undefined,
            }}
          >
            <span>{add ? '+' : '−'}</span>
            <span style={{ color: 'var(--vl-faint)' }}>{op.op}</span>
            <span className="vl-num" style={{ fontWeight: 600 }}>
              {op.path}
            </span>
            {op.value !== undefined && (
              <span
                className="vl-num truncate"
                style={{ color: 'var(--vl-muted)' }}
              >
                {JSON.stringify(op.value)}
              </span>
            )}
          </div>
        )
      })}
    </div>
  )
}

function PlannerReply({
  m,
  traderId,
  symbol,
  language,
  onApplied,
}: {
  m: PlanQAMessage
  traderId: string
  symbol: string
  language: Language
  onApplied: () => void
}) {
  // W16/R2 — TRI-state. This used to be one boolean that BOTH buttons set, so
  // declining rendered the applied confirmation ("Applied — card updated") while
  // persisting nothing. 'declined' is now its own outcome and its own record.
  const [outcome, setOutcome] = useState<'applied' | 'declined' | null>(
    m.applied ? 'applied' : null
  )
  const [busy, setBusy] = useState(false)
  const isNewInfo = m.point_class === 'NEW-INFO'
  const vColor =
    m.verdict === 'PROPOSE-MERGE'
      ? 'var(--vl-gold)'
      : m.verdict === 'DEFEND'
        ? 'var(--vl-short)'
        : 'var(--vl-muted)'
  const vLabel =
    m.verdict === 'PROPOSE-MERGE'
      ? tp('vMerge', language)
      : m.verdict === 'DEFEND'
        ? tp('vDefend', language)
        : tp('vConcede', language)

  const apply = async () => {
    setBusy(true)
    const res = await api.applyAsk(traderId, m.id, symbol)
    setBusy(false)
    if (!res.ok) {
      toast.error(tp('saveFailed', language), { description: res.error })
      return
    }
    setOutcome('applied')
    toast.success(tp('askApplied', language))
    onApplied()
  }

  const decline = async () => {
    setBusy(true)
    const res = await api.declineAsk(traderId, m.id)
    setBusy(false)
    if (!res.ok) {
      toast.error(tp('saveFailed', language), { description: res.error })
      return // stay undecided rather than claim an outcome we did not persist
    }
    setOutcome('declined')
    toast.success(tp('askDeclined', language))
  }

  return (
    <div
      className="self-start w-[96%]"
      style={{
        background: 'rgba(232,227,216,0.025)',
        border: '1px solid var(--vl-hair)',
        borderRadius: '12px 12px 12px 3px',
        overflow: 'hidden',
      }}
    >
      <div
        className="px-3 pt-2 pb-1 text-[9px] font-bold tracking-widest"
        style={{ color: 'var(--vl-muted)' }}
      >
        🤖 PLANNER · {vLabel}
      </div>
      {m.evidence && (
        <div
          className="px-3 py-1.5"
          style={{ borderTop: '1px solid var(--vl-hair)' }}
        >
          <div
            className="text-[8.5px] font-bold tracking-widest"
            style={{ color: 'var(--vl-gold)' }}
          >
            {tp('askEvidence', language)}
          </div>
          <div
            className="text-[11px] mt-1"
            style={{ color: 'var(--vl-ivory)', opacity: 0.85, lineHeight: 1.5 }}
          >
            {m.evidence}
          </div>
        </div>
      )}
      {m.point_class && (
        <div
          className="px-3 py-1.5"
          style={{ borderTop: '1px solid var(--vl-hair)' }}
        >
          <div
            className="text-[8.5px] font-bold tracking-widest flex items-center gap-1.5"
            style={{ color: 'var(--vl-gold)' }}
          >
            {tp('askYourPoint', language)}
            <span
              className="text-[9px] font-bold"
              style={{
                padding: '1.5px 6px',
                borderRadius: 4,
                color: isNewInfo ? 'var(--vl-long)' : 'var(--vl-faint)',
                border: `1px solid ${isNewInfo ? 'rgba(63,191,143,0.3)' : 'var(--vl-hair)'}`,
              }}
            >
              {isNewInfo ? tp('clsNewInfo', language) : tp('clsBare', language)}
            </span>
          </div>
        </div>
      )}
      <div
        className="flex items-center gap-2 px-3 py-2"
        style={{ borderTop: '1px solid var(--vl-hair)' }}
      >
        <span
          className="text-[9.5px] font-bold"
          style={{
            padding: '4px 9px',
            borderRadius: 6,
            color: m.verdict === 'PROPOSE-MERGE' ? 'var(--vl-ink)' : vColor,
            background:
              m.verdict === 'PROPOSE-MERGE' ? 'var(--vl-gold)' : 'transparent',
            border:
              m.verdict === 'PROPOSE-MERGE' ? undefined : `1px solid ${vColor}`,
          }}
        >
          {vLabel}
        </span>
        <span className="text-[10px]" style={{ color: 'var(--vl-muted)' }}>
          {m.content ||
            (m.verdict === 'PROPOSE-MERGE' ? tp('patchNote', language) : '')}
        </span>
      </div>
      {m.verdict === 'PROPOSE-MERGE' && m.patch && (
        <div className="px-3">
          <PatchPreview patch={m.patch} />
          {outcome === null ? (
            <div className="flex gap-2 pb-3">
              <button
                onClick={apply}
                disabled={busy}
                className="flex-1 text-[11px] font-semibold py-2"
                style={{
                  background: 'var(--vl-gold)',
                  color: 'var(--vl-ink)',
                  borderRadius: 8,
                  opacity: busy ? 0.6 : 1,
                }}
              >
                ✅ {tp('applyMerge', language)}
              </button>
              <button
                onClick={decline}
                disabled={busy}
                data-testid="ask-keep-as-is"
                className="flex-1 text-[11px] py-2"
                style={{
                  border: '1px solid var(--vl-hair)',
                  color: 'var(--vl-muted)',
                  borderRadius: 8,
                }}
              >
                {tp('keepAsIs', language)}
              </button>
            </div>
          ) : outcome === 'applied' ? (
            <div
              className="mb-3 text-[10.5px] px-3 py-2"
              data-testid="ask-outcome-applied"
              style={{
                color: 'var(--vl-long)',
                border: '1px solid rgba(63,191,143,0.25)',
                background: 'rgba(63,191,143,0.06)',
                borderRadius: 10,
              }}
            >
              ✔ {tp('askApplied', language)}
            </div>
          ) : (
            <div
              className="mb-3 text-[10.5px] px-3 py-2"
              data-testid="ask-outcome-declined"
              style={{
                color: 'var(--vl-muted)',
                border: '1px solid var(--vl-hair)',
                borderRadius: 10,
              }}
            >
              — {tp('askDeclinedNote', language)}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export function AskPlannerPanel({
  open,
  traderId,
  symbol,
  planId,
  planVersion,
  language,
  onClose,
  onApplied,
  contextLabel,
}: Props) {
  const wide = useIsWide()
  const [thread, setThread] = useState<PlanQAMessage[]>([])
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const bodyRef = useRef<HTMLDivElement>(null)

  const refresh = async () => {
    const res = await api.getPlanThread(traderId, planId)
    setThread(res.thread)
    setTimeout(() => {
      const el = bodyRef.current
      if (el && typeof el.scrollTo === 'function')
        el.scrollTo(0, el.scrollHeight)
    }, 30)
  }

  useEffect(() => {
    if (open) void refresh()
  }, [open, planId])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const send = async (q: string) => {
    const question = q.trim()
    if (!question || busy) return
    setBusy(true)
    setInput('')
    const res = await api.askPlanner(traderId, question, symbol)
    setBusy(false)
    if (!res.ok) {
      toast.error(tp('saveFailed', language), { description: res.error })
    }
    await refresh()
  }

  const shellStyle: React.CSSProperties = wide
    ? { position: 'fixed', top: 0, right: 0, bottom: 0, width: 400, zIndex: 60 }
    : {
        position: 'fixed',
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: 60,
        maxHeight: '85vh',
      }

  const panel = (
    <div
      className="fixed inset-0 z-[59]"
      style={{ background: 'rgba(0,0,0,0.5)' }}
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={tp('askPlannerTitle', language)}
        className="vl-sheet-in flex flex-col"
        style={{
          ...shellStyle,
          background: 'linear-gradient(180deg,var(--vl-card-2),var(--vl-card))',
          border: '1px solid var(--vl-gold-line)',
          borderRadius: wide ? 0 : '16px 16px 0 0',
          fontFamily: 'var(--vl-font-ui)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div
          className="flex items-center gap-2 px-4 py-3"
          style={{ borderBottom: '1px solid var(--vl-hair)' }}
        >
          <span
            style={{
              fontFamily: 'var(--vl-font-display)',
              fontSize: 18,
              fontWeight: 600,
              flex: 1,
              color: 'var(--vl-ivory)',
            }}
          >
            {tp('askPlannerTitle', language)}
          </span>
          <span
            className="text-[9px] font-bold tracking-wider"
            style={{
              color: 'var(--vl-gold)',
              border: '1px solid var(--vl-gold-line)',
              background: 'var(--vl-gold-dim)',
              borderRadius: 5,
              padding: '3px 7px',
            }}
          >
            {planId
              ? `${planId.split(':').pop()} · v${planVersion ?? 1}`
              : `v${planVersion ?? 1}`}
          </span>
          <button
            onClick={onClose}
            aria-label="close"
            style={{ color: 'var(--vl-muted)' }}
          >
            ✕
          </button>
        </div>
        {contextLabel && (
          <div
            data-testid="ask-context-label"
            className="px-4 py-2 text-[11px]"
            style={{
              background: 'rgba(224,108,108,0.08)',
              borderBottom: '1px solid rgba(224,108,108,0.3)',
              color: 'var(--vl-short)',
              fontFamily: 'var(--vl-font-ui)',
            }}
          >
            ⚠ {contextLabel}
          </div>
        )}

        <div
          ref={bodyRef}
          className="flex-1 flex flex-col gap-2.5 px-3 py-3"
          style={{ overflowY: 'auto', minHeight: 200 }}
        >
          {thread.length === 0 && (
            <div
              className="text-[11px] text-center py-6"
              style={{ color: 'var(--vl-faint)' }}
            >
              {tp('askPlaceholder', language)}
            </div>
          )}
          {thread.map((m) =>
            m.role === 'owner' ? (
              <div
                key={m.id}
                className="self-end max-w-[85%] text-[12px] px-3 py-2"
                style={{
                  background: 'var(--vl-gold-dim)',
                  border: '1px solid var(--vl-gold-line)',
                  borderRadius: '12px 12px 3px 12px',
                  color: 'var(--vl-ivory)',
                }}
              >
                {m.content}
              </div>
            ) : (
              <PlannerReply
                key={m.id}
                m={m}
                traderId={traderId}
                symbol={symbol}
                language={language}
                onApplied={onApplied}
              />
            )
          )}
          {busy && (
            <div
              className="self-start text-[11px] px-2"
              style={{ color: 'var(--vl-faint)' }}
            >
              🤖 {tp('thinking', language)}
            </div>
          )}
        </div>

        <div className="flex flex-wrap gap-1.5 px-3 pb-1">
          {[tp('qcWhyLevel', language), tp('qcWhatKills', language)].map(
            (qc) => (
              <button
                key={qc}
                onClick={() => send(qc)}
                disabled={busy}
                className="text-[9.5px] px-2.5 py-1"
                style={{
                  borderRadius: 999,
                  border: '1px solid var(--vl-hair)',
                  color: 'var(--vl-muted)',
                }}
              >
                {qc}
              </button>
            )
          )}
        </div>
        <div
          className="flex items-center gap-2 px-3 py-3"
          style={{ borderTop: '1px solid var(--vl-hair)' }}
        >
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && send(input)}
            placeholder={tp('askPlaceholder', language)}
            disabled={busy}
            className="flex-1 text-[12px] px-3.5 py-2"
            style={{
              border: '1px solid var(--vl-gold-line)',
              borderRadius: 999,
              background: 'rgba(201,162,75,0.04)',
              color: 'var(--vl-ivory)',
            }}
          />
          <button
            onClick={() => send(input)}
            disabled={busy || !input.trim()}
            aria-label={tp('send', language)}
            style={{
              width: 36,
              height: 36,
              borderRadius: '50%',
              background: 'var(--vl-gold)',
              color: 'var(--vl-ink)',
              fontWeight: 700,
              opacity: busy || !input.trim() ? 0.5 : 1,
            }}
          >
            ↑
          </button>
        </div>
      </div>
    </div>
  )

  return createPortal(panel, document.body)
}
