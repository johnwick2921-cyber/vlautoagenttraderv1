// INSTRUMENTS DRAWER (owner ruling 2026-09-03) — the three DESCRIPTIVE
// instruments, folded into one collapsed dropdown below the expectancy table.
//
// They are collapsed because none of them currently supports a decision:
// discipline is an aggregate whose seam rows are not yet excluded, MAE/MFE has
// no rows at all, and the level gate is a retired instrument kept only so its
// successor has something to be compared against. Giving them the same visual
// weight as the expectancy table would imply they carry the same authority.
//
// Every row names its SOURCE and its n. That is the whole point of the drawer:
// a number whose provenance is not on screen beside it is a number a reader has
// to trust rather than check.

import { useEffect, useState } from 'react'
import { planApi } from '../../lib/api/plan'

interface ExpRow {
  key: { condition?: string }
  n: number
  median_mae: number | null
  median_mfe: number | null
}

interface TradesLike {
  summary?: { counts?: Record<string, number>; total?: number; gpa?: number }
}

interface StatsLike {
  weekly?: {
    iso_week: string
    verdicts?: { level_type: string; status: string }[]
  } | null
  progress?: {
    level_type: string
    n: number
    target_n: number
    reactions: number
    warming: boolean
  }[]
}

const dash = '—'

export function InstrumentsDrawer({ traderId }: { traderId?: string }) {
  const [open, setOpen] = useState(false)
  const [trades, setTrades] = useState<TradesLike | null>(null)
  const [stats, setStats] = useState<StatsLike | null>(null)
  const [expectancyRows, setExpectancyRows] = useState<ExpRow[]>([])

  useEffect(() => {
    if (!traderId) return
    let alive = true
    void (async () => {
      const [t, s] = await Promise.all([
        planApi.getPlanTrades(traderId),
        planApi.getPlanStats(traderId),
      ])
      if (!alive) return
      setTrades(t as TradesLike)
      setStats(s as StatsLike)
    })()
    return () => {
      alive = false
    }
  }, [traderId])

  // Expectancy is read HERE rather than passed in. It used to arrive as a prop
  // from ExpectancyPanel, which returns null when the endpoint gives it
  // nothing — so an empty expectancy day removed the whole drawer, including
  // the two instruments that do not read expectancy at all. Same endpoint and
  // same 60s cadence as the table above, so neither surface can be showing
  // older rows than the other.
  useEffect(() => {
    let alive = true
    const load = async () => {
      const res = (await planApi.getExpectancy()) as { rows?: ExpRow[] } | null
      if (alive) setExpectancyRows(res?.rows ?? [])
    }
    void load()
    const t = setInterval(load, 60_000)
    return () => {
      alive = false
      clearInterval(t)
    }
  }, [])

  const src = { color: 'var(--vl-faint)', fontSize: '9px' } as const

  // MAE/MFE comes from the expectancy rows, which read trade_excursions and
  // return null when there is no excursion record. The legacy
  // trader_positions.mae/mfe columns are NOT consulted: they default to 0, so
  // averaging "the rows where they are non-zero" silently selects a biased
  // subset and reports it as the population (the class 40 shape).
  const withExc = expectancyRows.filter(
    (r) => r.median_mae !== null || r.median_mfe !== null
  )

  const summary = trades?.summary
  const graded = summary?.total ?? 0
  const progress = stats?.progress ?? []

  return (
    <div
      data-testid="instruments-drawer"
      className="flex flex-col gap-1 pt-1"
      style={{ borderTop: '1px dashed var(--vl-hair)' }}
    >
      <button
        type="button"
        data-testid="instruments-drawer-toggle"
        aria-expanded={open}
        className="flex w-full items-center gap-1.5 text-left text-[11px] uppercase tracking-wide"
        style={{
          color: 'var(--vl-muted)',
          background: 'none',
          border: 'none',
          padding: '2px 0',
          cursor: 'pointer',
          fontFamily: 'inherit',
        }}
        onClick={() => setOpen((o) => !o)}
      >
        <span aria-hidden style={{ width: '0.75em', flex: 'none' }}>
          {open ? '\u25be' : '\u25b8'}
        </span>
        <span>
          Instruments · discipline · MAE/MFE · level gate (descriptive)
        </span>
      </button>

      {open && (
        <div className="flex flex-col gap-2 pt-1">
          {/* ── DISCIPLINE ─────────────────────────────────────────── */}
          <div data-testid="instrument-discipline" className="flex flex-col">
            <span className="text-[10px]" style={{ color: 'var(--vl-ivory)' }}>
              DISCIPLINE —{' '}
              {graded > 0 ? (
                <>
                  adherence{' '}
                  <span className="vl-num">
                    GPA {(summary?.gpa ?? 0).toFixed(2)}
                  </span>{' '}
                  <span style={{ color: 'var(--vl-muted)' }}>n={graded}</span>{' '}
                  <span className="vl-num" style={{ color: 'var(--vl-muted)' }}>
                    {['A', 'B', 'C', 'D', 'F']
                      .map((g) => {
                        const c = summary?.counts?.[g] ?? 0
                        return c > 0 ? `${g}×${c}` : ''
                      })
                      .filter(Boolean)
                      .join(' ')}
                  </span>
                </>
              ) : (
                <span style={{ color: 'var(--vl-muted)' }}>
                  no graded trades yet
                </span>
              )}
            </span>
            {/* The seam-excluded aggregate is not in this build. Saying
                "seam-excluded: no" is the honest form; claiming an exclusion
                that did not happen would be worse than showing nothing. */}
            <span style={src}>
              source: GET /api/plan/trades · adherence summary · seam-excluded:
              no (pending the seam wave) · excluded count: {dash}
            </span>
          </div>

          {/* ── MAE / MFE ──────────────────────────────────────────── */}
          <div data-testid="instrument-maemfe" className="flex flex-col">
            <span className="text-[10px]" style={{ color: 'var(--vl-ivory)' }}>
              MAE/MFE —{' '}
              {withExc.length === 0 ? (
                <span style={{ color: 'var(--vl-muted)' }}>
                  no excursion rows yet
                </span>
              ) : (
                <span className="vl-num" style={{ color: 'var(--vl-muted)' }}>
                  {withExc.map((r) => (
                    <span key={r.key.condition ?? '?'} className="mr-2">
                      {r.key.condition}{' '}
                      <span style={{ color: 'var(--vl-short)' }}>
                        {r.median_mae === null
                          ? dash
                          : `−${r.median_mae.toFixed(1)}`}
                      </span>
                      /
                      <span style={{ color: 'var(--vl-long)' }}>
                        {r.median_mfe === null
                          ? dash
                          : `+${r.median_mfe.toFixed(1)}`}
                      </span>{' '}
                      n={r.n}
                    </span>
                  ))}
                </span>
              )}
            </span>
            <span style={src}>
              source: trade_excursions via GET /api/expectancy (median per
              condition). The legacy trader_positions.mae/mfe columns are never
              read here — they default to 0, so an average over the non-zero
              ones is a biased subset reported as the population. Blank means
              not measured, not zero.
            </span>
          </div>

          {/* ── LEVEL GATE (RETIRED) ───────────────────────────────── */}
          <div
            data-testid="instrument-randomgate"
            data-retired="true"
            className="flex flex-col"
          >
            <span className="text-[10px]" style={{ color: 'var(--vl-ivory)' }}>
              LEVEL GATE —{' '}
              <span style={{ color: 'var(--vl-faint)' }}>
                legacy — retired, pending D1′
              </span>
            </span>
            {/* The frozen weekly verdict — the only place this instrument ever
                made a claim. It stays INSIDE the retired scope: a verdict about
                an edge from a biased instrument is exactly the thing that must
                not be read as current. */}
            <span className="text-[10px]" style={{ color: 'var(--vl-muted)' }}>
              {stats?.weekly ? (
                <>
                  frozen {stats.weekly.iso_week}:{' '}
                  {(stats.weekly.verdicts ?? [])
                    .map((v) => `${v.level_type} ${v.status}`)
                    .join(' · ') || 'no verdicts'}
                </>
              ) : (
                'first eval Sunday'
              )}
            </span>
            {progress.length === 0 ? (
              <span
                className="text-[10px]"
                style={{ color: 'var(--vl-muted)' }}
              >
                no level rows yet
              </span>
            ) : (
              <span
                className="vl-num text-[10px]"
                style={{ color: 'var(--vl-muted)' }}
              >
                {progress.map((p) => (
                  <span key={p.level_type} className="mr-2" data-rate>
                    {p.level_type} {p.n}/{p.target_n}
                    {p.warming ? ' WARMING' : ''}
                  </span>
                ))}
              </span>
            )}
            <span style={src}>
              source: level_stats via GET /api/plan/stats — the BIASED
              instrument that wave 1B retires. Its touch/react counts are shown
              as raw progress, never as a rate with a significance claim. When
              1B boots this reads touch_outcomes p(hold) with n and a Wilson
              interval, and stays DESCRIPTIVE ONLY below n=200.
            </span>
          </div>
        </div>
      )}
    </div>
  )
}
