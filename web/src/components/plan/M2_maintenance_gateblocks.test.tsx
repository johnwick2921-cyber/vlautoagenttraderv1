// W-ONE-BUTTON M2 — the gate-block names the maintenance hold counts read as
// human labels, not raw gate names.

import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

vi.mock('../../lib/api/plan', () => ({
  planApi: {
    getGateBlocks: () =>
      Promise.resolve({
        session_day_utc: '2026-09-22T22:00:00Z',
        by_trader: {
          t1: {
            maintenance_hold: 3,
            maintenance_drop: 1,
            maintenance_drop_attempted: 1,
          },
        },
      }),
  },
}))

describe('GateBlocksPanel — maintenance hold labels', () => {
  it('labels the three maintenance gates', async () => {
    const { GateBlocksPanel } = await import('./GateBlocksPanel')
    render(<GateBlocksPanel traderId="t1" language="en" />)
    await waitFor(() =>
      expect(screen.getByTestId('gate-blocks-panel')).toBeTruthy()
    )
    expect(
      screen.getByTestId('gate-block-maintenance_hold').textContent
    ).toMatch(/update hold/i)
    expect(
      screen.getByTestId('gate-block-maintenance_drop').textContent
    ).toMatch(/dropped by the update hold/i)
    expect(
      screen.getByTestId('gate-block-maintenance_drop_attempted').textContent
    ).toMatch(/may have reached NT8/i)
  })
})
