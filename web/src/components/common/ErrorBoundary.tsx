import { Component, type ReactNode } from 'react'

/**
 * Root error boundary (W-CGOFREE-SQLITE-UPSTREAM, 2026-09-17). React unmounts
 * the whole tree on an uncaught render error, which the owner sees as a white
 * screen with nothing to click. This boundary keeps a one-line message and a
 * reload button on screen instead. Panel-local boundaries (PlanErrorBoundary)
 * still catch first; this is the last resort under <App />.
 */
export class ErrorBoundary extends Component<
  { children: ReactNode },
  { error: string | null }
> {
  state = { error: null as string | null }

  static getDerivedStateFromError(e: unknown) {
    return { error: e instanceof Error ? e.message : String(e) }
  }

  componentDidCatch(e: unknown) {
    console.error('[ErrorBoundary] render crash:', e)
  }

  render() {
    if (this.state.error !== null) {
      return (
        <div
          role="alert"
          data-testid="root-error-boundary"
          className="min-h-screen flex items-center justify-center p-6 text-sm"
          style={{ background: '#0b0e11', color: '#EAECEF' }}
        >
          <div className="text-center">
            <div className="mb-3">
              Something went wrong while rendering this page: {this.state.error}
            </div>
            <button
              type="button"
              className="px-3 py-1 rounded"
              style={{ background: '#F0B90B', color: '#0b0e11' }}
              onClick={() => window.location.reload()}
            >
              Reload
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
