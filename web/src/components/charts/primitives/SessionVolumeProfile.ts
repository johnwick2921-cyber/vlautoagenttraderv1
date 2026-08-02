/*
 * SessionVolumeProfile — a lightweight-charts v5 series primitive that renders
 * MULTIPLE Session Volume Profiles (one histogram per session/day) from OUR
 * server-computed data, in the style of TradingView's Session Volume Profile:
 * each session gets its own self-scaled histogram anchored at its session start,
 * a teal/red up-vs-down split per row, and a POC line spanning that session.
 *
 * Portions adapted (API shape only) from the Apache-2.0 licensed
 * tradingview/lightweight-charts plugin example
 * (plugin-examples/src/plugins/volume-profile/volume-profile.ts).
 * Copyright 2024 TradingView, Inc. Licensed under the Apache License, Version 2.0.
 *
 * The DATA contract below (SvpBin / SvpSession / SvpProfileData) matches the Go
 * backend JSON exactly (kernel/svp.go) and is NOT the example's shape.
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
  upVol: number
  downVol: number
  inVA: boolean
}

export interface SvpSession {
  /** UTC SECONDS — the chart x-anchor for this session. */
  sessionStart: number
  /** YYYY-MM-DD (session-day start, Chicago). */
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
  /** ALL sessions in the bar range, ascending by time; never null. */
  sessions: SvpSession[]
}

// ---------------------------------------------------------------------------
// Colors (all rgba; kept here so the chart/theme can reference them if needed).
// ---------------------------------------------------------------------------

const COLOR_UP = 'rgba(38, 166, 154, 0.75)' // teal — up-candle volume
const COLOR_DOWN = 'rgba(239, 83, 80, 0.75)' // red — down-candle volume
const COLOR_UP_POC = 'rgba(64, 200, 188, 0.95)' // brighter teal for the POC row
const COLOR_DOWN_POC = 'rgba(255, 110, 108, 0.95)' // brighter red for the POC row
const COLOR_POC_LINE = 'rgba(255, 255, 255, 0.9)' // white — POC level line
const COLOR_VA_LINE = 'rgba(38, 166, 154, 0.65)' // teal — VAH/VAL dashed lines
const COLOR_DIVIDER = 'rgba(120, 123, 134, 0.25)' // faint vertical session divider
const COLOR_LABEL_TEXT = 'rgba(0, 0, 0, 0.9)'

// Fraction of the pane width the widest bar of any single session may occupy.
const MAX_BAR_WIDTH_FRACTION = 0.18
// Fraction of the gap to the next session anchor that a bar may occupy.
const BAR_GAP_FRACTION = 0.9
// Minimum bar-width budget so a session squeezed against its neighbour is still legible.
const MIN_BAR_WIDTH_PX = 40

// ---------------------------------------------------------------------------
// Renderer data — everything the renderer needs, precomputed in media coords.
// ---------------------------------------------------------------------------

interface RowItem {
  y: Coordinate | null // center y of the bin
  halfThicknessPx: number // half of rowHeight expressed in px
  totalWidth: number // media px, already scaled to this session's budget
  upWidth: number // media px width of the teal (up-volume) segment
  isPoc: boolean
}

interface SessionRenderItem {
  anchorX: number // media x of this session's start
  endX: number // media x where this session's span ends (next anchor / pane edge)
  rows: RowItem[]
  pocY: Coordinate | null
  vahY: Coordinate | null
  valY: Coordinate | null
}

interface SvpRendererData {
  sessions: SessionRenderItem[]
  paneWidth: number
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

      for (const session of data.sessions) {
        const anchorXBitmap = Math.round(session.anchorX * hRatio)
        const endXBitmap = Math.round(session.endX * hRatio)

        // --- faint vertical divider at the session anchor ---
        ctx.save()
        ctx.strokeStyle = COLOR_DIVIDER
        ctx.lineWidth = Math.max(1, Math.floor(hRatio))
        ctx.setLineDash([])
        ctx.beginPath()
        ctx.moveTo(anchorXBitmap + 0.5, 0)
        ctx.lineTo(anchorXBitmap + 0.5, scope.bitmapSize.height)
        ctx.stroke()
        ctx.restore()

        // --- histogram rows (two-tone: up teal then down red) ---
        for (const row of session.rows) {
          if (row.y === null || row.totalWidth <= 0) continue

          const vertical = positionsBox(
            row.y - row.halfThicknessPx,
            row.y + row.halfThicknessPx,
            vRatio
          )
          const height = Math.max(1, vertical.length - 1)

          const upWidthBitmap = Math.max(0, Math.round(row.upWidth * hRatio))
          const totalWidthBitmap = Math.max(
            1,
            Math.round(row.totalWidth * hRatio)
          )
          const downWidthBitmap = Math.max(0, totalWidthBitmap - upWidthBitmap)

          // up segment (teal), extending RIGHT from the anchor
          if (upWidthBitmap > 0) {
            ctx.fillStyle = row.isPoc ? COLOR_UP_POC : COLOR_UP
            ctx.fillRect(
              anchorXBitmap,
              vertical.position,
              upWidthBitmap,
              height
            )
          }
          // down segment (red), stacked to the right of the up segment
          if (downWidthBitmap > 0) {
            ctx.fillStyle = row.isPoc ? COLOR_DOWN_POC : COLOR_DOWN
            ctx.fillRect(
              anchorXBitmap + upWidthBitmap,
              vertical.position,
              downWidthBitmap,
              height
            )
          }
        }

        // --- VAH / VAL dashed lines within the session span ---
        const dashOn = Math.round(3 * hRatio)
        const dashOff = Math.round(3 * hRatio)
        for (const levelY of [session.vahY, session.valY]) {
          if (levelY === null) continue
          const lineY = Math.round(levelY * vRatio)
          ctx.save()
          ctx.strokeStyle = COLOR_VA_LINE
          ctx.lineWidth = Math.max(1, Math.floor(vRatio))
          ctx.setLineDash([dashOn, dashOff])
          ctx.beginPath()
          ctx.moveTo(anchorXBitmap, lineY + 0.5)
          ctx.lineTo(endXBitmap, lineY + 0.5)
          ctx.stroke()
          ctx.restore()
        }

        // --- POC line (solid, spanning this session) — drawn last, on top ---
        if (session.pocY !== null) {
          const lineY = Math.round(session.pocY * vRatio)
          ctx.save()
          ctx.strokeStyle = COLOR_POC_LINE
          ctx.lineWidth = Math.max(1, Math.floor(vRatio))
          ctx.setLineDash([])
          ctx.beginPath()
          ctx.moveTo(anchorXBitmap, lineY + 0.5)
          ctx.lineTo(endXBitmap, lineY + 0.5)
          ctx.stroke()
          ctx.restore()
        }
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
  private _rendererData: SvpRendererData = { sessions: [], paneWidth: 0 }

  constructor(source: SessionVolumeProfile) {
    this._source = source
  }

  update(): void {
    const chart = this._source.chart()
    const series = this._source.series()
    const data = this._source.data()

    if (chart === undefined || series === undefined) {
      this._rendererData = { sessions: [], paneWidth: 0 }
      return
    }

    const timeScale = chart.timeScale()
    const paneWidth = timeScale.width()
    const sessions = data.sessions ?? []
    const maxVols = this._source.sessionMaxVols()
    const rowHeight = data.rowHeight > 0 ? data.rowHeight : 0

    // 1st pass: resolve each session's anchor x. The 17:00 CT session open often
    // isn't an EXACT chart bar (esp. on coarser intervals + the 16:00–17:00 CME
    // break), and timeScale.timeToCoordinate() returns null for a non-bar time —
    // so snap to the nearest actual bar within ±30 min (prefer forward = the
    // session's first bar). Skip only if nothing resolves (session off-screen).
    const resolveAnchor = (t0: number): number | null => {
      const direct = timeScale.timeToCoordinate(t0 as UTCTimestamp)
      if (direct !== null) return direct as number
      for (let off = 60; off <= 1800; off += 60) {
        const fwd = timeScale.timeToCoordinate((t0 + off) as UTCTimestamp)
        if (fwd !== null) return fwd as number
        const back = timeScale.timeToCoordinate((t0 - off) as UTCTimestamp)
        if (back !== null) return back as number
      }
      return null
    }
    const anchors: (number | null)[] = sessions.map((s) =>
      resolveAnchor(s.sessionStart)
    )

    const rendered: SessionRenderItem[] = []

    for (let i = 0; i < sessions.length; i++) {
      const anchorX = anchors[i]
      if (anchorX === null) continue // off the left edge — skip (never throw)

      const session = sessions[i]
      const maxVol = maxVols[i] ?? 0

      // Find the next VISIBLE anchor to the right (for span + width budget).
      let nextAnchorX: number | null = null
      for (let j = i + 1; j < sessions.length; j++) {
        if (anchors[j] !== null && (anchors[j] as number) > anchorX) {
          nextAnchorX = anchors[j] as number
          break
        }
      }

      // Bar-width budget: min(gap*0.9, pane*0.18) with a floor; last session
      // (no next anchor) uses pane*0.18. endX spans to next anchor / pane edge.
      let maxBarWidth: number
      let endX: number
      if (nextAnchorX !== null) {
        const gap = nextAnchorX - anchorX
        maxBarWidth = Math.max(
          MIN_BAR_WIDTH_PX,
          Math.min(gap * BAR_GAP_FRACTION, paneWidth * MAX_BAR_WIDTH_FRACTION)
        )
        endX = nextAnchorX
      } else {
        maxBarWidth = Math.max(
          MIN_BAR_WIDTH_PX,
          paneWidth * MAX_BAR_WIDTH_FRACTION
        )
        endX = paneWidth // naked POC extends to the right pane edge
      }

      const rows: RowItem[] = []
      if (maxVol > 0 && session.bins && session.bins.length > 0) {
        for (const bin of session.bins) {
          const yCenter = series.priceToCoordinate(bin.price)
          if (yCenter === null) continue // skip rows off the price axis

          // rowHeight → px thickness via two priceToCoordinate probes.
          let halfThickness = 1
          if (rowHeight > 0) {
            const yTop = series.priceToCoordinate(bin.price + rowHeight / 2)
            const yBot = series.priceToCoordinate(bin.price - rowHeight / 2)
            if (yTop !== null && yBot !== null) {
              halfThickness = Math.max(0.5, Math.abs(yBot - yTop) / 2)
            }
          }

          const totalWidth = (bin.vol / maxVol) * maxBarWidth
          // Split into up/down segments proportional to upVol/vol.
          const vol = bin.vol > 0 ? bin.vol : 1
          const upFrac = Math.min(1, Math.max(0, bin.upVol / vol))
          rows.push({
            y: yCenter,
            halfThicknessPx: halfThickness,
            totalWidth,
            upWidth: totalWidth * upFrac,
            isPoc: bin.price === session.poc,
          })
        }
      }

      rendered.push({
        anchorX,
        endX,
        rows,
        pocY: series.priceToCoordinate(session.poc),
        vahY: series.priceToCoordinate(session.vah),
        valY: series.priceToCoordinate(session.val),
      })
    }

    this._rendererData = { sessions: rendered, paneWidth }
  }

  renderer(): IPrimitivePaneRenderer | null {
    return new SessionVolumeProfileRenderer(this._rendererData)
  }

  zOrder(): 'top' {
    return 'top'
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
    sessions: [],
  }

  // Precomputed (in setData) so updateAllViews()/autoscaleInfo() stay O(1).
  // Per-session max(bin.vol), index-aligned with _data.sessions.
  private _sessionMaxVols: number[] = []
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
   * Replace the profile data. Precomputes per-session maxVol + autoscale
   * min/max, then asks the chart to redraw. Safe to call on every poll — it
   * mutates internal state instead of re-attaching (re-attaching would
   * duplicate drawings).
   */
  setData(data: SvpProfileData): void {
    this._data = {
      rowHeight: data.rowHeight,
      sessions: data.sessions ?? [],
    }

    // Per-session max(bin.vol) — each histogram scales to itself.
    const maxVols: number[] = []
    let min = Number.POSITIVE_INFINITY
    let max = Number.NEGATIVE_INFINITY
    for (const s of this._data.sessions) {
      let sMax = 0
      if (s.bins) {
        for (const bin of s.bins) {
          if (bin.vol > sMax) sMax = bin.vol
        }
      }
      maxVols.push(sMax)

      // autoscale union of every session's VAL..VAH (+POC guard); O(#sessions).
      if (Number.isFinite(s.val)) min = Math.min(min, s.val)
      if (Number.isFinite(s.vah)) max = Math.max(max, s.vah)
      if (Number.isFinite(s.poc)) {
        min = Math.min(min, s.poc)
        max = Math.max(max, s.poc)
      }
    }
    this._sessionMaxVols = maxVols
    this._autoMin = Number.isFinite(min) ? min : Number.NaN
    this._autoMax = Number.isFinite(max) ? max : Number.NaN

    this._rebuildAxisViews()
    this._requestUpdate?.()
  }

  data(): SvpProfileData {
    return this._data
  }

  sessionMaxVols(): number[] {
    return this._sessionMaxVols
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
    // Label the LAST (most recent) session's POC/VAH/VAL only — avoids clutter.
    const sessions = this._data.sessions
    const last = sessions.length > 0 ? sessions[sessions.length - 1] : null
    if (last) {
      views.push(
        new SvpPriceAxisView(
          this,
          last.poc,
          `POC ${fmt(last.poc)}`,
          COLOR_POC_LINE
        ),
        new SvpPriceAxisView(
          this,
          last.vah,
          `VAH ${fmt(last.vah)}`,
          COLOR_VA_LINE
        ),
        new SvpPriceAxisView(
          this,
          last.val,
          `VAL ${fmt(last.val)}`,
          COLOR_VA_LINE
        )
      )
    }
    this._priceAxisViews = views
  }
}

export default SessionVolumeProfile
