import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ScenarioList } from './ScenarioList'
import type { PlanOrderLeg } from '../../lib/api/plan'

const leg: PlanOrderLeg = {
  state: 'working',
  leg_index: 0,
  row_id: 3,
  placement_seq: 2,
  version: 6,
  armed_under_version: 5,
  side: 'short',
  intended: { entry: 100, stop: 110, target: 90, source: 'displayed plan v6' },
  composed: { entry: 102, stop: 112, target: 92, source: 'armed_orders row 3' },
  accepted: {
    entry: 102.25,
    stop: 112.25,
    target: null,
    source: 'current NT8 order_snapshot',
    reason: 'target: absent or ambiguous',
  },
  book_received_at_ms: 1788836242257,
  book_age_ms: 21000,
  build_id: '2026-09-07-h1',
}
const scenario = {
  id: 'S1',
  trigger: 'at 100',
  condition: 'reclaim',
  direction: 'short',
  target_chain: [90],
  invalid: '110',
  quality: 'A',
}

describe('scenario order price truth', () => {
  it('renders distinct intended, composed and accepted prices with placement provenance', () => {
    render(
      <ScenarioList
        scenarios={[scenario]}
        statusMap={{ S1: 'armed' }}
        armedStates={{ S1: { state: 'working', legs: [leg] } }}
        language="en"
      />
    )
    const table = screen.getByRole('table', { name: 'Leg 1 order prices' })
    const rows = within(table).getAllByRole('row')
    expect(
      within(rows[1])
        .getAllByRole('cell')
        .map((c) => c.textContent)
    ).toEqual(['100', '102', '102.25'])
    expect(
      within(rows[2])
        .getAllByRole('cell')
        .map((c) => c.textContent)
    ).toEqual(['110', '112', '112.25'])
    expect(
      within(rows[3])
        .getAllByRole('cell')
        .map((c) => c.textContent)
    ).toEqual(['90', '92', 'UNKNOWN'])
    expect(
      screen.getByText(
        /row 3 · placement 2 · last touched v6 · first authorized v5/
      )
    ).toBeTruthy()
    expect(screen.getByText(/age 21s · AddOn build 2026-09-07-h1/)).toBeTruthy()
  })
  it('keeps the order evidence visible when scenario activation is unevaluable', () => {
    render(
      <ScenarioList
        scenarios={[scenario]}
        armedStates={{ S1: { state: 'working', legs: [leg] } }}
        language="en"
      />
    )
    expect(screen.getByTestId('scenario-unevaluable-S1')).toBeTruthy()
    expect(
      screen.getByRole('table', { name: 'Leg 1 order prices' })
    ).toBeTruthy()
  })
})
