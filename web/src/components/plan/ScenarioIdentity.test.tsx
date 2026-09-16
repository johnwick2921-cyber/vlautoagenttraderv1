import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { ScenarioIdentity } from './ScenarioList'

describe('recorded scenario identity', () => {
  it('renders legacy NULL without guessing', () => {
    render(<ScenarioIdentity />)
    expect(screen.getByTestId('scenario-level-identity').textContent).toContain(
      'Level ID: NULL'
    )
    expect(screen.getByText(/no identity inferred/)).toBeTruthy()
  })
  it('shows the resolved candidate beside the unchanged evaluator anchor', () => {
    render(
      <ScenarioIdentity
        identity={{
          level_id: 'recorded-id',
          basis: 'candidate_id',
          evaluator_anchor: 20005,
          disagreed: true,
          level: {
            id: 'recorded-id',
            price: 20000,
            label: 'PDL',
            grade: 'A',
            instruction: 'hold',
            tf: '1m',
            formed_close_ms: 1789059900000,
          },
        }}
      />
    )
    const body = screen.getByTestId('scenario-level-identity').textContent
    expect(body).toContain('PDL @ 20000.00')
    expect(body).toContain(
      'evaluator anchor 20005.00 differs; decision unchanged'
    )
    expect(body).toContain('formation close 1789059900000')
  })
  it('keeps unknown authored IDs visible with WARN', () => {
    render(<ScenarioIdentity authoredId="unknown-id" />)
    expect(screen.getByTestId('scenario-level-identity').textContent).toContain(
      'unknown-id · Unresolved [WARN]'
    )
  })
})
