import type { StructuralGeometryView } from '../../lib/api/plan'

export function StructuralGeometry({ rows, language }: { rows?: StructuralGeometryView[] | null; language: string }) {
  if (!rows?.length) return null
  const labels = language === 'zh'
    ? { title: '结构止损与目标（研究候选）', entry: '入场', zone: '区域', buffer: '缓冲', stop: '止损', target: '目标', risk: '含成本风险', refused: '拒绝', admitted: '允许', pending: '待检查' }
    : language === 'id'
      ? { title: 'Stop dan target struktural (kandidat riset)', entry: 'Entri', zone: 'Zona', buffer: 'Buffer', stop: 'Stop', target: 'Target', risk: 'Risiko termasuk biaya', refused: 'Ditolak', admitted: 'Diizinkan', pending: 'Menunggu pemeriksaan' }
      : { title: 'Structural stop and target (research candidate)', entry: 'Entry', zone: 'Zone', buffer: 'Buffer', stop: 'Stop', target: 'Target', risk: 'Risk including costs', refused: 'Refused', admitted: 'Admitted', pending: 'Pending checks' }
  const number = (v?: number) => v === undefined ? '—' : v.toFixed(2)
  return <section className="px-3 py-2 text-xs" data-testid="structural-geometry">
    <h4 className="font-semibold">{labels.title}</h4>
    {rows.map(r => <div key={`${r.scenario}:${r.leg}`} className="mt-2 rounded border p-2">
      <strong>{r.scenario} · {r.reason === 'admitted' ? labels.admitted : r.reason === 'pending_gates' ? labels.pending : labels.refused} · {r.quantity} MNQ</strong>
      <p>{labels.entry} {number(r.entry)} · {labels.zone} [{number(r.zone_lo)}, {number(r.zone_hi)}] · {labels.buffer} {number(r.buffer)}</p>
      <p>{labels.stop} {number(r.stop)} ({r.stop_source}) · {labels.target} {number(r.target)} {r.target_names?.join(' + ')}</p>
      <p>{labels.risk} ${number(r.loss_usd)} · {r.reason}: {r.detail}</p>
    </div>)}
  </section>
}
