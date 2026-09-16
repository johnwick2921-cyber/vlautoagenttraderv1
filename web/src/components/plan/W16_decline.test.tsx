// W16/R2 — declining a proposal must not claim it was applied.
//
// Both buttons used to set the SAME boolean, so "Keep as-is" rendered the
// applied confirmation ("Applied — card updated") while persisting nothing.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'

const calls: string[] = []
vi.mock('../../lib/api', () => ({
  api: {
    applyAsk: () => {
      calls.push('apply')
      return Promise.resolve({ ok: true })
    },
    declineAsk: () => {
      calls.push('decline')
      return Promise.resolve({ ok: true })
    },
    askPlanner: () => Promise.resolve({ ok: true }),
    getPlanThread: () => Promise.resolve({ thread: threadFixture, kpi: null }),
  },
}))
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

let threadFixture: unknown[] = []

const proposal = {
  id: 7,
  role: 'planner' as const,
  content: 'I propose moving the fade level.',
  verdict: 'PROPOSE-MERGE',
  point_class: 'NEW-INFO',
  patch: '[{"op":"replace","path":"/levels/0/instruction","value":"fade"}]',
  applied: false,
  created_at: 0,
}

// The reply bubble is not exported; drive it through the panel, whose thread
// comes from api.getPlanThread (mocked above via threadFixture).
async function renderBubble(applied = false) {
  threadFixture = [{ ...proposal, applied }]
  const { AskPlannerPanel } = await import('./AskPlannerPanel')
  return render(
    <AskPlannerPanel
      open
      onClose={vi.fn()}
      traderId="t1"
      symbol="MNQ"
      language="en"
      planId="2026-08-17:NY"
      planVersion={1}
      onApplied={vi.fn()}
    />
  )
}

describe('Ask-Planner · decline is its own recorded outcome', () => {
  beforeEach(() => {
    calls.length = 0
  })

  it('declining calls the DECLINE endpoint, not apply', async () => {
    await renderBubble()
    fireEvent.click(await screen.findByTestId('ask-keep-as-is'))
    await waitFor(() => expect(calls).toContain('decline'))
    expect(calls).not.toContain('apply')
  })

  it('after declining the card says the plan was NOT changed', async () => {
    await renderBubble()
    fireEvent.click(await screen.findByTestId('ask-keep-as-is'))
    await waitFor(() =>
      expect(screen.getByTestId('ask-outcome-declined')).toBeTruthy()
    )
    expect(screen.queryByTestId('ask-outcome-applied')).toBeNull()
    expect(screen.getByTestId('ask-outcome-declined').textContent).toMatch(
      /nothing was changed/i
    )
  })

  it('an already-applied proposal still renders the applied state', async () => {
    await renderBubble(true)
    expect(await screen.findByTestId('ask-outcome-applied')).toBeTruthy()
    expect(screen.queryByTestId('ask-outcome-declined')).toBeNull()
  })
})
