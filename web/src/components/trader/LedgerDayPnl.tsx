// P&L-TRUTH WAVE (2026-09-01) — the dashboard-header ledger day total.
//
// The header used to show the NT8-native figure (permanently 0.00) beside a
// +212.00 day total in the position-history footer. This chip renders the
// SAME number the footer computes (server-side, strict pnl_corrected), with
// its resolved n and unresolved exclusion count — never a rate or a total
// without its n.
import type { AccountInfo } from '../../types/trading'

export function ledgerDayLabel(
  account: Pick<
    AccountInfo,
    | 'ledger_day_pnl'
    | 'ledger_day_resolved'
    | 'ledger_day_unresolved'
    | 'ledger_day_status'
  >
): string {
  if (typeof account.ledger_day_pnl !== 'number') {
    return `LEDGER_DAY::${account.ledger_day_status ?? 'UNRESOLVED'}`
  }
  const sign = account.ledger_day_pnl >= 0 ? '+' : ''
  const resolved = account.ledger_day_resolved ?? 0
  const unresolved = account.ledger_day_unresolved ?? 0
  return `LEDGER_DAY::${sign}${account.ledger_day_pnl.toFixed(2)} (${resolved} resolved, ${unresolved} unresolved excluded)`
}

export function LedgerDayPnl({ account }: { account: AccountInfo }) {
  const positive = (account.ledger_day_pnl ?? 0) >= 0
  return (
    <span
      data-testid="ledger-day-pnl"
      title="Ledger day total — same rule as the position-history footer: pnl_corrected only; unresolved rows counted and excluded, never coerced"
      style={{
        color:
          typeof account.ledger_day_pnl === 'number'
            ? positive
              ? '#0ECB81'
              : '#F6465D'
            : 'var(--vl-faint)',
      }}
    >
      {ledgerDayLabel(account)}
    </span>
  )
}
