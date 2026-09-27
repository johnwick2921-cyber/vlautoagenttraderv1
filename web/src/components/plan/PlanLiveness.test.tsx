import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SessionPlanCard } from './SessionPlanCard'
import type { PlanToday } from '../../lib/api/plan'

describe('plan liveness on the production card', () => {
  it('names exhaustion and renders the recorded death time', () => {
    const plan = {
      found: true,
      trade_date: '2026-09-08',
      session: 'NY',
      version: 2,
      lifecycle: 'active',
      mode: 'advisory',
      night: false,
      doc: {
        reasoning: '',
        bias: { direction: 'short', conviction: 'low', flip_condition: '' },
        levels: [],
        no_trade: [],
        death_condition: '',
        day_type: 'balance',
        scenarios: [
          {
            id: 'S1',
            trigger: 'reject 29753.25',
            condition: 'reject',
            direction: 'short',
            target_chain: [],
            invalid: '',
            quality: 'A',
          },
        ],
      },
      scenario_status: { S1: 'invalidated' },
      scenario_liveness: {
        total: 1,
        tradeable: 0,
        unknown: 0,
        observed_at: '2026-09-08T10:00:00-05:00',
      },
      scenario_deaths: {
        S1: {
          plan_id: '2026-09-08:NY',
          version: 2,
          scenario_id: 'S1',
          anchor: 29753.25,
          price: 29760,
          cause: 'invalidated',
          condition: 'accepted above',
          basis: 'heuristic',
          observed_at: '2026-09-08T10:00:00-05:00',
        },
      },
    } as PlanToday
    render(
      <SessionPlanCard
        plan={plan}
        traderId="test"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('plan-liveness').textContent).toContain(
      'tradeable 0/1'
    )
    expect(screen.getByTestId('plan-liveness').textContent).toContain(
      'EXHAUSTED'
    )
    expect(screen.getByTestId('scenario-death-S1').textContent).toContain(
      '2026-09-08T10:00:00-05:00'
    )
    expect(screen.getByTestId('scenario-death-S1').textContent).toContain(
      '29753.25'
    )
    // W2 — no born check on this (pre-W2) row: the card says n/a, never a policy.
    expect(screen.getByTestId('plan-authored-invalidation').textContent).toBe(
      'invalidation: n/a (no born check recorded on this plan row)'
    )
  })

  // W-EXEC-TRUTH W2 A1/A2 — the invalidation line is READ from the row's
  // born check (plans rowid 455's clocks: read 01:30:27 CT, publish 01:51:47 CT,
  // groups opening 01:30/01:35/01:40/01:45 CT).
  it('reads the born check recorded on the row', () => {
    const plan = {
      found: true,
      trade_date: '2026-09-23',
      session: 'LONDON',
      version: 1,
      lifecycle: 'active',
      mode: 'advisory',
      night: false,
      doc: {
        reasoning: '',
        bias: { direction: 'short', conviction: 'low', flip_condition: '' },
        levels: [],
        no_trade: [],
        death_condition: '',
        day_type: 'balance',
        scenarios: [],
      },
      authored_invalidation: {
        recorded: true,
        policy: 'enforced (grammar)',
        read_clock_ms: 1790145027000,
        publish_clock_ms: 1790146307000,
        groups: [1790145000000, 1790145300000, 1790145600000, 1790145900000],
      },
    } as PlanToday
    const { rerender } = render(
      <SessionPlanCard
        plan={plan}
        traderId="test"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('plan-authored-invalidation').textContent).toBe(
      'invalidation: enforced (grammar) · read 01:30:27 CT → publish 01:51:47 CT · 4 5m groups judged'
    )
    rerender(
      <SessionPlanCard
        plan={{
          ...plan,
          authored_invalidation: {
            recorded: true,
            policy: 'enforced (grammar)',
            read_clock_ms: null,
            publish_clock_ms: 1790146307000,
            groups: [1790145900000],
          },
        }}
        traderId="test"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('plan-authored-invalidation').textContent).toBe(
      'invalidation: enforced (grammar) · read n/a → publish 01:51:47 CT · 1 5m group judged'
    )
  })
})
