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

// P5.3 — detect owner/AI level conflicts: an OWNER level and an AI level at
// (nearly) the same price with DIFFERENT instructions. Owner wins → the AI index
// is ghosted; both indices are flagged for the ⚡ chip. Priced band ≈ tick noise.
export function detectConflicts(
  facts: PlanLevelFact[],
  bandPts = 3
): { ghosted: Set<number>; flagged: Set<number> } {
  const ghosted = new Set<number>()
  const flagged = new Set<number>()
  for (let i = 0; i < facts.length; i++) {
    const a = facts[i]
    if (a.origin !== 'OWNER') continue
    for (let j = 0; j < facts.length; j++) {
      if (i === j) continue
      const b = facts[j]
      if (b.origin === 'OWNER') continue // only owner-vs-AI
      if (
        Math.abs(a.price - b.price) <= bandPts &&
        a.instruction !== b.instruction
      ) {
        ghosted.add(j) // AI element loses
        flagged.add(i)
        flagged.add(j)
      }
    }
  }
  return { ghosted, flagged }
}
