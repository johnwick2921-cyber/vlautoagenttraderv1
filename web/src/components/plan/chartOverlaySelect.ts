// W-CHART-ZONE-WALL (2026-09-17) — what the plan mini chart DRAWS.
//
// The owner's screenshot (2026-09-17 16:40 CT): dozens of overlapping gold
// bands ("SUPPLY·1h 29828", "DEMAND·1h 29746.50", "OB·4h …") covering the whole
// chart. The chart was handed the WHOLE structure block — every zone of every
// TF (live read: 1h=6, 4h=6, D=1 → 13 bands, each 100–700 points wide) on top
// of the seated levels, and drew all of it.
//
// Rule: the chart draws what the card seats, not the zone universe.
//   1. seated levels (the plan's own level rows) — ALWAYS, every one;
//   2. HTF zones — OFF by default (owner, after the CLASS 143 build: six bands
//      = twelve edges + fills around price was "still a wall"). When the
//      viewer opts in: identical bands merged (same label = kind·tf, same lo,
//      same hi), then only the N NEAREST to price.
// Labels stay "KIND·tf price" (the overlay's axis view formats them).
//
// PURE: no React, no chart. PlanMiniChart calls it; the tests pin it.

import type { OverlayLevel } from './LevelOverlayPrimitive'

/** How many HTF zones draw by default (nearest to price). */
export const DEFAULT_NEAREST_ZONES = 6
/** The planner seats at most this many levels; the chart never trims them. */
export const SEATED_LEVEL_CAP = 12
/** Per-viewer view preference (NOT a knob): remembered in localStorage. */
export const SHOW_ZONES_KEY = 'vl.planChart.showZones'

export interface OverlaySelection {
  levels: OverlayLevel[]
  zonesShown: number
  /** distinct zones after the merge (what "show all" would draw) */
  zonesTotal: number
}

export function zoneKey(z: OverlayLevel): string {
  const lo = z.range ? z.range[0] : z.price
  const hi = z.range ? z.range[1] : z.price
  return `${z.label}|${lo}|${hi}`
}

/** Merge identical bands (same kind·tf label, same lo, same hi); first wins. */
export function dedupeZones(zones: OverlayLevel[]): OverlayLevel[] {
  const seen = new Set<string>()
  const out: OverlayLevel[] = []
  for (const z of zones) {
    const k = zoneKey(z)
    if (seen.has(k)) continue
    seen.add(k)
    out.push(z)
  }
  return out
}

/** 0 when price sits inside the band, else the distance to its nearer edge. */
export function zoneDistance(z: OverlayLevel, price: number): number {
  const lo = z.range ? Math.min(z.range[0], z.range[1]) : z.price
  const hi = z.range ? Math.max(z.range[0], z.range[1]) : z.price
  if (price >= lo && price <= hi) return 0
  return price < lo ? lo - price : price - hi
}

/**
 * The N zones nearest to price, in the caller's order (score order from the
 * structure block) so ties keep the scorer's preference. With no price the
 * first N stand — the scorer's order is the only ranking available.
 */
export function nearestZones(
  zones: OverlayLevel[],
  price: number | undefined,
  n: number
): OverlayLevel[] {
  if (n <= 0) return []
  if (zones.length <= n) return zones
  if (price === undefined || !Number.isFinite(price)) return zones.slice(0, n)
  const ranked = zones
    .map((z, i) => ({ z, i, d: zoneDistance(z, price) }))
    .sort((a, b) => a.d - b.d || a.i - b.i)
    .slice(0, n)
    .sort((a, b) => a.i - b.i)
  return ranked.map((r) => r.z)
}

export function selectChartOverlay(
  seated: OverlayLevel[],
  zones: OverlayLevel[],
  price: number | undefined,
  opts: { showZones: boolean; nearest?: number }
): OverlaySelection {
  const distinct = dedupeZones(zones)
  const n = opts.nearest ?? DEFAULT_NEAREST_ZONES
  const shown = opts.showZones ? nearestZones(distinct, price, n) : []
  return {
    levels: [...seated, ...shown],
    zonesShown: shown.length,
    zonesTotal: distinct.length,
  }
}

/** localStorage is a per-viewer convenience: every read/write is guarded. */
export function readShowZones(): boolean {
  try {
    return window.localStorage.getItem(SHOW_ZONES_KEY) === '1'
  } catch {
    return false
  }
}

export function writeShowZones(on: boolean): void {
  try {
    if (on) window.localStorage.setItem(SHOW_ZONES_KEY, '1')
    else window.localStorage.removeItem(SHOW_ZONES_KEY)
  } catch {
    /* private window / blocked storage — the toggle still works in-session */
  }
}
