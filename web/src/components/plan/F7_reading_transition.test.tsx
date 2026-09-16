// F7 (2026-08-30) — the plan card must TRANSITION off the writing state.
//
// 2026-08-30 live failure: wake re-reads held the in-flight claim back-to-back
// for hours while a committed, tradeable plan row sat in the DB, so the
// claim-keyed "writing a fresh plan" banner never cleared. The reading status
// now derives from the STORE: "writing" only while NO plan row exists; a
// committed row always renders the plan (a running read becomes the subtle
// re-reading chip). A failed read — tonight's benign wake failures — lands
// back on the kept plan, never on a stuck writing state.

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SessionPlanCard } from './SessionPlanCard'
import type { PlanToday } from '../../lib/api/plan'

const committedPlan: PlanToday = {
  found: true,
  trade_date: '2026-08-30',
  session: 'ASIA',
  night: false,
  mode: 'strict',
  version: 2,
  lifecycle: 'active',
  model_id: 'deepseek-v4-pro',
  warming: '',
  doc: {
    reasoning: 'Bias-tree: inside-day long LOW. Fade the open.',
    bias: {
      direction: 'long',
      conviction: 'low',
      flip_condition: '2x5m < 29494',
    },
    levels: [
      { price: 29535, label: 'weekly_open', grade: 'A', instruction: 'fade' },
      { price: 29494.38, label: 'POC', grade: 'B', instruction: 'monitor' },
    ],
    scenarios: [],
    no_trade: ['first 5m'],
    death_condition: 'acceptance above 29535',
    day_type: 'balance',
  },
  level_facts: [],
}

const prePlanWriting: PlanToday = {
  found: false,
  trade_date: '2026-08-30',
  session: 'ASIA',
  night: false,
  mode: 'strict',
  reading: true,
}

function renderCard(plan: PlanToday) {
  return render(
    <SessionPlanCard
      plan={plan}
      traderId="t1"
      symbol="MNQ"
      exchange="ninjatrader"
      language="en"
    />
  )
}

describe('SessionPlanCard — F7 writing→plan transition', () => {
  it('shows the writing state while a read is in flight and NO plan is committed', () => {
    renderCard(prePlanWriting)
    expect(screen.getByTestId('reading-banner').textContent).toContain(
      'writing a fresh plan'
    )
  })

  it('transitions to the rendered plan when the write commits (next poll)', () => {
    const { rerender } = renderCard(prePlanWriting)
    expect(screen.getByTestId('reading-banner')).toBeTruthy()

    // The next poll lands: row committed, reading cleared.
    rerender(
      <SessionPlanCard
        plan={committedPlan}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.queryByTestId('reading-banner')).toBeNull()
    expect(screen.getByRole('table')).toBeTruthy()
  })

  it('read-failed lands back on the kept plan, never a stuck writing state', () => {
    // Tonight's benign wake-read failures: the read ended without a write; the
    // committed plan stays. reading=false in the response (store-derived).
    renderCard({ ...committedPlan, reading: false, replan_in_flight: false })
    expect(screen.queryByTestId('reading-banner')).toBeNull()
    expect(screen.getByRole('table')).toBeTruthy()
  })

  it('a read running over a committed plan shows the subtle chip, not writing', () => {
    renderCard({ ...committedPlan, replan_in_flight: true })
    expect(screen.getByTestId('replan-chip').textContent).toContain(
      'stays live'
    )
    expect(screen.queryByTestId('reading-banner')).toBeNull()
    expect(screen.getByRole('table')).toBeTruthy()
  })
})
