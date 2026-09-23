import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ScenarioList } from './ScenarioList'
import { entryPolicyLine } from './EntryPolicyLine'
import type { PlanOrderLeg } from '../../lib/api/plan'

// W-EXEC-TRUTH W3 (h) — the entry line rendered through the REAL ScenarioList
// with armedStates (the pattern of OrderTerms.test.tsx), as the plan card
// mounts it.

const base: PlanOrderLeg = {
  state: 'armed',
  leg_index: 0,
  row_id: 21,
  placement_seq: 1,
  version: 2,
  armed_under_version: 2,
  side: 'long',
  intended: { entry: 31005, stop: 30990, target: 31050, source: 'plan v2' },
  composed: {
    entry: 31010,
    stop: 30990,
    target: 31050,
    source: 'armed_orders row 21',
  },
  accepted: { entry: null, stop: null, target: null, source: 'book' },
  book_age_ms: 0,
  build_id: 'h1',
  policy: 'market_in_zone',
  planned_entry: 31005,
  zone_lo: 31000,
  zone_hi: 31010,
}
const scenario = {
  id: 'S1',
  trigger: 'reclaim 31000',
  condition: 'reclaim',
  direction: 'long',
  target_chain: [31050],
  invalid: '30990',
  quality: 'A',
}

function lineFor(leg: PlanOrderLeg, statusMap: boolean = true): string | null {
  const { unmount } = render(
    <ScenarioList
      scenarios={[scenario]}
      statusMap={statusMap ? { S1: 'armed' } : undefined}
      armedStates={{ S1: { state: leg.state, legs: [leg] } }}
      language="en"
    />
  )
  const el = screen.queryByTestId('entry-policy-line-0')
  const text = el ? el.textContent : null
  unmount()
  return text
}

const HEAD = 'Entry: around 31,005 (zone 31,000–31,010) · '

describe('W3 entry-policy line on the plan card', () => {
  it('armed with no verdict, or a waiting/short_of_zone/unknown verdict → Waiting for price', () => {
    for (const verdict of [
      undefined,
      'short_of_zone',
      'unknown',
      'waiting: price below zone 31000.00–31010.00 (last 30990.25)',
    ]) {
      expect(lineFor({ ...base, verdict })).toBe(HEAD + 'Waiting for price')
    }
  })

  it('armed with a refused verdict → Blocked: <reason>', () => {
    expect(
      lineFor({ ...base, verdict: 'refused: arm_slot: S2 holds the slot' })
    ).toBe(HEAD + 'Blocked: arm_slot: S2 holds the slot')
  })

  it('place_pending / working → Placed at <the composed limit>', () => {
    for (const state of ['place_pending', 'working']) {
      expect(lineFor({ ...base, state, verdict: 'inside' })).toBe(
        HEAD + 'Placed at 31,010'
      )
    }
  })

  it('filled → Filled <fill_price> (+ ticks when measured; a 0 is a value)', () => {
    const filled = { ...base, state: 'filled', fill_price: 31006.25 }
    expect(lineFor({ ...filled, fill_slippage_ticks: -15 })).toBe(
      HEAD + 'Filled 31,006.25 (-15 ticks)'
    )
    expect(lineFor({ ...filled, fill_slippage_ticks: 0 })).toBe(
      HEAD + 'Filled 31,006.25 (0 ticks)'
    )
    expect(lineFor({ ...filled, fill_slippage_ticks: 1 })).toBe(
      HEAD + 'Filled 31,006.25 (1 tick)'
    )
    expect(lineFor(filled)).toBe(HEAD + 'Filled 31,006.25')
    expect(lineFor({ ...base, state: 'filled' })).toBe(HEAD + 'Filled UNKNOWN')
  })

  it('cancelled / rejected → Blocked: <state_reason>', () => {
    expect(
      lineFor({ ...base, state: 'cancelled', reason: 'zone rest expired' })
    ).toBe(HEAD + 'Blocked: zone rest expired')
    expect(
      lineFor({ ...base, state: 'rejected', reason: 'broker rejected' })
    ).toBe(HEAD + 'Blocked: broker rejected')
  })

  it('a legacy or planned_order leg renders nothing', () => {
    const {
      policy: _p,
      planned_entry: _e,
      zone_lo: _l,
      zone_hi: _h,
      ...legacy
    } = base
    expect(lineFor(legacy as PlanOrderLeg)).toBeNull()
    expect(lineFor({ ...base, policy: 'planned_order' })).toBeNull()
    // …and the rest of the row still renders for a legacy leg.
    render(
      <ScenarioList
        scenarios={[scenario]}
        statusMap={{ S1: 'armed' }}
        armedStates={{
          S1: { state: 'working', legs: [legacy as PlanOrderLeg] },
        }}
        language="en"
      />
    )
    expect(screen.getByTestId('armed-chip')).toBeTruthy()
    expect(screen.queryByText(/Entry: around/)).toBeNull()
  })

  it('absent numbers read n/a, never 0', () => {
    const bare: PlanOrderLeg = {
      ...base,
      planned_entry: undefined,
      zone_lo: undefined,
      zone_hi: undefined,
      composed: { entry: null, stop: null, target: null, source: 'x' },
    }
    expect(lineFor(bare)).toBe(
      'Entry: around n/a (zone n/a–n/a) · Waiting for price'
    )
    expect(lineFor({ ...bare, state: 'working' })).toBe(
      'Entry: around n/a (zone n/a–n/a) · Placed at n/a'
    )
    expect(lineFor({ ...bare, state: 'working' })).not.toMatch(/\b0\b/)
  })

  it('en-US numbers with up to 2 decimals', () => {
    expect(
      entryPolicyLine({
        ...base,
        planned_entry: 31005.25,
        zone_lo: 31000.5,
        zone_hi: 31010.75,
      })
    ).toBe(
      'Entry: around 31,005.25 (zone 31,000.5–31,010.75) · Waiting for price'
    )
  })

  it('stays visible when scenario activation is unevaluable (order evidence)', () => {
    expect(lineFor({ ...base, state: 'working' }, false)).toBe(
      HEAD + 'Placed at 31,010'
    )
  })
})
