// THE DESK STRIP — each pin is a way the strip could lie.
//
// The strip exists because the surfaces above it answered narrower questions
// than they looked like they were answering. These tests hold it to its own
// law: nothing undated, UNKNOWN carries its reason, a dead endpoint is stated
// rather than blanked, and the ledger's price is never shown as the broker's.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react'

vi.mock('../../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))

let stripPayload: unknown = null
vi.mock('../../lib/api', () => ({
  api: { getDeskStrip: () => Promise.resolve(stripPayload) },
}))

const line = (over: Record<string, unknown> = {}) => ({
  n: 1,
  key: 'mode',
  label: 'MODE',
  text: 'SIM · plan_mode=strict · session=NY',
  state: 'ok',
  source: 'strategy config',
  as_of_ms: Date.now(),
  age_ms: 3000,
  verified: true,
  ...over,
})

// Each case gets its OWN trader id, because the SWR key is derived from it and
// a shared key hands the next test the previous test's payload. (It did: four
// of these six passed against test one's data before the ids were split.)
let seq = 0
const renderStrip = async (payload: unknown) => {
  stripPayload = payload
  seq += 1
  const { DeskStrip } = await import('./DeskStrip')
  return render(<DeskStrip traderId={`t${seq}`} />)
}

beforeEach(() => {
  stripPayload = null
})

describe('DeskStrip', () => {
  it('renders every line with its age and never leaves one undated', async () => {
    await renderStrip({
      trader_id: 't1',
      generated_at_ms: Date.now(),
      cadence_ms: 15000,
      unknown_count: 0,
      stale_count: 0,
      lines: [
        line(),
        line({
          n: 2,
          key: 'position',
          label: 'POSITION',
          text: 'FLAT',
          state: 'flat',
        }),
      ],
    })
    await waitFor(() =>
      expect(screen.getByTestId('desk-age-mode')).toBeTruthy()
    )
    expect(screen.getByTestId('desk-age-mode').textContent).toContain('ago')
    expect(screen.getByTestId('desk-age-position').textContent).not.toBe('')
  })

  it('renders UNKNOWN with its reason, never a zero and never a dash', async () => {
    await renderStrip({
      trader_id: 't1',
      generated_at_ms: Date.now(),
      cadence_ms: 15000,
      unknown_count: 1,
      stale_count: 0,
      lines: [
        line({
          n: 3,
          key: 'protection',
          label: 'PROTECTION',
          text: 'UNKNOWN',
          state: 'unknown',
          reason: 'no accepted-risk record for this signal',
          as_of_ms: 0,
          age_ms: 0,
        }),
      ],
    })
    await waitFor(() =>
      expect(screen.getByTestId('desk-line-protection')).toBeTruthy()
    )
    const row = screen.getByTestId('desk-line-protection')
    expect(row.textContent).toContain('no accepted-risk record for this signal')
    // The rule is "no dash STANDING IN FOR A VALUE", not "no dash": the em-dash
    // here joins UNKNOWN to its reason and is punctuation. What must never
    // appear is a placeholder pretending to be a measurement.
    expect(row.textContent).not.toContain('0.00')
    for (const placeholder of ['UNKNOWN — —', 'UNKNOWN — -', 'UNKNOWN — n/a']) {
      expect(row.textContent).not.toContain(placeholder)
    }
    expect(row.textContent?.startsWith('PROTECTION')).toBe(true)
    // an undated row says so rather than implying "now"
    expect(screen.getByTestId('desk-age-protection').textContent).toBe(
      'undated'
    )
  })

  // C3 — THE LEDGER PRICE IS NEVER THE BROKER'S. The strip renders what the
  // server computed; this pins that the accepted stop is what reaches the
  // screen and the ledger's number is confined to the DRIFT row.
  it('shows the accepted stop in PROTECTION and the difference in DRIFT', async () => {
    await renderStrip({
      trader_id: 't1',
      generated_at_ms: Date.now(),
      cadence_ms: 5000,
      unknown_count: 0,
      stale_count: 0,
      lines: [
        line({
          n: 3,
          key: 'protection',
          label: 'PROTECTION',
          state: 'ok',
          text: 'accepted stop 29355.00 · 55.00 pts (220 ticks) away · 110.00 USD at risk if it fills',
        }),
        line({
          n: 4,
          key: 'drift',
          label: 'DRIFT',
          state: 'ok',
          text: 'ledger 29351.628473 vs accepted 29355.00 → +3.371527 pts (the ledger price is NOT your stop)',
        }),
      ],
    })
    await waitFor(() =>
      expect(screen.getByTestId('desk-line-protection')).toBeTruthy()
    )
    expect(screen.getByTestId('desk-line-protection').textContent).toContain(
      '29355'
    )
    expect(
      screen.getByTestId('desk-line-protection').textContent
    ).not.toContain('29351')
    expect(screen.getByTestId('desk-line-drift').textContent).toContain(
      '3.371527'
    )
  })

  it('marks a stale source amber rather than showing the last value as current', async () => {
    await renderStrip({
      trader_id: 't1',
      generated_at_ms: Date.now(),
      cadence_ms: 15000,
      unknown_count: 0,
      stale_count: 1,
      lines: [
        line({
          n: 9,
          key: 'feed',
          label: 'FEED',
          state: 'stale',
          reason: 'the newest 1m bar is older than 2 minutes',
          text: 'last bar 46h ago · link Disconnected · AddOn build 2026-09-03-f12',
          age_ms: 165_600_000,
        }),
      ],
    })
    await waitFor(() =>
      expect(screen.getByTestId('desk-line-feed')).toBeTruthy()
    )
    expect(
      screen.getByTestId('desk-line-feed').getAttribute('data-state')
    ).toBe('stale')
    expect(screen.getByTestId('desk-age-feed').textContent).toContain('h ago')
  })

  // A dead endpoint is STATED. A blank strip and a strip full of UNKNOWNs mean
  // different things and only the second one is honest.
  it('states an unreachable endpoint instead of blanking the card', async () => {
    await renderStrip(null)
    await waitFor(() =>
      expect(screen.getByTestId('desk-strip')).toHaveAttribute(
        'data-state',
        'unreachable'
      )
    )
    const el = screen.getByTestId('desk-strip')
    expect(el.getAttribute('data-state')).toBe('unreachable')
    expect(el.textContent).toContain('This is not a quiet desk')
  })

  it('shows the UNKNOWN count and the resolved cadence in its header', async () => {
    await renderStrip({
      trader_id: 't1',
      generated_at_ms: Date.now(),
      cadence_ms: 5000,
      unknown_count: 4,
      stale_count: 1,
      lines: [line()],
    })
    await waitFor(() =>
      expect(screen.getByTestId('desk-unknown-count')).toBeTruthy()
    )
    const hdr = screen.getByTestId('desk-unknown-count').textContent ?? ''
    expect(hdr).toContain('4 UNKNOWN')
    expect(hdr).toContain('1 stale')
    expect(hdr).toContain('5s')
  })
})

describe('Desk loading and accessible disclosure', () => {
  it('announces a pending read, then replaces loading with real facts', async () => {
    let resolve!: (data: unknown) => void
    await renderStrip(
      new Promise((done) => {
        resolve = done
      })
    )
    expect(screen.getByRole('status')).toHaveAttribute('data-state', 'loading')
    expect(screen.queryByTestId('desk-line-mode')).toBeNull()
    await act(async () =>
      resolve({
        trader_id: 'pending',
        generated_at_ms: Date.now(),
        cadence_ms: 5000,
        unknown_count: 0,
        stale_count: 0,
        lines: [line()],
      })
    )
    await screen.findByTestId('desk-line-mode')
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('announces expanded state and keeps the toggle reachable while collapsed', async () => {
    await renderStrip({
      trader_id: 'toggle',
      generated_at_ms: Date.now(),
      cadence_ms: 5000,
      unknown_count: 0,
      stale_count: 0,
      lines: [line()],
    })
    const toggle = await screen.findByRole('button', { name: 'Collapse Desk' })
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    fireEvent.click(toggle)
    expect(screen.getByRole('button', { name: 'Expand Desk' })).toHaveAttribute(
      'aria-expanded',
      'false'
    )
    expect(screen.queryByTestId('desk-line-mode')).toBeNull()
    fireEvent.click(toggle)
    expect(screen.getByTestId('desk-line-mode')).toBeInTheDocument()
  })
})

describe('Received book and missing link labels', () => {
  it('shows receipt age and received build, preserving known facts beside UNKNOWN link state', async () => {
    await renderStrip({
      generated_at_ms: Date.now(),
      cadence_ms: 15000,
      unknown_count: 1,
      stale_count: 0,
      lines: [
        line({
          key: 'book',
          label: 'BOOK',
          text: '0 working orders · received 21:00:00 CT · age 21s · AddOn build received-build',
          as_of_ms: Date.now() - 21000,
          age_ms: 21000,
        }),
        line({
          key: 'feed',
          label: 'FEED',
          state: 'unknown',
          reason: 'NT8 link state has not been received',
          text: 'last bar 3s ago · link UNKNOWN (no feed_status received) · AddOn build received-build',
        }),
      ],
    })
    await waitFor(() =>
      expect(screen.getByTestId('desk-line-book')).toBeInTheDocument()
    )
    expect(screen.getByTestId('desk-age-book')).toHaveTextContent('21s ago')
    expect(screen.getByTestId('desk-line-book')).toHaveTextContent(
      'received-build'
    )
    expect(screen.getByTestId('desk-line-feed')).toHaveTextContent(
      'last bar 3s ago · link UNKNOWN'
    )
    expect(screen.getByTestId('desk-line-feed')).toHaveAttribute(
      'data-state',
      'unknown'
    )
  })
})
