// CLASS 35 (2026-09-01) — the card READS the budget; it never computes one.
//
// The card carried a third formula for the NO-TRADE banner's "used X of Y":
// `noTradeVersion ? noTradeVersion − 2 : version − 1`. The API and the gate
// now agree on a RECORDED counter, and the card must show that number — a chip
// that infers spend from a version number can disagree with both.

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SessionPlanCard } from './SessionPlanCard'
import type { PlanToday, PlanVersionItem } from '../../lib/api/plan'

const doc = {
  reasoning: 'range day',
  bias: { direction: 'long', conviction: 'medium', flip_condition: 'n/a' },
  levels: [],
  scenarios: [],
  no_trade: [],
  death_condition: '2x5m close above 29231.63',
  day_type: 'range',
}

const plan = (over: Partial<PlanToday>): PlanToday => ({
  found: true,
  trade_date: '2026-09-01',
  session: 'LONDON',
  night: false,
  mode: 'advisory',
  version: 6,
  latest_version: 6,
  historical: false,
  lifecycle: 'active',
  model_id: 'deepseek-v4-pro',
  replans_left: 4,
  replan_cap: 4,
  is_active: true,
  warming: '',
  doc,
  level_facts: [],
  ...over,
})

const versions = (n: number, noTradeAt?: number): PlanVersionItem[] =>
  Array.from({ length: n }, (_, i) => i + 1).map((v) => ({
    version: v,
    lifecycle: v === noTradeAt ? 'no_trade' : 'active',
    trigger_reason: 'level_event',
    created_at: `2026-09-01T0${v}:00:00Z`,
    model_id: 'deepseek-v4-pro',
    is_latest: v === n,
    level_count: 6,
    scenario_count: 3,
    bias: 'long',
  }))

describe('SessionPlanCard — class 35 budget is read from the API', () => {
  it("today's LONDON chain: six rows, nothing spent → 4 re-reads left", () => {
    render(
      <SessionPlanCard
        plan={plan({})}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={versions(6)}
        latestVersion={6}
        onSelectVersion={vi.fn()}
      />
    )
    expect(screen.getByText('4 re-reads left')).toBeTruthy()
    expect(screen.queryByTestId('no-trade-banner')).toBeNull()
  })

  it('NO-TRADE banner states the RECORDED spend (cap − replans_left), not version − 2', () => {
    // Marker at v3 with the API saying 0 of 4 left: the old client formula
    // would have printed "1 of 4"; the recorded budget says 4 of 4.
    render(
      <SessionPlanCard
        plan={plan({
          version: 3,
          latest_version: 3,
          lifecycle: 'no_trade',
          replans_left: 0,
          replan_cap: 4,
        })}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={versions(3, 3)}
        latestVersion={3}
        onSelectVersion={vi.fn()}
      />
    )
    const banner = screen.getByTestId('no-trade-banner').textContent!
    expect(banner).toContain('4 of 4 re-plans')
    expect(banner).not.toContain('1 of 4')
  })

  it('never fabricates a spend when the API omits the numbers', () => {
    render(
      <SessionPlanCard
        plan={plan({
          version: 3,
          latest_version: 3,
          lifecycle: 'no_trade',
          replans_left: undefined,
          replan_cap: undefined,
        })}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={versions(3, 3)}
        latestVersion={3}
        onSelectVersion={vi.fn()}
      />
    )
    const banner = screen.getByTestId('no-trade-banner').textContent!
    expect(banner).toContain('? of ? re-plans')
    expect(banner).not.toMatch(/\b[0-9]+ of [0-9]+ re-plans/)
  })
})
