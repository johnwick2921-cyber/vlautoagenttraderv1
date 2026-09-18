// W-ARM-STATE-UI (2026-09-18) — the Studio strategy badge becomes TWO truths:
// `trading:` from the real trader binding (strategy_id on /api/my-traders rows)
// and `display:` from the existing is_active flag. The owner read "not active"
// as "my strategy is not trading"; the binding is what trading actually means.

interface Props {
  isActive: boolean
  /** undefined = the binding fetch has not answered (or failed) — render NO
   * trading claim; [] = loaded and genuinely unbound. */
  traderNames?: string[]
  tr: (key: string, params?: Record<string, string>) => string
}

export function StrategyTradingBadge({ isActive, traderNames, tr }: Props) {
  return (
    <span className="flex flex-wrap items-center gap-1">
      {traderNames !== undefined && (
        <span
          data-testid="strategy-trading-badge"
          className="px-1.5 py-0.5 text-[10px] rounded bg-nofx-gold/15 text-nofx-gold"
          title={
            traderNames.length > 0
              ? tr('tradingBoundTo', { names: traderNames.join(', ') })
              : tr('tradingNoBound')
          }
        >
          {traderNames.length > 0
            ? tr('tradingBoundTo', { names: traderNames.join(', ') })
            : tr('tradingNoBound')}
        </span>
      )}
      <span
        data-testid="strategy-display-badge"
        className={`px-1.5 py-0.5 text-[10px] rounded ${
          isActive
            ? 'bg-nofx-success/15 text-nofx-success'
            : 'bg-nofx-bg-lighter text-nofx-text-muted'
        }`}
      >
        {isActive ? tr('displayActive') : tr('displayInactive')}
      </span>
    </span>
  )
}
