// P7 — NO-TRADE PLANS MUST STILL SHOW THE MAP.
//
// Levels are market facts; the plan is an opinion about them. A no-trade
// decision must never erase the map: the fail-closed writers now populate the
// doc's levels from the current detector/scorer output, and the card renders
// them READ-ONLY under the ⛔ NO-TRADE banner — the edit door is shut, with the
// banner as the stated reason.

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SessionPlanCard } from './SessionPlanCard'
import type { PlanToday } from '../../lib/api/plan'

const noTradeWithMap: PlanToday = {
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
      'FAIL-CLOSED: re-plans exhausted (4/4) after 5 deaths — no valid plan produced; sitting out this session.',
    bias: { direction: 'neutral', conviction: 'low', flip_condition: 'n/a' },
    levels: [
      { price: 30203, label: 'ONH', grade: 'A', instruction: 'monitor' },
      { price: 30199.5, label: 'EQL', grade: 'B', instruction: 'monitor' },
    ],
    scenarios: [],
    no_trade: ['ENTIRE SESSION — planner fail-closed'],
    death_condition: 'already dead (fail-closed)',
    day_type: 'no-trade',
  },
  level_facts: [
    {
      price: 30203,
      label: 'ONH',
      grade: 'A',
      instruction: 'monitor',
      distance: 12,
      sweep: false,
      closes_beyond: 0,
      accept_have: 0,
      accept_need: 2,
      still_valid: true,
    },
    {
      price: 30199.5,
      label: 'EQL',
      grade: 'B',
      instruction: 'monitor',
      distance: -8.5,
      sweep: true,
      closes_beyond: 1,
      accept_have: 0,
      accept_need: 2,
      still_valid: true,
    },
  ],
}

describe('SessionPlanCard — NO-TRADE with a map', () => {
  it('renders the levels under the NO-TRADE banner', () => {
    render(
      <SessionPlanCard
        plan={noTradeWithMap}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('no-trade-banner')).toBeInTheDocument()
    expect(screen.getByText('ONH')).toBeInTheDocument()
    expect(screen.getByText('EQL')).toBeInTheDocument()
    // No empty-levels state — the map is present.
    expect(screen.queryByTestId('zone-table-empty')).toBeNull()
  })

  it('shuts the edit door: the map is read-only on a NO-TRADE plan', () => {
    render(
      <SessionPlanCard
        plan={noTradeWithMap}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    // Both header owner-door buttons must be disabled (✎ edit + 💬 ask) because
    // editing a plan the bot is NOT trading is the exact mistake the door exists
    // to prevent. The banner states the reason.
    const headerButtons = screen
      .getAllByRole('button')
      .filter((b) => b.title === 'Edit' || b.title === 'Ask Planner')
    expect(headerButtons.length).toBe(2)
    headerButtons.forEach((b) => expect(b).toBeDisabled())
  })

  it('keeps the old honest behavior: no map at all still says why', () => {
    render(
      <SessionPlanCard
        plan={{
          ...noTradeWithMap,
          doc: { ...noTradeWithMap.doc!, levels: [] },
          level_facts: [],
        }}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('zone-table-empty-reason').textContent).toContain(
      're-plans exhausted'
    )
  })
})
