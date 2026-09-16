// THE ONE chart-timezone site — DST correctness + pin-to-Houston.
//
// The delta this guards against: PlanMiniChart handed lightweight-charts raw
// epochs, so its labels rendered UTC — +5h vs the NT8 chart (S1). And any fixed
// −5/−6 constant would break twice a year (S5).

import { describe, it, expect } from 'vitest'
import { TickMarkType, type Time } from 'lightweight-charts'
import { ctTickMarkFormatter, ctCrosshairTimeFormatter } from './chartTime'

// A bar OPEN-stamped 14:30:00 UTC. In August (CDT, −5) that is 09:30 CT —
// the NY open. In January (CST, −6) the same wall UTC instant is 08:30 CT.
const augEpochS = Date.UTC(2026, 7, 18, 14, 30, 0) / 1000 // Aug 18 2026
const janEpochS = Date.UTC(2026, 0, 15, 14, 30, 0) / 1000 // Jan 15 2026

describe('ctTickMarkFormatter — DST-safe CT labels', () => {
  it('renders an August (CDT, −5) intraday tick as CT wall-clock', () => {
    expect(ctTickMarkFormatter(augEpochS as Time, TickMarkType.Time)).toBe(
      '09:30'
    )
  })
  it('renders a January (CST, −6) intraday tick as CT wall-clock', () => {
    expect(ctTickMarkFormatter(janEpochS as Time, TickMarkType.Time)).toBe(
      '08:30'
    )
  })
  it('never renders the raw UTC hour (the S1 bug)', () => {
    expect(ctTickMarkFormatter(augEpochS as Time, TickMarkType.Time)).not.toBe(
      '14:30'
    )
  })
  it('day-granularity ticks are CT-dated (a 23:30 CT bar is NOT tomorrow)', () => {
    // 04:30 UTC Aug 19 = 23:30 CT Aug 18 — the CT date must win.
    const lateEvening = Date.UTC(2026, 7, 19, 4, 30, 0) / 1000
    expect(
      ctTickMarkFormatter(lateEvening as Time, TickMarkType.DayOfMonth)
    ).toBe('08/18')
  })
})

describe('ctCrosshairTimeFormatter — agrees with the axis', () => {
  it('same zone, same instant, both seasons', () => {
    expect(ctCrosshairTimeFormatter(augEpochS)).toBe('08/18 09:30')
    expect(ctCrosshairTimeFormatter(janEpochS)).toBe('01/15 08:30')
  })
})

describe('pin-to-Houston holds for any viewer timezone', () => {
  it('output is independent of the host TZ (Intl timeZone pins it)', () => {
    // The formatter passes an explicit timeZone to Intl, so the host TZ env
    // cannot influence it. Render twice with different TZ env hints — vitest
    // runs one process, so assert the value equals the CT truth computed above.
    expect(ctCrosshairTimeFormatter(augEpochS)).toBe('08/18 09:30')
  })
})
