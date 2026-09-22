// W-ROLL-DAY-CHART (2026-09-19) — the chart's roll-day rendering logic: the
// {klines, roll} envelope unwrap, the one-line legend from the envelope only,
// and the muted derived-bar styling + raw/adjusted tooltip data.

import { describe, it, expect } from 'vitest'
import {
  unwrapKlinesResponse,
  rollLegend,
  candleFromRow,
  candleBarStyle,
} from './PlanMiniChart'

describe('unwrapKlinesResponse', () => {
  it('passes a bare array through untouched (crypto / legacy knob)', () => {
    const arr = [{ openTime: 1, open: 2, high: 3, low: 1, close: 2.5 }]
    expect(unwrapKlinesResponse(arr)).toEqual({ rows: arr })
  })
  it('unwraps the ninjatrader envelope', () => {
    const env = {
      klines: [{ openTime: 2, open: 3, high: 4, low: 2, close: 3.5 }],
      roll: { derived: true, adjusted: true, basis: 290.25 },
    }
    expect(unwrapKlinesResponse(env)).toEqual({
      rows: env.klines,
      roll: env.roll,
    })
  })
})

describe('rollLegend', () => {
  it('reads the month and basis from the envelope, no literals', () => {
    expect(
      rollLegend({ derived: true, adjusted: true, basis: 290.25 }, 'MNQ 09-26')
    ).toBe('pre-roll (Sep) · basis-adjusted +290.25')
  })
  it('says unadjusted with the reason when no basis was applied', () => {
    expect(
      rollLegend(
        {
          derived: true,
          adjusted: false,
          reason: 'basis pair unmeasurable within 5 minutes',
        },
        'MNQ 09-26'
      )
    ).toBe(
      'pre-roll (Sep) · unadjusted (basis pair unmeasurable within 5 minutes)'
    )
  })
  it('renders nothing without a derived segment', () => {
    expect(rollLegend(undefined, 'MNQ 09-26')).toBe('')
    expect(rollLegend({ derived: false, adjusted: false }, 'MNQ 09-26')).toBe(
      ''
    )
  })
})

describe('candleFromRow + candleBarStyle', () => {
  const roll = { derived: true, adjusted: true, basis: 290 }
  it('marks derived bars and carries the raw close for adjusted bars', () => {
    const c = candleFromRow(
      {
        openTime: 1726000000000,
        open: 30290,
        high: 30295,
        low: 30285,
        close: 30293,
        derived: true,
        adjusted: true,
        contract: 'MNQ 09-26',
      },
      roll
    )
    expect(c.derived).toBe(true)
    expect(c.rawClose).toBeCloseTo(30293 - 290, 6)
    expect(candleBarStyle(c)).toEqual({
      color: '#6B7078',
      borderColor: '#6B7078',
      wickColor: '#6B7078',
    })
  })
  it('leaves current-contract bars untouched', () => {
    const c = candleFromRow(
      {
        openTime: 1726000000001,
        open: 30290,
        high: 30295,
        low: 30285,
        close: 30293,
        contract: 'MNQ 12-26',
      },
      roll
    )
    expect(c.derived).toBeUndefined()
    expect(c.rawClose).toBeUndefined()
    expect(candleBarStyle(c)).toEqual({})
  })
})
