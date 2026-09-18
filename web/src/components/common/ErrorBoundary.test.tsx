import { render, screen, cleanup } from '@testing-library/react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { ErrorBoundary } from './ErrorBoundary'

function Bomb(): never {
  throw new Error('kaboom')
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('ErrorBoundary', () => {
  it('renders children when nothing throws', () => {
    render(
      <ErrorBoundary>
        <div data-testid="child">ok</div>
      </ErrorBoundary>
    )
    expect(screen.getByTestId('child')).toBeInTheDocument()
    expect(screen.queryByTestId('root-error-boundary')).toBeNull()
  })

  it('catches a throwing child and shows a one-line error with a reload button', () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    // jsdom re-reports the error React rethrows on the window; keep the run quiet.
    const quiet = (e: ErrorEvent) => e.preventDefault()
    window.addEventListener('error', quiet)
    render(
      <ErrorBoundary>
        <Bomb />
      </ErrorBoundary>
    )
    const box = screen.getByTestId('root-error-boundary')
    expect(box).toHaveTextContent('kaboom')
    expect(screen.getByRole('button', { name: 'Reload' })).toBeInTheDocument()
    window.removeEventListener('error', quiet)
  })
})
