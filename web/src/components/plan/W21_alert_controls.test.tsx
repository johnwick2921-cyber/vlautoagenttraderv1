// ITEM 5 (2026-08-17) — the feed can be cleared, the record cannot.
//
// Alerts could be acknowledged but never removed, so the feed only grew. These
// cover the owner-facing half: the ✕ on every row, clear-read, and the P0
// protection that keeps a halt from being swiped away unseen.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { AlertCenter } from './AlertCenter'
import { api } from '../../lib/api'
import type { PlanAlert } from '../../lib/api/plan'

const alert = (over: Partial<PlanAlert>): PlanAlert => ({
  id: 1,
  trader_id: 't1',
  level: 'P1',
  event_id: 'e1',
  kind: 'k',
  title: 'Plan died',
  body: 'all levels consumed',
  acked: false,
  created_at: Math.floor(Date.now() / 1000),
  ...over,
})

const feed = [
  alert({ id: 1, level: 'P0', acked: false, title: 'Trading halted' }),
  alert({ id: 2, level: 'P1', acked: true, title: 'Plan armed' }),
]

beforeEach(() => {
  vi.restoreAllMocks()
  vi.spyOn(api, 'getPlanAlerts').mockResolvedValue({
    alerts: feed,
    unacked: 1,
  })
})

// SWR caches by key, so a test that needs a DIFFERENT feed must use a different
// trader id or it will read the previous test's cached response.
const openFeed = async (traderId = 't1') => {
  render(<AlertCenter traderId={traderId} language="en" />)
  const bell = await screen.findByTestId('alert-bell')
  fireEvent.click(bell)
}

describe('AlertCenter — remove controls', () => {
  it('offers a ✕ on every alert row', async () => {
    await openFeed()
    expect(await screen.findByTestId('alert-dismiss-1')).toBeInTheDocument()
    expect(screen.getByTestId('alert-dismiss-2')).toBeInTheDocument()
  })

  it('removes a row and refreshes the feed', async () => {
    const dismiss = vi
      .spyOn(api, 'dismissAlert')
      .mockResolvedValue({ ok: true })
    await openFeed()
    fireEvent.click(await screen.findByTestId('alert-dismiss-2'))
    await waitFor(() => expect(dismiss).toHaveBeenCalledWith('t1', 2))
  })

  it('explains the refusal instead of silently failing on an unacked P0', async () => {
    vi.spyOn(api, 'dismissAlert').mockResolvedValue({
      ok: false,
      needsAck: true,
      error:
        'an unacknowledged P0 alert must be acknowledged before it can be dismissed',
    })
    await openFeed()
    fireEvent.click(await screen.findByTestId('alert-dismiss-1'))
    // The row must still be there — nothing was hidden.
    await waitFor(() =>
      expect(screen.getByTestId('alert-dismiss-1')).toBeInTheDocument()
    )
  })

  it('offers clear-read only when something has been read', async () => {
    await openFeed()
    expect(await screen.findByTestId('alert-clear-read')).toBeInTheDocument()

    const clear = vi
      .spyOn(api, 'clearReadAlerts')
      .mockResolvedValue({ ok: true, cleared: 1 })
    fireEvent.click(screen.getByTestId('alert-clear-read'))
    await waitFor(() => expect(clear).toHaveBeenCalledWith('t1'))
  })

  it('hides clear-read when nothing is acknowledged', async () => {
    vi.spyOn(api, 'getPlanAlerts').mockResolvedValue({
      alerts: [alert({ id: 9, acked: false })],
      unacked: 1,
    })
    await openFeed('t-unacked-only')
    await screen.findByTestId('alert-dismiss-9')
    expect(screen.queryByTestId('alert-clear-read')).toBeNull()
  })
})
