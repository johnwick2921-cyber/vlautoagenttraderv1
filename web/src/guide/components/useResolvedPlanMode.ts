// GUIDE CONTENT LAW — a mock that states a mode is making a CLAIM.
//
// MockPlanCard printed "Scenarios (advisory)" as a static string while the
// bound strategy resolved `strict`. The guide was telling the operator the plan
// was advisory while the engine ran strict, and GUIDE_BUILT_REV could not catch
// it: the rev proves when the guide was BUILT, never that its prose still
// matches behaviour.
//
// So the mock either READS the mode or says plainly that it is not reading one.
// It never asserts a mode it did not resolve.

import { useEffect, useState } from 'react'

/** The label to render when no mode could be resolved — never a guessed mode. */
export const MODE_NOT_READ = 'example — not a reading'

interface ResolvedField {
  path: string
  resolved: string
}

export function useResolvedPlanMode(): string {
  const [mode, setMode] = useState<string>(MODE_NOT_READ)

  useEffect(() => {
    let live = true
    const token = localStorage.getItem('auth_token') ?? ''
    const auth: Record<string, string> = token
      ? { Authorization: `Bearer ${token}` }
      : {}

    // The guide has no trader context of its own, so it asks which traders
    // exist and resolves against the first. If any step cannot be completed the
    // label stays MODE_NOT_READ — an unread mode is never rendered as a mode.
    ;(async () => {
      try {
        const tr = await fetch('/api/traders').then((r) => r.json())
        const id = Array.isArray(tr) ? tr[0]?.id : tr?.traders?.[0]?.id
        if (!id) return
        const cfg = await fetch(
          `/api/config/resolved?trader_id=${encodeURIComponent(id)}`,
          { headers: auth }
        ).then((r) => r.json())
        const fields: ResolvedField[] = cfg?.resolved ?? []
        const pm = fields.find((f) => f.path === 'day_plan.plan_mode')
        if (live && pm?.resolved) setMode(pm.resolved)
      } catch {
        // leave MODE_NOT_READ — silence is not a mode
      }
    })()
    return () => {
      live = false
    }
  }, [])

  return mode
}
