import { render, screen, cleanup } from '@testing-library/react'
import { describe, it, expect, vi, afterEach } from 'vitest'

// Drive the three SWR keys EquityChart reads with a per-test table; the
// component never touches the network in these tests.
const swrData: Record<string, unknown> = {}
vi.mock('swr', () => ({
  default: (key: string | null) => {
    if (!key) return { data: undefined, error: undefined, isLoading: false }
    const hit = Object.keys(swrData).find((k) => key.startsWith(k))
    return {
      data: hit ? swrData[hit] : undefined,
      error: undefined,
      isLoading: false,
    }
  },
}))
vi.mock('../../lib/api', () => ({ api: {} }))
vi.mock('../../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 'u' }, token: 't' }),
}))
vi.mock('recharts', () => {
  const Stub = ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  )
  return {
    ResponsiveContainer: Stub,
    LineChart: Stub,
    Line: () => null,
    XAxis: () => null,
    YAxis: () => null,
    CartesianGrid: () => null,
    Tooltip: () => null,
    ReferenceLine: () => null,
  }
})

import { EquityChart } from './EquityChart'

afterEach(() => {
  cleanup()
  for (const k of Object.keys(swrData)) delete swrData[k]
})

describe('EquityChart total_equity guards', () => {
  it('renders the empty state when history is undefined', () => {
    render(<EquityChart traderId="t1" />)
    expect(screen.getByText('No Historical Data')).toBeInTheDocument()
  })

  it('renders the empty state (no throw) when history points lack total_equity', () => {
    swrData['equity-history-'] = [
      {
        timestamp: '2026-09-17T10:00:00Z',
        pnl: 0,
        pnl_pct: 0,
        cycle_number: 1,
      },
      { timestamp: '2026-09-17T10:05:00Z', total_equity: null, pnl: 0 },
    ]
    swrData['account-'] = { initial_balance: 1000 }
    render(<EquityChart traderId="t1" />)
    expect(screen.getByText('No Historical Data')).toBeInTheDocument()
  })

  it('renders the empty state when the history body is not an array', () => {
    swrData['equity-history-'] = { error: 'nope' }
    render(<EquityChart traderId="t1" />)
    expect(screen.getByText('No Historical Data')).toBeInTheDocument()
  })

  it('renders the chart with 0.00 headline when account.total_equity is missing', () => {
    swrData['equity-history-'] = [
      {
        timestamp: '2026-09-17T10:00:00Z',
        total_equity: 1000,
        pnl: 0,
        pnl_pct: 0,
        cycle_number: 1,
      },
      {
        timestamp: '2026-09-17T10:05:00Z',
        total_equity: 1010,
        pnl: 10,
        pnl_pct: 1,
        cycle_number: 2,
      },
    ]
    swrData['account-'] = { initial_balance: 1000 } // no total_equity
    render(<EquityChart traderId="t1" />)
    expect(screen.queryByText('No Historical Data')).toBeNull()
    expect(screen.getByText('0.00')).toBeInTheDocument()
  })
})
