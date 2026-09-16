// P2 (ledger-close 2026-08-19) — stop_until owner pause control.
//
// One-tap, mobile-first: Pause 30m / 1h / until session end / custom minutes,
// and Resume while a pause is armed. Pairs with the banner rendered by
// TraderDashboardPage when status.stop_until is in the future.
//
// Blocks NEW entries only — stops, targets, EOD-flat and position management
// continue (see trader/auto_trader_pause.go). Labels carry en/zh/id inline
// (the sibling EmergencyFlatButton predates i18n; a local dict keeps this
// component self-contained without touching the 3k-line translations file).

import { useState } from 'react'
import { useLanguage } from '../../contexts/LanguageContext'
import { api } from '../../lib/api'

interface Props {
  traderId: string
  stopUntil?: string // RFC3339 from status.stop_until; zero/past = not paused
  onChanged?: () => void
}

const L: Record<string, Record<string, string>> = {
  pause: { en: 'Pause', zh: '暂停', id: 'Jeda' },
  resume: { en: 'Resume', zh: '恢复', id: 'Lanjutkan' },
  m30: { en: '30 min', zh: '30分钟', id: '30 mnt' },
  h1: { en: '1 hour', zh: '1小时', id: '1 jam' },
  sessionEnd: { en: 'Until session end', zh: '至本时段结束', id: 'Sampai akhir sesi' },
  custom: { en: 'Custom (min)', zh: '自定义(分钟)', id: 'Kustom (mnt)' },
  title: {
    en: 'Pause NEW entries — stops/targets/position management continue',
    zh: '暂停新开仓 — 止损/止盈/持仓管理继续运行',
    id: 'Jeda entri BARU — stop/target/manajemen posisi tetap berjalan',
  },
}

export function isPauseActive(stopUntil?: string): boolean {
  if (!stopUntil) return false
  const t = Date.parse(stopUntil)
  return Number.isFinite(t) && t > Date.now()
}

export function pauseUntilCT(stopUntil: string): string {
  return new Intl.DateTimeFormat('en-US', {
    timeZone: 'America/Chicago', // ALL UI pinned to Houston CT (standing rule)
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(stopUntil))
}

export function PauseButton({ traderId, stopUntil, onChanged }: Props) {
  const { language } = useLanguage()
  const t = (k: string) => L[k][language] ?? L[k].en
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [customMin, setCustomMin] = useState('')
  const active = isPauseActive(stopUntil)

  async function doPause(body: { minutes?: number; until?: string }) {
    setBusy(true)
    try {
      await api.pauseTrader(traderId, body)
      setOpen(false)
      onChanged?.()
    } catch (e) {
      console.error('pause failed', e)
    } finally {
      setBusy(false)
    }
  }

  async function doResume() {
    setBusy(true)
    try {
      await api.resumeTrader(traderId)
      onChanged?.()
    } catch (e) {
      console.error('resume failed', e)
    } finally {
      setBusy(false)
    }
  }

  if (active) {
    return (
      <button
        data-testid="resume-entries-button"
        type="button"
        disabled={busy}
        onClick={doResume}
        className="bg-nofx-gold/20 hover:bg-nofx-gold/30 border border-nofx-gold text-nofx-gold font-bold py-2 px-4 rounded-lg transition-colors disabled:opacity-50"
      >
        ▶ {t('resume')}
      </button>
    )
  }

  return (
    <div className="relative">
      <button
        data-testid="pause-entries-button"
        type="button"
        disabled={busy}
        onClick={() => setOpen((v) => !v)}
        title={L.title[language] ?? L.title.en}
        className="border border-white/20 hover:border-nofx-gold/60 text-nofx-text-muted hover:text-nofx-gold font-bold py-2 px-4 rounded-lg transition-colors disabled:opacity-50"
      >
        ⏸ {t('pause')}
      </button>
      {open && (
        <div className="absolute right-0 mt-2 z-40 nofx-glass border border-white/10 rounded-lg p-2 min-w-[180px] flex flex-col gap-1 shadow-xl">
          <button
            type="button"
            data-testid="pause-30m"
            className="text-left px-3 py-2 rounded hover:bg-white/10 text-sm"
            onClick={() => doPause({ minutes: 30 })}
          >
            {t('m30')}
          </button>
          <button
            type="button"
            data-testid="pause-1h"
            className="text-left px-3 py-2 rounded hover:bg-white/10 text-sm"
            onClick={() => doPause({ minutes: 60 })}
          >
            {t('h1')}
          </button>
          <button
            type="button"
            data-testid="pause-session-end"
            className="text-left px-3 py-2 rounded hover:bg-white/10 text-sm"
            onClick={() => doPause({ until: 'session_end' })}
          >
            {t('sessionEnd')}
          </button>
          <div className="flex items-center gap-1 px-1 pt-1 border-t border-white/10">
            <input
              data-testid="pause-custom-min"
              inputMode="numeric"
              placeholder={t('custom')}
              value={customMin}
              onChange={(e) => setCustomMin(e.target.value.replace(/\D/g, ''))}
              className="w-full bg-black/40 border border-white/10 rounded px-2 py-1 text-sm"
            />
            <button
              type="button"
              data-testid="pause-custom-go"
              disabled={!customMin || Number(customMin) <= 0}
              className="px-2 py-1 rounded bg-nofx-gold/20 border border-nofx-gold/50 text-nofx-gold text-sm disabled:opacity-40"
              onClick={() => doPause({ minutes: Number(customMin) })}
            >
              OK
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
