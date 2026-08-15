// P4.3 — derive the CARD's view state from the backend level facts. The UI
// computes nothing tradable: freshness/distance are read from the evaluator's
// facts, never recomputed here. This only maps facts → display buckets.

import type { PlanLevelFact } from '../../lib/api/plan'
import type { Fresh } from './chips'

// Default "near" threshold in points (≈ the design's 0.25×dATR highlight). The
// API doesn't return dATR, so we approximate with a fixed points band rather
// than fabricate an ATR client-side. Callers can override.
export const NEAR_THRESHOLD_PTS = 12

// still_valid=false → consumed · swept-but-valid → tested · else → fresh.
export function levelFresh(f: PlanLevelFact): Fresh {
  if (!f.still_valid) return 'consumed'
  if (f.sweep) return 'tested'
  return 'fresh'
}

// A level is "near" when |distance| is inside the threshold band.
export function levelNear(
  f: PlanLevelFact,
  thresholdPts = NEAR_THRESHOLD_PTS
): boolean {
  return Math.abs(f.distance) < thresholdPts
}

// Format a signed points distance for the mono cell (e.g. "+57", "−12").
export function fmtDistance(distance: number): string {
  const rounded = Math.round(distance)
  if (rounded === 0) return '0'
  const sign = rounded > 0 ? '+' : '−'
  return `${sign}${Math.abs(rounded)}`
}

// Format a price for the mono cell — MNQ uses quarter ticks.
export function fmtPrice(price: number): string {
  return Number.isInteger(price) ? String(price) : price.toFixed(2)
}
