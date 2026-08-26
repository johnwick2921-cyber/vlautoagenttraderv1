// ApproveButton — the spec-mandated PlanHeader approve action (PLAN-CARD
// component tree). When approval_required is ON for a session, entries stay
// gated until the owner reviews the plan and clicks Approve; the server grants
// entries for that CME session-day (W9, POST /api/plan/approve).
//
// The button renders only when the plan payload carries approval state, so an
// advisory-mode plan (approval OFF) shows nothing — no noise, no dead control.

import { useState } from 'react'
import { toast } from 'sonner'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import { planApi } from '../../lib/api/plan'
import { guardedCall } from '../../lib/api/guarded'

export function ApproveButton({
  traderId,
  language,
  onDone,
}: {
  traderId?: string
  language: Language
  /** re-fetch so the card reflects the approval */
  onDone?: () => void
}) {
  const [busy, setBusy] = useState(false)

  const run = async () => {
    if (!traderId || busy) return
    setBusy(true)
    try {
      const g = await guardedCall(() => planApi.postPlanApprove(traderId))
      if (g.ok && g.value.approved) {
        toast.success(tp('approveDone', language))
        onDone?.()
      } else {
        toast.error(g.ok ? tp('approveFailed', language) : g.error)
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <button
      onClick={() => void run()}
      disabled={busy || !traderId}
      className="px-3 py-1.5 rounded text-xs font-medium"
      style={{
        background: 'var(--vl-gold)',
        color: 'var(--vl-ink)',
        opacity: busy ? 0.6 : 1,
      }}
    >
      {tp('approve', language)}
    </button>
  )
}
