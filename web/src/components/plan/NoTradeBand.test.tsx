// NO-TRADE BAND (2026-09-02) — the ASIA-card pin, at the surface the owner saw.
//
// At 23:00 CT the card listed three no-trade rules and every one of them was
// dead: the first 5m had passed six hours earlier, the lunch window belongs to
// NY, and the red-news blackout had fired fourteen hours before. The server now
// stamps each machine window with a status; this block renders only the live
// ones as rules and demotes the model's prose to notes.

import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { RulesBlock, type NoTradeBandWindow } from './RulesBlock'

const PROSE = [
  'no entries in the first 5m after the open',
  'no entries 12:00–13:30 CT (lunch)',
  '🔴 ISM PMI 09:00 CT ±15m — HARD no-trade (red news)',
]

const asiaAt2300: NoTradeBandWindow[] = [
  {
    start_min: 1020,
    end_min: 1025,
    start_ct: '17:00',
    end_ct: '17:05',
    kind: 'first_n',
    label: 'first 5m after the ASIA open',
    source: 'code-constant',
    status: 'elapsed',
  },
  {
    start_min: 720,
    end_min: 810,
    start_ct: '12:00',
    end_ct: '13:30',
    kind: 'lunch',
    label: 'lunch',
    source: 'code-constant',
    status: 'other_session',
  },
  {
    start_min: 525,
    end_min: 555,
    start_ct: '08:45',
    end_ct: '09:15',
    kind: 't1',
    label: '🔴 ISM PMI 09:00 CT ±15m — HARD no-trade (red news)',
    source: 'calendar',
    status: 'other_session',
  },
]

describe('RulesBlock no-trade band', () => {
  it('shows no live constraint on the ASIA card at 23:00 CT', () => {
    render(
      <RulesBlock
        noTrade={PROSE}
        band={asiaAt2300}
        deathCondition=""
        language="en"
      />
    )
    expect(screen.getByText(/none live now/i)).toBeTruthy()
    // the three dead ones are counted, not listed
    expect(screen.getByText(/3 spent \/ other session/i)).toBeTruthy()
    expect(screen.queryByText(/17:00–17:05/)).toBeNull()
  })

  it('lists the spent windows with their reason when expanded', () => {
    const { container } = render(
      <RulesBlock
        noTrade={PROSE}
        band={asiaAt2300}
        deathCondition=""
        language="en"
      />
    )
    expect(container.textContent).not.toContain('17:00–17:05')
    // two toggles exist now (spent windows, model notes) — name the one meant
    fireEvent.click(
      screen.getByRole('button', { name: /spent \/ other session/i })
    )
    const text = container.textContent ?? ''
    expect(text).toContain('(17:00–17:05 CT) · spent')
    expect(text).toContain('(12:00–13:30 CT) · other session')
  })

  it('demotes the model prose to notes, COLLAPSED by default', () => {
    // SUPERSEDED SPEC (owner ruling 2026-09-03): this used to assert the prose
    // rendered inline under a "Model notes" label. Inline was the defect — the
    // ASIA card still printed the two dead windows, one line lower. The notes
    // are a toggle now, so the assertion moves from "is present" to "is present
    // only once asked for".
    const { container } = render(
      <RulesBlock
        noTrade={PROSE}
        band={asiaAt2300}
        deathCondition=""
        language="en"
      />
    )
    const notes = screen.getByRole('button', { name: /model notes/i })
    expect(container.textContent).not.toContain(
      'no entries 12:00–13:30 CT (lunch)'
    )
    fireEvent.click(notes)
    expect(container.textContent).toContain('no entries 12:00–13:30 CT (lunch)')
  })

  it('renders a live window as the rule, with its CT bounds', () => {
    const at1702: NoTradeBandWindow[] = [
      { ...asiaAt2300[0], status: 'live' },
      asiaAt2300[1],
      asiaAt2300[2],
    ]
    const { container } = render(
      <RulesBlock
        noTrade={PROSE}
        band={at1702}
        deathCondition=""
        language="en"
      />
    )
    expect(container.textContent).toContain(
      'first 5m after the ASIA open (17:00–17:05 CT)'
    )
    expect(screen.getByText(/first 5m after the ASIA open/)).toBeTruthy()
    expect(screen.queryByText(/none live now/i)).toBeNull()
    expect(screen.getByText(/2 spent \/ other session/i)).toBeTruthy()
  })

  it('keeps rendering prose as rules for a plan written before the band', () => {
    render(<RulesBlock noTrade={PROSE} deathCondition="" language="en" />)
    expect(screen.queryByText(/Model notes/i)).toBeNull()
    expect(screen.queryByText(/none live now/i)).toBeNull()
    expect(screen.getByText(/lunch/)).toBeTruthy()
  })
})
