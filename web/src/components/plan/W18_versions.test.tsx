// ITEM 15 — plan version history.
//
// The chips were inert <span>s whose labels were fabricated from a single
// integer, collapsing at n>=4 to "v1 … vN" so v2..v5 were unreachable — and the
// owner was at v6 with five superseded versions they could not open. There was
// no route serving an old version's doc, no store method listing versions, and
// no FE hook: the whole read path was missing while every version sat durable in
// an append-only table.

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SessionPlanCard } from './SessionPlanCard'
import { VersionChips } from './chips'
import type { PlanToday, PlanVersionItem } from '../../lib/api/plan'

// The real 2026-08-16:ASIA history: v1..v5 each died, v6 is the NO-TRADE plan.
const asiaVersions: PlanVersionItem[] = [1, 2, 3, 4, 5, 6].map((v) => ({
  version: v,
  lifecycle: v === 6 ? 'no_trade' : 'active',
  trigger_reason: v === 1 ? 'session open' : 'death condition',
  created_at: `2026-08-16T22:1${v}:00Z`,
  model_id: 'deepseek-reasoner',
  is_latest: v === 6,
  level_count: v === 6 ? 0 : 6,
  scenario_count: v === 6 ? 1 : 3,
  bias: v === 6 ? 'neutral' : 'long',
  death_condition: '1h close below 30146.75 PDL',
  ...(v < 6
    ? {
        superseded_by: v + 1,
        death_reason:
          v === 5
            ? 'replans_exhausted'
            : 'death condition (all levels consumed)',
        diff_vs_next:
          v === 5
            ? ['bias long → neutral', 'NO LEVELS in the replacement']
            : ['same 6 levels'],
      }
    : {}),
}))

const historicalPlan = (version: number): PlanToday => ({
  found: true,
  trade_date: '2026-08-16',
  session: 'ASIA',
  night: false,
  mode: 'advisory',
  version,
  latest_version: 6,
  historical: version !== 6,
  lifecycle: version === 6 ? 'no_trade' : 'active',
  model_id: 'deepseek-reasoner',
  replans_left: 0,
  is_active: true,
  warming: '',
  doc: {
    reasoning: 'range day',
    bias: { direction: 'long', conviction: 'medium', flip_condition: 'n/a' },
    levels: [],
    scenarios: [],
    no_trade: [],
    death_condition: '1h close below 30146.75 PDL',
    day_type: 'range',
  },
  level_facts: [],
})

describe('VersionChips', () => {
  it('renders EVERY version as a button — none hidden behind an ellipsis', () => {
    render(
      <VersionChips version={6} latest={6} count={6} onSelect={() => {}} />
    )
    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(6)
    expect(buttons.map((b) => b.textContent)).toEqual([
      'v1',
      'v2',
      'v3',
      'v4',
      'v5',
      'v6',
    ])
    // The old component collapsed at n>=4 and rendered a bare '…'.
    expect(screen.queryByText('…')).toBeNull()
  })

  it('marks the viewed version with aria-current, not colour alone', () => {
    render(
      <VersionChips version={2} latest={6} count={6} onSelect={() => {}} />
    )
    const current = screen
      .getAllByRole('button')
      .filter((b) => b.getAttribute('aria-current') === 'true')
    expect(current).toHaveLength(1)
    expect(current[0].textContent).toBe('v2')
  })

  it('calls onSelect with the clicked version', () => {
    const onSelect = vi.fn()
    render(
      <VersionChips version={6} latest={6} count={6} onSelect={onSelect} />
    )
    fireEvent.click(screen.getByText('v3'))
    expect(onSelect).toHaveBeenCalledWith(3)
  })

  it('stays inert (disabled) when no handler is supplied', () => {
    render(<VersionChips version={2} latest={2} count={2} />)
    screen.getAllByRole('button').forEach((b) => expect(b).toBeDisabled())
  })

  it('renders nothing at version 0', () => {
    const { container } = render(<VersionChips version={0} />)
    expect(container.firstChild).toBeNull()
  })

  it('uses the wrapping row class rather than a nowrap flex', () => {
    // The header overflowed its 316px interior at 390px with six chips; wrapping
    // is what keeps the ✎/💬 buttons on the card.
    const { container } = render(<VersionChips version={6} count={6} />)
    expect(container.querySelector('.vl-version-row')).toBeTruthy()
    expect(container.querySelectorAll('.vl-version-chip')).toHaveLength(6)
  })
})

describe('SessionPlanCard — historical version', () => {
  const renderCard = (version: number, onSelectVersion = vi.fn()) => {
    render(
      <SessionPlanCard
        plan={historicalPlan(version)}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={onSelectVersion}
      />
    )
    return onSelectVersion
  }

  it('marks a superseded version as HISTORICAL and read-only', () => {
    renderCard(3)
    expect(screen.getByTestId('historical-banner')).toBeInTheDocument()
    expect(screen.getByText(/HISTORICAL VERSION/)).toBeInTheDocument()
  })

  it('shows the death reason and what changed next', () => {
    renderCard(5)
    expect(screen.getByTestId('death-reason').textContent).toContain(
      'replans_exhausted'
    )
    expect(screen.getByTestId('death-reason').textContent).toContain(
      'Replaced by v6'
    )
    expect(screen.getByTestId('version-diff').textContent).toContain(
      'NO LEVELS in the replacement'
    )
  })

  it('offers the way back to the current version', () => {
    const onSelectVersion = renderCard(2)
    fireEvent.click(screen.getByTestId('back-to-active'))
    expect(onSelectVersion).toHaveBeenCalledWith(6)
  })

  it('CLOSES THE OWNER DOOR while a historical version is open', () => {
    // Every mutating endpoint writes the LATEST version of the active session,
    // so an edit made while reading v2 would silently land on v6.
    renderCard(2)
    screen
      .getAllByRole('button')
      .filter((b) => ['✎', '💬'].includes(b.textContent ?? ''))
      .forEach((b) => expect(b).toBeDisabled())
  })

  it('shows no historical banner on the current version', () => {
    renderCard(6)
    expect(screen.queryByTestId('historical-banner')).toBeNull()
  })
})

// ITEM 1 (2026-08-17) — THE CARD RENDERS TWO CHIP ROWS.
//
// Click-to-open shipped wired to the header only, so the FOOTER row stayed a set
// of disabled buttons. Tapping those did nothing — indistinguishable, from the
// owner's chair, from "the chips are still not clickable". Both rows must work.
describe('SessionPlanCard — both chip rows are live', () => {
  it('every version chip on the card is clickable, header AND footer', () => {
    const onSelectVersion = vi.fn()
    render(
      <SessionPlanCard
        plan={historicalPlan(6)}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={onSelectVersion}
      />
    )
    const rows = document.querySelectorAll('.vl-version-row')
    expect(rows.length).toBe(2) // header + footer

    rows.forEach((row, i) => {
      const chips = row.querySelectorAll<HTMLButtonElement>('.vl-version-chip')
      expect(chips).toHaveLength(6)
      chips.forEach((c) =>
        expect(
          c.disabled,
          `row ${i} chip ${c.textContent} must not be disabled`
        ).toBe(false)
      )
    })

    // A click on the FOOTER row opens that version, exactly like the header.
    const footerChips =
      rows[1].querySelectorAll<HTMLButtonElement>('.vl-version-chip')
    fireEvent.click(footerChips[2])
    expect(onSelectVersion).toHaveBeenCalledWith(3)
  })

  it('both rows agree on which version is current', () => {
    render(
      <SessionPlanCard
        plan={historicalPlan(2)}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={vi.fn()}
      />
    )
    document.querySelectorAll('.vl-version-row').forEach((row) => {
      const current = row.querySelectorAll('[aria-current="true"]')
      expect(current).toHaveLength(1)
      expect(current[0].textContent).toBe('v2')
    })
  })
})

// ITEM 2d — the death history must be visible on the LIVE card.
//
// Six plans died in 25 minutes on 2026-08-16 and the card said nothing about it:
// the owner saw only the final levels-less v6. A death streak must never again be
// invisible.
describe('SessionPlanCard — death history', () => {
  it('lists every superseded version with the condition that killed it', () => {
    render(
      <SessionPlanCard
        plan={historicalPlan(6)}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={vi.fn()}
      />
    )
    const panel = screen.getByTestId('death-history')
    expect(panel.textContent).toContain('lost 5 plan(s)')
    for (const v of [1, 2, 3, 4, 5]) {
      expect(screen.getByTestId(`death-row-v${v}`)).toBeInTheDocument()
    }
    expect(screen.getByTestId('death-row-v5').textContent).toContain(
      'replans_exhausted'
    )
    // The current version is not a death.
    expect(screen.queryByTestId('death-row-v6')).toBeNull()
  })

  it('opens a dead version straight from its row', () => {
    const onSelectVersion = vi.fn()
    render(
      <SessionPlanCard
        plan={historicalPlan(6)}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={onSelectVersion}
      />
    )
    fireEvent.click(screen.getByTestId('death-row-v4'))
    expect(onSelectVersion).toHaveBeenCalledWith(4)
  })

  it('shows nothing when the session has lost no plans', () => {
    render(
      <SessionPlanCard
        plan={historicalPlan(1)}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={[{ ...asiaVersions[0], version: 1, is_latest: true }]}
        latestVersion={1}
        onSelectVersion={vi.fn()}
      />
    )
    expect(screen.queryByTestId('death-history')).toBeNull()
  })
})

// ITEM 1 — THE NO-TRADE ROW IS A TERMINAL MARKER, NOT A PLAN VERSION.
//
// The owner set replan_cap = 4, saw a row labelled "v6", and read it as "the cap
// didn't work". It did: cap N allows N re-plans, i.e. real versions v1..v(N+1),
// and the (N+1)th death writes a NO-TRADE marker which consumes a version number
// because the plans table is append-only. Showing that marker as a plain "v6" is
// what made correct behaviour look broken.
describe('SessionPlanCard — the NO-TRADE marker', () => {
  const noTradePlan: PlanToday = {
    ...historicalPlan(6),
    lifecycle: 'no_trade',
    replan_cap: 4,
    replans_left: 0,
    doc: {
      ...historicalPlan(6).doc!,
      reasoning:
        'FAIL-CLOSED: re-plans exhausted (4/4) after 5 deaths — last: all 6 levels touched and accepted through',
    },
  }

  it('renders the marker as NO-TRADE, never as a version number', () => {
    render(
      <SessionPlanCard
        plan={noTradePlan}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={vi.fn()}
      />
    )
    const marker = document.querySelectorAll('[data-no-trade="1"]')
    expect(marker.length).toBe(2) // header + footer rows
    marker.forEach((m) => expect(m.textContent).toContain('NO-TRADE'))
    // No chip anywhere may still call it "v6".
    document
      .querySelectorAll('.vl-version-chip')
      .forEach((c) => expect(c.textContent).not.toBe('v6'))
  })

  it('states the budget honestly: 4 of 4 spent, not 5', () => {
    render(
      <SessionPlanCard
        plan={noTradePlan}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={vi.fn()}
      />
    )
    const banner = screen.getByTestId('no-trade-banner').textContent!
    expect(banner).toContain('re-read budget exhausted')
    // The marker consumes a version, so version-1 would overcount by one.
    expect(banner).toContain('4 of 4 re-plans')
  })

  it('keeps the planner reason visible under the new framing', () => {
    render(
      <SessionPlanCard
        plan={noTradePlan}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={vi.fn()}
      />
    )
    expect(screen.getByTestId('fail-closed-reason').textContent).toContain(
      're-plans exhausted (4/4)'
    )
  })

  it('shows no NO-TRADE framing on a normal active plan', () => {
    render(
      <SessionPlanCard
        plan={{ ...historicalPlan(6), lifecycle: 'active' }}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions.map((v) => ({ ...v, lifecycle: 'active' }))}
        latestVersion={6}
        onSelectVersion={vi.fn()}
      />
    )
    expect(screen.queryByTestId('no-trade-banner')).toBeNull()
    expect(document.querySelectorAll('[data-no-trade="1"]').length).toBe(0)
  })
})

// ITEM 4 (2026-08-17) — edits that could not carry must be VISIBLE, never dropped.
describe('SessionPlanCard — uncarried owner edits', () => {
  it('asks for review, naming each edit', () => {
    render(
      <SessionPlanCard
        plan={{
          ...historicalPlan(6),
          lifecycle: 'active',
          uncarried_edits: [
            {
              op: 'replace',
              path: '/bias/direction',
              reason: 'structural edits have no price to re-anchor to',
              summary: 'your edit to /bias/direction could not carry — review',
            },
            {
              op: 'remove',
              path: '/levels[30100]',
              reason: 'you removed this level and the new plan re-added it',
              summary:
                'you removed junk at 30100; the new plan added it back — review',
            },
          ],
        }}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={vi.fn()}
      />
    )
    const panel = screen.getByTestId('uncarried-edits')
    expect(panel.textContent).toContain('2 edit(s) could not carry')
    expect(panel.textContent).toContain('/bias/direction')
    expect(panel.textContent).toContain('added it back')
  })

  it('shows nothing when everything carried', () => {
    render(
      <SessionPlanCard
        plan={{ ...historicalPlan(6), lifecycle: 'active' }}
        traderId="t1"
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        versions={asiaVersions}
        latestVersion={6}
        onSelectVersion={vi.fn()}
      />
    )
    expect(screen.queryByTestId('uncarried-edits')).toBeNull()
  })
})
