// P4.5 — DayPlanEditor tests: master switch materializes defaults, filters/mode
// write through onChange, and per-session override toggles set/clear pointer
// fields (⚪ inherit / 🔸 override).

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { DayPlanEditor } from './DayPlanEditor'
import type { DayPlanConfig } from '../../types/strategy'

describe('DayPlanEditor', () => {
  it('enabling the master switch materializes the default block', () => {
    const onChange = vi.fn()
    render(
      <DayPlanEditor config={undefined} onChange={onChange} language="en" />
    )
    // the master toggle is the first switch
    fireEvent.click(screen.getAllByRole('switch')[0])
    expect(onChange).toHaveBeenCalledTimes(1)
    const next = onChange.mock.calls[0][0] as DayPlanConfig
    expect(next.plan_enabled).toBe(true)
    expect(next.max_levels).toBe(8) // spec default carried through
    expect(next.acceptance_rule).toBe('2x5m')
  })

  it('changing max levels writes through onChange', () => {
    const onChange = vi.fn()
    const cfg: DayPlanConfig = { plan_enabled: true, max_levels: 8 }
    render(<DayPlanEditor config={cfg} onChange={onChange} language="en" />)
    const input = screen.getByDisplayValue('8')
    fireEvent.change(input, { target: { value: '6' } })
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ max_levels: 6 })
    )
  })

  it('a per-session override toggle sets then clears the field', () => {
    const onChange = vi.fn()
    const cfg: DayPlanConfig = { plan_enabled: true }
    render(<DayPlanEditor config={cfg} onChange={onChange} language="en" />)
    // NY accordion is open by default; toggle the min-grade override on
    const minGradeBtn = screen.getByRole('button', { name: /Min grade/i })
    fireEvent.click(minGradeBtn)
    const call = onChange.mock.calls[0][0] as DayPlanConfig
    expect(call.sessions?.find((s) => s.session === 'NY')?.min_grade).toBe('B')
  })

  it('planner timeframes are an editable multiselect (toggle in/out)', () => {
    const onChange = vi.fn()
    const cfg: DayPlanConfig = {
      plan_enabled: true,
      planner_timeframes: ['D', '4h', '1h', '15m'],
    }
    render(<DayPlanEditor config={cfg} onChange={onChange} language="en" />)
    // toggling an OFF tf ('5m') adds it, preserving order
    const tf5 = screen.getByRole('switch', { name: '5m' })
    expect(tf5).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(tf5)
    let next = onChange.mock.calls[0][0] as DayPlanConfig
    expect(next.planner_timeframes).toEqual(['D', '4h', '1h', '15m', '5m'])
    // toggling an ON tf ('1h') removes it
    onChange.mockClear()
    fireEvent.click(screen.getByRole('switch', { name: '1h' }))
    next = onChange.mock.calls[0][0] as DayPlanConfig
    expect(next.planner_timeframes).toEqual(['D', '4h', '15m'])
  })

  it('the whole body is disabled when the plan is off', () => {
    const onChange = vi.fn()
    render(
      <DayPlanEditor
        config={{ plan_enabled: false }}
        onChange={onChange}
        language="en"
      />
    )
    // number fields exist but are disabled while off
    const levels = screen.getByDisplayValue('8') as HTMLInputElement
    expect(levels.disabled).toBe(true)
  })
})

// PART A — the reported bug: no enable toggle on ANY session row. These lock the
// fix: a toggle on every row, defaults matching the resolver, and a flip that
// writes the per-session `enable` override (the field the Go gates read).
describe('DayPlanEditor · session enable toggles', () => {
  it('renders an enable toggle on ALL THREE session rows', () => {
    render(
      <DayPlanEditor
        config={{ plan_enabled: true }}
        onChange={vi.fn()}
        language="en"
      />
    )
    for (const s of ['NY', 'ASIA', 'LONDON']) {
      const el = screen.getByTestId(`session-enable-${s}`)
      expect(el).toBeTruthy()
      expect(el.getAttribute('role')).toBe('switch')
      // accessible name (a bare role=switch announces as just "switch")
      expect(el.getAttribute('aria-label')).toContain(s)
    }
  })

  it('defaults mirror the Go resolver: NY on, ASIA/LONDON off', () => {
    render(
      <DayPlanEditor
        config={{ plan_enabled: true }}
        onChange={vi.fn()}
        language="en"
      />
    )
    expect(
      screen.getByTestId('session-enable-NY').getAttribute('aria-checked')
    ).toBe('true')
    expect(
      screen.getByTestId('session-enable-ASIA').getAttribute('aria-checked')
    ).toBe('false')
    expect(
      screen.getByTestId('session-enable-LONDON').getAttribute('aria-checked')
    ).toBe('false')
  })

  it('flipping ASIA on writes sessions[].enable = true (what the gates read)', () => {
    const onChange = vi.fn()
    render(
      <DayPlanEditor
        config={{ plan_enabled: true }}
        onChange={onChange}
        language="en"
      />
    )
    fireEvent.click(screen.getByTestId('session-enable-ASIA'))
    const next = onChange.mock.calls[0][0] as DayPlanConfig
    const asia = next.sessions?.find((x) => x.session === 'ASIA')
    expect(asia?.enable).toBe(true)
  })

  it('flipping NY off writes an explicit false (not just an absent field)', () => {
    const onChange = vi.fn()
    render(
      <DayPlanEditor
        config={{ plan_enabled: true }}
        onChange={onChange}
        language="en"
      />
    )
    fireEvent.click(screen.getByTestId('session-enable-NY'))
    const next = onChange.mock.calls[0][0] as DayPlanConfig
    const ny = next.sessions?.find((x) => x.session === 'NY')
    expect(ny?.enable).toBe(false) // explicit off — an absent field would inherit ON
  })

  it('an explicit override shows the 🔸 chip; inherit shows none', () => {
    const { rerender } = render(
      <DayPlanEditor
        config={{ plan_enabled: true }}
        onChange={vi.fn()}
        language="en"
      />
    )
    expect(screen.queryByTestId('session-enable-chip-ASIA')).toBeNull()
    rerender(
      <DayPlanEditor
        config={{
          plan_enabled: true,
          sessions: [{ session: 'ASIA', enable: true }],
        }}
        onChange={vi.fn()}
        language="en"
      />
    )
    expect(screen.getByTestId('session-enable-chip-ASIA')).toBeTruthy()
  })
})
