/*
 * LevelOverlayPrimitive — a lightweight-charts v5 series primitive that draws
 * the day-plan's levels on the mini chart. It is a focused clone of
 * SessionVolumeProfile.ts (same four-class shape: Renderer / AxisView / PaneView
 * / Primitive) adapted for plan levels.
 *
 * CONTRACT: it consumes the SAME level array the ZoneTable renders (props only,
 * never fetches) so the card and the chart can never diverge (design-system
 * "one array, three renderers" rule).
 *
 * Line style by provenance: structural levels (PDH/PDL/ONH/ONL/RTH…) → solid;
 * volume levels (POC/nPOC) → dashed; round numbers / equal-highs (RN/EQH/EQL) →
 * hairline. Grade drives opacity (A brightest). Zone BANDS + scenario markers
 * activate when a level carries a [proximal,distal] range / scenario linkage
 * (owner-overlay data, P5) — absent now, so only lines + axis labels draw.
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
} from 'lightweight-charts'

export type LevelKind = 'structural' | 'volume' | 'minor'

export interface OverlayLevel {
  price: number
  label: string
  grade: string // A | B | C
  instruction?: string
  /** optional zone band [proximal, distal] — owner-overlay (P5); absent now. */
  range?: [number, number]
}

export interface LevelOverlayData {
  levels: OverlayLevel[]
}

// Provenance → line kind. Case-insensitive prefix match on the label.
export function classifyLevel(label: string): LevelKind {
  const l = (label || '').toUpperCase()
  if (l.includes('POC')) return 'volume'
  if (l.startsWith('RN') || l.startsWith('EQ')) return 'minor'
  return 'structural'
}

const GOLD = '201,162,75' // --vl-gold rgb
const gradeAlpha = (grade: string): number =>
  grade === 'A' ? 0.9 : grade === 'B' ? 0.6 : 0.4

interface LineItem {
  y: Coordinate
  kind: LevelKind
  alpha: number
}

interface BandItem {
  yTop: Coordinate
  yBot: Coordinate
  alpha: number
}

interface OverlayRendererData {
  lines: LineItem[]
  bands: BandItem[]
  paneWidth: number
}

class LevelOverlayRenderer implements IPrimitivePaneRenderer {
  _data: OverlayRendererData
  constructor(data: OverlayRendererData) {
    this._data = data
  }

  draw(target: CanvasRenderingTarget2D): void {
    const data = this._data
    target.useBitmapCoordinateSpace((scope) => {
      const ctx = scope.context
      const hRatio = scope.horizontalPixelRatio
      const vRatio = scope.verticalPixelRatio
      const rightX = Math.round(scope.bitmapSize.width)

      // zone bands first (behind the lines)
      for (const band of data.bands) {
        const top = Math.round(band.yTop * vRatio)
        const bot = Math.round(band.yBot * vRatio)
        ctx.save()
        ctx.fillStyle = `rgba(${GOLD},${band.alpha})`
        ctx.fillRect(
          0,
          Math.min(top, bot),
          rightX,
          Math.max(1, Math.abs(bot - top))
        )
        ctx.restore()
      }

      // level lines, full pane width
      for (const line of data.lines) {
        const y = Math.round(line.y * vRatio)
        ctx.save()
        ctx.strokeStyle = `rgba(${GOLD},${line.alpha})`
        ctx.lineWidth = Math.max(
          1,
          Math.floor(vRatio) *
            (line.kind === 'minor' ? 1 : line.kind === 'volume' ? 1 : 2)
        )
        if (line.kind === 'volume')
          ctx.setLineDash([Math.round(4 * hRatio), Math.round(4 * hRatio)])
        else if (line.kind === 'minor')
          ctx.setLineDash([Math.round(1 * hRatio), Math.round(3 * hRatio)])
        else ctx.setLineDash([])
        ctx.beginPath()
        ctx.moveTo(0, y + 0.5)
        ctx.lineTo(rightX, y + 0.5)
        ctx.stroke()
        ctx.restore()
      }
    })
  }
}

class LevelAxisView implements ISeriesPrimitiveAxisView {
  constructor(
    private _source: LevelOverlayPrimitive,
    private _price: number,
    private _text: string
  ) {}
  coordinate(): number {
    const s = this._source.series()
    if (!s) return -100
    return s.priceToCoordinate(this._price) ?? -100
  }
  text(): string {
    return this._text
  }
  textColor(): string {
    return 'rgba(12,14,18,0.95)'
  }
  backColor(): string {
    return `rgba(${GOLD},0.9)`
  }
  visible(): boolean {
    return Number.isFinite(this._price)
  }
  tickVisible(): boolean {
    return true
  }
}

class LevelOverlayPaneView implements IPrimitivePaneView {
  private _rendererData: OverlayRendererData = {
    lines: [],
    bands: [],
    paneWidth: 0,
  }
  constructor(private _source: LevelOverlayPrimitive) {}

  update(): void {
    const chart = this._source.chart()
    const series = this._source.series()
    if (!chart || !series) {
      this._rendererData = { lines: [], bands: [], paneWidth: 0 }
      return
    }
    const paneWidth = chart.timeScale().width()
    const lines: LineItem[] = []
    const bands: BandItem[] = []
    for (const lvl of this._source.data().levels) {
      const y = series.priceToCoordinate(lvl.price)
      if (y === null) continue
      lines.push({
        y,
        kind: classifyLevel(lvl.label),
        alpha: gradeAlpha(lvl.grade),
      })
      if (lvl.range) {
        const yTop = series.priceToCoordinate(lvl.range[0])
        const yBot = series.priceToCoordinate(lvl.range[1])
        if (yTop !== null && yBot !== null)
          bands.push({ yTop, yBot, alpha: 0.1 })
      }
    }
    this._rendererData = { lines, bands, paneWidth }
  }

  renderer(): IPrimitivePaneRenderer | null {
    return new LevelOverlayRenderer(this._rendererData)
  }
  zOrder(): 'top' {
    return 'top'
  }
}

export class LevelOverlayPrimitive implements ISeriesPrimitive<Time> {
  private _chart: IChartApi | undefined
  private _series: ISeriesApi<SeriesType> | undefined
  private _requestUpdate: (() => void) | undefined
  private _data: LevelOverlayData = { levels: [] }
  private _autoMin = Number.NaN
  private _autoMax = Number.NaN
  private _paneViews: LevelOverlayPaneView[]
  private _axisViews: LevelAxisView[] = []

  constructor() {
    this._paneViews = [new LevelOverlayPaneView(this)]
  }

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

  setData(data: LevelOverlayData): void {
    this._data = { levels: data.levels ?? [] }
    let min = Number.POSITIVE_INFINITY
    let max = Number.NEGATIVE_INFINITY
    for (const l of this._data.levels) {
      if (Number.isFinite(l.price)) {
        min = Math.min(min, l.price)
        max = Math.max(max, l.price)
      }
    }
    this._autoMin = Number.isFinite(min) ? min : Number.NaN
    this._autoMax = Number.isFinite(max) ? max : Number.NaN
    this._rebuildAxisViews()
    this._requestUpdate?.()
  }

  data(): LevelOverlayData {
    return this._data
  }
  chart(): IChartApi | undefined {
    return this._chart
  }
  series(): ISeriesApi<SeriesType> | undefined {
    return this._series
  }

  updateAllViews(): void {
    for (const v of this._paneViews) v.update()
  }
  paneViews(): readonly IPrimitivePaneView[] {
    return this._paneViews
  }
  priceAxisViews(): readonly ISeriesPrimitiveAxisView[] {
    return this._axisViews
  }
  autoscaleInfo(_s: Logical, _e: Logical): AutoscaleInfo | null {
    if (!Number.isFinite(this._autoMin) || !Number.isFinite(this._autoMax))
      return null
    return { priceRange: { minValue: this._autoMin, maxValue: this._autoMax } }
  }

  private _rebuildAxisViews(): void {
    const fmt = (p: number) => (Number.isInteger(p) ? String(p) : p.toFixed(2))
    // Label grade-A structural levels only (avoids axis clutter).
    this._axisViews = this._data.levels
      .filter((l) => l.grade === 'A')
      .slice(0, 6)
      .map(
        (l) => new LevelAxisView(this, l.price, `${l.label} ${fmt(l.price)}`)
      )
  }
}

export default LevelOverlayPrimitive
