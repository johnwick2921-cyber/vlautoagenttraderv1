// W-PICTURE-HTF (2026-09-20) — panel pins: rows render intended vs broker
// answer side by side; a missing broker answer renders as a dash, never a
// fabricated success; the empty ledger renders the dormant state.

import { describe, expect, it, vi, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { PictureHtfPanel } from './PictureHtfPanel'

afterEach(() => {
  vi.restoreAllMocks()
})

const ROW = {
  opp_key: 't1|||long|resistance|1000|2000',
  stage: 'place_pending',
  stage_reason: '',
  symbol: 'MNQ',
  direction: 'long',
  level_role: 'resistance',
  level_body_top: 101,
  h1_close_time_ms: 2_000,
  window_close_ms: 3_000,
  entry_ref: 101.49,
  stop_px: 98.25,
  target_px: 110,
  rr_estimate: 2.63,
  rr_configured: 2.5,
  momentum_stall: false,
  signal_id: 'nt8-uuid-12345',
  submitted_at_ms: Date.now() - 65_000,
  broker_order_id: '',
  broker_status: '',
  fill_price: 0,
  fill_qty: 0,
  fill_rr: 0,
  reject_reason: '',
  created_at_ms: Date.now(),
}

describe('PictureHtfPanel', () => {
  it('renders intended geometry and the broker answer side by side', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        json: async () => ({ rows: [ROW], count: 1 }),
      })
    )
    render(<PictureHtfPanel traderId="trader-1" />)
    await waitFor(() => expect(screen.getByText('LONG')).toBeTruthy())
    expect(screen.getByText(/resistance 101\.00/)).toBeTruthy()
    expect(screen.getByText(/R:R 2\.63 \(min 2\.50\)/)).toBeTruthy()
    expect(screen.getByText(/sent 1m ago/)).toBeTruthy()
    expect(screen.getByText('place_pending')).toBeTruthy()
    expect(screen.getByText(/no fill yet/)).toBeTruthy()
  })

  it('shows the dormant state when nothing is recorded', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        json: async () => ({ rows: [], count: 0 }),
      })
    )
    render(<PictureHtfPanel traderId="trader-1" />)
    await waitFor(() =>
      expect(
        screen.getByText(/No two-picture opportunities recorded/)
      ).toBeTruthy()
    )
  })
})
