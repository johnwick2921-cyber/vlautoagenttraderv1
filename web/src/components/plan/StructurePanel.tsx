// S5 — the STRUCTURE panel (bias only, never entries). Renders NOTHING when
// the plan doc carries no structure block (absent ≠ [] — no fabricated rows).
// One row per TF (D / 4h / 1h): trend arrow, last swings, impulse range with
// premium/discount, and scorer-ranked zones with a TF badge. Prices are raw,
// contract-scoped — never back-adjusted across the roll.

import type { StructureMapView, StructureZoneView } from '../../lib/api/plan'
import type { OverlayLevel } from './LevelOverlayPrimitive'

const TF_ORDER = ['D', '4h', '1h'] as const

function trendArrow(trend: string): string {
  switch (trend) {
    case 'up':
      return '↑'
    case 'down':
      return '↓'
    default:
      return '⇄'
  }
}

// D1 (review): mirror the scorer's freshMult ladder exactly — the dev
// vocabulary is ""/"a"/"b"/"c"/"done"/"consumed"/"tested" (levels_score.go);
// "fresh"/"tested-1"/"tested-2"/"stale" are S2's display vocabulary.
function zoneGrade(fresh: string): 'A' | 'B' | 'C' {
  switch (fresh) {
    case '':
    case 'a':
    case 'fresh':
      return 'A'
    case 'b':
    case 'tested-1':
      return 'B'
    default:
      return 'C'
  }
}

/** D1: the chip shows the scorer's real label; empty means fresh. */
function zoneFreshChip(fresh: string): string {
  return fresh === '' ? 'fresh' : fresh
}

/** The chart's slice of the structure block: one overlay entry per zone. */
export function structureZonesToOverlay(
  structure?: StructureMapView | null
): OverlayLevel[] {
  if (!structure?.tfs) return []
  const out: OverlayLevel[] = []
  for (const [tf, st] of Object.entries(structure.tfs)) {
    for (const z of st.zones ?? []) {
      out.push({
        price: z.hi,
        label: `${z.kind}\u00b7${tf}`,
        grade: zoneGrade(z.fresh),
        range: [z.lo, z.hi],
      })
    }
  }
  return out
}

function ZoneRow({ z }: { z: StructureZoneView }) {
  return (
    <span
      className="mr-1 inline-flex items-center gap-1 rounded border border-white/10 px-1.5 py-0.5 text-[10px]"
      data-testid="structure-zone"
    >
      <span className="font-bold">{z.kind}</span>
      <span className="rounded bg-white/10 px-1 text-[9px] font-mono" data-testid="structure-zone-tf">
        {z.tf}
      </span>
      <span>
        {z.lo.toFixed(2)}–{z.hi.toFixed(2)}
      </span>
      <span className="opacity-70">{zoneFreshChip(z.fresh)}</span>
    </span>
  )
}

export function StructurePanel({
  structure,
}: {
  structure?: StructureMapView | null
}) {
  if (!structure?.tfs) return null
  const asOf = structure.as_of_ms ? new Date(structure.as_of_ms).toISOString().slice(11, 19) : ''
  return (
    <section
      className="rounded border border-white/10 p-3"
      data-testid="structure-panel"
    >
      <header className="mb-1 flex items-baseline gap-2">
        <h4 className="text-xs font-bold tracking-wide">
          STRUCTURE — bias only, not entries
        </h4>
        {structure.contract && (
          <span className="text-[10px] font-mono opacity-70">{structure.contract}</span>
        )}
        {asOf && <span className="text-[10px] opacity-50">@ {asOf}</span>}
      </header>
      <div className="space-y-1">
        {TF_ORDER.filter((tf) => structure.tfs[tf]).map((tf) => {
          const st = structure.tfs[tf]
          const pd = st.premium_discount
          return (
            <div
              key={tf}
              className="flex flex-wrap items-center gap-x-2 gap-y-0.5 border-t border-white/5 pt-1 text-[11px]"
              data-testid={`structure-tf-${tf}`}
            >
              <span className="font-mono font-bold" data-testid="structure-tf-badge">
                {tf}
              </span>
              <span data-testid="structure-trend">
                {trendArrow(st.trend)} {st.trend}
              </span>
              {st.last_swing_high && (
                <span className="opacity-80">
                  H {st.last_swing_high.price.toFixed(2)}
                </span>
              )}
              {st.last_swing_low && (
                <span className="opacity-80">
                  L {st.last_swing_low.price.toFixed(2)}
                </span>
              )}
              {typeof st.impulse_lo === 'number' && typeof st.impulse_hi === 'number' && (
                <span className="opacity-80">
                  {st.impulse_lo.toFixed(2)}–{st.impulse_hi.toFixed(2)}
                </span>
              )}
              {typeof st.impulse_lo === 'number' &&
                typeof st.impulse_hi === 'number' &&
                typeof pd === 'number' && (
                  <span className="opacity-60">
                    pd {(pd * 100).toFixed(0)}%
                  </span>
                )}
              <span className="opacity-50">bars {st.bars}</span>
              {(st.zones ?? []).length > 0 && (
                <span className="flex flex-wrap gap-1">
                  {(st.zones ?? []).map((z, i) => (
                    <ZoneRow key={i} z={z} />
                  ))}
                </span>
              )}
            </div>
          )
        })}
      </div>
    </section>
  )
}
