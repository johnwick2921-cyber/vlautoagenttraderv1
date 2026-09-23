import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PictureGateChip } from './PictureGateChip'
import { SessionPlanCard } from './SessionPlanCard'
import type { PlanToday } from '../../lib/api/plan'

// W-EXEC-TRUTH W0 (CTO Q6) — the card shows the server's strict refusal
// verbatim, and nothing when Picture is off or admitted.
const REFUSAL =
  'refused under strict until W5 (source not yet a Day Plan scenario)'

describe('PictureGateChip', () => {
  it('renders the server refusal verbatim when Picture is on and refused', () => {
    render(<PictureGateChip picture={{ enabled: true, refusal: REFUSAL }} />)
    expect(screen.getByTestId('picture-gate-chip').textContent).toContain(
      REFUSAL
    )
  })
  it('renders nothing when plan mode admits Picture', () => {
    const { container } = render(
      <PictureGateChip picture={{ enabled: true, refusal: '' }} />
    )
    expect(container.textContent).toBe('')
  })
  it('renders nothing when Picture is off or the read is absent', () => {
    const { container } = render(
      <>
        <PictureGateChip picture={{ enabled: false, refusal: REFUSAL }} />
        <PictureGateChip picture={null} />
        <PictureGateChip />
      </>
    )
    expect(container.textContent).toBe('')
  })
})

describe('SessionPlanCard — the Picture strict refusal mounts on the card', () => {
  const base = {
    trade_date: '2026-09-15',
    session: 'NY',
    night: false,
    mode: 'strict',
    is_active: true,
  }
  const found = (picture: PlanToday['picture']): PlanToday => ({
    ...base,
    found: true,
    version: 1,
    lifecycle: 'active',
    model_id: 'deepseek-v4-pro',
    replans_left: 2,
    weekly: null,
    picture,
    doc: {
      reasoning: 'n/a',
      bias: { direction: 'long', conviction: 'medium', flip_condition: 'n/a' },
      levels: [],
      scenarios: [],
      no_trade: [],
      death_condition: 'n/a',
      day_type: 'range',
    },
    level_facts: [],
  })
  const mount = (plan: PlanToday) =>
    render(
      <SessionPlanCard
        plan={plan}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )

  it('a plan card under strict shows the refusal', () => {
    mount(found({ enabled: true, refusal: REFUSAL }))
    expect(screen.getByTestId('picture-gate-chip').textContent).toContain(
      REFUSAL
    )
  })

  it('the no-plan-yet card shows it too', () => {
    mount({
      ...base,
      found: false,
      picture: { enabled: true, refusal: REFUSAL },
    })
    expect(screen.getByTestId('picture-gate-chip').textContent).toContain(
      REFUSAL
    )
  })

  it('outside strict the card shows no Picture refusal', () => {
    mount(found({ enabled: true, refusal: '' }))
    expect(screen.queryByTestId('picture-gate-chip')).toBeNull()
  })
})
