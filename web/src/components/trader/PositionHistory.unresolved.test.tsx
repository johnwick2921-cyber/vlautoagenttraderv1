import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { PositionHistory } from './PositionHistory'
import type { HistoricalPosition } from '../../types'

// Trading-P2 RED→GREEN (audit/0926-trading-pipeline, the 618 class): a NULL
// pnl_corrected must READ "unresolved" in the row and be COUNTED in the
// footer — never 0, never a fabricated realized_pnl number.

vi.mock('../../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))
vi.mock('../../lib/autoRefresh', () => ({
  useAutoRefresh: () => {},
  REFRESH_HISTORY_MS: 60000,
}))
vi.mock('../../lib/api', () => ({
  api: { getPositionHistory: vi.fn() },
}))
vi.mock('../common/MetricTooltip', () => ({
  MetricTooltip: () => null,
}))

import { api } from '../../lib/api'

const mockGet = api.getPositionHistory as ReturnType<typeof vi.fn>

function row(over: Partial<HistoricalPosition>): HistoricalPosition {
  const base: HistoricalPosition = {
    id: 618,
    trader_id: 't1',
    exchange_id: 'e',
    exchange_type: 'ninjatrader',
    symbol: 'MNQ',
    side: 'LONG',
    quantity: 1,
    entry_quantity: 1,
    entry_price: 30736.5,
    entry_order_id: '',
    entry_time: '2026-09-24T14:00:00Z',
    exit_price: 0,
    exit_order_id: '',
    exit_time: new Date().toISOString(),
    realized_pnl: 777, // distinctive: must NOT appear once pnl_corrected is NULL
    fee: 0,
    leverage: 1,
    status: 'CLOSED',
    close_reason: 'sync',
    created_at: '',
    updated_at: '',
  }
  return { ...base, ...over }
}

function mockResponse(positions: HistoricalPosition[]) {
  mockGet.mockResolvedValue({
    positions,
    stats: null,
    symbol_stats: [],
    direction_stats: [],
  })
}

beforeEach(() => mockGet.mockReset())
afterEach(() => cleanup())

describe('PositionHistory — NULL pnl_corrected reads unresolved', () => {
  it('renders "unresolved" (never the realized number) and counts it in the footer', async () => {
    const p = row({ pnl_corrected: undefined })
    delete (p as Partial<HistoricalPosition>).pnl_corrected // wire-shape: field ABSENT
    mockResponse([p])

    render(<PositionHistory traderId="t1" />)

    // The row's P&L cell says unresolved…
    expect(await screen.findByText('unresolved')).toBeTruthy()
    // …and the fabricated realized_pnl number never appears in the cell.
    expect(screen.queryByText('+777.00')).toBeNull()
    expect(screen.queryByText('777.00')).toBeNull()
    // The footer shows the count (canon class 40: count shown).
    expect(screen.getByText('1 unresolved')).toBeTruthy()
  })

  it('renders the corrected number for a RESOLVED row and no unresolved count', async () => {
    mockResponse([row({ id: 619, pnl_corrected: -32, realized_pnl: -32 })])

    render(<PositionHistory traderId="t1" />)

    expect((await screen.findAllByText('-32.00')).length).toBeGreaterThan(0)
    expect(screen.queryByText('unresolved')).toBeNull()
    expect(screen.queryByText(/unresolved$/)).toBeNull()
  })
})
