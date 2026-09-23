import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { EffectiveChip, formatEffective } from './EffectiveChip'
import type { EffectiveKnob } from '../../lib/api/strategyEffective'

function knob(over: Partial<EffectiveKnob>): EffectiveKnob {
  return {
    path: 'ai_config.risk_control.consecutive_loss_halt',
    status: 'live',
    ui_label: 'live',
    stored: { present: false },
    effective: 8,
    origin: 'shipped default',
    scope: 'strategy',
    resolver: 'trader.breakerHaltN',
    resolved: true,
    ...over,
  }
}

describe('EffectiveChip', () => {
  it('renders a saved value with its origin and scope', () => {
    render(
      <EffectiveChip
        knob={knob({
          stored: { present: true, value: 3 },
          effective: 3,
          origin: 'saved value',
        })}
      />
    )
    const chip = screen.getByTestId('effective-chip')
    expect(chip).toHaveTextContent('eff 3 · saved value · strategy')
    expect(chip.getAttribute('title')).toContain('stored: 3')
  })

  it('renders a default the server resolved, and says nothing was stored', () => {
    render(<EffectiveChip knob={knob({})} />)
    const chip = screen.getByTestId('effective-chip')
    expect(chip).toHaveTextContent('eff 8 · shipped default · strategy')
    expect(chip.getAttribute('title')).toContain('stored: absent')
  })

  it('renders "n/a — no resolver registered" verbatim, with no eff value', () => {
    render(
      <EffectiveChip
        knob={knob({
          path: 'grid_config.grid_count',
          effective: 'n/a — no resolver registered',
          origin: 'unset',
          resolver: 'none',
          resolved: false,
        })}
      />
    )
    const chip = screen.getByTestId('effective-chip')
    expect(chip).toHaveTextContent('n/a — no resolver registered')
    expect(chip.textContent).not.toContain('eff')
  })

  it('renders a secret as "redacted" and never its stored value', () => {
    render(
      <EffectiveChip
        knob={knob({
          path: 'ai_config.indicators.nofxos_api_key',
          stored: { present: true, value: 'redacted' },
          effective: 'redacted',
          origin: 'saved value',
          resolver: 'none',
          resolved: false,
        })}
      />
    )
    const chip = screen.getByTestId('effective-chip')
    expect(chip).toHaveTextContent(/^redacted$/)
    expect(chip.getAttribute('title')).toContain('stored: redacted')
  })

  it('never renders 0 for an absent or unknown value', () => {
    const { container, rerender } = render(<EffectiveChip knob={undefined} />)
    expect(container.textContent).toBe('')
    rerender(<EffectiveChip knob={null} />)
    expect(container.textContent).toBe('')
    rerender(
      <EffectiveChip
        knob={knob({
          stored: { present: false },
          effective: 'n/a — no resolver registered',
        })}
      />
    )
    expect(container.textContent).not.toMatch(/\b0\b/)
    rerender(<EffectiveChip knob={knob({ effective: null })} />)
    expect(container.textContent).toContain('eff n/a')
    expect(container.textContent).not.toMatch(/\b0\b/)
  })

  it('renders a suspension, a session scope and a lost saved value as sent', () => {
    const { rerender } = render(
      <EffectiveChip
        knob={knob({
          path: 'ai_config.risk_control.breakeven_enabled',
          stored: { present: true, value: true },
          effective: false,
          origin: 'suspended (EXIT_MECHS_SUSPENDED) — saved true not used',
          scope: 'process env',
        })}
      />
    )
    expect(screen.getByTestId('effective-chip')).toHaveTextContent(
      'eff off · suspended (EXIT_MECHS_SUSPENDED) — saved true not used · process env'
    )
    rerender(
      <EffectiveChip
        knob={knob({
          path: 'day_plan.sessions.plan_mode',
          effective: 'strict',
          origin: 'session override',
          scope: 'session:NY',
        })}
      />
    )
    expect(screen.getByTestId('effective-chip')).toHaveTextContent(
      'eff strict · session override · session:NY'
    )
  })

  // W1 (CTO R2): an explicit 0 the Studio never confirmed refuses its trader at
  // load — its chip must not wear the owner-green of a confirmed setting.
  it('warns on an UNCONFIRMED explicit 0 and stays green when a Studio save confirmed it', () => {
    const { rerender } = render(
      <EffectiveChip
        knob={knob({
          stored: { present: true, value: 0 },
          effective: 0,
          origin: 'saved value — explicit 0 UNCONFIRMED — re-save in Studio',
        })}
      />
    )
    const chip = screen.getByTestId('effective-chip')
    expect(chip).toHaveTextContent('explicit 0 UNCONFIRMED — re-save in Studio')
    expect(chip.className).toContain('text-amber-400')
    expect(chip.className).not.toContain('text-emerald-400')
    rerender(
      <EffectiveChip
        knob={knob({
          stored: { present: true, value: 0 },
          effective: 0,
          origin:
            'saved value — OFF — confirmed by Studio save 2026-09-23 10:04 CT',
        })}
      />
    )
    expect(screen.getByTestId('effective-chip').className).toContain(
      'text-emerald-400'
    )
  })
})

describe('formatEffective', () => {
  it('formats without inventing values', () => {
    expect(formatEffective(undefined)).toBe('n/a')
    expect(formatEffective(null)).toBe('n/a')
    expect(formatEffective(0)).toBe('0') // a RESOLVED 0 is a real value
    expect(formatEffective(true)).toBe('on')
    expect(formatEffective('')).toBe('none')
    expect(formatEffective([])).toBe('none')
    expect(formatEffective(['NY', 'ASIA'])).toBe('NY, ASIA')
    expect(formatEffective({ fvg_entry: 'shadow', hold: 'live' })).toBe(
      'fvg_entry=shadow, hold=live'
    )
  })
})
