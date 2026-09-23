// W-EXEC-TRUTH W1 (g) — GET /api/strategies/:id/effective (api/strategy_effective.go).
//
// Every settings row with its EFFECTIVE value, ORIGIN and SCOPE, resolved by the
// server from the STORED strategy row with the production resolvers. The UI
// renders what arrives; it never derives an effective value, an origin or a
// default of its own.

import { httpClient } from '../httpClient'
import { API_BASE } from './helpers'

/** What the stored row carries at the path. `value` is ABSENT when nothing is
 *  stored — absent and a stored 0 / [] are different answers. */
export interface EffectiveStored {
  present: boolean
  value?: unknown
}

/** One settings row (mirrors trader.EffectiveKnob). `effective` is the value
 *  the runtime uses — or a string starting "n/a" when it is not known, or
 *  "redacted" for a secret. */
export interface EffectiveKnob {
  path: string
  status: string
  ui_label: string
  stored: EffectiveStored
  effective: unknown
  origin: string
  scope: string
  resolver: string
  resolved: boolean
}

export interface EffectiveCoverage {
  resolved: number
  total: number
  unresolved: string[]
  not_enumerated: string[]
}

export interface StrategyEffectiveResponse {
  strategy_id: string
  session: string | null
  venue: string | null
  settings: EffectiveKnob[]
  coverage: EffectiveCoverage
}

export type EffectiveByPath = Record<string, EffectiveKnob>

export const strategyEffectiveApi = {
  /** Silent GET; any failure resolves to null so a row renders no chip rather
   *  than a fabricated value. */
  async getStrategyEffective(
    strategyId: string,
    session?: string,
    venue?: string
  ): Promise<StrategyEffectiveResponse | null> {
    const params = new URLSearchParams()
    if (session) params.set('session', session)
    if (venue) params.set('venue', venue)
    const qs = params.toString()
    try {
      const res = await httpClient.request<StrategyEffectiveResponse>(
        `${API_BASE}/strategies/${encodeURIComponent(strategyId)}/effective${qs ? `?${qs}` : ''}`,
        { silent: true }
      )
      return res.success && res.data ? res.data : null
    } catch {
      return null
    }
  },
}

/** path → row, for the editors to look a row up by its schema path. */
export function effectiveByPath(
  resp: StrategyEffectiveResponse | null | undefined
): EffectiveByPath {
  const out: EffectiveByPath = {}
  for (const k of resp?.settings ?? []) out[k.path] = k
  return out
}
