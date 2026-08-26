import type { TraderInfo } from '../types'
import { findTraderBySlug } from './traderSlug'

// Trader selection persistence + resolution (the "View must stick" fix).
//
// Selection was URL-only (?trader=) plus in-memory state, and the first-run /
// deleted-selection default was traders[0]. Because the traders API returns
// creation-DESC, traders[0] is the LAST-CREATED trader — so any fresh load
// WITHOUT the ?trader= param (the header "Dashboard" nav to a bare /dashboard, a
// new tab, a bookmark, an F5 after the param was dropped) snapped the view back to
// the newest trader. This module makes the selection first-class: it survives new
// tabs / refresh / bookmarks via localStorage, and the fallback is a STABLE sort,
// never creation-order.

const STORAGE_KEY = 'vl.selectedTraderId'

export function loadStoredTraderId(): string | undefined {
  try {
    return window.localStorage.getItem(STORAGE_KEY) || undefined
  } catch {
    return undefined // storage unavailable (private mode / SSR) → no-op
  }
}

export function saveStoredTraderId(id: string): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, id)
  } catch {
    /* ignore */
  }
}

export function clearStoredTraderId(): void {
  try {
    window.localStorage.removeItem(STORAGE_KEY)
  } catch {
    /* ignore */
  }
}

// byStableTraderOrder is a deterministic order (name, then the immutable id as a
// tiebreak) — independent of creation order, so the fallback trader is stable
// across reloads instead of "whichever was created last".
export function byStableTraderOrder(a: TraderInfo, b: TraderInfo): number {
  const byName = (a.trader_name || '').localeCompare(b.trader_name || '')
  if (byName !== 0) {
    return byName
  }
  return a.trader_id.localeCompare(b.trader_id)
}

export type TraderSelectionSource =
  | 'url'
  | 'memory'
  | 'storage'
  | 'fallback'
  | 'none'

export interface TraderResolution {
  trader: TraderInfo | undefined
  source: TraderSelectionSource
  // true when a persisted id no longer resolves (the trader was deleted) — the
  // caller should clear localStorage so the stale id can't keep losing.
  clearStored: boolean
}

// resolveSelectedTrader is the single, pure selection authority. Priority:
//   1. url    — an explicit ?trader= (a View click / bookmark) always wins.
//   2. memory — the current in-session selection (survives the header nav dropping
//               ?trader= while the page stays mounted).
//   3. storage— the persisted selection (survives a full reload / new tab).
//   4. fallback — a STABLE-sorted first trader (NOT creation-order traders[0]).
// A stored id that no longer exists is reported via clearStored and skipped.
export function resolveSelectedTrader(
  traders: TraderInfo[] | undefined,
  urlSlug: string | undefined,
  currentId: string | undefined,
  storedId: string | undefined
): TraderResolution {
  if (!traders || traders.length === 0) {
    return { trader: undefined, source: 'none', clearStored: false }
  }

  if (urlSlug) {
    const fromUrl = findTraderBySlug(urlSlug, traders)
    if (fromUrl) {
      return { trader: fromUrl, source: 'url', clearStored: false }
    }
  }

  if (currentId) {
    const fromMemory = traders.find((t) => t.trader_id === currentId)
    if (fromMemory) {
      return { trader: fromMemory, source: 'memory', clearStored: false }
    }
  }

  if (storedId) {
    const fromStorage = traders.find((t) => t.trader_id === storedId)
    if (fromStorage) {
      return { trader: fromStorage, source: 'storage', clearStored: false }
    }
    // stored id resolves to nothing → it was deleted; clear it below.
  }

  const first = [...traders].sort(byStableTraderOrder)[0]
  return { trader: first, source: 'fallback', clearStored: Boolean(storedId) }
}
