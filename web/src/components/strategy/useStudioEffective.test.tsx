// W1 (g) — the Studio's effective-values plumbing at its production call
// site: the real hook → the real client → a mocked transport answering with
// the server's wire shape (the mount fixture).

import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { SWRConfig } from 'swr'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import fixture from './__fixtures__/strategyEffective.fixture.json'
import { useStudioEffective } from './useStudioEffective'
import {
  studioEffective,
  type StrategyEffectiveResponse,
} from '../../lib/api/strategyEffective'

const mocks = vi.hoisted(() => ({ request: vi.fn() }))
vi.mock('../../lib/httpClient', () => ({
  httpClient: { request: mocks.request },
}))

type Scope = 'strategy' | 'NY' | 'ASIA' | 'LONDON'
const R = fixture.responses as unknown as Record<
  Scope,
  StrategyEffectiveResponse
>

const BASE = '/api/strategies/fx-1/effective'

function wrapper({ children }: { children: ReactNode }) {
  return (
    <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
      {children}
    </SWRConfig>
  )
}

function answer(url: string) {
  const m = /\?session=([A-Z]+)$/.exec(url)
  const scope = (m ? m[1] : 'strategy') as Scope
  return Promise.resolve({ success: true, data: R[scope] })
}

describe('useStudioEffective', () => {
  beforeEach(() => {
    mocks.request.mockReset()
    mocks.request.mockImplementation(answer)
  })

  it('reads four times — no session, NY, ASIA, LONDON — and builds the lookup', async () => {
    const { result } = renderHook(() => useStudioEffective('fx-1'), {
      wrapper,
    })
    await waitFor(() =>
      expect(Object.keys(result.current.effective.bySession).sort()).toEqual([
        'ASIA',
        'LONDON',
        'NY',
      ])
    )
    const urls = mocks.request.mock.calls.map((c) => c[0]).sort()
    expect(urls).toEqual(
      [
        BASE,
        `${BASE}?session=ASIA`,
        `${BASE}?session=LONDON`,
        `${BASE}?session=NY`,
      ].sort()
    )
    const eff = result.current.effective
    // strategy row from the session-less read, session row from ITS read
    expect(eff.byPath['day_plan.plan_mode']?.effective).toBe('advisory')
    expect(eff.bySession.NY?.['day_plan.sessions.plan_mode']?.effective).toBe(
      'strict'
    )
    expect(
      eff.bySession.ASIA?.['day_plan.sessions.replan_cap']?.effective
    ).toBe(1)
  })

  it('refresh() re-reads all four (the page calls it after a successful save)', async () => {
    const { result } = renderHook(() => useStudioEffective('fx-1'), {
      wrapper,
    })
    await waitFor(() => expect(mocks.request).toHaveBeenCalledTimes(4))
    act(() => result.current.refresh())
    await waitFor(() => expect(mocks.request).toHaveBeenCalledTimes(8))
    const urls = mocks.request.mock.calls
      .slice(4)
      .map((c) => c[0])
      .sort()
    expect(urls).toEqual(
      [
        BASE,
        `${BASE}?session=ASIA`,
        `${BASE}?session=LONDON`,
        `${BASE}?session=NY`,
      ].sort()
    )
  })

  it('no strategy → no read and an empty lookup', () => {
    const { result } = renderHook(() => useStudioEffective(null), { wrapper })
    expect(mocks.request).not.toHaveBeenCalled()
    expect(result.current.effective).toEqual({ byPath: {}, bySession: {} })
  })

  it('a failed read leaves that scope empty — no borrowed rows', async () => {
    mocks.request.mockImplementation((url: string) =>
      url.endsWith('?session=ASIA')
        ? Promise.resolve({ success: false, statusCode: 500 })
        : answer(url)
    )
    const { result } = renderHook(() => useStudioEffective('fx-1'), {
      wrapper,
    })
    await waitFor(() =>
      expect(Object.keys(result.current.effective.bySession).sort()).toEqual([
        'LONDON',
        'NY',
      ])
    )
    expect(result.current.effective.bySession.ASIA).toBeUndefined()
  })
})

describe('studioEffective', () => {
  it('a reply for the wrong scope fills nothing', () => {
    const out = studioEffective(R.NY, { ASIA: R.NY, NY: R.NY })
    expect(out.byPath).toEqual({}) // a session reply never answers strategy rows
    expect(out.bySession.ASIA).toBeUndefined() // NY's reply never fills ASIA
    expect(out.bySession.NY?.['day_plan.sessions.max_trades']?.effective).toBe(
      3
    )
  })
})
