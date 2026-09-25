import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { AdvancedChart } from './AdvancedChart'
const state = vi.hoisted(() => ({
  line: vi.fn(),
  removeLine: vi.fn(),
  candles: vi.fn(),
  attach: vi.fn(),
  request: vi.fn(),
  remove: vi.fn(),
}))
vi.mock('../../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))
vi.mock('../../lib/httpClient', () => ({
  httpClient: { request: state.request },
}))
vi.mock('./primitives/SessionVolumeProfile', () => ({
  SessionVolumeProfile: class {
    setData = vi.fn()
  },
}))
vi.mock('lightweight-charts', () => ({
  CandlestickSeries: 'candles',
  HistogramSeries: 'volume',
  LineSeries: 'line',
  createSeriesMarkers: () => ({ setMarkers: vi.fn() }),
  createChart: () => ({
    addSeries: (kind: string) => ({
      setData: kind === 'candles' ? state.candles : vi.fn(),
      priceScale: () => ({ applyOptions: vi.fn() }),
      attachPrimitive: state.attach,
      detachPrimitive: vi.fn(),
      createPriceLine: state.line,
      removePriceLine: state.removeLine,
    }),
    applyOptions: vi.fn(),
    remove: state.remove,
    removeSeries: vi.fn(),
    subscribeCrosshairMove: vi.fn(),
    timeScale: () => ({ fitContent: vi.fn() }),
  }),
}))
const bars = (price: number) => ({
  success: true,
  data: [
    {
      openTime: 1700000000000,
      open: price,
      high: price + 1,
      low: price - 1,
      close: price,
      volume: 1,
    },
  ],
})
beforeEach(() => {
  state.line.mockReset().mockReturnValue({})
  state.removeLine.mockReset()
  state.candles.mockReset()
  state.attach.mockReset()
  state.request.mockReset()
  state.remove.mockReset()
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      disconnect() {}
    }
  )
  vi.spyOn(console, 'log').mockImplementation(() => {})
})
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

it('retains a snapshot on failure, clears on confirmed empty, and resets account scope', async () => {
  vi.useFakeTimers()
  let orders: unknown = {
    success: true,
    data: [
      {
        type: 'LIMIT',
        status: 'NEW',
        price: 200,
        stop_price: 190,
        quantity: 1,
        side: 'BUY',
      },
    ],
  }
  state.request.mockImplementation((url: string) =>
    Promise.resolve(url.includes('/open-orders?') ? orders : bars(200))
  )
  const { rerender } = render(
    <AdvancedChart symbol="MNQ" traderID="trader" selectedAccount="SimA" />
  )
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1000)
  })
  expect(state.line).toHaveBeenCalledOnce()
  orders = { success: false, error: 'offline' }
  await act(async () => {
    await vi.advanceTimersByTimeAsync(60000)
  })
  expect(state.removeLine).not.toHaveBeenCalled()
  expect(screen.getByRole('status').textContent).toContain(
    'UNKNOWN — refresh failed'
  )
  expect(screen.getByRole('status').textContent).toContain(
    'retained; lines may be stale'
  )
  orders = { success: true, data: { error: 'malformed response' } }
  await act(async () => {
    await vi.advanceTimersByTimeAsync(60000)
  })
  expect(state.removeLine).not.toHaveBeenCalled()
  expect(screen.getByRole('status').textContent).toContain(
    'UNKNOWN — refresh failed'
  )
  orders = { success: true, data: [] }
  await act(async () => {
    await vi.advanceTimersByTimeAsync(60000)
  })
  expect(state.removeLine).toHaveBeenCalledOnce()
  expect(screen.getByRole('status').textContent).toContain('0 orders')
  orders = { success: false }
  rerender(
    <AdvancedChart symbol="MNQ" traderID="trader" selectedAccount="SimB" />
  )
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1000)
  })
  expect(screen.getByRole('status').textContent).toContain(
    'UNKNOWN — refresh failed'
  )
  expect(screen.getByRole('status').textContent).not.toContain('retained')
  expect(screen.getByRole('status').textContent).toContain(
    'SimB is not applied'
  )
})

it('discards a pending order result after changing the selected account', async () => {
  vi.useFakeTimers()
  let finish!: (value: unknown) => void
  state.request.mockImplementation((url: string) =>
    url.includes('/open-orders?')
      ? new Promise((resolve) => {
          finish = resolve
        })
      : Promise.resolve(bars(200))
  )
  const { rerender } = render(
    <AdvancedChart symbol="MNQ" traderID="trader" selectedAccount="SimA" />
  )
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1000)
  })
  rerender(
    <AdvancedChart symbol="MNQ" traderID="trader" selectedAccount="SimB" />
  )
  await act(async () =>
    finish({
      success: true,
      data: [{ type: 'LIMIT', price: 200, stop_price: 190, quantity: 1 }],
    })
  )
  expect(state.line).not.toHaveBeenCalled()
  expect(screen.getByRole('status').textContent).toContain('awaiting snapshot')
})
