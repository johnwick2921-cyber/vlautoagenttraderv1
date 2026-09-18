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
  const [failed, setFailed] = useState(false)
  // W-CHART-ZONE-WALL — a per-viewer VIEW preference (localStorage), not a knob.
  const [showZones, setShowZones] = useState<boolean>(() => readShowZones())
  // the last loaded close: the price the "nearest zones" rule measures from
  const [lastClose, setLastClose] = useState<number | undefined>(undefined)

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
        const res = await httpClient.request<
          Array<{
            openTime: number
            open: number
            high: number
            low: number
            close: number
          }>
        >(url, { silent: true })
        if (
          stop ||
          !res.success ||
          !Array.isArray(res.data) ||
          !seriesRef.current
        )
          return
        const candles: Candle[] = res.data
          .map((c) => ({
            time: Math.floor(c.openTime / 1000) as UTCTimestamp,
            open: c.open,
            high: c.high,
            low: c.low,
            close: c.close,
          }))
          .sort((a, b) => (a.time as number) - (b.time as number))
          .filter((c, i, arr) => i === 0 || c.time !== arr[i - 1].time)
        seriesRef.current.setData(candles)
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
        style={{
          width: '100%',
          height,
          borderRadius: 'var(--vl-radius-inner)',
          overflow: 'hidden',
        }}
        aria-hidden
      />
      {zoneControl}
    </div>
  )
}
