// Wave 1D — the expectancy table's honesty properties, pinned.
//
// Every assertion here is about a way the panel could LIE: showing a verdict on
// a sample too small to carry one, rendering an unmeasured statistic as 0,
// letting a counterfactual row sit in the realized table, hiding the sample, or
// timestamping stale data with the moment the page was opened.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

let payload: unknown = null
vi.mock('../../lib/api/plan', () => ({
  planApi: { getExpectancy: () => Promise.resolve(payload) },
}))

const renderPanel = async () => {
  const { ExpectancyPanel } = await import('./ExpectancyPanel')
  return render(<ExpectancyPanel />)
}

// The table is a collapsed dropdown now (owner ruling 2026-09-03). Every test
// that asserts on the table itself opens it first; the pins for the collapsed
// state use renderPanel directly.
const renderOpen = async () => {
  const r = await renderPanel()
  fireEvent.click(await screen.findByTestId('expectancy-toggle'))
  return r
}

const row = (over: Record<string, unknown> = {}) => ({
  key: { condition: 'reject', session: 'NY' },
  n: 20,
  wins: 12,
  losses: 8,
  flats: 0,
  sum_pnl_corrected: 40,
  mean: 2,
  sd: 10.052493799,
  win_rate: 0.6,
  wilson_lo: 0.3865779423,
  wilson_hi: 0.7811960326,
  mean_lo: -2.4,
  mean_hi: 6.4,
  t_stat: 0.8897565,
  avg_realized_r: null,
  avg_planned_rr: null,
  median_mae: null,
  median_mfe: null,
  stop_hit_share: null,
  target_hit_share: null,
  excluded_unresolved: 3,
  row_ids: [584, 586, 591],
  descriptive_only: true,
  status: 'NOT ENOUGH DATA',
  ...over,
})

const base = (over: Record<string, unknown> = {}) => ({
  by: 'condition',
  rows: [row()],
  counterfactual_e8: [],
  excluded: {
    unresolved_pnl: 3,
    unresolvable: 1,
    test_seam: 2,
    no_condition: 0,
    crypto_era: 0,
  },
  min_n: 30,
  as_of_ms: 1788000000000,
  as_of_utc: '2026-09-01T18:41:14Z',
  built_at_ms: 1788200000000,
  era_0b_start: '2026-09-02T12:49:06Z',
  promotion_rule:
    'n >= min_n AND mean > 0 AND the 95% interval on the mean excludes 0',
  ...over,
})

describe('ExpectancyPanel', () => {
  beforeEach(() => {
    payload = null
  })

  it('shows n before any statistic and marks a sub-floor row DESCRIPTIVE ONLY', async () => {
    payload = base()
    await renderOpen()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-panel')).toBeTruthy()
    )

    const cells = screen.getByTestId('expectancy-row-0').textContent ?? ''
    // n is the first number the eye meets — the floor is a property of n.
    expect(cells.indexOf('20')).toBeLessThan(cells.indexOf('2.00'))
    expect(screen.getByTestId('expectancy-status-0').textContent).toMatch(
      /DESCRIPTIVE ONLY/i
    )
    // and NO verdict is rendered from a sample below the floor
    expect(cells).not.toMatch(/PASSES|FAILS/)
  })

  it('renders a verdict only at or above the floor that the payload carries', async () => {
    payload = base({
      rows: [
        row({
          n: 30,
          mean: 16.6666667,
          mean_lo: 9.8037171,
          descriptive_only: false,
          status: 'PASSES',
        }),
      ],
    })
    await renderOpen()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-panel')).toBeTruthy()
    )
    expect(screen.getByTestId('expectancy-status-0').textContent).toMatch(
      /PASSES/
    )
  })

  it('renders an unmeasured statistic blank, never as zero', async () => {
    payload = base()
    await renderOpen()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-panel')).toBeTruthy()
    )
    const mae = screen.getByTestId('expectancy-mae-0').textContent ?? ''
    expect(mae.trim()).toBe('—')
    expect(mae).not.toMatch(/0/)
  })

  it('exposes the row ids so any figure can be recomputed', async () => {
    payload = base()
    await renderOpen()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-panel')).toBeTruthy()
    )
    fireEvent.click(screen.getByTestId('expectancy-ids-toggle-0'))
    expect(screen.getByTestId('expectancy-ids-0').textContent).toMatch(
      /584.*586.*591/
    )
  })

  it('dates the table by the last closed trade, not by page load', async () => {
    payload = base()
    await renderOpen()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-panel')).toBeTruthy()
    )
    const asOf = screen.getByTestId('expectancy-as-of').textContent ?? ''
    expect(asOf).toMatch(/2026-09-01/)
    expect(asOf).not.toMatch(/2026-09-03/)
  })

  it('states what was excluded and why, beside the table', async () => {
    payload = base()
    await renderOpen()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-panel')).toBeTruthy()
    )
    const ex = screen.getByTestId('expectancy-excluded').textContent ?? ''
    expect(ex).toMatch(/3/)
    expect(ex).toMatch(/unresolved/i)
    expect(ex).toMatch(/test.seam/i)
  })

  it('keeps counterfactual rows in their own table, flagged', async () => {
    payload = base({
      counterfactual_e8: [
        {
          key: { condition: 'sweep_reclaim', session: 'NY' },
          rule: 'touch',
          n: 9,
          usable_n: 4,
          wins: 4,
          losses: 5,
          sum_net_pnl: -12,
          mean: -1.3333,
          excluded_price_scale: 3,
          excluded_zero_pnl: 2,
          counterfactual: true,
          short_suspect: true,
          note: 'counterfactual (E8) · SHORT ROWS SUSPECT (E8 sign bug)',
        },
      ],
    })
    await renderOpen()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-panel')).toBeTruthy()
    )
    const cf = screen.getByTestId('expectancy-counterfactual')
    expect(cf.textContent).toMatch(/counterfactual/i)
    expect(cf.textContent).toMatch(/SUSPECT/i)
    // the realized table must not have absorbed it
    expect(screen.getByTestId('expectancy-row-0').textContent).not.toMatch(
      /sweep_reclaim/
    )
  })

  it('says so when there is nothing to show, instead of rendering an empty table', async () => {
    payload = base({ rows: [] })
    await renderOpen()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-empty')).toBeTruthy()
    )
  })

  it('shows why, not a number, when no counterfactual row is usable', async () => {
    payload = base({
      counterfactual_e8: [
        {
          key: { condition: 'reject', session: 'NY' },
          rule: '1x5m_close',
          n: 78,
          usable_n: 0,
          wins: 30,
          losses: 48,
          sum_net_pnl: null,
          mean: null,
          excluded_price_scale: 40,
          excluded_zero_pnl: 38,
          counterfactual: true,
          short_suspect: true,
          note: 'NO USABLE net_pnl',
        },
      ],
    })
    await renderOpen()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-panel')).toBeTruthy()
    )
    const cf = screen.getByTestId('expectancy-cf-0').textContent ?? ''
    expect(cf).toMatch(/0 usable/)
    expect(cf).toMatch(/40 not a P&L/)
    expect(cf).toMatch(/mean\s*—/)
  })
})

// SIBLING, NOT CHILD (owner ruling 2026-09-03). The drawer used to be rendered
// from inside this panel, below the table. The panel returns null whenever the
// expectancy endpoint gives it nothing — so on a day with no expectancy the
// three instruments vanished with it, for a reason no reader could connect to
// the drawer. The drawer is a sibling in PlanCard now, and this pins that the
// coupling cannot come back.
describe('the instruments drawer is not inside the expectancy panel', () => {
  it('does not render the drawer even when the panel has data', async () => {
    payload = base()
    await renderPanel()
    await waitFor(() =>
      expect(screen.getByTestId('expectancy-panel')).toBeTruthy()
    )
    expect(screen.queryByTestId('instruments-drawer')).toBeNull()
  })
})

// FOLDED (owner ruling 2026-09-03) — the expectancy table is a collapsed
// dropdown, same convention as the instruments drawer below it: a visible
// labelled toggle, not a 9px caption. The "as of" clock stays on the CLOSED
// toggle, because a reader deciding whether to open the table needs to know how
// old it is before they open it, not after.
describe('the expectancy table is a collapsed dropdown', () => {
  beforeEach(() => {
    payload = null
  })

  it('is COLLAPSED on load, showing the label and the as-of only', async () => {
    payload = base()
    await renderPanel()
    const btn = await screen.findByTestId('expectancy-toggle')
    expect(btn.textContent).toMatch(/Expectancy · by condition/)
    expect(btn.textContent).toMatch(/as of 2026-09-01/)
    // nothing from the table itself until it is opened
    expect(screen.queryByTestId('expectancy-row-0')).toBeNull()
    expect(screen.queryByTestId('expectancy-excluded')).toBeNull()
  })

  it('expands to the full table, status column and all', async () => {
    payload = base()
    await renderPanel()
    fireEvent.click(await screen.findByTestId('expectancy-toggle'))
    expect(screen.getByTestId('expectancy-row-0')).toBeTruthy()
    expect(screen.getByTestId('expectancy-status-0').textContent).toMatch(
      /DESCRIPTIVE ONLY/
    )
    expect(screen.getByTestId('expectancy-excluded')).toBeTruthy()
  })

  it('keeps the as-of on the closed toggle, not only inside', async () => {
    payload = base()
    await renderPanel()
    await screen.findByTestId('expectancy-toggle')
    const asOf = screen.getByTestId('expectancy-as-of')
    expect(
      screen.getByTestId('expectancy-toggle').contains(asOf),
      'the as-of clock must be readable without opening the table'
    ).toBe(true)
  })

  it('folds the E8 counterfactual block inside, under its own sub-heading', async () => {
    payload = base({
      counterfactual_e8: [
        {
          key: { condition: 'reject', session: 'ASIA' },
          rule: 'touch',
          n: 7,
          usable_n: 0,
          wins: 0,
          losses: 0,
          mean: null,
          excluded_price_scale: 0,
          excluded_zero_pnl: 7,
          short_suspect: false,
        },
      ],
    })
    await renderPanel()
    await screen.findByTestId('expectancy-toggle')
    expect(screen.queryByTestId('expectancy-counterfactual')).toBeNull()
    fireEvent.click(screen.getByTestId('expectancy-toggle'))
    const cf = screen.getByTestId('expectancy-counterfactual')
    expect(cf.textContent).toMatch(/counterfactual \(E8\)/)
    expect(screen.getByTestId('expectancy-cf-0').textContent).toMatch(/usable/)
  })

  it('is a real control at the drawer convention, not a 9px caption', async () => {
    payload = base()
    await renderPanel()
    const btn = await screen.findByTestId('expectancy-toggle')
    expect(btn.getAttribute('type')).toBe('button')
    expect(btn.getAttribute('aria-expanded')).toBe('false')
    expect(btn.style.color).toBe('var(--vl-muted)')
    expect(btn.className).toMatch(/text-\[11px\]/)
    expect(btn.textContent).toMatch(/▸/)
    fireEvent.click(btn)
    expect(btn.getAttribute('aria-expanded')).toBe('true')
    expect(btn.textContent).toMatch(/▾/)
  })
})
