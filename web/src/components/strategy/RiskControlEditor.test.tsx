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
