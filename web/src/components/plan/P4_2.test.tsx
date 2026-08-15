// P4.2 — timeline / tabs / handover tests.

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SessionTimelineStrip } from './SessionTimelineStrip'
import { SessionTabs, type SessionTab } from './SessionTabs'
import { HandoverBanner } from './HandoverBanner'
import {
  ctToMinutes,
  toSegments,
  inWindow,
  nowCTMinutes,
} from './sessionConfig'

describe('sessionConfig', () => {
  it('ctToMinutes parses HH:MM', () => {
    expect(ctToMinutes('08:30')).toBe(510)
    expect(ctToMinutes('00:00')).toBe(0)
    expect(ctToMinutes('17:00')).toBe(1020)
  })

  it('toSegments splits a midnight-wrapping window into two', () => {
    // Asia 17:00 → 02:00 wraps.
    expect(toSegments(1020, 120)).toEqual([
      [1020, 1440],
      [0, 120],
    ])
    // NY 08:30 → 15:00 does not wrap.
    expect(toSegments(510, 900)).toEqual([[510, 900]])
  })

  it('inWindow handles wrap correctly', () => {
    expect(inWindow(1030, 1020, 120)).toBe(true) // 17:10 in Asia
    expect(inWindow(60, 1020, 120)).toBe(true) // 01:00 in Asia (wrapped)
    expect(inWindow(600, 1020, 120)).toBe(false) // 10:00 not in Asia
    expect(inWindow(600, 510, 900)).toBe(true) // 10:00 in NY
  })

  it('nowCTMinutes returns a valid minute-of-day', () => {
    const m = nowCTMinutes(new Date('2026-08-15T18:30:00Z'))
    expect(m).toBeGreaterThanOrEqual(0)
    expect(m).toBeLessThan(1440)
  })
})

describe('SessionTimelineStrip', () => {
  it('renders session legend labels + now marker', () => {
    render(
      <SessionTimelineStrip
        activeSession="NY"
        language="en"
        now={new Date('2026-08-15T14:00:00Z')}
      />
    )
    expect(screen.getAllByText('New York').length).toBeGreaterThan(0)
    expect(screen.getByRole('img')).toHaveAttribute('aria-label')
  })
})

describe('SessionTabs', () => {
  const tabs: SessionTab[] = [
    { name: 'ASIA', state: 'disabled' },
    { name: 'LONDON', state: 'disabled' },
    { name: 'NY', state: 'active' },
  ]

  it('renders a tablist with the active tab selected', () => {
    render(
      <SessionTabs
        tabs={tabs}
        selected="NY"
        onSelect={() => {}}
        language="en"
      />
    )
    expect(screen.getByRole('tablist')).toBeInTheDocument()
    const ny = screen.getByRole('tab', { name: /New York/i })
    expect(ny).toHaveAttribute('aria-selected', 'true')
  })

  it('does not fire onSelect for a disabled tab', () => {
    const onSelect = vi.fn()
    render(
      <SessionTabs
        tabs={tabs}
        selected="NY"
        onSelect={onSelect}
        language="en"
      />
    )
    fireEvent.click(screen.getByRole('tab', { name: /Asia/i }))
    expect(onSelect).not.toHaveBeenCalled()
  })
})

describe('HandoverBanner', () => {
  it('renders the born phase with the session name', () => {
    render(<HandoverBanner phase="born" session="NY" language="en" />)
    expect(screen.getByRole('status')).toHaveTextContent(
      /New York plan is live/i
    )
  })

  it('renders the read-failed phase (fail-closed)', () => {
    render(
      <HandoverBanner phase="read-failed" session="LONDON" language="en" />
    )
    expect(screen.getByRole('status')).toHaveTextContent(/read failed/i)
  })

  it('fires onDismiss when the close button is clicked', () => {
    const onDismiss = vi.fn()
    render(
      <HandoverBanner
        phase="expired"
        session="NY"
        language="en"
        onDismiss={onDismiss}
      />
    )
    fireEvent.click(screen.getByLabelText(/Dismiss/i))
    expect(onDismiss).toHaveBeenCalled()
  })
})
