// W1 (g) — the Studio's effective-values hook. Keyed by strategy + session so a
// session switch refetches instead of reusing another session's answer; null
// key (no strategy) skips the fetch entirely.

import useSWR from 'swr'
import {
  effectiveByPath,
  strategyEffectiveApi,
  type EffectiveByPath,
  type StrategyEffectiveResponse,
} from '../../lib/api/strategyEffective'

export function useStrategyEffective(
  strategyId?: string | null,
  session?: string
): {
  data: StrategyEffectiveResponse | null
  byPath: EffectiveByPath
  isLoading: boolean
  mutate: () => void
} {
  const key = strategyId
    ? `strategy-effective-${strategyId}${session ? `-${session}` : ''}`
    : null
  const { data, isLoading, mutate } = useSWR<StrategyEffectiveResponse | null>(
    key,
    () =>
      strategyEffectiveApi.getStrategyEffective(strategyId as string, session),
    { revalidateOnFocus: false, dedupingInterval: 5000 }
  )
  return {
    data: data ?? null,
    byPath: effectiveByPath(data),
    isLoading,
    mutate: () => {
      void mutate()
    },
  }
}
