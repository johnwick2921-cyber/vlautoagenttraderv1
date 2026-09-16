// P4.4 — AlertCenter tests. usePlanAlerts + sonner + api are mocked so the test
// drives the feed/badge/banner/ack purely from props-shaped data.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import type { PlanAlert } from '../../lib/api/plan'

const ackMock = vi.fn().mockResolvedValue(true)
const mutateMock = vi.fn().mockResolvedValue(undefined)
const toastError = vi.fn()

let mockAlerts: PlanAlert[] = []
let mockUnacked = 0

vi.mock('./usePlan', () => ({
  usePlanAlerts: () => ({
    alerts: mockAlerts,
    unacked: mockUnacked,
    mutate: mutateMock,
  }),
}))
vi.mock('../../lib/api', () => ({
  api: { ackPlanAlert: (...a: unknown[]) => ackMock(...a) },
}))
vi.mock('sonner', () => ({
  toast: { error: (...a: unknown[]) => toastError(...a) },
}))

import { AlertCenter } from './AlertCenter'

const alert = (over: Partial<PlanAlert>): PlanAlert => ({
  id: 1,
  trader_id: 't1',
  level: 'P1',
  event_id: '',
  kind: 'armed',
  title: 'S2 armed',
  body: '',
  acked: false,
  created_at: Math.floor(Date.now() / 1000),
  ...over,
})

describe('AlertCenter', () => {
  beforeEach(() => {
    mockAlerts = []
    mockUnacked = 0
    ackMock.mockClear()
    mutateMock.mockClear()
    toastError.mockClear()
  })

  it('renders nothing without a trader', () => {
    const { container } = render(<AlertCenter language="en" />)
    expect(container.firstChild).toBeNull()
  })

  it('shows the unacked badge count', () => {
    mockAlerts = [
      alert({ id: 1 }),
      alert({ id: 2, level: 'P0', title: 'Halt' }),
    ]
    mockUnacked = 2
    render(<AlertCenter traderId="t1" language="en" />)
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('renders a persistent P0 banner until ack', async () => {
    mockAlerts = [alert({ id: 5, level: 'P0', title: 'Daily halt tripped' })]
    mockUnacked = 1
    render(<AlertCenter traderId="t1" language="en" />)
    const banner = screen.getByRole('alert')
    expect(banner).toHaveTextContent(/Daily halt tripped/i)
    // ack calls the API + revalidates
    fireEvent.click(banner.querySelector('button')!)
    await waitFor(() => expect(ackMock).toHaveBeenCalledWith('t1', 5))
    expect(mutateMock).toHaveBeenCalled()
  })

  it('opens the feed and lists non-P2 alerts', () => {
    mockAlerts = [
      alert({ id: 1, title: 'S1 armed' }),
      alert({ id: 2, level: 'P2', title: 'digest thing' }),
    ]
    mockUnacked = 1
    render(<AlertCenter traderId="t1" language="en" />)
    fireEvent.click(screen.getByRole('button', { name: /Alerts/i }))
    expect(screen.getByText('S1 armed')).toBeInTheDocument()
    // P2 is collapsed to a digest count, not shown as its own row
    expect(screen.queryByText('digest thing')).not.toBeInTheDocument()
  })

  it('does not toast the initial backlog', () => {
    mockAlerts = [alert({ id: 9, level: 'P0', title: 'old halt' })]
    mockUnacked = 1
    render(<AlertCenter traderId="t1" language="en" />)
    expect(toastError).not.toHaveBeenCalled()
  })
})
