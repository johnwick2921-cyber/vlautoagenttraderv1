import { fireEvent, render, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import { RiskControlEditor } from './RiskControlEditor'
import type { RiskControlConfig } from '../../types'

it('keeps the owner daily-loss control without requiring a per-trade cap', () => {
  const onChange = vi.fn()
  const config = {
    guardrails_enabled: true,
    daily_loss_enabled: true,
    daily_loss_limit_usd: 450,
  } as RiskControlConfig
  render(
    <RiskControlEditor
      config={config}
      onChange={onChange}
      language="en"
      isFutures
    />
  )
  expect(
    screen.queryByRole('spinbutton', { name: 'MNQ per-trade risk cap' })
  ).not.toBeInTheDocument()
  fireEvent.change(screen.getByDisplayValue('450'), {
    target: { value: '400' },
  })
  expect(onChange).toHaveBeenCalledWith({
    ...config,
    daily_loss_limit_usd: 400,
  })
})

// ── W1 (settings truth, 2026-09-23) — the breaker row is PRESENCE-AWARE ─────
// absent/null = inherit (the runtime enforces env BREAKER_HALT_N, else 8 — ON),
// an explicit 0 = OFF, N = N. The old row showed an absent breaker as OFF while
// the runtime enforced 8, and ON wrote a literal 2.

function renderBreaker(config: RiskControlConfig) {
  const onChange = vi.fn()
  render(
    <RiskControlEditor
      config={config}
      onChange={onChange}
      language="en"
      isFutures
    />
  )
  return {
    onChange,
    toggle: screen.getByTestId('breaker-toggle'),
    input: screen.getByTestId('breaker-halt-input') as HTMLInputElement,
  }
}

it('W1: an ABSENT breaker reads ON (inherit) with an empty box, never a fake number', () => {
  const { toggle, input } = renderBreaker({} as RiskControlConfig)
  expect(toggle.getAttribute('aria-checked')).toBe('true')
  expect(input.value).toBe('')
  expect(input.placeholder).toBe('inherit')
})

it('W1: OFF writes an explicit 0', () => {
  const config = {} as RiskControlConfig
  const { onChange, toggle } = renderBreaker(config)
  fireEvent.click(toggle)
  expect(onChange).toHaveBeenCalledWith({ ...config, consecutive_loss_halt: 0 })
})

it('W1: a stored 0 reads OFF, and ON writes null (inherit) — never a literal 2', () => {
  const config = { consecutive_loss_halt: 0 } as RiskControlConfig
  const { onChange, toggle } = renderBreaker(config)
  expect(toggle.getAttribute('aria-checked')).toBe('false')
  fireEvent.click(toggle)
  const next = onChange.mock.calls[0][0] as RiskControlConfig
  expect(next.consecutive_loss_halt).toBeNull()
  // The wire shape the PUT sends: the key is PRESENT as null — the merge keeps
  // absent keys, so an omitted key could never clear the stored 0.
  expect(JSON.stringify(next)).toContain('"consecutive_loss_halt":null')
})

it('W1: clearing the number box writes null (inherit), never 0', () => {
  const config = { consecutive_loss_halt: 5 } as RiskControlConfig
  const { onChange, input } = renderBreaker(config)
  expect(input.value).toBe('5')
  fireEvent.change(input, { target: { value: '' } })
  expect(onChange).toHaveBeenCalledWith({
    ...config,
    consecutive_loss_halt: null,
  })
})

it('W1: typing a number writes it; typing 0 writes OFF', () => {
  const config = { consecutive_loss_halt: 5 } as RiskControlConfig
  const { onChange, input } = renderBreaker(config)
  fireEvent.change(input, { target: { value: '3' } })
  expect(onChange).toHaveBeenLastCalledWith({
    ...config,
    consecutive_loss_halt: 3,
  })
  fireEvent.change(input, { target: { value: '0' } })
  expect(onChange).toHaveBeenLastCalledWith({
    ...config,
    consecutive_loss_halt: 0,
  })
})
