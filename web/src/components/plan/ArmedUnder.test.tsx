// T4 (owner ruling 2026-09-03) — a position armed under v2 while the live plan
// is v3: both lines present, armed-under FIRST.

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ArmedUnderBlock, type OpenPositionProvenance } from './ArmedUnderBlock'

// the real 2026-09-03 shape: filled 09:03 under v2 S1 short, card showing v3
const NY_0903: OpenPositionProvenance = {
  symbol: 'MNQ',
  side: 'short',
  entry_price: 29285,
  quantity: 1,
  cited_scenario_id: 'S1',
  armed_under_version: 2,
  live_plan_version: 3,
  version_differs: true,
}

describe('ArmedUnderBlock', () => {
  it('THE PIN: states the position version and the live plan version, armed-under first', () => {
    const { container } = render(
      <ArmedUnderBlock position={NY_0903} language="en" />
    )
    const text = container.textContent ?? ''
    expect(text).toContain('Position armed under')
    expect(text).toContain('v2 S1 short @ 29285.00')
    expect(text).toContain('Plan now')
    expect(text).toContain('v3')
    // ORDER: the position's own terms come before the plan on screen
    expect(text.indexOf('Position armed under')).toBeLessThan(
      text.indexOf('Plan now')
    )
  })

  it('renders nothing when flat', () => {
    const { container } = render(
      <ArmedUnderBlock position={null} language="en" />
    )
    expect(container.textContent).toBe('')
  })

  it('renders nothing when the position and the plan on screen agree', () => {
    const { container } = render(
      <ArmedUnderBlock
        position={{
          ...NY_0903,
          armed_under_version: 3,
          version_differs: false,
        }}
        language="en"
      />
    )
    expect(container.textContent).toBe('')
  })

  it('says "version not recorded" rather than v0 for a pre-attribution row', () => {
    // armed rows 35 and 36 were created 09:02 and 09:20, before the
    // attribution wave booted at 10:28 — both carry 0.
    const { container } = render(
      <ArmedUnderBlock
        position={{
          ...NY_0903,
          armed_under_version: null,
          armed_under_note: 'version not recorded',
        }}
        language="en"
      />
    )
    const text = container.textContent ?? ''
    expect(text).toContain('version not recorded')
    expect(text).not.toContain('v0')
  })
})
