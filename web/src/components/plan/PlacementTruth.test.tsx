import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { ArmedChip } from './ScenarioList'

it('distinguishes sending from received acceptance and exposes rejection evidence', () => {
  const { rerender } = render(<ArmedChip arm={{ state: 'place_pending' }} />)
  expect(screen.getByTestId('armed-chip').textContent).toBe(
    '⏳ placement pending'
  )
  rerender(<ArmedChip arm={{ state: 'working' }} />)
  expect(screen.getByTestId('armed-chip').textContent).toBe('📌 working')
  rerender(
    <ArmedChip
      arm={{
        state: 'rejected',
        reason: 'reason unavailable (NT8 frame omitted reason)',
      }}
    />
  )
  expect(screen.getByTestId('armed-chip').textContent).toContain(
    'rejected · reason unavailable'
  )
  rerender(
    <ArmedChip
      arm={{ state: 'rejected', reason: 'stale signal age=715.3s (max 60s)' }}
    />
  )
  expect(screen.getByTestId('armed-chip').textContent).toContain(
    'stale signal age=715.3s (max 60s)'
  )
})
