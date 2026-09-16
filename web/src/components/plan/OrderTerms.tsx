import type { PlanOrderLeg } from '../../lib/api/plan'

const price = (value: number | null | undefined) =>
  typeof value === 'number' && Number.isFinite(value) && value > 0
    ? value.toLocaleString('en-US', { maximumFractionDigits: 4 })
    : 'UNKNOWN'

export function OrderTerms({ legs }: { legs?: PlanOrderLeg[] }) {
  if (!legs?.length) return null
  return (
    <div className="space-y-2 text-[10px]" style={{ color: 'var(--vl-muted)' }}>
      {legs.map((leg) => (
        <div key={leg.leg_index} data-testid={`order-terms-${leg.leg_index}`}>
          <p>
            Leg {leg.leg_index + 1} · order {leg.state}
            {leg.side ? ` · ${leg.side}` : ''}
            {leg.row_id
              ? ` · row ${leg.row_id} · placement ${leg.placement_seq} · last touched v${leg.version} · first authorized ${leg.armed_under_version ? `v${leg.armed_under_version}` : 'UNKNOWN'}`
              : ' · no selected authorization'}
            {leg.reason ? ` · ${leg.reason}` : ''}
          </p>
          <table
            className="w-full text-left"
            aria-label={`Leg ${leg.leg_index + 1} order prices`}
          >
            <thead>
              <tr>
                <th>Price</th>
                <th>Intended</th>
                <th>Composed</th>
                <th>ACCEPTED · current book</th>
              </tr>
            </thead>
            <tbody>
              {(['entry', 'stop', 'target'] as const).map((key) => (
                <tr key={key}>
                  <th className="capitalize">{key}</th>
                  <td>{price(leg.intended[key])}</td>
                  <td>{price(leg.composed[key])}</td>
                  <td>{price(leg.accepted[key])}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p>
            Intended: {leg.intended.source}
            {leg.intended.reason ? ` · ${leg.intended.reason}` : ''}
          </p>
          <p>
            Composed: {leg.composed.source}
            {leg.composed.reason ? ` · ${leg.composed.reason}` : ''}
          </p>
          <p>
            ACCEPTED: {leg.accepted.source} · received{' '}
            {leg.book_received_at_ms
              ? new Date(leg.book_received_at_ms).toLocaleString('en-US', {
                  timeZone: 'America/Chicago',
                }) + ' CT'
              : 'UNKNOWN'}
            {' · age '}
            {leg.book_received_at_ms && leg.book_age_ms >= 0
              ? `${Math.floor(leg.book_age_ms / 1000)}s`
              : 'UNKNOWN'}
            {' · AddOn build '}
            {leg.build_id || 'UNKNOWN'}
            {leg.accepted.reason ? ` · ${leg.accepted.reason}` : ''}
          </p>
        </div>
      ))}
    </div>
  )
}
