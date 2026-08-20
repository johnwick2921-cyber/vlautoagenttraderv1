// reset-dialog-stuck hotfix (2026-08-19) — regression tests. The bug: the 30s
// axios default vs the reset's SYNCHRONOUS 60-300s planner read threw past an
// unguarded await → busy latched, confirm box froze (F5-only), and the
// unguarded Yes button double-fired resets (owner log: same-pair duplicates
// at 01:27/01:30 CT).
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ResetButton } from './ResetButton'

const forceReset = vi.fn()
const getResetGate = vi.fn()

vi.mock('../../lib/api', () => ({
  api: {
    forceReset: (...a: unknown[]) => forceReset(...a),
    getResetGate: (...a: unknown[]) => getResetGate(...a),
  },
}))
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}))

const openConfirm = async () => {
  render(<ResetButton traderId="t1" language="en" onDone={() => {}} />)
  await waitFor(() => screen.getByTestId('reset-button'))
  fireEvent.click(screen.getByTestId('reset-button'))
  await waitFor(() => screen.getByTestId('reset-go'))
}

beforeEach(() => {
  forceReset.mockReset()
  getResetGate.mockReset()
  getResetGate.mockResolvedValue({ allowed: true })
})

describe('ResetButton confirm dialog', () => {
  it('round-trip: success closes the confirm box and re-enables', async () => {
    forceReset.mockResolvedValue({ ok: true })
    await openConfirm()
    fireEvent.click(screen.getByTestId('reset-go'))
    await waitFor(() => expect(forceReset).toHaveBeenCalledTimes(1))
    await waitFor(() =>
      expect(screen.queryByTestId('reset-confirm')).toBeNull()
    )
    expect(screen.getByTestId('reset-button')).not.toBeDisabled()
  })

  it('forced 500 ({ok:false}): dialog recovers, never latches', async () => {
    forceReset.mockResolvedValue({ ok: false, error: 'server error' })
    await openConfirm()
    fireEvent.click(screen.getByTestId('reset-go'))
    await waitFor(() => expect(forceReset).toHaveBeenCalledTimes(1))
    // confirm box stays open for retry, buttons re-enabled — no F5 needed
    await waitFor(() =>
      expect(screen.getByTestId('reset-go')).not.toBeDisabled()
    )
  })

  it('forced timeout (REJECTED promise): dialog recovers — the latch bug', async () => {
    forceReset.mockRejectedValue(new Error('timeout of 30000ms exceeded'))
    await openConfirm()
    fireEvent.click(screen.getByTestId('reset-go'))
    await waitFor(() => expect(forceReset).toHaveBeenCalledTimes(1))
    await waitFor(() =>
      expect(screen.getByTestId('reset-go')).not.toBeDisabled()
    )
    // …and a retry works
    forceReset.mockResolvedValue({ ok: true })
    fireEvent.click(screen.getByTestId('reset-go'))
    await waitFor(() => expect(forceReset).toHaveBeenCalledTimes(2))
  })

  it('double-click fires exactly ONE reset (the 01:27/01:30 duplicate class)', async () => {
    let release: (v: { ok: boolean }) => void = () => {}
    forceReset.mockImplementation(() => new Promise((r) => (release = r)))
    await openConfirm()
    fireEvent.click(screen.getByTestId('reset-go'))
    fireEvent.click(screen.getByTestId('reset-go'))
    fireEvent.click(screen.getByTestId('reset-go'))
    release({ ok: true })
    await waitFor(() =>
      expect(screen.queryByTestId('reset-confirm')).toBeNull()
    )
    expect(forceReset).toHaveBeenCalledTimes(1)
  })
})
