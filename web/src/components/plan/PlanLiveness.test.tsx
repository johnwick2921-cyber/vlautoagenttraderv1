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
  })
})
