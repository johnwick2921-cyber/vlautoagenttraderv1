// W13 (FE) — plan re-alignment on owner edit. api + sonner mocked so the tests
// drive the flow from props and assert the calls + rendered states.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import type { PlanLevelFact, RealignResponse } from '../../lib/api/plan'

const realignPlan = vi.fn()
const applyAsk = vi.fn().mockResolvedValue({ ok: true })
const addOwnerLevel = vi.fn().mockResolvedValue({ ok: true, id: 1 })
const postOverlay = vi.fn().mockResolvedValue({ ok: true })

vi.mock('../../lib/api', () => ({
  api: {
    realignPlan: (...a: unknown[]) => realignPlan(...a),
    applyAsk: (...a: unknown[]) => applyAsk(...a),
    addOwnerLevel: (...a: unknown[]) => addOwnerLevel(...a),
    postOverlay: (...a: unknown[]) => postOverlay(...a),
  },
}))
vi.mock('sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { RealignPanel, RealignButton, type RealignState } from './RealignPanel'
import { BulkAddSheet } from './BulkAddSheet'

const proposal: RealignResponse = {
  status: 'proposal',
  qa_id: 77,
  plan_version: 1,
  would_become: 'v1+o2',
  cost_usd: 0.0021,
  latency_ms: 4100,
  reply: {
    evidence: '30246.5 marked supply from the 1h lower-high.',
    point_class: 'NEW-INFO',
    verdict: 'PROPOSE-MERGE',
    summary: 'HTF origin outranks the 1h read.',
    patch: '[{"op":"replace","path":"/bias/conviction","value":"high"}]',
  },
}

const panel = (state: RealignState, onApplied = vi.fn(), onDismiss = vi.fn()) =>
  render(
    <RealignPanel
      state={state}
      traderId="t1"
      symbol="MNQ"
      language="en"
      onDismiss={onDismiss}
      onApplied={onApplied}
    />
  )

describe('W13 · re-align panel states', () => {
  beforeEach(() => vi.clearAllMocks())

  it('renders the quiet inline reviewing line', () => {
    panel({ phase: 'reviewing' })
    expect(screen.getByTestId('realign-reviewing').textContent).toMatch(
      /reviewing your change/i
    )
  })

  it('renders the no-change chip (and auto-fades it)', () => {
    vi.useFakeTimers()
    try {
      const onDismiss = vi.fn()
      panel({ phase: 'no-change' }, vi.fn(), onDismiss)
      expect(screen.getByTestId('realign-no-change').textContent).toMatch(
        /no plan change/i
      )
      expect(onDismiss).not.toHaveBeenCalled()
      vi.advanceTimersByTime(5000) // the chip fades itself; nothing to click
      expect(onDismiss).toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
    }
  })

  it('renders the proposal card with the target version + patch rows', () => {
    panel({ phase: 'proposal', res: proposal })
    const card = screen.getByTestId('realign-proposal')
    expect(card.textContent).toMatch(/would become/i)
    expect(card.textContent).toContain('v1+o2')
    expect(card.textContent).toMatch(/HTF origin outranks/)
    expect(card.textContent).toMatch(/replace/) // PatchPreview op row
  })

  it('Apply routes through applyAsk and reports applied; Keep never mutates', async () => {
    const onApplied = vi.fn()
    const onDismiss = vi.fn()
    panel({ phase: 'proposal', res: proposal }, onApplied, onDismiss)

    fireEvent.click(screen.getByText(/Apply merge/i))
    await waitFor(() => expect(applyAsk).toHaveBeenCalledWith('t1', 77, 'MNQ'))
    await waitFor(() => expect(onApplied).toHaveBeenCalled())

    vi.clearAllMocks()
    panel({ phase: 'proposal', res: proposal }, onApplied, onDismiss)
    fireEvent.click(screen.getAllByText(/Keep as-is/i)[0])
    expect(applyAsk).not.toHaveBeenCalled() // reject path: plan untouched
  })

  it('capped state points the owner at the manual button', () => {
    panel({ phase: 'capped' })
    expect(screen.getByTestId('realign-capped').textContent).toMatch(
      /Re-align plan/i
    )
  })

  it('failed state says the plan is unchanged (fail-closed)', () => {
    panel({ phase: 'failed' })
    expect(screen.getByTestId('realign-failed').textContent).toMatch(
      /plan unchanged/i
    )
  })

  it('the manual button is always available as the fallback affordance', () => {
    const onClick = vi.fn()
    render(<RealignButton language="en" onClick={onClick} />)
    fireEvent.click(screen.getByTestId('realign-button'))
    expect(onClick).toHaveBeenCalled()
  })
})

describe('W13 · bulk-add fires ONE re-align for the batch', () => {
  beforeEach(() => vi.clearAllMocks())

  it('emits a single bulk-add change carrying the batch count', async () => {
    const onSaved = vi.fn()
    render(
      <BulkAddSheet
        open
        traderId="t1"
        symbol="MNQ"
        language="en"
        onClose={vi.fn()}
        onSaved={onSaved}
      />
    )
    const ta = document.querySelector('textarea') as HTMLTextAreaElement
    fireEvent.change(ta, {
      target: { value: '30156 D-zone\n30246 S-zone\n30288 level' },
    })
    fireEvent.click(screen.getAllByText(/Save/i)[0])

    await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1)) // ONE, not per row
    const change = onSaved.mock.calls[0][0]
    expect(change.kind).toBe('bulk-add')
    expect(change.batch_count).toBe(3)
    expect(addOwnerLevel).toHaveBeenCalledTimes(3) // rows still saved individually
  })
})
