// ITEM 2 (2026-08-17) — THE 💬 THREAD MUST OPEN WITH NO ACTIVE PLAN.
//
// Every no-plan state returned a bare StatePanel, so there was no ask affordance
// at all — and "why did the plan die?" / "why no levels tonight?" are exactly the
// questions that only exist once the plan is gone.

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SessionPlanCard } from './SessionPlanCard'
import { AskPlannerPanel } from './AskPlannerPanel'
import type { PlanToday } from '../../lib/api/plan'

const nightPlan: PlanToday = {
  found: false,
  trade_date: '2026-08-16',
  session: '',
  night: true,
  mode: 'advisory',
}

describe('SessionPlanCard — asking with no plan', () => {
  it('offers the planner thread in the NIGHT state', () => {
    render(
      <SessionPlanCard
        plan={nightPlan}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByText(/Night/i)).toBeInTheDocument()
    expect(screen.getByTestId('ask-anyway')).toBeInTheDocument()
  })

  it('offers it in the plan-not-armed-yet state', () => {
    render(
      <SessionPlanCard
        plan={{ ...nightPlan, night: false, session: 'NY' }}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('ask-anyway')).toBeInTheDocument()
  })

  it('opens the thread labelled NO PLAN so it cannot read as live', () => {
    render(
      <SessionPlanCard
        plan={nightPlan}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    fireEvent.click(screen.getByTestId('ask-anyway'))
    const label = screen.getByTestId('ask-context-label').textContent!
    expect(label).toContain('NO PLAN')
    expect(label).toContain('live market facts only')
  })

  it('does not offer it without a trader (nothing to scope the thread to)', () => {
    render(
      <SessionPlanCard
        plan={nightPlan}
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.queryByTestId('ask-anyway')).toBeNull()
  })
})

describe('AskPlannerPanel — context labelling', () => {
  it('shows no banner for a live plan', () => {
    render(
      <AskPlannerPanel
        open
        traderId="t1"
        symbol="MNQ"
        language="en"
        onClose={vi.fn()}
        onApplied={vi.fn()}
      />
    )
    expect(screen.queryByTestId('ask-context-label')).toBeNull()
  })

  it('warns when the thread is historical', () => {
    render(
      <AskPlannerPanel
        open
        traderId="t1"
        symbol="MNQ"
        language="en"
        contextLabel="HISTORICAL CONTEXT — answering about the last stored plan, not a live one"
        onClose={vi.fn()}
        onApplied={vi.fn()}
      />
    )
    expect(screen.getByTestId('ask-context-label').textContent).toContain(
      'HISTORICAL'
    )
  })
})
