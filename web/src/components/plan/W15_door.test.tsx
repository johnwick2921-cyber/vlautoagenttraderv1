// W15.C REGRESSION GUARD — the owner door must only open on the LIVE session.
//
// Every mutating endpoint (/plan/overlay, /plan/ask, /plan/realign) resolves
// reg.ActiveSession(now) SERVER-SIDE and writes the active session's plan; none
// takes a session argument. Before the tabs became real the card always showed
// the active session, so "what you see" == "what you edit". Once a sibling
// session could be displayed, an edit made while looking at ASIA would have
// landed silently on NY's plan.

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SessionPlanCard } from './SessionPlanCard'
import type { PlanToday } from '../../lib/api/plan'

const base: PlanToday = {
  found: true,
  trade_date: '2026-08-17',
  session: 'NY',
  night: false,
  mode: 'advisory',
  version: 1,
  lifecycle: 'active',
  replans_left: 2,
  doc: {
    reasoning: 'r',
    bias: { direction: 'long', conviction: 'medium', flip_condition: 'f' },
    levels: [],
    scenarios: [],
    no_trade: [],
    death_condition: 'd',
  },
  level_facts: [],
  price: 30000,
}

const renderCard = (plan: PlanToday) =>
  render(
    <SessionPlanCard
      plan={plan}
      traderId="t1"
      symbol="MNQ"
      exchange="ninjatrader"
      language="en"
      isLoading={false}
      errored={false}
      onChanged={vi.fn()}
    />
  )

describe('SessionPlanCard · owner door is scoped to the live session', () => {
  it('viewing a SIBLING session shows the read-only note', () => {
    renderCard({
      ...base,
      session: 'ASIA',
      active_session: 'NY',
      is_active: false,
    })
    expect(screen.getByTestId('sibling-session-readonly')).toBeTruthy()
  })

  it('viewing the LIVE session does NOT show it', () => {
    renderCard({
      ...base,
      session: 'NY',
      active_session: 'NY',
      is_active: true,
    })
    expect(screen.queryByTestId('sibling-session-readonly')).toBeNull()
  })

  it('an older payload without is_active keeps the pre-W15.C behavior (door open)', () => {
    renderCard(base) // is_active undefined
    expect(screen.queryByTestId('sibling-session-readonly')).toBeNull()
  })
})

// A fail-closed / re-plans-exhausted plan writes lifecycle='no_trade' — the ONE
// non-active value production ever writes. It was missing from LifecycleChip's
// map, so the `?? map.active` fallback painted it GOLD "ACTIVE": a plan sitting
// the session out looked live.
describe('LifecycleChip · no_trade must not look active', () => {
  it('renders NO-TRADE, not ACTIVE', async () => {
    const { LifecycleChip } = await import('./chips')
    render(<LifecycleChip lifecycle="no_trade" language="en" />)
    expect(screen.getByText('NO-TRADE')).toBeTruthy()
    expect(screen.queryByText('ACTIVE')).toBeNull()
  })

  it('still renders ACTIVE for an active plan', async () => {
    const { LifecycleChip } = await import('./chips')
    render(<LifecycleChip lifecycle="active" language="en" />)
    expect(screen.getByText('ACTIVE')).toBeTruthy()
  })
})
