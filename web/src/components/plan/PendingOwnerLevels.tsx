// W-OWNER-LEVELS-UI (2026-09-17) — the "Pending owner levels" block. WHY: the
// edit sheet's ＋ Add writes a STICKY owner level (owner_levels, user+symbol
// scoped) that the planner only seats at its NEXT read, so the card showed
// nothing, the toast said "Plan updated", and there was no way to see or delete
// what was pending. This block lists GET /api/plan/owner-levels for the current
// trader + session — price, label, note, a pending/applied chip judged at read
// time, created time (CT) — with a Delete per row. It polls on the card's own
// plan cadence (useOwnerLevels → PLAN_REFRESH_MS), never a faster timer.
// "applied" means the card's plan for this session carries a level at that
// tick; NO applied_version exists and none is claimed here.

import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import { api } from '../../lib/api'
import { guardedCall } from '../../lib/api/guarded'
import { fmtPrice } from './levelState'
import { useOwnerLevels } from './usePlan'

interface Props {
  traderId: string
  symbol: string
  session?: string
  language: Language
  /** bump after an add elsewhere (edit sheet / bulk sheet) → refetch now */
  refreshTick?: number
}

// HH:MM in America/Chicago from unix seconds — the card's clock (sessionConfig).
export function fmtCreatedCT(unixSec: number): string {
  if (!unixSec || !Number.isFinite(unixSec)) return '—'
  return new Date(unixSec * 1000).toLocaleTimeString('en-US', {
    timeZone: 'America/Chicago',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

export function PendingOwnerLevels({
  traderId,
  symbol,
  session,
  language,
  refreshTick = 0,
}: Props) {
  const { levels, mutate } = useOwnerLevels(traderId, symbol, session)
  const [busyId, setBusyId] = useState<number | null>(null)

  // An add in the sheets bumps the tick → refetch the pending list at once.
  useEffect(() => {
    if (refreshTick > 0) void mutate()
  }, [refreshTick, mutate])

  const del = async (id: number) => {
    if (busyId !== null) return
    setBusyId(id)
    const g = await guardedCall(() => api.deleteOwnerLevel(traderId, id))
    setBusyId(null)
    if (!g.ok || !g.value) {
      toast.error(tp('ownerLevelDeleteFailed', language), {
        description: g.ok ? undefined : g.error,
      })
      return
    }
    await mutate()
  }

  return (
    <div className="px-3 pb-2" data-testid="pending-owner-levels">
      <div
        className="text-[10px] uppercase tracking-widest mb-1"
        style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
      >
        👤 {tp('pendingOwnerLevelsTitle', language)}
      </div>
      {levels.length === 0 ? (
        <div
          className="text-[11px]"
          style={{ color: 'var(--vl-faint)' }}
          data-testid="pending-owner-levels-empty"
        >
          {tp('pendingOwnerLevelsEmpty', language)}
        </div>
      ) : (
        <ul className="flex flex-col gap-1">
          {levels.map((l) => {
            const applied = l.status === 'applied'
            return (
              <li
                key={l.id}
                className="flex items-center gap-2 text-[11px]"
                data-testid="pending-owner-level-row"
                data-status={l.status}
              >
                <span
                  className="font-mono tabular-nums"
                  style={{ color: 'var(--vl-ivory)' }}
                >
                  {fmtPrice(l.price)}
                </span>
                <span style={{ color: 'var(--vl-muted)' }}>{l.label}</span>
                {l.note && (
                  <span
                    className="truncate"
                    style={{ color: 'var(--vl-faint)', maxWidth: 160 }}
                    title={l.note}
                  >
                    {l.note}
                  </span>
                )}
                <span
                  className="text-[10px] px-1.5 py-0.5 leading-none"
                  data-testid="pending-owner-level-status"
                  style={{
                    borderRadius: 6,
                    border: `1px solid ${applied ? 'var(--vl-gold-line)' : 'var(--vl-hair)'}`,
                    background: applied ? 'var(--vl-gold-dim)' : 'transparent',
                    color: applied ? 'var(--vl-ivory)' : 'var(--vl-faint)',
                  }}
                >
                  {applied
                    ? tp('ownerLevelApplied', language)
                    : tp('ownerLevelPending', language)}
                </span>
                <span
                  className="ml-auto tabular-nums"
                  style={{ color: 'var(--vl-faint)' }}
                >
                  {fmtCreatedCT(l.created_at)} CT
                </span>
                <button
                  type="button"
                  onClick={() => void del(l.id)}
                  disabled={busyId !== null}
                  aria-label={`${tp('ownerLevelDelete', language)} ${fmtPrice(l.price)}`}
                  className="text-[10px] px-2 py-0.5"
                  style={{
                    borderRadius: 6,
                    border: '1px solid var(--vl-hair)',
                    color: 'var(--vl-faint)',
                    background: 'transparent',
                    opacity: busyId === l.id ? 0.5 : 1,
                  }}
                >
                  {tp('ownerLevelDelete', language)}
                </button>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
