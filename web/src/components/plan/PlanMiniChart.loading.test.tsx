import { act, cleanup, render } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { PlanMiniChart } from './PlanMiniChart'

const mocks = vi.hoisted(() => ({
  request: vi.fn(),
  setData: vi.fn(),
  remove: vi.fn(),
}))

vi.mock('../../lib/httpClient', () => ({
  httpClient: { request: mocks.request },
}))
vi.mock('lightweight-charts', () => ({
  CandlestickSeries: {},
  createChart: () => ({
    addSeries: () => ({ setData: mocks.setData, attachPrimitive: vi.fn() }),
    applyOptions: vi.fn(),
    subscribeCrosshairMove: vi.fn(),
    remove: mocks.remove,
  }),
}))

const bar = {
  openTime: 1790089200000,
  open: 30900,
  high: 30925,
  low: 30895,
  close: 30920,
}
const success = { success: true, data: [bar] }
const props = {
  symbol: 'MNQ',
  exchange: 'ninjatrader',
  facts: [],
  language: 'en' as const,
}

describe('PlanMiniChart response lifecycle', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    mocks.request.mockReset()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
      {} as CanvasRenderingContext2D
    )
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe() {}
        disconnect() {}
      }
    )
  })

  afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('loads successful candles while mounted and refreshes on the polling interval', async () => {
    mocks.request.mockResolvedValue(success)
    render(<PlanMiniChart {...props} />)
    await act(async () => {
      await Promise.resolve()
    })
    expect(mocks.setData).toHaveBeenCalledWith([
      {
        time: bar.openTime / 1000,
        open: bar.open,
        high: bar.high,
        low: bar.low,
        close: bar.close,
      },
    ])
    await act(async () => {
      await vi.advanceTimersByTimeAsync(15000)
    })
    expect(mocks.setData).toHaveBeenCalledTimes(2)
  })

  it('ignores a previous interval response after a new interval has loaded', async () => {
    let finishOld!: (value: typeof success) => void
    mocks.request
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finishOld = resolve
          })
      )
      .mockResolvedValueOnce(success)
    const view = render(<PlanMiniChart {...props} interval="5m" />)
    view.rerender(<PlanMiniChart {...props} interval="1h" />)
    await act(async () => {
      await Promise.resolve()
    })
    expect(mocks.setData).toHaveBeenCalledTimes(1)
    await act(async () => {
      finishOld({ success: true, data: [{ ...bar, close: 30899 }] })
    })
    expect(mocks.setData).toHaveBeenCalledTimes(1)
  })

  it('ignores a response after unmount and stops polling', async () => {
    let finish!: (value: typeof success) => void
    mocks.request.mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve
        })
    )
    const view = render(<PlanMiniChart {...props} />)
    view.unmount()
    await act(async () => {
      finish(success)
      await vi.advanceTimersByTimeAsync(30000)
    })
    expect(mocks.setData).not.toHaveBeenCalled()
    expect(mocks.request).toHaveBeenCalledTimes(1)
    expect(mocks.remove).toHaveBeenCalledTimes(1)
  })
})
