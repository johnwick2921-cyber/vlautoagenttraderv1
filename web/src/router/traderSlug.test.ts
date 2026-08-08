import { describe, it, expect } from 'vitest'
import type { TraderInfo } from '../types'
import { getTraderSlug, findTraderBySlug } from './traderSlug'

// Regression coverage for the "trader view selection must stick" fix.
// Root cause was a fragile slug: getTraderSlug used trader_id.slice(0, 4),
// which is the SHARED user prefix for every trader a user owns — so resolution
// fell back to trader_name alone and silently collided, and an unresolvable
// slug snapped the view to traders[0]. The fix makes the slug the full,
// unique trader_id and never silently falls back.

const mk = (id: string, name: string, running = true): TraderInfo => ({
  trader_id: id,
  trader_name: name,
  ai_model: 'deepseek-v4-pro',
  is_running: running,
})

// The owner's real data: two traders that SHARE the id prefix (8d5c8af5_…).
const FIFTEEN = mk(
  '8d5c8af5_396db319-d3fe-4d63-9b97-516ff0008f53_deepseek_0751c0b6_1786221895',
  '15m ' // note the trailing space — real data
)
const HOANG = mk(
  '8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265',
  'hoang'
)
const TRADERS = [FIFTEEN, HOANG]

describe('getTraderSlug', () => {
  it('is the full trader_id — unique and rename-proof', () => {
    expect(getTraderSlug(HOANG)).toBe(HOANG.trader_id)
    expect(getTraderSlug(FIFTEEN)).toBe(FIFTEEN.trader_id)
  })

  it('does not collapse to a shared prefix for traders under one user', () => {
    // The old slug was `${name}-${id.slice(0,4)}` → both "…-8d5c". The new
    // slugs must differ so the two traders never map to the same URL.
    expect(getTraderSlug(HOANG)).not.toBe(getTraderSlug(FIFTEEN))
  })
})

describe('findTraderBySlug — id-based resolution', () => {
  it('resolves each trader to ITSELF via its slug (round-trip)', () => {
    expect(findTraderBySlug(getTraderSlug(HOANG), TRADERS)).toBe(HOANG)
    expect(findTraderBySlug(getTraderSlug(FIFTEEN), TRADERS)).toBe(FIFTEEN)
  })

  it('resolves the non-default trader (hoang) — never snaps to traders[0]', () => {
    // The reported symptom: selecting the non-default trader returned to the
    // running traders[0]. With the id slug, hoang resolves to hoang.
    const resolved = findTraderBySlug(getTraderSlug(HOANG), TRADERS)
    expect(resolved).toBe(HOANG)
    expect(resolved).not.toBe(FIFTEEN) // FIFTEEN is traders[0]
  })

  it('disambiguates DUPLICATE names by id (old scheme collided here)', () => {
    const dupA = mk('8d5c8af5_aaaa1111_deepseek_1', 'scalp')
    const dupB = mk('8d5c8af5_bbbb2222_deepseek_2', 'scalp')
    const list = [dupA, dupB]
    // Selecting the SECOND "scalp" must resolve to dupB, not the first match.
    expect(findTraderBySlug(getTraderSlug(dupB), list)).toBe(dupB)
    expect(findTraderBySlug(getTraderSlug(dupA), list)).toBe(dupA)
  })

  it('returns undefined for an unresolvable slug — NO silent traders[0] fallback', () => {
    // This is the core of the bounce fix: the resolver itself must not invent a
    // trader. The effect decides the default; the resolver stays honest.
    expect(findTraderBySlug('does-not-exist-9999', TRADERS)).toBeUndefined()
    expect(
      findTraderBySlug('8d5c8af5_deleted_trader_id', TRADERS)
    ).toBeUndefined()
  })
})

describe('findTraderBySlug — legacy "<name>-<idPrefix>" bookmarks', () => {
  it('still resolves an old-format slug when the name is unique', () => {
    expect(findTraderBySlug('hoang-8d5c', TRADERS)).toBe(HOANG)
  })

  it('preserves a trailing space in the legacy name segment', () => {
    // "15m " (trailing space) → legacy slug "15m -8d5c"; the space must survive
    // the lastIndexOf('-') split so trader_name === "15m " matches.
    expect(findTraderBySlug('15m -8d5c', TRADERS)).toBe(FIFTEEN)
  })

  it('does not resolve a legacy slug whose name no longer matches (renamed)', () => {
    // A stale bookmark for a renamed trader must NOT resolve — the effect then
    // keeps the current selection or defaults, honestly, instead of masquerading.
    expect(findTraderBySlug('oldname-8d5c', TRADERS)).toBeUndefined()
  })
})
