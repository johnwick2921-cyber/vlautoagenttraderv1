// W-T1-CURRENCIES (2026-09-18) — a red event outside the hard-block currency
// set is an ADVISORY line: always visible on the card, never a window, never
// hidden behind the model-notes toggle. Born the night the BOJ rate decision
// (JPY) hard-blocked the MNQ bot.

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { RulesBlock, type NoTradeBandWindow } from './RulesBlock'

const BOJ_ADVISORY =
  '🟠 BOJ Policy Rate 21:54 CT (JPY) — red news, advisory only (t1_currencies=USD)'
const BAND: NoTradeBandWindow[] = [
  {
    start_min: 1185,
    end_min: 1215,
    start_ct: '19:45',
    end_ct: '20:15',
    kind: 't1',
    label: 'Fed Chair Powell Speaks 20:00 CT ±15m',
    source: 'calendar',
    status: 'elapsed',
  },
]

describe('RulesBlock — T1 advisory line', () => {
  it('shows the advisory beside the band, not behind the notes toggle', () => {
    const { container } = render(
      <RulesBlock
        noTrade={['first 5m', BOJ_ADVISORY]}
        band={BAND}
        deathCondition=""
        language="en"
      />
    )
    expect(screen.getByTestId('no-trade-advisory').textContent).toContain(
      BOJ_ADVISORY
    )
    // the model prose is still collapsed, and the advisory is not counted as a note
    expect(container.textContent).not.toContain('first 5m')
    expect(
      screen.getByRole('button', { name: /model notes/i }).textContent
    ).toContain('(1)')
  })

  it('renders on a doc with no machine band too', () => {
    render(
      <RulesBlock noTrade={[BOJ_ADVISORY]} deathCondition="" language="en" />
    )
    expect(screen.getByTestId('no-trade-advisory').textContent).toContain(
      'BOJ Policy Rate'
    )
  })
})
