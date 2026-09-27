import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({ request: vi.fn() }))
vi.mock('../httpClient', () => ({
  httpClient: { request: mocks.request },
}))

import { effectiveByPath, strategyEffectiveApi } from './strategyEffective'

describe('strategyEffectiveApi.getStrategyEffective', () => {
  beforeEach(() => {
    // A block body: returning the mock (a function) would register it as a
    // teardown that vitest CALLS after each test.
    mocks.request.mockReset()
  })

  it('asks the effective route for the strategy and the named session, silently', async () => {
    mocks.request.mockResolvedValue({
      success: true,
      data: {
        strategy_id: 's/1',
        session: 'NY',
        venue: null,
        settings: [],
        coverage: {},
      },
    })
    const out = await strategyEffectiveApi.getStrategyEffective('s/1', 'NY')
    expect(mocks.request).toHaveBeenCalledWith(
      '/api/strategies/s%2F1/effective?session=NY',
      {
        silent: true,
      }
    )
    expect(out?.session).toBe('NY')
  })

  it('resolves null on a refusal — never a fabricated payload', async () => {
    mocks.request.mockResolvedValue({ success: false, statusCode: 404 })
    expect(await strategyEffectiveApi.getStrategyEffective('x')).toBeNull()
  })

  it('resolves null when the request throws', async () => {
    mocks.request.mockImplementation(() => Promise.reject(new Error('network')))
    expect(await strategyEffectiveApi.getStrategyEffective('x')).toBeNull()
  })
})

describe('effectiveByPath', () => {
  it('indexes rows by schema path; no payload is an empty map', () => {
    expect(effectiveByPath(null)).toEqual({})
    const row = {
      path: 'day_plan.plan_mode',
      status: 'live',
      ui_label: 'live',
      stored: { present: false },
      effective: 'advisory',
      origin: 'shipped default',
      scope: 'strategy',
      resolver: 'store.ResolvePlanMode',
      resolved: true,
    }
    const m = effectiveByPath({
      strategy_id: 's',
      session: null,
      venue: null,
      settings: [row],
      coverage: { resolved: 1, total: 1, unresolved: [], not_enumerated: [] },
    })
    expect(m['day_plan.plan_mode']).toBe(row)
  })
})
