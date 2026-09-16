// Plan 4 Task 23.4 — Decisions audit table.
//
// Consumes GET /api/audit/decisions?trader_id=X&limit=N and renders one row
// per DecisionAction inside each DecisionRecord. The 8 audit-trail fields
// (ai_model, ai_latency_ms, risk_check_passed, risk_check_error,
// execution_status, fill_price, fill_latency_ms, prompt_version) live on the
// parent record; per-trade fields (symbol, action, price/stop/tp, confidence,
// reasoning) live on each nested action.
//
// Required columns per plan spec lines 6174-6176:
//   Symbol, Action, Entry, SL, TP, Confidence, Risk Check,
//   Execution Status, Fill Price, Latency (+ Time as 11th column)
//
// Click a row to expand the Reasoning panel.

import { Fragment, useEffect, useState } from 'react'
import { useAutoRefresh, REFRESH_HISTORY_MS } from '../../lib/autoRefresh'

interface DecisionActionRecord {
  action: string
  symbol: string
  price?: number
  stop_loss?: number
  take_profit?: number
  confidence?: number
  reasoning?: string
  // W16/R3 — already on the wire (store.DecisionAction Success/Error), never
  // declared here, so a REFUSED action rendered identically to an executed one.
  // The reason names the gate: "session_gate: ASIA session not enabled".
  success?: boolean
  error?: string
}

interface DecisionAuditRow {
  id: number
  trader_id: string
  cycle_number: number
  timestamp: string
  created_at: string
  decisions: DecisionActionRecord[]
  prompt_version: string
  ai_model: string
  ai_latency_ms: number
  risk_check_passed: boolean
  risk_check_error: string
  execution_status: string
  fill_price: number | null
  fill_latency_ms: number | null
}

interface Props {
  traderId: string
  account?: string
}

// Flatten one cycle into one or more rows (one per nested decision action).
interface FlatRow {
  cycleId: number
  actionIndex: number
  rowKey: string
  parent: DecisionAuditRow
  action: DecisionActionRecord
}

function flattenDecisions(records: DecisionAuditRow[]): FlatRow[] {
  const rows: FlatRow[] = []
  for (const rec of records) {
    const actions =
      rec.decisions && rec.decisions.length > 0
        ? rec.decisions
        : [
            {
              action: 'wait',
              symbol: '-',
              price: 0,
              stop_loss: 0,
              take_profit: 0,
              confidence: 0,
              reasoning: '',
            },
          ]
    actions.forEach((a, i) => {
      rows.push({
        cycleId: rec.id,
        actionIndex: i,
        rowKey: `${rec.id}-${i}`,
        parent: rec,
        action: a,
      })
    })
  }
  return rows
}

function fmtNum(n: number | undefined | null, digits = 2): string {
  if (n == null || isNaN(n as number)) return '—'
  return (n as number).toFixed(digits)
}

function fmtConfidence(c: number | undefined): string {
  if (c == null) return '—'
  // Backend stores confidence as 0-100 integer (see store.DecisionAction).
  return `${Math.round(c)}%`
}

export function DecisionAudit({ traderId, account }: Props) {
  const [records, setRecords] = useState<DecisionAuditRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [expandedKey, setExpandedKey] = useState<string | null>(null)

  useEffect(() => {
    if (!traderId) return
    let cancelled = false
    setLoading(true)
    setError(null)
    const token = localStorage.getItem('auth_token') ?? ''
    fetch(
      `/api/audit/decisions?trader_id=${encodeURIComponent(traderId)}&limit=100` +
        (account ? `&account=${encodeURIComponent(account)}` : ''),
      { headers: token ? { Authorization: `Bearer ${token}` } : {} }
    )
      .then(async (r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then((data) => {
        if (cancelled) return
        setRecords(Array.isArray(data) ? (data as DecisionAuditRow[]) : [])
        setLoading(false)
      })
      .catch((e) => {
        if (cancelled) return
        setError(e instanceof Error ? e.message : String(e))
        setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [traderId, account])

  // Silent background refresh (auto-refresh layer): new decision cycles
  // appear without re-opening the tab; no loading flash, errors keep the
  // last good data (the hook backs off on failures).
  useAutoRefresh(async () => {
    if (!traderId) return
    const token = localStorage.getItem('auth_token') ?? ''
    const r = await fetch(
      `/api/audit/decisions?trader_id=${encodeURIComponent(traderId)}&limit=100` +
        (account ? `&account=${encodeURIComponent(account)}` : ''),
      { headers: token ? { Authorization: `Bearer ${token}` } : {} }
    )
    if (!r.ok) throw new Error(`HTTP ${r.status}`)
    const data = await r.json()
    setRecords(Array.isArray(data) ? (data as DecisionAuditRow[]) : [])
  }, REFRESH_HISTORY_MS)

  if (loading) {
    return (
      <div
        className="p-4 text-sm text-nofx-text-muted"
        data-testid="decision-audit-loading"
      >
        Loading decisions...
      </div>
    )
  }
  if (error) {
    return (
      <div
        className="p-4 text-sm text-nofx-red"
        data-testid="decision-audit-error"
      >
        Error: {error}
      </div>
    )
  }

  const rows = flattenDecisions(records)

  if (rows.length === 0) {
    return (
      <div
        className="p-6 text-center text-nofx-text-muted opacity-70"
        data-testid="decision-audit-empty"
      >
        <div className="text-4xl mb-2 opacity-40">🧠</div>
        <div className="text-sm">No decisions yet</div>
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table data-testid="decision-audit-table" className="w-full text-xs">
        <thead className="text-left border-b border-white/10">
          <tr className="text-nofx-text-muted font-mono uppercase tracking-wider">
            <th className="px-2 py-2 font-semibold whitespace-nowrap">Time</th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap">
              Symbol
            </th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap">
              Action
            </th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap text-right">
              Entry
            </th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap text-right">
              SL
            </th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap text-right">
              TP
            </th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap text-right">
              Confidence
            </th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap text-center">
              Risk Check
            </th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap">
              Execution Status
            </th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap text-right">
              Fill Price
            </th>
            <th className="px-2 py-2 font-semibold whitespace-nowrap text-right">
              Latency
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const p = row.parent
            const a = row.action
            const isExpanded = expandedKey === row.rowKey
            const tsRaw = p.created_at || p.timestamp
            const tsDisplay = tsRaw
              ? new Date(tsRaw).toLocaleString(undefined, {
                  timeZone: 'America/Chicago',
                })
              : '—'
            const latencyMs =
              p.fill_latency_ms != null ? p.fill_latency_ms : p.ai_latency_ms
            const riskOk = p.risk_check_passed
            return (
              <Fragment key={row.rowKey}>
                <tr
                  data-testid={`decision-row-${row.cycleId}`}
                  className="border-b border-white/5 hover:bg-white/5 cursor-pointer transition-colors"
                  onClick={() => setExpandedKey(isExpanded ? null : row.rowKey)}
                >
                  <td className="px-2 py-2 text-nofx-text-muted whitespace-nowrap font-mono">
                    {tsDisplay}
                  </td>
                  <td className="px-2 py-2 font-mono font-semibold text-nofx-text-main whitespace-nowrap">
                    {a.symbol || '—'}
                  </td>
                  <td className="px-2 py-2 whitespace-nowrap">
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-white/5 text-nofx-text-main">
                      {a.action || '—'}
                    </span>
                  </td>
                  <td className="px-2 py-2 font-mono text-right text-nofx-text-main whitespace-nowrap">
                    {fmtNum(a.price)}
                  </td>
                  <td className="px-2 py-2 font-mono text-right text-nofx-red whitespace-nowrap">
                    {fmtNum(a.stop_loss)}
                  </td>
                  <td className="px-2 py-2 font-mono text-right text-nofx-green whitespace-nowrap">
                    {fmtNum(a.take_profit)}
                  </td>
                  <td className="px-2 py-2 text-right font-mono text-nofx-gold whitespace-nowrap">
                    {fmtConfidence(a.confidence)}
                  </td>
                  <td
                    className="px-2 py-2 text-center whitespace-nowrap"
                    title={p.risk_check_error || (riskOk ? 'Passed' : 'Failed')}
                  >
                    <span
                      className={`inline-flex items-center justify-center w-5 h-5 rounded text-[11px] font-bold ${
                        riskOk
                          ? 'bg-nofx-green/15 text-nofx-green'
                          : 'bg-nofx-red/15 text-nofx-red'
                      }`}
                    >
                      {riskOk ? '✓' : '✗'}
                    </span>
                  </td>
                  <td className="px-2 py-2 whitespace-nowrap">
                    {/* W16/R3 — a gate refusal is now visible AS a refusal, with the
                        gate that caused it. Before this the column showed the
                        execution status only, so a blocked entry was indistinguishable
                        from an executed one. */}
                    {a.error ? (
                      <span
                        data-testid={`decision-refused-${row.cycleId}`}
                        title={a.error}
                        className="px-1.5 py-0.5 rounded text-[10px] font-mono uppercase tracking-wider bg-nofx-red/15 border border-nofx-red/30 text-nofx-red"
                      >
                        ⛔ {a.error.split(':')[0]}
                      </span>
                    ) : (
                      <span className="px-1.5 py-0.5 rounded text-[10px] font-mono uppercase tracking-wider bg-black/40 border border-white/10 text-nofx-text-muted">
                        {p.execution_status || '—'}
                      </span>
                    )}
                  </td>
                  <td className="px-2 py-2 font-mono text-right text-nofx-text-main whitespace-nowrap">
                    {p.fill_price != null ? fmtNum(p.fill_price) : '—'}
                  </td>
                  <td className="px-2 py-2 font-mono text-right text-nofx-text-muted whitespace-nowrap">
                    {latencyMs > 0 ? `${latencyMs} ms` : '—'}
                  </td>
                </tr>
                {isExpanded && (
                  <tr data-testid={`decision-row-${row.cycleId}-reasoning`}>
                    <td
                      colSpan={11}
                      className="px-3 py-3 bg-white/[0.03] text-nofx-text-main text-xs"
                    >
                      <div className="flex items-center gap-3 mb-2 text-[10px] font-mono uppercase tracking-wider text-nofx-text-muted">
                        <span>Reasoning</span>
                        {p.ai_model && (
                          <span className="px-1.5 py-0.5 rounded bg-white/5">
                            model: {p.ai_model}
                          </span>
                        )}
                        {p.prompt_version && (
                          <span className="px-1.5 py-0.5 rounded bg-white/5">
                            prompt: {p.prompt_version.slice(0, 12)}
                          </span>
                        )}
                        {p.risk_check_error && (
                          <span className="px-1.5 py-0.5 rounded bg-nofx-red/15 text-nofx-red">
                            risk: {p.risk_check_error}
                          </span>
                        )}
                      </div>
                      <pre className="whitespace-pre-wrap font-mono text-xs text-nofx-text-main opacity-90">
                        {a.reasoning || '(no reasoning recorded)'}
                      </pre>
                    </td>
                  </tr>
                )}
              </Fragment>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
