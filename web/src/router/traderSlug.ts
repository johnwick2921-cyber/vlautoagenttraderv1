import type { TraderInfo } from '../types'

// A trader's URL slug is its full, immutable trader_id. Using the id (not
// name + short prefix) makes selection collision-proof and rename-proof: every
// trader_id is unique, whereas trader_id.slice(0, 4) is the SHARED user prefix
// ("8d5c…") for every trader a user owns — so the old "<name>-<prefix>" slug
// resolved by trader_name ALONE and silently collided on duplicate or renamed
// names, snapping the view to whichever match came first (traders[0]).
export function getTraderSlug(trader: TraderInfo): string {
  return trader.trader_id
}

export function findTraderBySlug(
  slug: string,
  traderList: TraderInfo[]
): TraderInfo | undefined {
  // Primary: exact trader_id match (current, unambiguous slugs).
  const byId = traderList.find((trader) => trader.trader_id === slug)
  if (byId) {
    return byId
  }

  // Legacy fallback for old "<name>-<idPrefix>" bookmarks so existing links
  // still resolve. Best-effort (matches name + id prefix); may be ambiguous for
  // duplicate names — which is exactly why new slugs carry the full id.
  const lastDashIndex = slug.lastIndexOf('-')
  if (lastDashIndex === -1) {
    return traderList.find((trader) => trader.trader_name === slug)
  }

  const name = slug.slice(0, lastDashIndex)
  const idPrefix = slug.slice(lastDashIndex + 1)
  return traderList.find(
    (trader) =>
      trader.trader_name === name && trader.trader_id.startsWith(idPrefix)
  )
}
