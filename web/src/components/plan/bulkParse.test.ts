// P5.2 / P5.7 — the bulk-add + blind-mark grammar parser. Locks that a marking
// line parses to {price[, range][, type], note} and that junk lines are skipped.

import { describe, it, expect } from 'vitest'
import { parseBulkLevels } from './bulkParse'

describe('parseBulkLevels', () => {
  it('parses price + type + note', () => {
    const [r] = parseBulkLevels('30246 D-zone strong 4h OB')
    expect(r.price).toBe(30246)
    expect(r.type).toBe('D-zone')
    expect(r.note).toBe('strong 4h OB')
  })

  it('parses a price range', () => {
    const [r] = parseBulkLevels('30150-30152 S-zone')
    expect(r.price).toBe(30150)
    expect(r.priceHi).toBe(30152)
    expect(r.type).toBe('S-zone')
  })

  it('price-only line has empty type + note', () => {
    const [r] = parseBulkLevels('30000')
    expect(r.price).toBe(30000)
    expect(r.type).toBeUndefined()
    expect(r.note).toBe('')
  })

  it('an unknown 2nd word becomes part of the note (not a type)', () => {
    const [r] = parseBulkLevels('30000 magnet please')
    expect(r.type).toBeUndefined()
    expect(r.note).toBe('magnet please')
  })

  it('skips blank + price-less lines, keeps valid ones', () => {
    const rows = parseBulkLevels('\n30100 Level\nnot a level\n  \n29950 RN gap')
    expect(rows).toHaveLength(2)
    expect(rows[0].price).toBe(30100)
    expect(rows[1].price).toBe(29950)
  })

  it('the blind-mark grammar (owner hour) parses', () => {
    const rows = parseBulkLevels(
      [
        '30246 D-zone strong 4h OB — tin zone nay',
        '30150-30152 S-zone',
        '30000 Level round',
        '29900 Liquidity sweep target',
      ].join('\n')
    )
    expect(rows).toHaveLength(4)
    expect(rows.map((r) => r.price)).toEqual([30246, 30150, 30000, 29900])
    expect(rows[3].type).toBe('Liquidity')
  })
})
