// Pin: strict resolved → the mock says strict. And when nothing resolves, the
// mock says it is an example rather than naming a mode it did not read.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MockPlanCard } from './MockPlanCard'
import { MODE_NOT_READ } from './useResolvedPlanMode'

function stub(traders: unknown, cfg: unknown) {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockImplementation((url: string) => {
      if (String(url).includes('/api/traders')) {
        return Promise.resolve({ json: () => Promise.resolve(traders) })
      }
      return Promise.resolve({ json: () => Promise.resolve(cfg) })
    })
  )
}

describe('MockPlanCard scenarios header', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('says strict when strict is what resolves', async () => {
    stub([{ id: 't1' }], {
      resolved: [{ path: 'day_plan.plan_mode', resolved: 'strict' }],
    })
    render(<MockPlanCard />)
    await waitFor(() =>
      expect(screen.getByTestId('mock-scenarios-header').textContent).toContain(
        'strict'
      )
    )
    expect(
      screen.getByTestId('mock-scenarios-header').textContent
    ).not.toContain('advisory')
  })

  it('never names a mode it could not read', async () => {
    stub([], {})
    render(<MockPlanCard />)
    await waitFor(() =>
      expect(screen.getByTestId('mock-scenarios-header').textContent).toContain(
        MODE_NOT_READ
      )
    )
  })

  it('does not assert advisory when the fetch fails outright', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('down')))
    render(<MockPlanCard />)
    await waitFor(() =>
      expect(screen.getByTestId('mock-scenarios-header').textContent).toContain(
        MODE_NOT_READ
      )
    )
  })
})
