// W15.B — the session tabs were DISPLAY-ONLY: clicking ASIA moved the highlight
// while the card kept rendering the LIVE session's plan, because the fetch had no
// session dimension and enablement came from a hardcoded frontend constant.
// These lock both halves of the fix.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { computeSessionTabs } from './PlanCard'
import { planApi } from '../../lib/api/plan'
import { httpClient } from '../../lib/httpClient'

// Assert the URL the client BUILDS; the transport itself is not under test.
const requestSpy = vi.spyOn(httpClient, 'request')

describe('computeSessionTabs · enablement comes from the server', () => {
  it('a session the SERVER reports runnable is selectable even though the hardcoded constant says off', () => {
    const tabs = computeSessionTabs('NY', ['NY', 'ASIA'])
    const asia = tabs.find((t) => t.name === 'ASIA')
    expect(asia?.state).toBe('inactive') // selectable, just not the live one
    expect(tabs.find((t) => t.name === 'NY')?.state).toBe('active')
    // absent from the server list → still disabled
    expect(tabs.find((t) => t.name === 'LONDON')?.state).toBe('disabled')
  })

  it('a session the owner switched OFF goes disabled even though the constant says on', () => {
    const tabs = computeSessionTabs(null, ['ASIA'])
    expect(tabs.find((t) => t.name === 'NY')?.state).toBe('disabled')
    expect(tabs.find((t) => t.name === 'ASIA')?.state).toBe('inactive')
  })

  it('no server list (older payload / still loading) falls back to the constant — pre-W15 rendering', () => {
    const tabs = computeSessionTabs('NY', undefined)
    expect(tabs.find((t) => t.name === 'NY')?.state).toBe('active')
    expect(tabs.find((t) => t.name === 'ASIA')?.state).toBe('disabled')
  })

  it('the live session reads active regardless of the list', () => {
    const tabs = computeSessionTabs('ASIA', ['NY', 'ASIA'])
    expect(tabs.find((t) => t.name === 'ASIA')?.state).toBe('active')
  })
})

describe('getPlanToday · the selected session reaches the request', () => {
  const urls: string[] = []
  beforeEach(() => {
    urls.length = 0
    requestSpy.mockImplementation((url: string) => {
      urls.push(url)
      return Promise.resolve({ success: true, data: { found: false } })
    })
  })

  it('omits the param when no session is chosen (unchanged behavior)', async () => {
    await planApi.getPlanToday('t1', 'MNQ')
    expect(urls[0]).toContain('trader_id=t1')
    expect(urls[0]).not.toContain('session=')
  })

  it('sends session=ASIA when the ASIA tab is selected', async () => {
    await planApi.getPlanToday('t1', 'MNQ', true, 'ASIA')
    expect(urls[0]).toContain('session=ASIA')
  })
})
