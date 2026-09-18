// W-ARM-STATE-UI — the dormant chip must render the LIFECYCLE reason (death vs
// flip), never trigger_reason. NY v2 went dormant on its death line and the
// card said "dormant: structure_flip" — the owner read that as a flip.

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { dormantReasonLabel, SessionPlanCard } from './SessionPlanCard'
import type { PlanToday } from '../../lib/api/plan'

describe('dormantReasonLabel', () => {
  it('labels a death marker as death, not the authoring trigger', () => {
    expect(
      dormantReasonLabel(
        'dormant:death:death-condition: 5m_close close below 29767.00'
      )
    ).toBe('death — 5m_close close below 29767.00')
  })
  it('labels a flip marker as flip', () => {
    expect(
      dormantReasonLabel(
        'dormant:flip:flip-condition: 2x5m close above 100 → bias long'
      )
    ).toBe('flip — 2x5m close above 100 → bias long')
  })
  it('returns nothing when the marker is absent', () => {
    expect(dormantReasonLabel(undefined)).toBe('')
    expect(dormantReasonLabel('')).toBe('')
  })
})

describe('SessionPlanCard · dormant banner uses the lifecycle reason', () => {
  const base: PlanToday = {
    found: true,
    trade_date: '2026-09-18',
    session: 'NY',
    night: false,
    mode: 'advisory',
    version: 2,
    lifecycle: 'dormant',
    trigger_reason: 'structure_flip',
    lifecycle_reason:
      'dormant:death:death-condition: 5m_close close below 29767.00',
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

  it('renders death, not the structure_flip authoring trigger', () => {
    render(
      <SessionPlanCard
        plan={base}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        isLoading={false}
        errored={false}
        onChanged={() => {}}
      />
    )
    const banner = screen.getByTestId('dormant-banner')
    expect(banner.textContent).toContain('dormant: death —')
    expect(banner.textContent).not.toContain('structure_flip')
  })

  it('renders the bare dormant line when no marker exists', () => {
    const { lifecycle_reason: _omit, ...noReason } = base
    render(
      <SessionPlanCard
        plan={noReason}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        isLoading={false}
        errored={false}
        onChanged={() => {}}
      />
    )
    const banner = screen.getByTestId('dormant-banner')
    expect(banner.textContent).toContain('dormant —')
    expect(banner.textContent).not.toContain('structure_flip')
    expect(banner.textContent).not.toContain('death')
  })
})
