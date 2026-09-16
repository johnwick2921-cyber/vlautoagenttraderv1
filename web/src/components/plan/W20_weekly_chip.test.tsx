// CLASS 50 (refs-only wave) — WEEKLY chip: refs-only state + the card-level
// mount. The chip never carries a direction anymore.
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { WeeklyChip } from './WeeklyChip'
import { SessionPlanCard } from './SessionPlanCard'
import type { PlanToday, PlanWeekly } from '../../lib/api/plan'

const refs: PlanWeekly = {
  refs_only: true,
  pwh: 30500.25,
  pwl: 29980,
  narrative: 'PWH and PWL bracket the accepted range',
  weekly_levels: [
    { name: 'PWH', px: 30500.25 },
    { name: 'PWL', px: 29980 },
  ],
  thin_history: false,
}
const thin: PlanWeekly = { ...refs, pwh: 0, pwl: 0, thin_history: true }

describe('WeeklyChip', () => {
  it('refs-only state — PWH/PWL, no direction token', () => {
    render(<WeeklyChip weekly={refs} />)
    const chip = screen.getByTestId('weekly-chip')
    expect(chip.textContent).toContain('WEEKLY refs')
    expect(chip.textContent).toContain('PWH 30500.25')
    expect(chip.textContent).toContain('PWL 29980.00')
    expect(chip.textContent).not.toContain('bull')
    expect(chip.textContent).not.toContain('bear')
    expect(chip.getAttribute('title')).toContain('refs only')
  })

  it('thin state — refs label with thin marker', () => {
    render(<WeeklyChip weekly={thin} />)
    const chip = screen.getByTestId('weekly-chip')
    expect(chip.textContent).toContain('WEEKLY refs (thin)')
  })

  it('none state — grey WEEKLY none', () => {
    render(<WeeklyChip weekly={null} />)
    const chip = screen.getByTestId('weekly-chip')
    expect(chip.textContent).toContain('WEEKLY none')
    expect(chip.getAttribute('title')).toContain('WEEKLY: none')
  })
})

describe('SessionPlanCard — WEEKLY chip mounts on the card', () => {
  const plan = (weekly: PlanWeekly | null): PlanToday => ({
    found: true,
    trade_date: '2026-08-31',
    session: 'NY',
    night: false,
    mode: 'advisory',
    version: 1,
    lifecycle: 'active',
    model_id: 'deepseek-v4-pro',
    replans_left: 2,
    is_active: true,
    weekly,
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

  it('renders the refs chip when a weekly doc exists', () => {
    render(
      <SessionPlanCard
        plan={plan(refs)}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('weekly-chip').textContent).toContain(
      'WEEKLY refs'
    )
  })

  it('renders the grey none chip when no weekly doc exists', () => {
    render(
      <SessionPlanCard
        plan={plan(null)}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByTestId('weekly-chip').textContent).toContain(
      'WEEKLY none'
    )
  })
})
