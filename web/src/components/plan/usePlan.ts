// P4 — Day-Plan SWR hooks. The PlanCard self-fetches (matches DecisionAudit /
// GridRiskPanel): pass a trader_id, get back the plan + alerts with polling. Keys
// are null when there's no trader so SWR skips the fetch entirely.

import useSWR from 'swr'
import { api } from '../../lib/api'
import type { PlanToday, PlanAlertsResponse } from '../../lib/api/plan'

const PLAN_REFRESH_MS = 15_000
const ALERTS_REFRESH_MS = 20_000

export function usePlanToday(traderId?: string, symbol?: string) {
  const key = traderId
    ? `plan-today-${traderId}${symbol ? `-${symbol}` : ''}`
    : null
  const { data, error, isLoading, mutate } = useSWR<PlanToday | null>(
    key,
    () => api.getPlanToday(traderId as string, symbol),
    {
      refreshInterval: PLAN_REFRESH_MS,
      revalidateOnFocus: false,
      dedupingInterval: 5000,
    }
  )
  return { plan: data ?? null, error: error as unknown, isLoading, mutate }
}

export function usePlanAlerts(traderId?: string) {
  const key = traderId ? `plan-alerts-${traderId}` : null
  const { data, mutate } = useSWR<PlanAlertsResponse>(
    key,
    () => api.getPlanAlerts(traderId as string),
    { refreshInterval: ALERTS_REFRESH_MS, revalidateOnFocus: false }
  )
  return { alerts: data?.alerts ?? [], unacked: data?.unacked ?? 0, mutate }
}
