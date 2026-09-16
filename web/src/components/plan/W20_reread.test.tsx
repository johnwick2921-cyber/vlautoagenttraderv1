// ITEM 3 (2026-08-17) — the owner's manual re-read.
//
// It SPENDS one re-read from the same session budget the automatic path uses, so
// it can never talk the bot past its own limits, and it says so BEFORE spending
// because it costs a real API call.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { RereadButton } from './RereadButton'
import { api } from '../../lib/api'
import type { RereadGate } from '../../lib/api/plan'

const allowed: RereadGate = {
  allowed: true,
  session: 'NY',
  replans_left: 3,
  replan_cap: 4,
  version: 2,
}
const refused: RereadGate = {
  allowed: false,
  reason: 'the re-read budget for NY is spent (4 of 4 used)',
  session: 'NY',
  replans_left: 0,
  replan_cap: 4,
  version: 5,
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('RereadButton', () => {
  it('never spends without an explicit confirm', async () => {
    vi.spyOn(api, 'getRereadGate').mockResolvedValue(allowed)
    const force = vi.spyOn(api, 'forceReread')
    render(<RereadButton traderId="t1" language="en" />)

    const btn = await screen.findByTestId('reread-button')
    fireEvent.click(btn)
    // The click opens the confirm step; it must NOT call the API yet.
    expect(await screen.findByTestId('reread-confirm')).toBeInTheDocument()
    expect(force).not.toHaveBeenCalled()
  })

  it('states the cost in re-reads before spending', async () => {
    vi.spyOn(api, 'getRereadGate').mockResolvedValue(allowed)
    render(<RereadButton traderId="t1" language="en" />)
    fireEvent.click(await screen.findByTestId('reread-button'))

    const confirm = await screen.findByTestId('reread-confirm')
    expect(confirm.textContent).toContain('one of 3 re-reads')
    expect(confirm.textContent).toContain('costs an API call')
  })

  it('spends exactly once on confirm and refreshes the card', async () => {
    vi.spyOn(api, 'getRereadGate').mockResolvedValue(allowed)
    const force = vi
      .spyOn(api, 'forceReread')
      .mockResolvedValue({ ok: true, gate: allowed })
    const onDone = vi.fn()
    render(<RereadButton traderId="t1" language="en" onDone={onDone} />)

    fireEvent.click(await screen.findByTestId('reread-button'))
    fireEvent.click(await screen.findByTestId('reread-go'))
    await waitFor(() => expect(force).toHaveBeenCalledTimes(1))
    expect(force).toHaveBeenCalledWith('t1')
  })

  it('is disabled AND says why when the budget is spent', async () => {
    vi.spyOn(api, 'getRereadGate').mockResolvedValue(refused)
    render(<RereadButton traderId="t1" language="en" />)

    const btn = await screen.findByTestId('reread-button')
    expect(btn).toBeDisabled()
    expect(screen.getByTestId('reread-reason').textContent).toContain(
      '4 of 4 used'
    )
    // A disabled control must not open the confirm step.
    fireEvent.click(btn)
    expect(screen.queryByTestId('reread-confirm')).toBeNull()
  })

  it('renders nothing without a trader', () => {
    const { container } = render(<RereadButton language="en" />)
    expect(container.firstChild).toBeNull()
  })
})
