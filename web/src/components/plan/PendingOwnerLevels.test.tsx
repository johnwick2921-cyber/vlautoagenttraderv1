// W-OWNER-LEVELS-UI (2026-09-17) — the "Pending owner levels" block. api +
// sonner mocked; SWR drives the fetch, so each test asserts the GET it makes,
// the rows it renders, and the delete → refetch round trip.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { SWRConfig } from 'swr'
import type { OwnerLevelsResponse } from '../../lib/api/plan'

const getOwnerLevels = vi.fn()
const deleteOwnerLevel = vi.fn().mockResolvedValue(true)

vi.mock('../../lib/api', () => ({
  api: {
    getOwnerLevels: (...a: unknown[]) => getOwnerLevels(...a),
    deleteOwnerLevel: (...a: unknown[]) => deleteOwnerLevel(...a),
  },
}))
vi.mock('sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { PendingOwnerLevels, fmtCreatedCT } from './PendingOwnerLevels'

const empty: OwnerLevelsResponse = {
  levels: [],
  count: 0,
  symbol: 'MNQ',
  as_of_ms: 1,
  judged_against: null,
}

const two: OwnerLevelsResponse = {
  levels: [
    {
      id: 7,
      symbol: 'MNQ',
      price: 30150.25,
      label: '👤',
      note: 'gap fill',
      scenario_tag: '',
      created_at: 1758133800, // 2026-09-17 13:30:00 CT
      consumed: false,
      status: 'applied',
    },
    {
      id: 9,
      symbol: 'MNQ',
      price: 30010,
      label: 'PDH',
      note: '',
      scenario_tag: '',
      created_at: 1758137400,
      consumed: false,
      status: 'pending',
    },
  ],
  count: 2,
  symbol: 'MNQ',
  as_of_ms: 1,
  judged_against: {
    plan_id: 'p1',
    version: 3,
    session: 'NY',
    trade_date: '2026-09-17',
  },
}

// Fresh SWR cache per render — otherwise the second test reads the first's data.
function mount(refreshTick = 0) {
  return render(
    <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
      <PendingOwnerLevels
        traderId="t1"
        symbol="MNQ"
        session="NY"
        language="en"
        refreshTick={refreshTick}
      />
    </SWRConfig>
  )
}

describe('PendingOwnerLevels', () => {
  beforeEach(() => {
    getOwnerLevels.mockReset()
    deleteOwnerLevel.mockClear()
  })

  it('renders the empty line when the list is []', async () => {
    getOwnerLevels.mockResolvedValue(empty)
    mount()
    await waitFor(() => expect(getOwnerLevels).toHaveBeenCalled())
    expect(getOwnerLevels.mock.calls[0]).toEqual(['t1', 'MNQ', 'NY'])
    expect(
      await screen.findByText('No pending owner levels')
    ).toBeInTheDocument()
    expect(screen.queryAllByTestId('pending-owner-level-row')).toHaveLength(0)
  })

  it('renders two rows with the right chips, prices, notes and CT times', async () => {
    getOwnerLevels.mockResolvedValue(two)
    mount()
    const rows = await screen.findAllByTestId('pending-owner-level-row')
    expect(rows).toHaveLength(2)
    const chips = screen.getAllByTestId('pending-owner-level-status')
    expect(chips.map((c) => c.textContent)).toEqual(['applied', 'pending'])
    expect(rows[0].getAttribute('data-status')).toBe('applied')
    expect(rows[1].getAttribute('data-status')).toBe('pending')
    expect(screen.getByText('30150.25')).toBeInTheDocument()
    expect(screen.getByText('30010')).toBeInTheDocument()
    expect(screen.getByText('gap fill')).toBeInTheDocument()
    expect(
      screen.getByText(`${fmtCreatedCT(1758133800)} CT`)
    ).toBeInTheDocument()
    expect(screen.queryByText('No pending owner levels')).toBeNull()
  })

  it('delete posts the row id and refetches the list', async () => {
    getOwnerLevels.mockResolvedValueOnce(two).mockResolvedValue(empty)
    mount()
    await screen.findAllByTestId('pending-owner-level-row')
    fireEvent.click(screen.getByLabelText('Delete 30010'))
    await waitFor(() => expect(deleteOwnerLevel).toHaveBeenCalledWith('t1', 9))
    await waitFor(() => expect(getOwnerLevels).toHaveBeenCalledTimes(2))
    expect(
      await screen.findByText('No pending owner levels')
    ).toBeInTheDocument()
  })

  it('a refreshTick bump refetches without a delete', async () => {
    getOwnerLevels.mockResolvedValue(empty)
    const { rerender } = mount()
    await waitFor(() => expect(getOwnerLevels).toHaveBeenCalledTimes(1))
    rerender(
      <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
        <PendingOwnerLevels
          traderId="t1"
          symbol="MNQ"
          session="NY"
          language="en"
          refreshTick={1}
        />
      </SWRConfig>
    )
    await waitFor(() => expect(getOwnerLevels).toHaveBeenCalledTimes(2))
  })
})

describe('fmtCreatedCT', () => {
  it('formats unix seconds as HH:MM in America/Chicago', () => {
    // 2026-09-17T18:30:00Z = 13:30 CDT
    expect(fmtCreatedCT(1758133800)).toBe('13:30')
    expect(fmtCreatedCT(0)).toBe('—')
  })
})
