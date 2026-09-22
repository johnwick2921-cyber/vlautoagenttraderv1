// P4.3 — the plan mini chart: a small candlestick chart with the
// LevelOverlayPrimitive attached. Levels arrive as PROPS (the same array the
// ZoneTable renders) — the chart never fetches levels, only bars. Chart init is
// guarded so a headless/canvas-less environment (jsdom tests) degrades to a
// placeholder instead of throwing.

import { useEffect, useMemo, useRef, useState } from 'react'
import {
  createChart,
  CandlestickSeries,
  type IChartApi,
  type ISeriesApi,
  type UTCTimestamp,
} from 'lightweight-charts'
import {
  ctTickMarkFormatter,
  ctCrosshairTimeFormatter,
  logBarDebug,
} from '../../lib/chartTime'
import { httpClient } from '../../lib/httpClient'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { PlanLevelFact } from '../../lib/api/plan'
import {
  LevelOverlayPrimitive,
  type OverlayLevel,
} from './LevelOverlayPrimitive'
import {
  selectChartOverlay,
  readShowZones,
  writeShowZones,
  DEFAULT_NEAREST_ZONES,
} from './chartOverlaySelect'

interface Props {
  symbol: string
  exchange: string
  facts: PlanLevelFact[]
  language: Language
  interval?: string
  height?: number
  /**
   * S5 — HTF structure zones (kind·tf labels + [lo,hi] bands), raw prices only.
   * W-CHART-ZONE-WALL: the chart draws the seated levels ALWAYS and NO zones
   * by default; "Show zones (N)" under the chart draws the `nearestZones`
   * zones nearest to the last close (duplicates merged).
   */
  structureZones?: OverlayLevel[]
  /** how many HTF zones draw when "show all" is off (default 6) */
  nearestZones?: number
}

// PlanLevelFact[] → OverlayLevel[] (the chart's slice of the shared array).
export function factsToOverlay(facts: PlanLevelFact[]): OverlayLevel[] {
  return facts.map((f) => ({
    price: f.price,
    label: f.label,
    grade: f.grade,
    instruction: f.instruction,
  }))
}

interface Candle {
  time: UTCTimestamp
  open: number
  high: number
  low: number
  close: number
  /** W-ROLL-DAY-CHART — derived from the prior contract's 1m rows. */
  derived?: boolean
  /** raw (pre-basis) close for the tooltip; present only on adjusted bars. */
  rawClose?: number
}

/** W-ROLL-DAY-CHART — the /klines roll envelope (ninjatrader only). */
export interface RollInfo {
  derived: boolean
  adjusted: boolean
  basis?: number
  reason?: string
}

interface KlinesRow {
  openTime: number
  open: number
  high: number
  low: number
  close: number
  contract?: string
  derived?: boolean
  adjusted?: boolean
}

/**
 * Pure unwrap of the /klines response: today's bare array (crypto + legacy
 * knob + no-prior-segment) or the W-ROLL-DAY-CHART {klines, roll} envelope.
 */
export function unwrapKlinesResponse(data: unknown): {
  rows: KlinesRow[]
  roll?: RollInfo
} {
  if (Array.isArray(data)) return { rows: data as KlinesRow[] }
  const obj = data as { klines?: unknown; roll?: RollInfo } | null
  if (obj && Array.isArray(obj.klines)) {
    return { rows: obj.klines as KlinesRow[], roll: obj.roll }
  }
  return { rows: [] }
}

const MONTH_NAMES = [
  'Jan',
  'Feb',
  'Mar',
  'Apr',
  'May',
  'Jun',
  'Jul',
  'Aug',
  'Sep',
  'Oct',
  'Nov',
  'Dec',
]

function contractMonthName(contract?: string): string {
  if (!contract) return ''
  const mm = contract.split(/\s+/)[1]?.split('-')[0]
  const n = Number(mm)
  return Number.isFinite(n) && n >= 1 && n <= 12 ? MONTH_NAMES[n - 1] : ''
}

/**
 * The one-line legend, from the envelope ONLY (no literals):
 * "pre-roll (Sep) · basis-adjusted +290.00" / "pre-roll (Sep) · unadjusted".
 */
export function rollLegend(
  roll: RollInfo | undefined,
  priorContract?: string
): string {
  if (!roll?.derived) return ''
  const month = contractMonthName(priorContract) || 'prior'
  if (roll.adjusted && typeof roll.basis === 'number') {
    return `pre-roll (${month}) · basis-adjusted +${roll.basis.toFixed(2)}`
  }
  return `pre-roll (${month}) · unadjusted${roll.reason ? ` (${roll.reason})` : ''}`
}

/** the muted colour derived bars draw in */
const DERIVED_MUTED = '#6B7078'

/**
 * Row → candle + per-bar styling. Derived bars draw muted; adjusted bars carry
 * the raw close (close − basis from the envelope) for the tooltip.
 */
export function candleFromRow(
  c: KlinesRow,
  roll: RollInfo | undefined
): Candle {
  const derived = c.derived === true
  const adjusted =
    derived && roll?.adjusted === true && typeof roll.basis === 'number'
  const candle: Candle = {
    time: Math.floor(c.openTime / 1000) as UTCTimestamp,
    open: c.open,
    high: c.high,
    low: c.low,
    close: c.close,
  }
  if (derived) {
    candle.derived = true
    if (adjusted) {
      candle.rawClose = c.close - (roll!.basis as number)
    }
  }
  return candle
}

/** per-bar colours for the candlestick series (muted when derived) */
export function candleBarStyle(candle: Candle) {
  return candle.derived
    ? {
        color: DERIVED_MUTED,
        borderColor: DERIVED_MUTED,
        wickColor: DERIVED_MUTED,
      }
    : {}
}

export function PlanMiniChart({
  symbol,
  exchange,
  facts,
  language,
  interval = '5m',
  height = 200,
  structureZones = [],
  nearestZones = DEFAULT_NEAREST_ZONES,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const overlayRef = useRef<LevelOverlayPrimitive | null>(null)
  // W-ROLL-DAY-CHART — candles kept for the crosshair tooltip lookup.
  const candlesRef = useRef<Candle[]>([])
  const [failed, setFailed] = useState(false)
  // W-CHART-ZONE-WALL — a per-viewer VIEW preference (localStorage), not a knob.
  const [showZones, setShowZones] = useState<boolean>(() => readShowZones())
  // the last loaded close: the price the "nearest zones" rule measures from
  const [lastClose, setLastClose] = useState<number | undefined>(undefined)
  // W-ROLL-DAY-CHART — the roll envelope (prior segment info) + tooltip ref.
  const [roll, setRoll] = useState<RollInfo | undefined>(undefined)
  const [priorContract, setPriorContract] = useState<string | undefined>(
    undefined
  )
  const tooltipRef = useRef<HTMLDivElement>(null)

  // init chart once
  useEffect(() => {
    if (!containerRef.current) return
    // Bail in canvas-less environments (jsdom/SSR): lightweight-charts draws on
    // an async animation frame that a try/catch here can't guard, so detect the
    // missing 2d context up front and degrade to the placeholder instead.
    let canvasOk = false
    try {
      canvasOk = !!document.createElement('canvas').getContext('2d')
    } catch {
      canvasOk = false
    }
    if (!canvasOk) {
      setFailed(true)
      return
    }
    let disposed = false
    try {
      const chart = createChart(containerRef.current, {
        height,
        layout: {
          background: { color: 'transparent' },
          textColor: '#8A8F98',
          fontFamily: 'JetBrains Mono, monospace',
        },
        grid: {
          vertLines: { visible: false },
          horzLines: { color: 'rgba(232,227,216,0.05)' },
        },
        rightPriceScale: { borderColor: 'rgba(232,227,216,0.08)' },
        timeScale: {
          borderColor: 'rgba(232,227,216,0.08)',
          timeVisible: true,
          secondsVisible: false,
          // CT labels via THE ONE shared chart-tz site. Without a formatter the
          // lib renders UTC — this chart showed +5h vs the NT8 chart (S1).
          tickMarkFormatter: ctTickMarkFormatter,
        },
        localization: { timeFormatter: ctCrosshairTimeFormatter },
        crosshair: { mode: 0 },
      })
      const series = chart.addSeries(CandlestickSeries, {
        upColor: '#3FBF8F',
        downColor: '#E06C6C',
        borderVisible: false,
        wickUpColor: '#3FBF8F',
        wickDownColor: '#E06C6C',
      })
      const overlay = new LevelOverlayPrimitive()
      series.attachPrimitive(overlay)
      chartRef.current = chart
      seriesRef.current = series
      overlayRef.current = overlay

      const ro = new ResizeObserver(() => {
        if (containerRef.current && chartRef.current) {
          chartRef.current.applyOptions({
            width: containerRef.current.clientWidth,
          })
        }
      })
      ro.observe(containerRef.current)

      // W-ROLL-DAY-CHART — crosshair tooltip: raw vs adjusted close on derived
      // bars (the only bars whose displayed price was shifted).
      chart.subscribeCrosshairMove((param) => {
        const tip = tooltipRef.current
        if (!tip) return
        if (
          param.point === undefined ||
          !param.time ||
          param.point.x < 0 ||
          param.point.y < 0
        ) {
          tip.style.display = 'none'
          return
        }
        const candle = candlesRef.current.find(
          (cd) => (cd.time as number) === (param.time as number)
        )
        if (!candle || !candle.derived) {
          tip.style.display = 'none'
          return
        }
        tip.style.display = 'block'
        tip.style.left = `${param.point.x + 8}px`
        tip.style.top = `${param.point.y + 8}px`
        tip.innerHTML =
          candle.rawClose !== undefined
            ? `raw ${candle.rawClose.toFixed(2)} · adj ${candle.close.toFixed(2)}`
            : `close ${candle.close.toFixed(2)}`
      })

      return () => {
        disposed = true
        ro.disconnect()
        chart.remove()
        chartRef.current = null
        seriesRef.current = null
        overlayRef.current = null
      }
    } catch {
      if (!disposed) setFailed(true)
      return
    }
  }, [height])

  // load bars + poll
  useEffect(() => {
    if (failed) return
    let stop = false
    const load = async () => {
      try {
        const url = `/api/klines?symbol=${encodeURIComponent(symbol)}&interval=${encodeURIComponent(interval)}&limit=5000&exchange=${encodeURIComponent(exchange)}`
        const res = await httpClient.request<unknown>(url, { silent: true })
        if (stop || !res.success || !seriesRef.current) return
        const { rows, roll: envRoll } = unwrapKlinesResponse(res.data)
        if (rows.length === 0) return
        const candles: Candle[] = rows
          .map((c) => candleFromRow(c, envRoll))
          .sort((a, b) => (a.time as number) - (b.time as number))
          .filter((c, i, arr) => i === 0 || c.time !== arr[i - 1].time)
        // per-bar muted styling for derived candles
        candles.forEach((c) => {
          const style = candleBarStyle(c)
          if (Object.keys(style).length > 0) Object.assign(c, style)
        })
        candlesRef.current = candles
        seriesRef.current.setData(candles)
        if (envRoll?.derived) {
          setRoll(envRoll)
          setPriorContract(rows.find((r) => r.derived)?.contract)
        }
        if (candles.length) setLastClose(candles[candles.length - 1].close)
        logBarDebug(
          'PlanMiniChart',
          candles.length
            ? (candles[candles.length - 1].time as number) * 1000
            : undefined
        )
      } catch {
        /* silent — the table still shows the levels */
      }
    }
    void load()
    const id = setInterval(load, 15_000)
    return () => {
      stop = true
      clearInterval(id)
    }
  }, [symbol, exchange, interval, failed])

  // What the chart draws: every seated level + the nearest N distinct zones
  // (or all of them). The zones prop is a fresh array every parent render, so
  // its CONTENT keys the memo — the old effect keyed on [facts] alone and a
  // structure block that changed under a stable facts array never re-drew.
  const zonesKey = JSON.stringify(structureZones)
  const selection = useMemo(
    () =>
      selectChartOverlay(factsToOverlay(facts), structureZones, lastClose, {
        showZones,
        nearest: nearestZones,
      }),

    [facts, zonesKey, lastClose, showZones, nearestZones]
  )

  // push levels to the overlay whenever the selection changes
  useEffect(() => {
    if (overlayRef.current)
      overlayRef.current.setData({ levels: selection.levels })
  }, [selection])

  const toggleShowZones = (on: boolean) => {
    setShowZones(on)
    writeShowZones(on)
  }

  // The zone control renders whenever the block carries zones — also on the
  // placeholder, so the count is visible (and testable) without a canvas.
  const zoneControl = selection.zonesTotal > 0 && (
    <label
      className="flex items-center gap-1 text-[10px]"
      style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
      data-testid="chart-zone-control"
    >
      <input
        type="checkbox"
        checked={showZones}
        onChange={(e) => toggleShowZones(e.target.checked)}
        data-testid="chart-show-zones"
      />
      <span>
        {tp('chartShowZones', language, {
          total: String(selection.zonesTotal),
        })}
      </span>
      {showZones && (
        <span data-testid="chart-zone-count">
          {tp('chartZonesShown', language, {
            shown: String(selection.zonesShown),
            total: String(selection.zonesTotal),
          })}
        </span>
      )}
    </label>
  )

  if (failed) {
    return (
      <div className="flex flex-col gap-1">
        <div
          className="flex items-center justify-center text-[11px]"
          style={{
            height,
            background: 'var(--vl-card-2)',
            borderRadius: 'var(--vl-radius-inner)',
            color: 'var(--vl-faint)',
            fontFamily: 'var(--vl-font-ui)',
          }}
        >
          {tp('loading', language)}
        </div>
        {zoneControl}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-1">
      <div
        ref={containerRef}
        className="relative"
        style={{
          width: '100%',
          height,
          borderRadius: 'var(--vl-radius-inner)',
          overflow: 'hidden',
        }}
        aria-hidden
      >
        <div
          ref={tooltipRef}
          data-testid="chart-roll-tooltip"
          className="absolute z-10 text-[10px] px-1.5 py-0.5 rounded"
          style={{
            display: 'none',
            background: 'var(--vl-card-2)',
            border: '1px solid var(--vl-hair)',
            color: 'var(--vl-ivory)',
            fontFamily: 'var(--vl-font-ui)',
            pointerEvents: 'none',
          }}
        />
      </div>
      {roll?.derived && (
        <div
          data-testid="chart-roll-legend"
          className="text-[10px] px-1"
          style={{ color: DERIVED_MUTED, fontFamily: 'var(--vl-font-ui)' }}
        >
          {rollLegend(roll, priorContract)}
        </div>
      )}
      {zoneControl}
    </div>
  )
}
