import { Component, type ReactNode } from 'react'

/**
 * Stuck-send fix (final-bundle Phase 5, 2026-08-19): a render crash inside the
 * Ask-Planner thread (malformed reply, unguarded undefined — the June family)
 * used to white-screen the panel and require F5. This boundary contains the
 * crash to the thread area; the input row below it stays alive so the owner
 * can keep asking.
 */
export class PlanErrorBoundary extends Component<
  { children: ReactNode },
  { error: string | null }
> {
  state = { error: null as string | null }

  static getDerivedStateFromError(e: unknown) {
    return { error: e instanceof Error ? e.message : String(e) }
  }

  render() {
    if (this.state.error) {
      return (
        <div
          className="text-[11px] px-3 py-4 text-center"
          style={{ color: 'var(--vl-muted)' }}
          data-testid="plan-thread-crashed"
        >
          ⚠ This thread failed to render ({this.state.error}). Your questions
          are saved — reopen the panel to reload, no page refresh needed.
          <button
            className="ml-2 underline"
            onClick={() => this.setState({ error: null })}
          >
            retry
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
