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
  // W-EXEC-TRUTH W2 — the chip reads the recorded verdict of the ONE confirm
  // resolver: a waterfall scenario that STORES 2x5m_close is recorded (and
  // chipped) as 2×5m, never the BD_MIN_CLOSES default the old evaluator
  // synthesized; a default is labelled as one.
  it('shows the stored 2x5m_close rule of a recorded two-leg verdict', () => {
    render(
      <ConfirmChip
        id="S1"
        c={{
          rule: '2x5m_close',
          rule_source: 'stored',
          outcome: 'MET',
          ref_price: 30264,
          side: 'below',
          met: true,
          detail: 'pullback failed to reclaim — entry live',
          legs: [
            {
              met: true,
              rule: '2x5m_close',
              ref_price: 30264,
              side: 'below',
              detail: 'best run 3/2 completed closes below 30264.00',
            },
            {
              met: true,
              rule: 'retest_fail',
              ref_price: 30264,
              side: 'below',
              detail: 'pullback failed to reclaim — entry live',
            },
          ],
        }}
      />
    )
    const text = screen.getByTestId('confirm-chip-S1').textContent
    expect(text).toContain('Recorded 2×5m MET (1/2 MET · 2/2 MET)')
    expect(text).not.toContain('1×5m')
    expect(text).not.toContain('authoring default')
  })
  it('labels an authoring-default rule as a default', () => {
    render(
      <ConfirmChip
        id="S2"
        c={{
          rule: '1x5m_close',
          rule_source: 'authoring_default',
          outcome: 'NOT MET',
          ref_price: 100,
          side: 'below',
          met: false,
          detail: 'waiting for the breakdown leg',
        }}
      />
    )
    expect(screen.getByTestId('confirm-chip-S2').textContent).toContain(
      'Recorded 1×5m (authoring default) NOT MET'
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
