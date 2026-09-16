// P6 (2026-08-17) — the owner reset: abandon the chain, restore the full
// budget, fresh plan. Distinct from the re-read; the two sit side by side with
// one explanatory line each.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ResetButton } from './ResetButton'
import { api } from '../../lib/api'
import type { ResetGate } from '../../lib/api/plan'

const allowed: ResetGate = {
  allowed: true,
  session: 'NY',
  version: 6,
  replan_cap: 4,
}
const refused: ResetGate = {
  allowed: false,
  reason: 'no plan has been written yet — the first read costs nothing',
  session: 'NY',
  version: 0,
  replan_cap: 4,
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('ResetButton', () => {
  it('never resets without an explicit confirm', async () => {
    vi.spyOn(api, 'getResetGate').mockResolvedValue(allowed)
    const force = vi.spyOn(api, 'forceReset')
    render(<ResetButton traderId="t1" language="en" />)

    fireEvent.click(await screen.findByTestId('reset-button'))
    expect(await screen.findByTestId('reset-confirm')).toBeInTheDocument()
    expect(force).not.toHaveBeenCalled()
  })

  it('says what it will do before spending: abandon + restore + untouched positions', async () => {
    vi.spyOn(api, 'getResetGate').mockResolvedValue(allowed)
    render(<ResetButton traderId="t1" language="en" />)
    fireEvent.click(await screen.findByTestId('reset-button'))

    const confirm = await screen.findByTestId('reset-confirm')
    expect(confirm.textContent).toContain('ABANDONED')
    expect(confirm.textContent).toContain('never touched')
  })

  it('fires exactly once on confirm', async () => {
    vi.spyOn(api, 'getResetGate').mockResolvedValue(allowed)
    const force = vi
      .spyOn(api, 'forceReset')
      .mockResolvedValue({ ok: true, gate: allowed })
    render(<ResetButton traderId="t1" language="en" />)

    fireEvent.click(await screen.findByTestId('reset-button'))
    fireEvent.click(await screen.findByTestId('reset-go'))
    await waitFor(() => expect(force).toHaveBeenCalledTimes(1))
    expect(force).toHaveBeenCalledWith('t1')
  })

  it('is disabled AND says why when the reset is refused', async () => {
    vi.spyOn(api, 'getResetGate').mockResolvedValue(refused)
    render(<ResetButton traderId="t1" language="en" />)

    const btn = await screen.findByTestId('reset-button')
    expect(btn).toBeDisabled()
    expect(screen.getByTestId('reset-reason').textContent).toContain(
      'first read costs nothing'
    )
    fireEvent.click(btn)
    expect(screen.queryByTestId('reset-confirm')).toBeNull()
  })

  it('renders nothing without a trader', () => {
    const { container } = render(<ResetButton language="en" />)
    expect(container.firstChild).toBeNull()
  })
})
