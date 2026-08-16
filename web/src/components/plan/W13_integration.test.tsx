// W13 integration — the exact path the owner used in the sandbox: open the card,
// add a level, and expect the proposal card to appear. Mocks return the REAL
// backend JSON the sandbox produced (verbatim shape from .sandbox/api.log).

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import type { PlanToday } from '../../lib/api/plan'

const realignPlan = vi.fn()
const addOwnerLevel = vi.fn().mockResolvedValue({ ok: true, id: 9 })
const postOverlay = vi.fn().mockResolvedValue({ ok: true })
const applyAsk = vi.fn().mockResolvedValue({ ok: true })
const getPlanThread = vi.fn().mockResolvedValue({ thread: [], kpi: {} })

vi.mock('../../lib/api', () => ({
  api: {
    realignPlan: (...a: unknown[]) => realignPlan(...a),
    addOwnerLevel: (...a: unknown[]) => addOwnerLevel(...a),
    postOverlay: (...a: unknown[]) => postOverlay(...a),
    applyAsk: (...a: unknown[]) => applyAsk(...a),
    getPlanThread: (...a: unknown[]) => getPlanThread(...a),
  },
}))
vi.mock('sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { SessionPlanCard } from './SessionPlanCard'

const plan: PlanToday = {
  found: true,
  trade_date: '2026-08-16',
  session: 'NY',
  night: false,
  mode: 'advisory',
  version: 1,
  lifecycle: 'active',
  price: 30231.5,
  replans_left: 2,
  level_facts: [
    {
      price: 30288,
      label: 'PDH',
      grade: 'A',
      instruction: 'target only',
      distance: 56.5,
      sweep: false,
      closes_beyond: 0,
      accept_have: 0,
      accept_need: 2,
      still_valid: true,
    },
  ],
  doc: {
    reasoning: 'r',
    bias: {
      direction: 'long',
      conviction: 'medium',
      flip_condition: 'flips short below 30148',
    },
    levels: [
      { price: 30288, label: 'PDH', grade: 'A', instruction: 'target only' },
    ],
    scenarios: [
      {
        id: 'S1',
        trigger: 't',
        condition: 'sweep_reclaim',
        direction: 'long',
        target_chain: [30288],
        invalid: 'x',
        quality: 'A+',
      },
    ],
    no_trade: ['first 5 min'],
    death_condition: '2×5m closes < 30148',
  },
}

// the exact body the sandbox backend returned for an add-level (status 200)
const backendProposal = {
  status: 'proposal',
  qa_id: 12,
  plan_id: '2026-08-16:NY',
  plan_version: 1,
  would_become: 'v1+o2',
  latency_ms: 0,
  cost_usd: 0.0002,
  reply: {
    evidence:
      'S1 triggers on a sweep+reclaim of 30160-52; the level you added sits just under that band.',
    point_class: 'NEW-INFO',
    verdict: 'PROPOSE-MERGE',
    summary: 'Your zone EXTENDS S1 rather than needing a new play.',
    patch:
      '[{"op":"replace","path":"/scenarios/0/trigger","value":"sweep 30160-30156 then reclaim"}]',
  },
}

describe('W13 integration — add a level on the card', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows the proposal card after an ＋Add level save', async () => {
    realignPlan.mockResolvedValue(backendProposal)
    render(
      <SessionPlanCard
        plan={plan}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        onChanged={vi.fn()}
      />
    )

    // open ＋Add level → fill a price → Save (the owner's exact gesture)
    fireEvent.click(screen.getByText(/Add level/i))
    const price = document.querySelector('input') as HTMLInputElement
    fireEvent.change(price, { target: { value: '30156' } })
    fireEvent.click(screen.getByText(/^Save/i))

    await waitFor(() => expect(addOwnerLevel).toHaveBeenCalled())
    await waitFor(() => expect(realignPlan).toHaveBeenCalled())

    // THE ASSERTION THAT MATTERS: the proposal must be on screen
    await waitFor(() => {
      expect(screen.getByTestId('realign-proposal')).toBeTruthy()
    })
    expect(screen.getByTestId('realign-proposal').textContent).toContain(
      'v1+o2'
    )
  })
})
