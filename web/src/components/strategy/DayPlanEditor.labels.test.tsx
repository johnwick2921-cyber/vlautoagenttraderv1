// W1 (d)(e) — two switches whose labels did not say what they do.
// (d) One setup is REJECT-only in code (kernel/one_setup.go OneSetupPlay =
//     "reject"); its label said "the single best level".
// (e) The Picture HTF switch borrowed the master's key and read
//     "Enable Day Plan", so the page showed that label twice.
// Both render the real editor; no behaviour changes.

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { DayPlanEditor } from './DayPlanEditor'
import type { DayPlanConfig } from '../../types/strategy'

function rowOf(label: HTMLElement): HTMLElement {
  // FieldRow: the label and its control share one row element.
  let el: HTMLElement | null = label
  while (el && !el.querySelector('[role="switch"]')) el = el.parentElement
  if (!el) throw new Error('label has no switch in its row')
  return el
}

describe('DayPlanEditor labels (W1 d/e)', () => {
  it('(d) the One setup switch says it arms only the reject (fade) level', () => {
    render(
      <DayPlanEditor
        config={{ plan_enabled: true }}
        onChange={vi.fn()}
        language="en"
      />
    )
    const label = screen.getByText(
      'One setup — arm only the single best reject (fade) level'
    )
    expect(
      rowOf(label).querySelector('[data-testid="one-setup-toggle"]')
    ).not.toBeNull()
  })

  it('(e) the Picture switch has its own label; "Enable Day Plan" appears once', () => {
    const onChange = vi.fn()
    render(
      <DayPlanEditor
        config={{ plan_enabled: true }}
        onChange={onChange}
        language="en"
      />
    )
    expect(screen.getAllByText('Enable Day Plan')).toHaveLength(1)
    const label = screen.getByText('Include Picture HTF setups')
    const sw = rowOf(label).querySelector(
      '[data-testid="picture-htf-toggle"]'
    ) as HTMLElement
    expect(sw).not.toBeNull()
    fireEvent.click(sw)
    const next = onChange.mock.calls[0][0] as DayPlanConfig
    expect(next.picture_htf?.enabled).toBe(true)
  })

  it('(e) the Picture switch sits under the Day Plan master: disabled when the master is off', () => {
    render(
      <DayPlanEditor
        config={{ plan_enabled: false }}
        onChange={vi.fn()}
        language="en"
      />
    )
    expect(screen.getByTestId('picture-htf-toggle')).toBeDisabled()
  })

  it('(d)(e) zh and id carry their own strings', () => {
    const { unmount } = render(
      <DayPlanEditor
        config={{ plan_enabled: true }}
        onChange={vi.fn()}
        language="zh"
      />
    )
    expect(
      screen.getByText('单一设置 — 仅对最佳拒绝（fade）价位挂单')
    ).toBeTruthy()
    expect(screen.getByText('纳入双图（Picture HTF）设置')).toBeTruthy()
    unmount()
    render(
      <DayPlanEditor
        config={{ plan_enabled: true }}
        onChange={vi.fn()}
        language="id"
      />
    )
    expect(
      screen.getByText('One setup — pasang hanya level reject (fade) terbaik')
    ).toBeTruthy()
    expect(screen.getByText('Sertakan setup Picture HTF')).toBeTruthy()
  })
})
