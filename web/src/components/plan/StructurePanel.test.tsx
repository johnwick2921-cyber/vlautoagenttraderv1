// S5 — StructurePanel pins: renders NOTHING when the structure block is absent
// (no placeholder rows), and one row per TF with trend + zone TF badges when
// present. Mirrors the S1 contract field names verbatim.

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StructurePanel, structureZonesToOverlay } from './StructurePanel'
import type { StructureMapView } from '../../lib/api/plan'

const structure: StructureMapView = {
  as_of_ms: 1760000000000,
  contract: 'MNQ 12-26',
  tfs: {
    D: {
      trend: 'up',
      last_swing_high: { price: 29600, time_ms: 1759999000000 },
      last_swing_low: { price: 29200, time_ms: 1759900000000 },
      impulse_lo: 29200,
      impulse_hi: 29600,
      premium_discount: 0.25,
      zones: [{ kind: 'Supply', lo: 29550, hi: 29600, tf: 'D', fresh: 'fresh', score: 3.1 }],
      bars: 60,
    },
    '4h': { trend: 'down', premium_discount: 0.6, zones: [], bars: 120 },
    '1h': { trend: 'range', premium_discount: 0.5, zones: [{ kind: 'OB', lo: 29400, hi: 29440, tf: '1h', fresh: 'tested-1', score: 1.9 }], bars: 40 },
  },
}

describe('StructurePanel', () => {
  it('renders NOTHING when the structure block is absent', () => {
    const { container } = render(<StructurePanel structure={undefined} />)
    expect(container).toBeEmptyDOMElement()
    render(<StructurePanel structure={null} />)
    expect(screen.queryByTestId('structure-panel')).toBeNull()
  })

  it('renders the bias-only header and one row per TF with badges', () => {
    render(<StructurePanel structure={structure} />)
    expect(screen.getByTestId('structure-panel')).toBeTruthy()
    expect(screen.getByText(/STRUCTURE — bias only, not entries/)).toBeTruthy()
    expect(screen.getByTestId('structure-tf-D')).toBeTruthy()
    expect(screen.getByTestId('structure-tf-4h')).toBeTruthy()
    expect(screen.getByTestId('structure-tf-1h')).toBeTruthy()
    expect(screen.getAllByTestId('structure-zone')).toHaveLength(2)
    const badges = screen.getAllByTestId('structure-zone-tf').map((n) => n.textContent)
    expect(badges).toEqual(['D', '1h'])
    expect(screen.getByText(/pd 25%/)).toBeTruthy()
  })

  it('maps zones to chart overlay entries with kind·tf labels and raw bands', () => {
    const overlay = structureZonesToOverlay(structure)
    expect(overlay).toHaveLength(2)
    expect(overlay[0].label).toBe('Supply\u00b7D')
    expect(overlay[0].range).toEqual([29550, 29600])
    expect(overlay[1].label).toBe('OB\u00b71h')
    expect(structureZonesToOverlay(undefined)).toEqual([])
  })

  // D1 (review): the dev freshness vocabulary — '' is fresh and grades A,
  // 'b' grades B; the chip renders 'fresh' for the empty string.
  it('grades the dev freshness vocabulary, not the S2 display words', () => {
    const freshBlank: StructureMapView = {
      as_of_ms: 1760000000000,
      tfs: {
        '4h': {
          trend: 'range',
          premium_discount: 0,
          bars: 1,
          zones: [
            { kind: 'OB', lo: 1, hi: 2, tf: '4h', fresh: '', score: 1 },
            { kind: 'FVG', lo: 2, hi: 3, tf: '4h', fresh: 'b', score: 1 },
            { kind: 'Supply', lo: 3, hi: 4, tf: '4h', fresh: 'consumed', score: 1 },
          ],
        },
      },
    }
    const overlay = structureZonesToOverlay(freshBlank)
    expect(overlay[0].grade).toBe('A')
    expect(overlay[1].grade).toBe('B')
    expect(overlay[2].grade).toBe('C')
    render(<StructurePanel structure={freshBlank} />)
    expect(screen.getAllByTestId('structure-zone')[0].textContent).toContain('fresh')
  })

  // D2 (review): no impulse lo/hi -> no pd text, ever.
  it('prints no pd when the impulse range is absent', () => {
    const noImpulse: StructureMapView = {
      as_of_ms: 1760000000000,
      tfs: { '4h': { trend: 'range', premium_discount: 0, bars: 1, zones: [] } },
    }
    const { container } = render(<StructurePanel structure={noImpulse} />)
    expect(container.textContent).not.toContain('pd')
  })
})
