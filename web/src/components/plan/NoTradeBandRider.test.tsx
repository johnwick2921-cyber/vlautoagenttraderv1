// NO-TRADE BAND rider (owner ruling 2026-09-03) — the card shows the MACHINE's
// live constraints and nothing else by default.
//
// The first cut demoted the model's prose to "Model notes" but still rendered
// it inline, so an ASIA card at 00:00 read "No-trade: none live now · Model
// notes · first 5m (CT) · 12:00-13:30 CT lunch" — the same two dead windows the
// wave exists to stop showing, one line lower. Notes are behind a toggle now.

import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { RulesBlock, type NoTradeBandWindow } from './RulesBlock'

// exactly what plan 2026-09-02 ASIA v14 carries
const V14_PROSE = ['first 5m (CT)', '12:00-13:30 CT lunch']
const V14_BAND: NoTradeBandWindow[] = [
  {
    start_min: 1020,
    end_min: 1025,
    start_ct: '17:00',
    end_ct: '17:05',
    kind: 'first_n',
    label: 'first 5m after the ASIA open (17:00–17:05 CT)',
    source: 'code-constant',
    status: 'elapsed',
  },
  {
    start_min: 720,
    end_min: 810,
    start_ct: '12:00',
    end_ct: '13:30',
    kind: 'lunch',
    label: 'lunch 12:00–13:30 CT',
    source: 'code-constant',
    status: 'other_session',
  },
]

describe('RulesBlock rider — machine windows only, notes collapsed', () => {
  it('THE PIN: an ASIA doc at 00:00 says none live and shows no prose', () => {
    const { container } = render(
      <RulesBlock
        noTrade={V14_PROSE}
        band={V14_BAND}
        deathCondition=""
        language="en"
      />
    )
    expect(screen.getByText(/none live/i)).toBeTruthy()
    const text = container.textContent ?? ''
    for (const line of V14_PROSE) {
      expect(text).not.toContain(line)
    }
    // and no elapsed window's bounds are inline either
    expect(text).not.toContain('17:00–17:05')
    expect(text).not.toContain('12:00–13:30 CT)')
  })

  it('model notes are behind a toggle, and open on demand', () => {
    const { container } = render(
      <RulesBlock
        noTrade={V14_PROSE}
        band={V14_BAND}
        deathCondition=""
        language="en"
      />
    )
    const notes = screen.getByRole('button', { name: /model notes/i })
    expect(container.textContent).not.toContain('first 5m (CT)')
    fireEvent.click(notes)
    expect(container.textContent).toContain('first 5m (CT)')
    expect(container.textContent).toContain('12:00-13:30 CT lunch')
    fireEvent.click(notes)
    expect(container.textContent).not.toContain('first 5m (CT)')
  })

  it('a live window is the rule, and prose stays hidden beside it', () => {
    const live: NoTradeBandWindow[] = [
      { ...V14_BAND[0], status: 'live' },
      V14_BAND[1],
    ]
    const { container } = render(
      <RulesBlock
        noTrade={V14_PROSE}
        band={live}
        deathCondition=""
        language="en"
      />
    )
    // the machine label already carries its bounds — they render ONCE
    expect(container.textContent).toContain(
      'first 5m after the ASIA open (17:00–17:05 CT)'
    )
    expect(container.textContent).not.toContain('CT) (17:00–17:05 CT)')
    expect(container.textContent).not.toContain('first 5m (CT)')
  })

  it('a plan with no machine band at all still renders its prose as rules', () => {
    const { container } = render(
      <RulesBlock noTrade={V14_PROSE} deathCondition="" language="en" />
    )
    expect(container.textContent).toContain('first 5m (CT)')
    expect(screen.queryByRole('button', { name: /model notes/i })).toBeNull()
  })
})
