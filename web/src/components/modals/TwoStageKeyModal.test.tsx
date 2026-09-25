import { render, fireEvent, screen, act } from '@testing-library/react'
import { it, expect, vi } from 'vitest'
import { TwoStageKeyModal } from './TwoStageKeyModal'
vi.mock('sonner', () => ({
  toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn() }),
}))
vi.mock('../common/WebCryptoEnvironmentCheck', () => ({
  WebCryptoEnvironmentCheck: () => null,
}))
it('clears key parts and ignores the delayed next stage after closing/reopening', async () => {
  vi.useFakeTimers()
  const props = {
    isOpen: true,
    language: 'en' as const,
    onCancel: vi.fn(),
    onComplete: vi.fn(),
  }
  const { rerender } = render(<TwoStageKeyModal {...props} />)
  fireEvent.change(screen.getByPlaceholderText('0x1234...'), {
    target: { value: 'a'.repeat(58) },
  })
  const buttons = screen.getAllByRole('button')
  const next = buttons.find((b) => /next|continue/i.test(b.textContent || ''))!
  fireEvent.click(next)
  rerender(<TwoStageKeyModal {...props} isOpen={false} />)
  rerender(<TwoStageKeyModal {...props} />)
  await act(async () => {
    vi.advanceTimersByTime(2500)
  })
  expect(screen.getByPlaceholderText('0x1234...')).toHaveValue('')
  expect(screen.queryByPlaceholderText('...5678')).toBeNull()
  vi.useRealTimers()
})
