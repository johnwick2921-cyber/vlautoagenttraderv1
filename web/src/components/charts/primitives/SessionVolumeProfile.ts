/*
 * SessionVolumeProfile — a lightweight-charts v5 series primitive that renders a
 * Session Volume Profile (DEV + prior session POC/VAH/VAL) from OUR server-computed data.
 *
 * Portions adapted (API shape only) from the Apache-2.0 licensed
 * tradingview/lightweight-charts plugin example
 * (plugin-examples/src/plugins/volume-profile/volume-profile.ts).
 * Copyright 2024 TradingView, Inc. Licensed under the Apache License, Version 2.0.
 *
 * The DATA contract below (SvpBin / SvpSession / SvpProfileData) matches the Go
 * backend JSON exactly and is NOT the example's shape.
 */

import { CanvasRenderingTarget2D } from 'fancy-canvas'
import {
  AutoscaleInfo,
  Coordinate,
  IChartApi,
  IPrimitivePaneRenderer,
  IPrimitivePaneView,
  ISeriesApi,
  ISeriesPrimitive,
  ISeriesPrimitiveAxisView,
  Logical,
  SeriesAttachedParameter,
  SeriesType,
  Time,
  UTCTimestamp,
} from 'lightweight-charts'
import { positionsBox } from './positions'

// ---------------------------------------------------------------------------
// DATA CONTRACT — mirrors the Go JSON exactly. Do NOT change field names/shape.
// ---------------------------------------------------------------------------

export interface SvpBin {
  price: number
  vol: number
  inVA: boolean
}

export interface SvpSession {
  date: string
  partial: boolean
  poc: number
  vah: number
  val: number
  bins: SvpBin[]
  frozen: boolean
}

export interface SvpProfileData {
  rowHeight: number
  dev: SvpSession | null
  prior: SvpSession | null
  /** UTCTimestamp (seconds) x-anchor of the DEV histogram; may be null. */
  sessionStartTime: number | null
}

// ---------------------------------------------------------------------------
// Colors (all rgba; kept here so the chart/theme can reference them if needed).
// ---------------------------------------------------------------------------

const COLOR_VA = 'rgba(38, 166, 154, 0.55)' // teal — bins inside the Value Area
const COLOR_NON_VA = 'rgba(120, 123, 134, 0.35)' // muted grey — bins outside VA
const COLOR_POC_BAR = 'rgba(240, 185, 11, 0.8)' // gold — the max-volume (POC) bar
const COLOR_POC_LINE = 'rgba(240, 185, 11, 0.9)' // gold — POC level line
const COLOR_VA_LINE = 'rgba(38, 166, 154, 0.9)' // teal — VAH/VAL dashed lines
const COLOR_PRIOR_LINE = 'rgba(150, 150, 160, 0.6)' // muted — prior POC/VAH/VAL
const COLOR_LABEL_TEXT = 'rgba(0, 0, 0, 0.9)'

// Fraction of the pane width the widest (max-vol) DEV bar may occupy.
const MAX_BAR_WIDTH_FRACTION = 0.3

// ---------------------------------------------------------------------------
// Renderer data — everything the renderer needs, precomputed in media coords.
// ---------------------------------------------------------------------------

interface DevBarItem {
  y: Coordinate | null // center y of the bin
  halfThicknessPx: number // half of rowHeight expressed in px
  width: number // media px, already scaled to 0..(paneWidth*fraction)
  color: string
}

interface LevelLine {
  y: Coordinate | null
  color: string
  dashed: boolean
}

interface SvpRendererData {
  anchorX: Coordinate | null // DEV session-start x; null → fall back to left edge (0)
  bars: DevBarItem[]
  lines: LevelLine[]
}

// ---------------------------------------------------------------------------
// Renderer
// ---------------------------------------------------------------------------

class SessionVolumeProfileRenderer implements IPrimitivePaneRenderer {
  _data: SvpRendererData

  constructor(data: SvpRendererData) {
    this._data = data
  }

  draw(target: CanvasRenderingTarget2D): void {
    const data = this._data
    target.useBitmapCoordinateSpace((scope) => {
      const ctx = scope.context
      const hRatio = scope.horizontalPixelRatio
      const vRatio = scope.verticalPixelRatio
      const paneWidthBitmap = scope.bitmapSize.width

      // GUARD: null anchor → fall back to the left edge (media x = 0).
      const anchorXMedia = data.anchorX ?? (0 as Coordinate)
      const anchorXBitmap = Math.round(anchorXMedia * hRatio)

      // --- DEV histogram bars (draw first so level lines sit on top) ---
      for (const bar of data.bars) {
        if (bar.y === null || bar.width <= 0) continue

        const vertical = positionsBox(
          bar.y - bar.halfThicknessPx,
          bar.y + bar.halfThicknessPx,
          vRatio
        )
        const barWidthBitmap = Math.max(1, Math.round(bar.width * hRatio))

        ctx.fillStyle = bar.color
        ctx.fillRect(
          anchorXBitmap,
          vertical.position,
          barWidthBitmap,
          // shave 1px so adjacent rows don't visually merge
          Math.max(1, vertical.length - 1)
        )
      }

      // --- POC / VAH / VAL horizontal level lines across the full pane ---
      for (const line of data.lines) {
        if (line.y === null) continue

        const lineY = Math.round(line.y * vRatio)
        ctx.save()
        ctx.strokeStyle = line.color
        ctx.lineWidth = Math.max(1, Math.floor(vRatio))
        if (line.dashed) {
          ctx.setLineDash([Math.round(4 * hRatio), Math.round(4 * hRatio)])
        } else {
          ctx.setLineDash([])
        }
        ctx.beginPath()
        ctx.moveTo(0, lineY + 0.5)
        ctx.lineTo(paneWidthBitmap, lineY + 0.5)
        ctx.stroke()
        ctx.restore()
      }
    })
  }
}

// ---------------------------------------------------------------------------
// Price-axis label view (POC / VAH / VAL text)
// ---------------------------------------------------------------------------

class SvpPriceAxisView implements ISeriesPrimitiveAxisView {
  private _source: SessionVolumeProfile
  private _price: number
  private _text: string
  private _backColor: string

  constructor(
    source: SessionVolumeProfile,
    price: number,
    text: string,
    backColor: string
  ) {
    this._source = source
    this._price = price
    this._text = text
    this._backColor = backColor
  }

  coordinate(): number {
    const series = this._source.series()
    if (series === undefined) return -100
    const y = series.priceToCoordinate(this._price)
    return y ?? -100
  }

  text(): string {
    return this._text
  }

  textColor(): string {
    return COLOR_LABEL_TEXT
  }

  backColor(): string {
    return this._backColor
  }

  visible(): boolean {
    return Number.isFinite(this._price)
  }

  tickVisible(): boolean {
    return true
  }
}

// ---------------------------------------------------------------------------
// Pane view — recomputes media-space renderer data from _data each update.
// ---------------------------------------------------------------------------

class SessionVolumeProfilePaneView implements IPrimitivePaneView {
  private _source: SessionVolumeProfile
  private _rendererData: SvpRendererData = {
    anchorX: null,
    bars: [],
    lines: [],
  }

  constructor(source: SessionVolumeProfile) {
    this._source = source
  }

  update(): void {
    const chart = this._source.chart()
    const series = this._source.series()
    const data = this._source.data()

    if (chart === undefined || series === undefined) {
      this._rendererData = { anchorX: null, bars: [], lines: [] }
      return
    }

    const timeScale = chart.timeScale()

    // x-anchor of the DEV histogram.
    let anchorX: Coordinate | null = null
    if (data.sessionStartTime !== null) {
      anchorX = timeScale.timeToCoordinate(
        data.sessionStartTime as UTCTimestamp
      )
    }

    const paneWidth = timeScale.width()
    const maxBarWidth = paneWidth * MAX_BAR_WIDTH_FRACTION

    // thickness (px) for a single row of height `rowHeight` in price space.
    const rowHeight = data.rowHeight > 0 ? data.rowHeight : 0

    const bars: DevBarItem[] = []
    const dev = data.dev
    if (
      dev &&
      dev.bins &&
      dev.bins.length > 0 &&
      this._source.maxDevVol() > 0
    ) {
      const maxVol = this._source.maxDevVol()
      const pocPrice = dev.poc
      for (const bin of dev.bins) {
        const yCenter = series.priceToCoordinate(bin.price)
        // rowHeight → px thickness via two priceToCoordinate probes.
        let halfThickness = 1
        if (rowHeight > 0) {
          const yTop = series.priceToCoordinate(bin.price + rowHeight / 2)
          const yBot = series.priceToCoordinate(bin.price - rowHeight / 2)
          if (yTop !== null && yBot !== null) {
            halfThickness = Math.max(0.5, Math.abs(yBot - yTop) / 2)
          }
        }
        const isPoc = bin.price === pocPrice
        const color = isPoc ? COLOR_POC_BAR : bin.inVA ? COLOR_VA : COLOR_NON_VA
        bars.push({
          y: yCenter,
          halfThicknessPx: halfThickness,
          width: (bin.vol / maxVol) * maxBarWidth,
          color,
        })
      }
    }

    const lines: LevelLine[] = []
    if (dev) {
      lines.push(
        {
          y: series.priceToCoordinate(dev.poc),
          color: COLOR_POC_LINE,
          dashed: false,
        },
        {
          y: series.priceToCoordinate(dev.vah),
          color: COLOR_VA_LINE,
          dashed: true,
        },
        {
          y: series.priceToCoordinate(dev.val),
          color: COLOR_VA_LINE,
          dashed: true,
        }
      )
    }
    // PRIOR session: POC/VAH/VAL only (histogram intentionally skipped — see notes).
    const prior = data.prior
    if (prior) {
      lines.push(
        {
          y: series.priceToCoordinate(prior.poc),
          color: COLOR_PRIOR_LINE,
          dashed: true,
        },
        {
          y: series.priceToCoordinate(prior.vah),
          color: COLOR_PRIOR_LINE,
          dashed: true,
        },
        {
          y: series.priceToCoordinate(prior.val),
          color: COLOR_PRIOR_LINE,
          dashed: true,
        }
      )
    }

    this._rendererData = { anchorX, bars, lines }
  }

  renderer(): IPrimitivePaneRenderer | null {
    return new SessionVolumeProfileRenderer(this._rendererData)
  }
}

// ---------------------------------------------------------------------------
// Primitive
// ---------------------------------------------------------------------------

export class SessionVolumeProfile implements ISeriesPrimitive<Time> {
  private _chart: IChartApi | undefined
  private _series: ISeriesApi<SeriesType> | undefined
  private _requestUpdate: (() => void) | undefined

  private _data: SvpProfileData = {
    rowHeight: 0,
    dev: null,
    prior: null,
    sessionStartTime: null,
  }

  // Precomputed (in setData) so updateAllViews()/autoscaleInfo() stay O(1).
  private _maxDevVol = 0
  private _autoMin: number = Number.NaN
  private _autoMax: number = Number.NaN

  private _paneViews: SessionVolumeProfilePaneView[]
  private _priceAxisViews: SvpPriceAxisView[] = []

  constructor(
    chart?: IChartApi,
    series?: ISeriesApi<SeriesType>,
    data?: SvpProfileData
  ) {
    this._chart = chart
    this._series = series
    this._paneViews = [new SessionVolumeProfilePaneView(this)]
    if (data) {
      this.setData(data)
    }
  }

  // --- lifecycle ---------------------------------------------------------

  attached(param: SeriesAttachedParameter<Time>): void {
    this._chart = param.chart as unknown as IChartApi
    this._series = param.series as ISeriesApi<SeriesType>
    this._requestUpdate = param.requestUpdate
    this._rebuildAxisViews()
    this._requestUpdate?.()
  }

  detached(): void {
    this._requestUpdate = undefined
    this._chart = undefined
    this._series = undefined
  }

  // --- data --------------------------------------------------------------

  /**
   * Replace the profile data. Precomputes maxDevVol + autoscale min/max, then
   * asks the chart to redraw. Safe to call on every poll — it mutates internal
   * state instead of re-attaching (re-attaching would duplicate drawings).
   */
  setData(data: SvpProfileData): void {
    this._data = data

    // maxDevVol — used to normalize bar widths.
    let maxVol = 0
    if (data.dev && data.dev.bins) {
      for (const bin of data.dev.bins) {
        if (bin.vol > maxVol) maxVol = bin.vol
      }
    }
    this._maxDevVol = maxVol

    // autoscale union of dev + prior VAL..VAH (O(1); never scans bins).
    let min = Number.POSITIVE_INFINITY
    let max = Number.NEGATIVE_INFINITY
    const consider = (s: SvpSession | null): void => {
      if (!s) return
      if (Number.isFinite(s.val)) min = Math.min(min, s.val)
      if (Number.isFinite(s.vah)) max = Math.max(max, s.vah)
      // POC can sit outside VAL..VAH in edge cases; include it too.
      if (Number.isFinite(s.poc)) {
        min = Math.min(min, s.poc)
        max = Math.max(max, s.poc)
      }
    }
    consider(data.dev)
    consider(data.prior)
    this._autoMin = Number.isFinite(min) ? min : Number.NaN
    this._autoMax = Number.isFinite(max) ? max : Number.NaN

    this._rebuildAxisViews()
    this._requestUpdate?.()
  }

  data(): SvpProfileData {
    return this._data
  }

  maxDevVol(): number {
    return this._maxDevVol
  }

  chart(): IChartApi | undefined {
    return this._chart
  }

  series(): ISeriesApi<SeriesType> | undefined {
    return this._series
  }

  // --- views -------------------------------------------------------------

  updateAllViews(): void {
    for (const view of this._paneViews) {
      view.update()
    }
  }

  paneViews(): readonly IPrimitivePaneView[] {
    return this._paneViews
  }

  priceAxisViews(): readonly ISeriesPrimitiveAxisView[] {
    return this._priceAxisViews
  }

  autoscaleInfo(_start: Logical, _end: Logical): AutoscaleInfo | null {
    if (!Number.isFinite(this._autoMin) || !Number.isFinite(this._autoMax)) {
      return null
    }
    return {
      priceRange: {
        minValue: this._autoMin,
        maxValue: this._autoMax,
      },
    }
  }

  // --- internals ---------------------------------------------------------

  private _rebuildAxisViews(): void {
    const views: SvpPriceAxisView[] = []
    const fmt = (p: number): string => {
      // Trim to a sensible precision; futures like MNQ use quarter ticks.
      return Number.isInteger(p) ? String(p) : p.toFixed(2)
    }
    const dev = this._data.dev
    if (dev && dev.bins && dev.bins.length > 0) {
      views.push(
        new SvpPriceAxisView(
          this,
          dev.poc,
          `POC ${fmt(dev.poc)}`,
          COLOR_POC_LINE
        ),
        new SvpPriceAxisView(
          this,
          dev.vah,
          `VAH ${fmt(dev.vah)}`,
          COLOR_VA_LINE
        ),
        new SvpPriceAxisView(
          this,
          dev.val,
          `VAL ${fmt(dev.val)}`,
          COLOR_VA_LINE
        )
      )
    }
    const prior = this._data.prior
    if (prior) {
      views.push(
        new SvpPriceAxisView(
          this,
          prior.poc,
          `pPOC ${fmt(prior.poc)}`,
          COLOR_PRIOR_LINE
        ),
        new SvpPriceAxisView(
          this,
          prior.vah,
          `pVAH ${fmt(prior.vah)}`,
          COLOR_PRIOR_LINE
        ),
        new SvpPriceAxisView(
          this,
          prior.val,
          `pVAL ${fmt(prior.val)}`,
          COLOR_PRIOR_LINE
        )
      )
    }
    this._priceAxisViews = views
  }
}

export default SessionVolumeProfile
