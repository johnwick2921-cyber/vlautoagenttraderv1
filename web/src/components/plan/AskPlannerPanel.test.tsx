// Phase 5 (final-bundle 2026-08-19) — Ask-Planner stuck-send regression tests.
// The bug: send() awaited api.askPlanner with no try/catch; the 30s axios
// timeout (vs the planner's 300s budget) threw, busy latched true, the cleared
// input lost the question, and only F5 recovered.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { AskPlannerPanel } from './AskPlannerPanel'

const askPlanner = vi.fn()
const getPlanThread = vi.fn()

vi.mock('../../lib/api', () => ({
  api: {
    askPlanner: (...a: unknown[]) => askPlanner(...a),
    getPlanThread: (...a: unknown[]) => getPlanThread(...a),
    applyAsk: vi.fn(),
    declineAsk: vi.fn(),
  },
}))
vi.mock('sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

function mount() {
  return render(
    <AskPlannerPanel
      open
      traderId="t1"
      symbol="MNQ"
      planId="p:NY"
      planVersion={1}
      language="en"
      onClose={() => {}}
      onApplied={() => {}}
    />
  )
}

const type = (q: string) => {
  fireEvent.change(screen.getByRole('textbox'), { target: { value: q } })
}
const clickSend = () => {
  fireEvent.click(screen.getByRole('button', { name: /send/i }))
}

beforeEach(() => {
  askPlanner.mockReset()
  getPlanThread.mockReset()
  getPlanThread.mockResolvedValue({ thread: [], kpi: {} })
})

describe('AskPlannerPanel send', () => {
  it('normal round-trip clears the input and re-enables the button', async () => {
    askPlanner.mockResolvedValue({ ok: true, data: {} })
    mount()
    type('why this level?')
    clickSend()
    await waitFor(() => expect(askPlanner).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(screen.getByRole('textbox')).not.toBeDisabled())
    expect((screen.getByRole('textbox') as HTMLInputElement).value).toBe('')
  })

  it('a REJECTED call (timeout/network) recovers: input restored, button alive — never needs F5', async () => {
    askPlanner.mockRejectedValue(new Error('timeout of 30000ms exceeded'))
    mount()
    type('what kills the plan?')
    clickSend()
    await waitFor(() => expect(askPlanner).toHaveBeenCalledTimes(1))
    // busy must ALWAYS reset (finally) and the question must come back.
    await waitFor(() => expect(screen.getByRole('textbox')).not.toBeDisabled())
    expect((screen.getByRole('textbox') as HTMLInputElement).value).toBe(
      'what kills the plan?'
    )
    // …and the panel can send again immediately.
    askPlanner.mockResolvedValue({ ok: true, data: {} })
    clickSend()
    await waitFor(() => expect(askPlanner).toHaveBeenCalledTimes(2))
  })

  it('a 500-style {ok:false} result toasts and preserves the input', async () => {
    askPlanner.mockResolvedValue({ ok: false, error: 'server error' })
    mount()
    type('question')
    clickSend()
    await waitFor(() => expect(screen.getByRole('textbox')).not.toBeDisabled())
    expect((screen.getByRole('textbox') as HTMLInputElement).value).toBe(
      'question'
    )
  })

  it('rapid double-click sends exactly one message (busy guard)', async () => {
    let release: (v: { ok: boolean }) => void = () => {}
    askPlanner.mockImplementation(() => new Promise((r) => (release = r)))
    mount()
    type('once')
    clickSend()
    clickSend()
    clickSend()
    release({ ok: true })
    await waitFor(() => expect(screen.getByRole('textbox')).not.toBeDisabled())
    expect(askPlanner).toHaveBeenCalledTimes(1)
  })
})
