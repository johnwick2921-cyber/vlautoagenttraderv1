import type { LevelZoneMap as ZoneMap } from '../../lib/api/plan'

// Frozen at the planner read. Never reorders the editable authored-level array.
export function LevelZoneMap({ map }: { map?: ZoneMap }) {
  if (!map) return null
  return (
    <details className="rounded border border-white/10 p-3">
      <summary>
        Level zones · {map.zones.length} zones · {map.broad} broad context ·{' '}
        {map.null_widths} unknown source widths
      </summary>
      <p className="text-xs opacity-70">
        At read: {map.at}. Bounds and all source names are preserved. Ranking is
        experimental [I]; a shortlist place gives no trading permission.
      </p>
      <div className="max-h-96 overflow-auto">
        {map.zones.map((zone, index) => (
          <div key={index} className="border-t border-white/10 py-2 text-xs">
            <strong>
              {zone.lo === null || zone.hi === null
                ? 'NULL width'
                : `${zone.lo.toFixed(2)}–${zone.hi.toFixed(2)}`}
            </strong>
            {' · anchor '}
            {zone.anchor.toFixed(2)}
            {' · '}
            {zone.broad
              ? 'broad context'
              : zone.shortlisted
                ? 'shortlist [I]'
                : 'local reference'}
            {' · families '}
            {zone.family_count}
            {zone.incomplete_width && ' · some source widths unknown'}
            <div>Prior touches: {zone.prior_touches ?? 'UNKNOWN'}</div>
            {zone.sources.map((source, i) => (
              <div key={i}>
                {source.label} · {source.tf || 'timeframe unknown'} ·{' '}
                {source.price.toFixed(2)} · formed:{' '}
                {source.formed_at ?? 'UNKNOWN'} · {source.width_rule}
              </div>
            ))}
          </div>
        ))}
      </div>
    </details>
  )
}
