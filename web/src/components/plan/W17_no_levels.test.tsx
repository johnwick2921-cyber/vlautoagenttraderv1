// P0 2026-08-17 — A LEVELS-LESS PLAN MUST SAY WHY.
//
// What happened: 2026-08-16:ASIA v1..v5 each held 6 good levels and each was
// killed on arrival (activePlanIsDead judged them against the whole 33h bar
// cache), the re-plan budget ran out, and writeNoTradePlan stored a levels:null
// NO-TRADE plan as v6. The card rendered that faithfully — a bare "No levels in
// this plan", no reason, and (because the chart was gated on having levels) no
// chart either. For a whole session that was indistinguishable from a rendering
// bug, so the real fault stayed invisible.
//
// These tests pin the loud version: the reason is always on the card, an
// unexplained emptiness names itself a fault, and the bars render regardless.

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SessionPlanCard } from './SessionPlanCard'
import { ZoneTable } from './ZoneTable'
import type { PlanToday } from '../../lib/api/plan'

// The real v6 doc, as NoTradePlanDoc() writes it (kernel/plan_doc.go:181).
const noTradePlan: PlanToday = {
  found: true,
  trade_date: '2026-08-16',
  session: 'ASIA',
  night: false,
  mode: 'advisory',
  version: 6,
  lifecycle: 'no_trade',
  model_id: 'deepseek-reasoner',
  replans_left: 0,
  warming: '',
  doc: {
    reasoning:
      'FAIL-CLOSED: re-plans exhausted after death condition — no valid plan produced; sitting out this session.',
    bias: { direction: 'neutral', conviction: 'low', flip_condition: 'n/a' },
    levels: [],
    scenarios: [],
    no_trade: ['ENTIRE SESSION — planner fail-closed'],
    death_condition: 'already dead (fail-closed)',
    day_type: 'no-trade',
  },
  level_facts: [],
}

describe('ZoneTable — zero levels', () => {
  it('states the supplied reason, not just "no levels"', () => {
    render(
      <ZoneTable
        facts={[]}
        language="en"
        emptyReason="FAIL-CLOSED: re-plans exhausted after death condition"
      />
    )
    expect(screen.getByText('No levels in this plan')).toBeInTheDocument()
    expect(screen.getByTestId('zone-table-empty-reason').textContent).toContain(
      're-plans exhausted after death condition'
    )
  })

  it('names an UNEXPLAINED emptiness as a fault rather than a quiet session', () => {
    render(<ZoneTable facts={[]} language="en" />)
    const reason = screen.getByTestId('zone-table-empty-reason').textContent!
    expect(reason).toContain('gave no reason')
    expect(reason).toContain('this is a fault')
  })

  it('shows no empty-state at all once levels exist', () => {
    render(
      <ZoneTable
        facts={[
          {
            price: 30203,
            label: 'ONH',
            grade: 'A',
            instruction: 'breakout_retest',
            distance: 12,
            sweep: false,
            closes_beyond: 0,
            accept_have: 0,
            accept_need: 2,
            still_valid: true,
          },
        ]}
        language="en"
        emptyReason="should not be rendered"
      />
    )
    expect(screen.queryByTestId('zone-table-empty')).toBeNull()
    expect(screen.queryByText(/should not be rendered/)).toBeNull()
  })
})

describe('SessionPlanCard — the v6 NO-TRADE plan', () => {
  it('carries the planner reason down to the empty level table', () => {
    render(
      <SessionPlanCard
        plan={noTradePlan}
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('zone-table-empty-reason').textContent).toContain(
      're-plans exhausted after death condition'
    )
  })

  it('puts the reason in the fail-closed banner instead of a generic hint', () => {
    render(
      <SessionPlanCard
        plan={noTradePlan}
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    const banner = screen.getByTestId('fail-closed-reason').textContent!
    expect(banner).toContain('FAIL-CLOSED')
    expect(banner).toContain('re-plans exhausted')
  })

  it('still renders the chart region when the plan has no levels', () => {
    // The bars are real even when the plan is empty — hiding the chart removed
    // the one view that showed the market was fine and the PLAN was the problem.
    render(
      <SessionPlanCard
        plan={noTradePlan}
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('mini-chart-no-levels')).toBeInTheDocument()
  })

  it('falls back to no_trade when the doc carries no reasoning', () => {
    const noReasoning: PlanToday = {
      ...noTradePlan,
      doc: { ...noTradePlan.doc!, reasoning: '' },
    }
    render(
      <SessionPlanCard
        plan={noReasoning}
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('zone-table-empty-reason').textContent).toContain(
      'ENTIRE SESSION'
    )
  })
})
