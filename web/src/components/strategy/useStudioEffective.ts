// W1 (g) — the Strategy Studio's effective-values plumbing.
//
// FOUR reads of GET /api/strategies/:id/effective for the selected strategy:
// one WITHOUT ?session= (the strategy-level rows — a strategy row must never be
// answered with one session's override) and one per session NY / ASIA / LONDON
// (the per-session `day_plan.sessions.<leaf>` rows only resolve with a session
// named). The server reads the SAVED strategy, so unsaved edits do not move a
// chip; the page calls refresh() after a successful save.

import { useMemo } from 'react'
import { useStrategyEffective } from './useStrategyEffective'
import {
  studioEffective,
  type StudioEffective,
} from '../../lib/api/strategyEffective'
import type { SessionName } from '../plan/sessionConfig'

/** The sessions the Studio's per-session accordion edits (DayPlanEditor). */
export const STUDIO_EFFECTIVE_SESSIONS: readonly SessionName[] = [
  'NY',
  'ASIA',
  'LONDON',
]

export function useStudioEffective(strategyId?: string | null): {
  effective: StudioEffective
  refresh: () => void
} {
  // Four explicit calls (hooks may not sit in a loop); the order matches
  // STUDIO_EFFECTIVE_SESSIONS.
  const base = useStrategyEffective(strategyId)
  const ny = useStrategyEffective(strategyId, 'NY')
  const asia = useStrategyEffective(strategyId, 'ASIA')
  const london = useStrategyEffective(strategyId, 'LONDON')

  const effective = useMemo(
    () =>
      studioEffective(base.data, {
        NY: ny.data,
        ASIA: asia.data,
        LONDON: london.data,
      }),
    [base.data, ny.data, asia.data, london.data]
  )

  const refresh = () => {
    base.mutate()
    ny.mutate()
    asia.mutate()
    london.mutate()
  }

  return { effective, refresh }
}
