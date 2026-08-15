// P4.4 — in-app alert center (NO external push). A bell with an unacked badge, a
// dropdown feed, a persistent P0 banner until ack, and a toast for each NEW P0.
// The backend bus is idempotent (deduped by event_id), and the FE dedupes toasts
// by alert id so a poll refresh never re-toasts. Levels: P0 = toast + banner ·
// P1 = feed · P2 = digest (muted, collapsed count).

import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import { usePlanAlerts } from './usePlan'
import { api } from '../../lib/api'
import type { PlanAlert, AlertLevel } from '../../lib/api/plan'

const levelColor = (l: AlertLevel) =>
  l === 'P0'
    ? 'var(--vl-short)'
    : l === 'P1'
      ? 'var(--vl-gold)'
      : 'var(--vl-faint)'

function relTime(unixSec: number): string {
  const diff = Math.max(0, Math.floor(Date.now() / 1000) - unixSec)
  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff / 60)}m`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`
  return `${Math.floor(diff / 86400)}d`
}

function AlertRow({
  alert,
  onAck,
  language,
}: {
  alert: PlanAlert
  onAck: (id: number) => void
  language: Language
}) {
  return (
    <div
      className="flex items-start gap-2 py-2 px-1"
      style={{
        borderBottom: '1px solid var(--vl-hair)',
        opacity: alert.acked ? 0.5 : 1,
      }}
    >
      <span
        className="mt-0.5 text-[9px] font-bold"
        style={{
          color: levelColor(alert.level),
          border: `1px solid ${levelColor(alert.level)}`,
          borderRadius: 'var(--vl-radius-chip)',
          padding: '0 4px',
          fontFamily: 'var(--vl-font-ui)',
        }}
      >
        {alert.level}
      </span>
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <span
            className="text-[12px] font-semibold truncate"
            style={{
              color: 'var(--vl-ivory)',
              fontFamily: 'var(--vl-font-ui)',
            }}
          >
            {alert.title}
          </span>
          <span
            className="text-[10px] vl-num"
            style={{ color: 'var(--vl-faint)' }}
          >
            {relTime(alert.created_at)}
          </span>
        </div>
        {alert.body && (
          <div
            className="text-[11px] mt-0.5"
            style={{
              color: 'var(--vl-muted)',
              fontFamily: 'var(--vl-font-ui)',
            }}
          >
            {alert.body}
          </div>
        )}
      </div>
      {!alert.acked && (
        <button
          onClick={() => onAck(alert.id)}
          className="text-[10px] px-1.5 py-0.5 shrink-0"
          style={{
            color: 'var(--vl-muted)',
            border: '1px solid var(--vl-hair)',
            borderRadius: 'var(--vl-radius-chip)',
          }}
        >
          {tp('markRead', language)}
        </button>
      )}
    </div>
  )
}

export function AlertCenter({
  traderId,
  language,
}: {
  traderId?: string
  language: Language
}) {
  const { alerts, unacked, mutate } = usePlanAlerts(traderId)
  const [open, setOpen] = useState(false)
  // ids we've already toasted — seeded on first load so the backlog never toasts.
  const toastedRef = useRef<Set<number>>(new Set())
  const seededRef = useRef(false)

  // fire a toast for each NEW unacked P0 (never the initial backlog).
  useEffect(() => {
    if (!seededRef.current) {
      for (const a of alerts) toastedRef.current.add(a.id)
      seededRef.current = true
      return
    }
    for (const a of alerts) {
      if (a.level === 'P0' && !a.acked && !toastedRef.current.has(a.id)) {
        toastedRef.current.add(a.id)
        toast.error(a.title, {
          description: a.body || undefined,
          duration: 8000,
        })
      }
    }
  }, [alerts])

  const ack = async (id: number) => {
    if (!traderId) return
    await api.ackPlanAlert(traderId, id)
    await mutate()
  }
  const ackAll = async () => {
    if (!traderId) return
    await Promise.all(
      alerts
        .filter((a) => !a.acked)
        .map((a) => api.ackPlanAlert(traderId, a.id))
    )
    await mutate()
  }

  if (!traderId) return null

  // the top unacked P0 → persistent banner until ack
  const bannerAlert = alerts.find((a) => a.level === 'P0' && !a.acked)
  // P2 digest: collapse to a count line rather than individual rows
  const p2Count = alerts.filter((a) => a.level === 'P2' && !a.acked).length
  const feedAlerts = alerts.filter((a) => a.level !== 'P2')

  return (
    <div className="flex flex-col gap-2">
      {/* header row: bell + unacked badge */}
      <div className="flex items-center justify-between">
        <button
          onClick={() => setOpen((o) => !o)}
          className="relative inline-flex items-center gap-1.5 text-[12px]"
          aria-label={tp('bellLabel', language, { n: unacked })}
          aria-expanded={open}
          style={{
            color: unacked > 0 ? 'var(--vl-gold)' : 'var(--vl-muted)',
            fontFamily: 'var(--vl-font-ui)',
          }}
        >
          <span aria-hidden style={{ fontSize: 14 }}>
            🔔
          </span>
          <span>{tp('alerts', language)}</span>
          {unacked > 0 && (
            <span
              className="vl-num text-[9px] font-bold"
              style={{
                color: 'var(--vl-ink)',
                background: 'var(--vl-short)',
                borderRadius: 999,
                minWidth: 15,
                height: 15,
                padding: '0 4px',
                display: 'inline-flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              {unacked}
            </span>
          )}
        </button>
        {open && unacked > 0 && (
          <button
            onClick={ackAll}
            className="text-[10px]"
            style={{
              color: 'var(--vl-muted)',
              fontFamily: 'var(--vl-font-ui)',
            }}
          >
            {tp('markAllRead', language)}
          </button>
        )}
      </div>

      {/* persistent P0 banner (until ack) */}
      {bannerAlert && (
        <div
          role="alert"
          className="flex items-center justify-between gap-2 px-2.5 py-1.5 text-[11px]"
          style={{
            background: 'rgba(224,108,108,0.12)',
            border: '1px solid var(--vl-short)',
            borderRadius: 'var(--vl-radius-inner)',
            color: 'var(--vl-short)',
            fontFamily: 'var(--vl-font-ui)',
          }}
        >
          <span className="flex items-center gap-2 min-w-0">
            <span aria-hidden>⛔</span>
            <span className="font-semibold truncate">{bannerAlert.title}</span>
          </span>
          <button
            onClick={() => ack(bannerAlert.id)}
            className="shrink-0 text-[10px] px-1.5 py-0.5"
            style={{
              color: 'var(--vl-short)',
              border: '1px solid var(--vl-short)',
              borderRadius: 'var(--vl-radius-chip)',
            }}
          >
            {tp('markRead', language)}
          </button>
        </div>
      )}

      {/* feed dropdown */}
      {open && (
        <div
          className="vl-sheet-in flex flex-col"
          style={{
            background: 'var(--vl-card-2)',
            border: '1px solid var(--vl-hair)',
            borderRadius: 'var(--vl-radius-inner)',
            maxHeight: 280,
            overflowY: 'auto',
          }}
        >
          {feedAlerts.length === 0 && p2Count === 0 ? (
            <div
              className="py-4 text-center text-[11px]"
              style={{
                color: 'var(--vl-faint)',
                fontFamily: 'var(--vl-font-ui)',
              }}
            >
              {tp('noAlerts', language)}
            </div>
          ) : (
            <>
              {feedAlerts.map((a) => (
                <AlertRow
                  key={a.id}
                  alert={a}
                  onAck={ack}
                  language={language}
                />
              ))}
              {p2Count > 0 && (
                <div
                  className="py-1.5 px-2 text-[10px]"
                  style={{
                    color: 'var(--vl-faint)',
                    fontFamily: 'var(--vl-font-ui)',
                  }}
                >
                  {tp('unread', language, { n: p2Count })} ·{' '}
                  {tp('digest', language)}
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}
