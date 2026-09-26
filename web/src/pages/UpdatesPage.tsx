// W-ONE-BUTTON M5 (U2) — the Updates page: the owner's face of M2/M3/M4.
//
// Truth rules (CLAUDE-canon "Updates page truth rules"): this page renders
// ONLY what the API returns and nothing it does not. A value the binary
// cannot know prints n/a. An absent job is "no update job", never an empty
// timeline. Every blocker is shown with the server's exact text.

import { useCallback, useEffect, useState } from 'react'
import { Download, Loader2, RefreshCw, ShieldAlert } from 'lucide-react'
import { useLanguage } from '../contexts/LanguageContext'
import { up } from '../i18n/updates-translations'
import { GUIDE_BUILT_REV } from '../guide/types'
import {
  INSTALL_AUTHZ_UNDER_REVIEW,
  updatesApi,
  type HealthStatus,
  type InstallationGate,
  type MaintenanceStatusView,
  type UpdateJobView,
  type UpdatesCheck,
  type UpdatesStatus,
} from '../lib/api/updates'
import { strategyEffectiveApi } from '../lib/api/strategyEffective'
import { traderApi } from '../lib/api/traders'
import { configApi } from '../lib/api/config'
import { httpClient } from '../lib/httpClient'
import { loadStoredTraderId } from '../router/selectedTrader'

const POLL_MS = 10_000

function na(lang: 'en' | 'zh' | 'id') {
  return up('notApplicable', lang)
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-4 py-1.5">
      <span className="text-xs text-zinc-500 shrink-0">{label}</span>
      <span className="text-sm text-zinc-200 text-right break-all font-mono">
        {value}
      </span>
    </div>
  )
}

function Panel({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <div className="bg-zinc-900/60 backdrop-blur-xl border border-zinc-800/80 rounded-2xl p-5">
      <h3 className="text-sm font-semibold text-white mb-3">{title}</h3>
      {children}
    </div>
  )
}

type UpdateButtonState =
  | 'update-now'
  | 'checking'
  | 'installing'
  | 'retry'
  | 'up-to-date'
  | 'blocked'

export default function UpdatesPage() {
  const { language } = useLanguage()

  const [health, setHealth] = useState<HealthStatus | null>(null)
  const [maintenance, setMaintenance] = useState<MaintenanceStatusView | null>(
    null
  )
  const [gate, setGate] = useState<InstallationGate | null>(null)
  const [status, setStatus] = useState<UpdatesStatus | null>(null)
  const [check, setCheck] = useState<UpdatesCheck | null>(null)
  const [checking, setChecking] = useState(false)
  const [installError, setInstallError] = useState<string | null>(null)
  const [notEnrolled, setNotEnrolled] = useState(false)
  const [job, setJob] = useState<UpdateJobView | null>(null)

  // Panel E — the resolved PivotWindow (W1 effective settings), or null.
  const [pivotWindow, setPivotWindow] = useState<number | null>(null)
  const [backfillTraderId, setBackfillTraderId] = useState<string | null>(null)
  const [futuresSymbol, setFuturesSymbol] = useState<string | null>(null)
  const [confirmBackfill, setConfirmBackfill] = useState(false)
  const [historyReloading, setHistoryReloading] = useState(false)
  const [historyReply, setHistoryReply] = useState<string | null>(null)

  const poll = useCallback(async () => {
    const [h, m, g, sr] = await Promise.all([
      updatesApi.health(),
      updatesApi.maintenance(),
      updatesApi.installationGate(),
      updatesApi.updatesStatus(),
    ])
    if (h) setHealth(h)
    if (m) setMaintenance(m)
    if (g) setGate(g)
    if (sr.status) setStatus(sr.status)
    return sr.statusCode
  }, [])

  useEffect(() => {
    let alive = true
    let timer: number | null = null
    const tick = async () => {
      const code = await poll()
      if (!alive) return
      if (code === 403) {
        // not-enrolled: the server refuses with its own text and logs a WARN
        // per request — stop the page poll instead of 6 more per minute ([5]).
        setNotEnrolled(true)
        return
      }
      timer = window.setTimeout(tick, POLL_MS)
    }
    tick()
    return () => {
      alive = false
      if (timer !== null) window.clearTimeout(timer)
    }
  }, [poll])

  // Job fetch: while the API names a job, AND after the hold clears — the
  // LAST id the API named keeps being polled, so the terminal state
  // (complete / rolled_back) is the last thing shown, never a frozen
  // snapshot of the moment the hold appeared (#206 review fold). Absent job
  // = "no update job".
  const [lastJobID, setLastJobID] = useState<string | null>(null)
  const polledJobID = maintenance?.job_id ?? lastJobID
  useEffect(() => {
    if (!polledJobID) {
      setJob(null)
      return
    }
    let alive = true
    const fetch = () => {
      updatesApi.job(polledJobID).then((j) => {
        if (alive) setJob(j)
      })
    }
    fetch()
    const id = window.setInterval(fetch, POLL_MS)
    return () => {
      alive = false
      window.clearInterval(id)
    }
  }, [polledJobID])

  // Remember the id the API named: it stays the polled id once the hold
  // clears (the job route reads the worker's own job file, which outlives
  // the hold).
  useEffect(() => {
    if (maintenance?.job_id) setLastJobID(maintenance.job_id)
  }, [maintenance?.job_id])

  // The receipt download (OQ-7): the route sits behind the M3 gate, so a bare
  // navigation 403s — the receipt is fetched through the API client (which
  // sends X-NOFX-Update) and saved as a file. A refusal shows the server's
  // own text, never a fabricated one.
  const [receiptBusy, setReceiptBusy] = useState(false)
  const [receiptError, setReceiptError] = useState<string | null>(null)
  const downloadReceipt = useCallback(async () => {
    const id = polledJobID
    if (!id || receiptBusy) return
    setReceiptBusy(true)
    setReceiptError(null)
    try {
      const res = await updatesApi.receipt(id)
      if (!res?.data) {
        // The server's own text when it said one; nothing fabricated when it
        // did not (the button simply stops spinning).
        setReceiptError(res?.error ?? null)
        return
      }
      const blob = new Blob([JSON.stringify(res.data, null, 2)], {
        type: 'application/json',
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `receipt-${id}.json`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
    } finally {
      setReceiptBusy(false)
    }
  }, [polledJobID, receiptBusy, language])

  // Panel E — resolve PivotWindow, the trader id and the trader's futures
  // symbol (READ from the trader row via its exchange config — never a
  // literal) from the selected trader.
  useEffect(() => {
    let alive = true
    Promise.all([traderApi.getTraders(true), configApi.getExchangeConfigs()])
      .then(([traders, exchanges]) => {
        const selected = loadStoredTraderId()
        const trader =
          traders.find((t) => t.trader_id === selected) ??
          (traders.length ? traders[0] : undefined)
        const strategyId = trader?.strategy_id
        if (!alive) return
        if (trader) setBackfillTraderId(trader.trader_id)
        const match = trader?.exchange_id
          ? exchanges.find((e) => e.id === trader.exchange_id)
          : undefined
        // The list endpoint returns nt_instrument_name (api/handler_exchange.go:96);
        // the shared Exchange type predates it — read it here, page-local.
        const symbol =
          (match as { nt_instrument_name?: string } | undefined)
            ?.nt_instrument_name ?? null
        setFuturesSymbol(symbol ? (symbol.length > 0 ? symbol : null) : null)
        if (!strategyId) {
          setPivotWindow(null)
          return
        }
        strategyEffectiveApi
          .getStrategyEffective(strategyId)
          .then((resp) => {
            if (!alive) return
            const knob = resp?.settings.find(
              (k) => k.path === 'picture_htf.pivot_window'
            )
            const v = knob?.effective
            setPivotWindow(
              typeof v === 'number' && Number.isFinite(v) ? v : null
            )
          })
          .catch(() => alive && setPivotWindow(null))
      })
      .catch(() => {
        if (alive) {
          setPivotWindow(null)
          setFuturesSymbol(null)
          setBackfillTraderId(null)
        }
      })
    return () => {
      alive = false
    }
  }, [])

  // ── Panel B button state, driven ONLY by the API ──────────────────────────
  let buttonState: UpdateButtonState = 'update-now'
  if (checking) buttonState = 'checking'
  else if (status?.install_enabled === false) buttonState = 'blocked'
  else if (status?.worker_listening === false) buttonState = 'blocked'
  else if (check?.checked) buttonState = 'up-to-date'
  else if (installError) buttonState = 'retry'

  const buttonLabel = {
    'update-now': up('updateNow', language),
    checking: up('checking', language),
    installing: up('installing', language),
    retry: up('retry', language),
    'up-to-date': up('upToDate', language),
    blocked: up('blocked', language),
  }[buttonState]

  const doCheck = useCallback(async () => {
    setChecking(true)
    setInstallError(null)
    const c = await updatesApi.check()
    setCheck(c)
    setChecking(false)
  }, [])

  // The install action exists but is unreachable while M3's adversarial review
  // is open (INSTALL_AUTHZ_UNDER_REVIEW ships ON): the disabled button carries
  // the exact text and no install POST can fire. #206's ruling: BOTH
  // install_enabled (configuration) AND worker_listening (measured) must be
  // true, and the exact reason is shown when the worker is not running.
  const workerDown =
    status?.install_enabled === true && status?.worker_listening === false
  const installDisabled =
    INSTALL_AUTHZ_UNDER_REVIEW ||
    status?.install_enabled !== true ||
    status?.worker_listening !== true

  const askReloadHistory = useCallback(() => {
    setHistoryReply(null)
    setConfirmBackfill(true)
  }, [])

  const cancelReloadHistory = useCallback(() => {
    setConfirmBackfill(false)
    setHistoryReply(null)
  }, [])

  const confirmReloadHistory = useCallback(async () => {
    if (pivotWindow === null || !futuresSymbol) return
    setHistoryReloading(true)
    setHistoryReply(null)
    try {
      const res = await httpClient.request<{
        ok?: boolean
        note?: string
        error?: string
      }>('/api/nt/bar-arbiter', {
        method: 'POST',
        data: {
          trader_id: backfillTraderId ?? '',
          action: 'backfill',
          symbol: futuresSymbol,
          timeframe: '4h',
          bars_back: pivotWindow + 4,
        },
      })
      setHistoryReply(
        res.data?.error ||
          res.data?.note ||
          (res.success ? 'backfill request accepted' : res.message) ||
          'no server text'
      )
    } catch (e) {
      setHistoryReply(e instanceof Error ? e.message : 'request failed')
    } finally {
      setHistoryReloading(false)
      setConfirmBackfill(false)
    }
  }, [pivotWindow, futuresSymbol, backfillTraderId])

  const state = maintenance?.state ?? null
  const ack = maintenance?.addon_ack ?? null
  const completedBarsN = pivotWindow !== null ? pivotWindow + 4 : null

  return (
    <div className="space-y-5" data-testid="updates-page">
      {notEnrolled && (
        <div className="rounded-lg border border-amber-600/40 bg-amber-500/10 px-4 py-2 text-xs text-amber-300">
          {up('pollingStopped', language)}
        </div>
      )}
      {/* Panel A — Running now */}
      <Panel title={up('runningNow', language)}>
        <Row
          label={up('revision', language)}
          value={health?.revision || na(language)}
        />
        <Row
          label={up('guideRev', language)}
          value={GUIDE_BUILT_REV || na(language)}
        />
        <Row
          label={up('addonBuild', language)}
          value={ack?.build_id || na(language)}
        />
        <Row label={up('mvidMatch', language)} value={na(language)} />
      </Panel>

      {/* Panel B — Update */}
      <Panel title={up('updatePanel', language)}>
        <div className="flex items-center gap-3">
          <button
            type="button"
            disabled={checking || installDisabled}
            onClick={doCheck}
            className="inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium bg-nofx-gold text-black disabled:opacity-50"
            data-testid="update-button"
          >
            {checking && <Loader2 size={15} className="animate-spin" />}
            {buttonLabel}
          </button>
        </div>
        {INSTALL_AUTHZ_UNDER_REVIEW && (
          <p className="mt-2 text-xs text-amber-400 flex items-center gap-1.5">
            <ShieldAlert size={13} />
            {up('installUnderReview', language)}
          </p>
        )}
        {workerDown && (
          <p
            className="mt-2 text-xs text-amber-400 flex items-center gap-1.5"
            data-testid="worker-not-running"
          >
            <ShieldAlert size={13} />
            {up('workerNotRunning', language)}
          </p>
        )}
        {installError && (
          <p className="mt-2 text-xs text-red-400" data-testid="install-error">
            {installError}
          </p>
        )}
        {check && (
          <p className="mt-2 text-xs text-zinc-400">
            {up('checkReason', language)}: {check.reason || na(language)}
          </p>
        )}
      </Panel>

      {/* Panel C — Maintenance hold + gate */}
      <Panel title={up('maintenancePanel', language)}>
        <Row label={up('holdState', language)} value={state || na(language)} />
        {maintenance?.job_id !== undefined && maintenance?.job_id !== null && (
          <Row label={up('jobId', language)} value={maintenance.job_id} />
        )}
        {maintenance?.since !== undefined && maintenance?.since !== null && (
          <Row label="Since" value={maintenance.since} />
        )}
        <Row
          label={up('inFlightSends', language)}
          value={String(maintenance?.in_flight_sends ?? 0)}
        />
        <Row
          label={up('drained', language)}
          value={maintenance ? String(maintenance.drained) : na(language)}
        />
        <Row
          label={up('addonAck', language)}
          value={
            ack
              ? `${ack.held ? 'held' : 'released'} · job=${ack.job_id || na(language)} · build=${ack.build_id || na(language)}`
              : up('noAckYet', language)
          }
        />

        <div className="border-t border-zinc-800 mt-4 pt-4">
          <Row
            label={up('gateReady', language)}
            value={gate ? String(gate.ready) : na(language)}
          />
          <div className="mt-2 space-y-1.5">
            {(gate?.legs ?? []).map((leg) => (
              <div
                key={leg.name}
                className="flex items-start justify-between gap-3 text-xs"
              >
                <span
                  className={`shrink-0 px-1.5 py-0.5 rounded ${
                    leg.pass
                      ? 'bg-emerald-500/15 text-emerald-400'
                      : 'bg-red-500/15 text-red-400'
                  }`}
                >
                  {leg.pass ? 'PASS' : 'FAIL'}
                </span>
                <span className="text-zinc-300 font-mono">{leg.name}</span>
                <span className="text-zinc-400 text-right flex-1">
                  {leg.detail}
                </span>
              </div>
            ))}
            {!gate && <p className="text-xs text-zinc-500">{na(language)}</p>}
          </div>
        </div>
      </Panel>

      {/* Panel D — Job */}
      <Panel title={up('jobPanel', language)}>
        {!maintenance?.job_id && !job?.job_id ? (
          <p className="text-sm text-zinc-400" data-testid="no-job">
            {up('noUpdateJob', language)}
          </p>
        ) : (
          <>
            <Row
              label={up('jobId', language)}
              value={job?.job_id || maintenance?.job_id || na(language)}
            />
            <Row
              label={up('jobState', language)}
              value={
                job?.state ||
                (job?.error ? `${job.status ?? ''} ${job.error}` : na(language))
              }
            />
            {job?.step && (
              <Row label={up('jobStep', language)} value={job.step} />
            )}
            {job?.blocker && (
              <Row label={up('jobBlocker', language)} value={job.blocker} />
            )}
            {job?.timestamps && Object.keys(job.timestamps).length > 0 && (
              <div
                className="mt-3 border-t border-zinc-800 pt-3 space-y-1.5"
                data-testid="job-timestamps"
              >
                {Object.entries(job.timestamps).map(([state, at]) => (
                  <Row key={state} label={state} value={at} />
                ))}
              </div>
            )}
            {job?.receipt_url && (
              <div className="mt-2">
                <button
                  type="button"
                  onClick={downloadReceipt}
                  disabled={receiptBusy}
                  className="inline-flex items-center gap-1.5 text-xs text-nofx-gold hover:underline disabled:opacity-60"
                  data-testid="receipt-link"
                >
                  {receiptBusy ? (
                    <Loader2 size={13} className="animate-spin" />
                  ) : (
                    <Download size={13} />
                  )}
                  {up('downloadReceipt', language)}
                </button>
                {receiptError && (
                  <p className="mt-1 text-xs text-red-400">{receiptError}</p>
                )}
              </div>
            )}
          </>
        )}
      </Panel>

      {/* Panel E — Native history readiness */}
      <Panel title={up('historyPanel', language)}>
        <Row label={up('tradingPaused', language)} value={na(language)} />
        <Row
          label={up('completedBars', language)}
          value={
            completedBarsN !== null
              ? `${na(language)} of ${completedBarsN} (${up('pivotWindow', language)}+4)`
              : `${na(language)} of ${na(language)} (${up('pivotWindow', language)}+4)`
          }
        />
        <Row label={up('lastProgress', language)} value={na(language)} />
        <div className="mt-3">
          <button
            type="button"
            onClick={askReloadHistory}
            disabled={pivotWindow === null || futuresSymbol === null}
            className="inline-flex items-center gap-2 rounded-lg px-3 py-1.5 text-xs font-medium bg-zinc-800 hover:bg-zinc-700 text-zinc-200 disabled:opacity-50"
            data-testid="reload-history"
          >
            <RefreshCw size={13} />
            {up('reloadHistory', language)}
          </button>
          {pivotWindow === null && (
            <p
              className="mt-2 text-xs text-amber-400"
              data-testid="pivot-unknown"
            >
              {up('pivotWindowUnknown', language)}
            </p>
          )}
          {futuresSymbol === null && (
            <p
              className="mt-2 text-xs text-amber-400"
              data-testid="symbol-unknown"
            >
              {up('futuresSymbolUnknown', language)}
            </p>
          )}
          {confirmBackfill &&
            pivotWindow !== null &&
            futuresSymbol !== null && (
              <div className="mt-2 rounded-lg border border-zinc-700 p-3">
                <p
                  className="text-xs text-zinc-300"
                  data-testid="backfill-confirm"
                >
                  {up('confirmBackfill', language, { n: pivotWindow + 4 })}
                </p>
                <div className="mt-2 flex gap-2">
                  <button
                    type="button"
                    onClick={confirmReloadHistory}
                    disabled={historyReloading}
                    className="rounded-lg px-3 py-1.5 text-xs font-medium bg-nofx-gold text-black disabled:opacity-50"
                    data-testid="confirm-backfill"
                  >
                    {historyReloading
                      ? up('reloading', language)
                      : up('confirm', language)}
                  </button>
                  <button
                    type="button"
                    onClick={cancelReloadHistory}
                    className="rounded-lg px-3 py-1.5 text-xs font-medium bg-zinc-700 text-zinc-200"
                    data-testid="cancel-backfill"
                  >
                    {up('cancel', language)}
                  </button>
                </div>
              </div>
            )}
          {historyReply && (
            <p
              className="mt-2 text-xs text-zinc-400"
              data-testid="backfill-reply"
            >
              {up('checkReason', language)}: {historyReply}
            </p>
          )}
        </div>
      </Panel>
    </div>
  )
}
