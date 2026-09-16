import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LevelZoneMap } from './LevelZoneMap'

describe('frozen level zones', () => {
  it('renders both source names and honest missing widths without dropping a context band', () => {
    render(
      <LevelZoneMap
        map={{
          at: '2026-09-11T10:30:27-05:00',
          detected: 2,
          broad: 1,
          merged: 1,
          null_widths: 1,
          widest_merged: 0,
          zones: [
            {
              anchor: 29006.625,
              lo: 28810.75,
              hi: 29202.5,
              incomplete_width: true,
              broad: true,
              family_count: 1,
              prior_touches: null,
              rank_value: null,
              shortlisted: false,
              sources: [
                {
                  kind: 'SWG-H',
                  tf: '5m',
                  price: 29475,
                  label: 'SWG-H·5m',
                  formed_at: null,
                  lo: null,
                  hi: null,
                  width_rule: 'NULL: defining wick unavailable',
                },
                {
                  kind: 'SWG-H',
                  tf: '15m',
                  price: 29475,
                  label: 'SWG-H·15m',
                  formed_at: null,
                  lo: null,
                  hi: null,
                  width_rule: 'NULL: defining wick unavailable',
                },
              ],
            },
          ],
        }}
      />
    )
    expect(screen.getByText('28810.75–29202.50')).toBeTruthy()
    expect(screen.getByText(/SWG-H·5m/)).toBeTruthy()
    expect(screen.getByText(/SWG-H·15m/)).toBeTruthy()
    expect(screen.getByText(/Prior touches: UNKNOWN/)).toBeTruthy()
  })
})
