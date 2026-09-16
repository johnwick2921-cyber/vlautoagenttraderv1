import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ConfirmChip } from './ScenarioList'

describe('recorded confirmation evidence', () => {
  it('shows the forming bucket reason outside the tooltip', () => {
    render(
      <ConfirmChip
        id="S3"
        c={{
          rule: '1x5m_close',
          ref_price: 29661.25,
          side: 'below',
          met: false,
          detail:
            'forming bucket 08:15–08:20 CT; closes 08:20 CT; closed=false',
        }}
      />
    )
    expect(screen.getByTestId('confirm-chip-S3').textContent).toContain(
      '08:20 CT'
    )
    expect(screen.getByTestId('confirm-chip-S3').textContent).toContain(
      'closed=false'
    )
  })
  it('shows UNKNOWN and its missing reference reason', () => {
    const c = {
      rule: 'touch',
      ref_price: 100,
      side: 'above',
      met: false,
      outcome: 'UNKNOWN',
      detail: 'part one reference instant is missing',
    }
    render(<ConfirmChip id="S1" c={c} />)
    expect(screen.getByTestId('confirm-chip-S1').textContent).toContain(
      'UNKNOWN'
    )
    expect(screen.getByTestId('confirm-chip-S1').textContent).toContain(
      'reference instant is missing'
    )
  })
})
