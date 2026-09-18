// W-CHART-ZONE-WALL (zones OFF by default, owner 2026-09-17 after the CLASS
// 143 build): a 40-zone structure block draws the seated lines and ZERO bands
// by default; "Show zones (30)" is OFF, clicking it draws ≤ 6 bands and
// reports "6 of 30"; the choice is remembered per viewer in localStorage.
//
// jsdom has no canvas, so the chart degrades to its placeholder; the zone
// control renders on the placeholder too, which is what this pins.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { PlanMiniChart, factsToOverlay } from './PlanMiniChart'
import { SHOW_ZONES_KEY, selectChartOverlay } from './chartOverlaySelect'
import type { OverlayLevel } from './LevelOverlayPrimitive'
import type { PlanLevelFact } from '../../lib/api/plan'
import { httpClient } from '../../lib/httpClient'

vi.spyOn(httpClient, 'request').mockResolvedValue({
  success: true,
  data: [],
} as never)

const facts: PlanLevelFact[] = Array.from({ length: 12 }, (_, i) => ({
  price: 29000 + i * 25,
  label: `L${i}`,
  grade: 'A',
  instruction: 'watch',
  distance: 0,
  sweep: false,
  closes_beyond: 0,
  accept_have: 0,
  accept_need: 0,
  still_valid: true,
}))

function fortyZones(): OverlayLevel[] {
  const distinct: OverlayLevel[] = Array.from({ length: 30 }, (_, i) => {
    const lo = 28000 + i * 100
    return {
      price: lo + 40,
      label: i % 2 ? 'SUPPLY·1h' : 'DEMAND·4h',
      grade: 'A',
      range: [lo, lo + 40] as [number, number],
    }
  })
  return [...distinct, ...distinct.slice(0, 10).map((z) => ({ ...z }))]
}

describe('PlanMiniChart · zone wall', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  const mount = () =>
    render(
      <PlanMiniChart
        symbol="MNQ"
        exchange="ninjatrader"
        facts={facts}
        language="en"
        structureZones={fortyZones()}
      />
    )

  it('DEFAULT: seated lines only, zero zone bands, control OFF and no count caption', () => {
    mount()
    const box = screen.getByTestId('chart-show-zones') as HTMLInputElement
    expect(box.checked).toBe(false)
    expect(screen.getByTestId('chart-zone-control').textContent).toContain(
      'Show zones (30)'
    )
    expect(screen.queryByTestId('chart-zone-count')).toBeNull()
    // the SAME selection the component hands the overlay, on the same inputs
    const sel = selectChartOverlay(
      factsToOverlay(facts),
      fortyZones(),
      undefined,
      { showZones: false }
    )
    expect(sel.levels).toHaveLength(12)
    expect(sel.levels.filter((l) => l.range)).toHaveLength(0)
  })

  it('"Show zones" ON draws ≤ 6 bands, reports "6 of 30", and is remembered', () => {
    mount()
    fireEvent.click(screen.getByTestId('chart-show-zones'))
    expect(screen.getByTestId('chart-zone-count').textContent).toContain(
      '6 of 30'
    )
    expect(window.localStorage.getItem(SHOW_ZONES_KEY)).toBe('1')
    const sel = selectChartOverlay(
      factsToOverlay(facts),
      fortyZones(),
      undefined,
      { showZones: true }
    )
    expect(sel.levels.length).toBeLessThanOrEqual(12 + 6)
    expect(sel.levels.filter((l) => l.range)).toHaveLength(6)
    // a fresh mount reads the preference back
    mount()
    const boxes = screen.getAllByTestId(
      'chart-show-zones'
    ) as HTMLInputElement[]
    expect(boxes[boxes.length - 1].checked).toBe(true)
  })

  it('no zones → no control (nothing to show)', () => {
    render(
      <PlanMiniChart
        symbol="MNQ"
        exchange="ninjatrader"
        facts={facts}
        language="en"
        structureZones={[]}
      />
    )
    expect(screen.queryByTestId('chart-zone-control')).toBeNull()
  })
})
