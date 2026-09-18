// W-CHART-ZONE-WALL — the pure selection: seated levels always, identical
// bands merged, and only the N nearest zones unless "show all".
import { describe, it, expect } from 'vitest'
import {
  dedupeZones,
  nearestZones,
  selectChartOverlay,
  zoneDistance,
  DEFAULT_NEAREST_ZONES,
} from './chartOverlaySelect'
import type { OverlayLevel } from './LevelOverlayPrimitive'

const seated: OverlayLevel[] = Array.from({ length: 12 }, (_, i) => ({
  price: 29000 + i * 25,
  label: `L${i}`,
  grade: 'A',
}))

// 40 zones: 30 distinct bands spread ±1500 around 29500, plus 10 exact
// duplicates of the first ten (same kind·tf label, same lo, same hi) — the
// shape a structure block produces when a zone is seated on more than one TF
// row and the caller concatenates them.
function fortyZones(): OverlayLevel[] {
  const distinct: OverlayLevel[] = Array.from({ length: 30 }, (_, i) => {
    const lo = 28000 + i * 100
    return {
      price: lo + 40,
      label: i % 2 ? `SUPPLY·1h` : `DEMAND·4h`,
      grade: 'A',
      range: [lo, lo + 40] as [number, number],
    }
  })
  const dupes = distinct.slice(0, 10).map((z) => ({ ...z }))
  return [...distinct, ...dupes]
}

describe('chartOverlaySelect', () => {
  it('merges identical bands (same label, lo, hi) and keeps distinct ones', () => {
    const zones = fortyZones()
    expect(zones).toHaveLength(40)
    const d = dedupeZones(zones)
    expect(d).toHaveLength(30)
    // a band that differs only in hi is NOT a duplicate
    const twin = {
      ...zones[0],
      range: [zones[0].range![0], zones[0].range![1] + 5] as [number, number],
    }
    expect(dedupeZones([zones[0], twin])).toHaveLength(2)
  })

  it('distance is 0 inside a band and edge-distance outside', () => {
    const z: OverlayLevel = {
      price: 100,
      label: 'OB·4h',
      grade: 'A',
      range: [90, 100],
    }
    expect(zoneDistance(z, 95)).toBe(0)
    expect(zoneDistance(z, 80)).toBe(10)
    expect(zoneDistance(z, 130)).toBe(30)
  })

  it('nearest N to price, in the caller (score) order; first N when no price', () => {
    const zones = dedupeZones(fortyZones())
    const near = nearestZones(zones, 29500, 6)
    expect(near).toHaveLength(6)
    // the six bands nearest 29500 are the ones at 29200..29700 (lo values)
    const los = near.map((z) => z.range![0])
    expect(los).toEqual([29200, 29300, 29400, 29500, 29600, 29700])
    // no price → the scorer's first N
    expect(nearestZones(zones, undefined, 3).map((z) => z.range![0])).toEqual([
      28000, 28100, 28200,
    ])
    // fewer than N → all of them, untouched
    expect(nearestZones(zones.slice(0, 2), 29500, 6)).toHaveLength(2)
  })

  it('DEFAULT (zones off): a 40-zone block draws the seated levels and ZERO bands; the total still counts the distinct zones', () => {
    const sel = selectChartOverlay(seated, fortyZones(), 29500, {
      showZones: false,
    })
    expect(sel.zonesShown).toBe(0)
    expect(sel.zonesTotal).toBe(30)
    expect(sel.levels).toEqual(seated)
    expect(sel.levels.filter((l) => l.range)).toHaveLength(0)
  })

  it('zones ON: at most 12 seated + 6 nearest zones, labels intact', () => {
    const sel = selectChartOverlay(seated, fortyZones(), 29500, {
      showZones: true,
    })
    expect(DEFAULT_NEAREST_ZONES).toBe(6)
    expect(sel.zonesShown).toBe(6)
    expect(sel.zonesTotal).toBe(30)
    expect(sel.levels.length).toBeLessThanOrEqual(12 + 6)
    for (const l of seated) expect(sel.levels).toContain(l)
    expect(sel.levels.slice(0, 12)).toEqual(seated)
    const bands = sel.levels.filter((l) => l.range)
    expect(bands).toHaveLength(6)
    for (const b of bands) expect(b.label).toMatch(/^(SUPPLY|DEMAND)·(1h|4h)$/)
  })

  it('no zones → seated levels only, counts 0/0', () => {
    const sel = selectChartOverlay(seated, [], 29500, { showZones: true })
    expect(sel.levels).toEqual(seated)
    expect(sel.zonesShown).toBe(0)
    expect(sel.zonesTotal).toBe(0)
  })
})
