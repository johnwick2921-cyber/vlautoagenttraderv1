// W16/R3 — the gate-block tally has counted every refusal since B6 and had ZERO
// frontend consumers. These pin that it renders, aggregates the process-wide
// bucket, and is HONEST when empty (an empty table must not read as "nothing was
// ever blocked" when the counter simply reset).

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

let payload: unknown = null
vi.mock('../../lib/api/plan', () => ({
  planApi: { getGateBlocks: () => Promise.resolve(payload) },
}))

const renderPanel = async (traderId = 't1') => {
  const { GateBlocksPanel } = await import('./GateBlocksPanel')
  return render(<GateBlocksPanel traderId={traderId} language="en" />)
}

describe('GateBlocksPanel', () => {
  beforeEach(() => {
    payload = null
  })

  it('lists refusals with their counts, most frequent first', async () => {
    payload = {
      session_day_utc: '2026-08-17T22:00:00Z',
      by_trader: { t1: { session_gate: 5, last_entry: 2 } },
    }
    await renderPanel()
    await waitFor(() =>
      expect(screen.getByTestId('gate-blocks-panel')).toBeTruthy()
    )
    expect(screen.getByTestId('gate-block-session_gate').textContent).toMatch(
      /5/
    )
    expect(screen.getByTestId('gate-block-last_entry').textContent).toMatch(/2/)
    // human label, not the raw gate name
    expect(screen.getByTestId('gate-block-session_gate').textContent).toMatch(
      /session window/i
    )
  })

  it('folds in the process-wide bucket (B3 files under the empty trader id)', async () => {
    payload = {
      by_trader: { t1: { session_gate: 1 }, '': { b3_rate_breaker: 3 } },
    }
    await renderPanel()
    await waitFor(() =>
      expect(screen.getByTestId('gate-blocks-panel')).toBeTruthy()
    )
    expect(
      screen.getByTestId('gate-block-b3_rate_breaker').textContent
    ).toMatch(/3/)
  })

  it('says nothing was refused rather than rendering a bare empty box', async () => {
    payload = { by_trader: {} }
    await renderPanel()
    await waitFor(() =>
      expect(screen.getByTestId('gate-blocks-empty')).toBeTruthy()
    )
    expect(screen.getByTestId('gate-blocks-empty').textContent).toMatch(
      /no entries refused/i
    )
  })

  it('always states that the counter resets — an empty table must not imply "never blocked"', async () => {
    payload = { by_trader: { t1: { frozen: 1 } } }
    await renderPanel()
    await waitFor(() =>
      expect(screen.getByTestId('gate-blocks-panel')).toBeTruthy()
    )
    expect(screen.getByTestId('gate-blocks-panel').textContent).toMatch(
      /reset/i
    )
  })

  it('renders nothing without a trader', async () => {
    payload = { by_trader: { t1: { frozen: 1 } } }
    const { container } = await renderPanel('')
    expect(
      container.querySelector('[data-testid="gate-blocks-panel"]')
    ).toBeNull()
  })
})
