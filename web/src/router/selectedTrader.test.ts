import { describe, it, expect } from 'vitest'
import type { TraderInfo } from '../types'
import { resolveSelectedTrader, byStableTraderOrder } from './selectedTrader'

const mk = (id: string, name: string): TraderInfo => ({
  trader_id: id,
  trader_name: name,
  ai_model: 'deepseek-v4-pro',
  is_running: true,
})

// Two traders sharing the real user id-prefix. In the traders API they arrive
// creation-DESC, so index 0 is the LAST-CREATED — the old snap-back default.
const HOANG = mk('8d5c8af5_hoang_1781246265', 'hoang')
const FIFTEEN = mk('8d5c8af5_15m_1786221895', '15m')
// API order: newest first → [FIFTEEN, HOANG] if 15m was created last.
const TRADERS = [FIFTEEN, HOANG]

describe('resolveSelectedTrader — priority order', () => {
  it('URL wins over everything', () => {
    const r = resolveSelectedTrader(
      TRADERS,
      HOANG.trader_id, // url
      FIFTEEN.trader_id, // memory
      FIFTEEN.trader_id // storage
    )
    expect(r.source).toBe('url')
    expect(r.trader).toBe(HOANG)
    expect(r.clearStored).toBe(false)
  })

  it('in-memory wins when no URL', () => {
    const r = resolveSelectedTrader(
      TRADERS,
      undefined,
      HOANG.trader_id, // memory
      FIFTEEN.trader_id // storage
    )
    expect(r.source).toBe('memory')
    expect(r.trader).toBe(HOANG)
  })

  it('localStorage restores the selection on a fresh load (no url, no memory)', () => {
    const r = resolveSelectedTrader(
      TRADERS,
      undefined,
      undefined,
      HOANG.trader_id // storage
    )
    expect(r.source).toBe('storage')
    expect(r.trader).toBe(HOANG)
  })

  it('fallback is the STABLE-sorted first, NOT creation-order traders[0]', () => {
    // Nothing selected anywhere → must NOT snap to the last-created (FIFTEEN at
    // index 0); stable order is by name → "15m" < "hoang", so 15m; if names were
    // reversed the id tiebreak still makes it deterministic (never index-based).
    const r = resolveSelectedTrader(TRADERS, undefined, undefined, undefined)
    expect(r.source).toBe('fallback')
    expect(r.trader).toBe([...TRADERS].sort(byStableTraderOrder)[0])
    expect(r.clearStored).toBe(false)
  })
})

describe('resolveSelectedTrader — deleted-id fallback', () => {
  it('a stored id that no longer exists → fallback + clearStored', () => {
    const r = resolveSelectedTrader(
      TRADERS,
      undefined,
      undefined,
      'deleted_trader_id_gone'
    )
    expect(r.source).toBe('fallback')
    expect(r.clearStored).toBe(true) // caller clears the stale localStorage entry
    expect(r.trader).toBeDefined()
  })

  it('a stale URL slug falls through to the next source (never hijacks)', () => {
    const r = resolveSelectedTrader(
      TRADERS,
      'no-such-slug',
      undefined,
      HOANG.trader_id
    )
    expect(r.source).toBe('storage') // storage still resolves
    expect(r.trader).toBe(HOANG)
  })

  it('empty / missing trader list → none', () => {
    expect(resolveSelectedTrader([], 'x', 'y', 'z').source).toBe('none')
    expect(
      resolveSelectedTrader(undefined, 'x', 'y', 'z').trader
    ).toBeUndefined()
  })
})

describe('byStableTraderOrder', () => {
  it('is deterministic and independent of array order', () => {
    const a = [FIFTEEN, HOANG].sort(byStableTraderOrder)
    const b = [HOANG, FIFTEEN].sort(byStableTraderOrder)
    expect(a.map((t) => t.trader_id)).toEqual(b.map((t) => t.trader_id))
  })
})
