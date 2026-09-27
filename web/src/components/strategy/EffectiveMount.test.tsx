// W1 (g) — the effective chip MOUNTED in the real Studio editors.
//
// The map comes from a fixture in the server's exact wire shape
// (__fixtures__/strategyEffective.fixture.json — hand-written from the
// production resolvers, see its _about) and is built by studioEffective, the
// SAME function the page's hook (useStudioEffective) calls. Assertions are on
// what the real RiskControlEditor / DayPlanEditor render.

import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import fixture from './__fixtures__/strategyEffective.fixture.json'
import { RiskControlEditor } from './RiskControlEditor'
import { DayPlanEditor } from './DayPlanEditor'
import {
  studioEffective,
  type EffectiveKnob,
  type StrategyEffectiveResponse,
  type StudioEffective,
} from '../../lib/api/strategyEffective'
import type { RiskControlConfig } from '../../types'
import type { DayPlanConfig } from '../../types/strategy'

type Scope = 'strategy' | 'NY' | 'ASIA' | 'LONDON'
const R = fixture.responses as unknown as Record<
  Scope,
  StrategyEffectiveResponse
>

function lookup(): StudioEffective {
  return studioEffective(R.strategy, {
    NY: R.NY,
    ASIA: R.ASIA,
    LONDON: R.LONDON,
  })
}

function chips(root: ParentNode = document): HTMLElement[] {
  return Array.from(
    root.querySelectorAll<HTMLElement>('[data-testid="effective-chip"]')
  )
}

function chipAt(path: string, root: ParentNode = document) {
  return root.querySelector<HTMLElement>(
    `[data-testid="effective-chip"][data-path="${path}"]`
  )
}

/** The chip of the row that holds `el`: the first ancestor carrying a chip. */
function rowChip(el: HTMLElement): HTMLElement {
  let cur: HTMLElement | null = el
  while (cur) {
    const c = cur.querySelector<HTMLElement>('[data-testid="effective-chip"]')
    if (c) return c
    cur = cur.parentElement
  }
  throw new Error('no chip above this element')
}

// ── RiskControlEditor ─────────────────────────────────────────────────────

const RC = 'ai_config.risk_control.'
const RC_MOUNTED = [
  'hold_discipline',
  'breakeven_enabled',
  'breakeven_trigger_points',
  'trailing_enabled',
  'trailing_atr_mult',
  'trailing_atr_period',
  'trailing_arm',
  'trailing_arm_points',
  'max_positions',
  'min_risk_reward_ratio',
  'min_confidence',
  'guardrails_enabled',
  'daily_loss_enabled',
  'daily_loss_limit_usd',
  'daily_profit_enabled',
  'daily_profit_target_usd',
  'max_daily_trades_enabled',
  'max_daily_trades',
  'consecutive_loss_halt',
  'reentry_cooldown_minutes',
  'consistency_enabled',
  'consistency_max_day_pct',
  'max_contracts_per_order',
  'max_notional_leverage',
  'blackout_enabled',
  'blackout_start_ct',
  'blackout_end_ct',
].map((l) => RC + l)

// `null` = render WITHOUT an effective map (undefined would take the default).
function renderRisk(
  config: RiskControlConfig = {} as RiskControlConfig,
  effective: StudioEffective | null = lookup()
) {
  const onChange = vi.fn()
  const utils = render(
    <RiskControlEditor
      config={config}
      onChange={onChange}
      language="en"
      isFutures
      effective={effective ?? undefined}
    />
  )
  return { onChange, ...utils }
}

describe('RiskControlEditor — effective chips (W1 g)', () => {
  it('mounts exactly one chip per covered row, keyed to the server path', () => {
    renderRisk()
    const paths = chips().map((c) => c.getAttribute('data-path'))
    expect([...paths].sort()).toEqual([...RC_MOUNTED].sort())
  })

  it('renders a saved value, a shipped default, an env origin, a clamp and a suspension as the server sent them', () => {
    renderRisk()
    expect(chipAt(RC + 'min_risk_reward_ratio')).toHaveTextContent(
      'eff 2.5 · saved value · strategy'
    )
    expect(chipAt(RC + 'breakeven_trigger_points')).toHaveTextContent(
      'eff 50 · shipped default · venue:ninjatrader'
    )
    expect(chipAt(RC + 'consecutive_loss_halt')).toHaveTextContent(
      'eff 5 · env BREAKER_HALT_N · process env'
    )
    expect(chipAt(RC + 'max_contracts_per_order')).toHaveTextContent(
      'eff 1 · clamp (kernel.ClampStageAContracts) · process env'
    )
    expect(chipAt(RC + 'breakeven_enabled')).toHaveTextContent(
      'eff off · suspended (EXIT_MECHS_SUSPENDED) — saved true not used · process env'
    )
    // the breaker chip sits in the breaker's own card
    expect(rowChip(screen.getByTestId('breaker-halt-input'))).toBe(
      chipAt(RC + 'consecutive_loss_halt')
    )
  })

  it('min confidence: the literal "unset/0 → default 60" hint is gone; the chip prints the server value', () => {
    const { unmount } = renderRisk({} as RiskControlConfig)
    expect(screen.queryByText(/unset\/0 → default 60/)).toBeNull()
    expect(chipAt(RC + 'min_confidence')).toHaveTextContent(
      'eff 60 · clamp (StrategyConfig.ClampLimits) · strategy'
    )
    unmount()
    // no map at all: still no literal default, and no chip
    renderRisk({} as RiskControlConfig, null)
    expect(screen.queryByText(/default 60/)).toBeNull()
    expect(chipAt(RC + 'min_confidence')).toBeNull()
  })

  it('futures panel: the "≤ ?? 10" / "equity × ?? 20" literals are gone — it prints the server value, n/a without one', () => {
    const { unmount } = renderRisk({} as RiskControlConfig)
    expect(screen.getByTestId('futures-panel-contracts').textContent).toBe(
      '≤ 1'
    )
    expect(screen.getByTestId('futures-panel-notional').textContent).toBe(
      'equity × 20'
    )
    expect(screen.queryByText(/≤ 10/)).toBeNull()
    unmount()

    // Unsaved form edits do not move it: the server read the SAVED row.
    const { unmount: u2 } = renderRisk({
      max_contracts_per_order: 3,
      max_notional_leverage: 25,
    } as RiskControlConfig)
    expect(screen.getByTestId('futures-panel-contracts').textContent).toBe(
      '≤ 1'
    )
    expect(screen.getByTestId('futures-panel-notional').textContent).toBe(
      'equity × 20'
    )
    u2()

    // No map: n/a, never a typed fallback.
    renderRisk({} as RiskControlConfig, null)
    expect(screen.getByTestId('futures-panel-contracts').textContent).toBe(
      'n/a'
    )
    expect(screen.getByTestId('futures-panel-notional').textContent).toBe('n/a')
    expect(screen.queryByText(/≤ 10/)).toBeNull()
    expect(screen.queryByText(/equity × 20/)).toBeNull()
  })

  it('a row absent from the map renders no chip and no empty line; no map renders no chip at all', () => {
    const eff = lookup()
    delete eff.byPath[RC + 'hold_discipline']
    const { unmount } = renderRisk({} as RiskControlConfig, eff)
    expect(chipAt(RC + 'hold_discipline')).toBeNull()
    expect(chips()).toHaveLength(RC_MOUNTED.length - 1)
    // every rendered line carries a chip — nothing empty was mounted
    expect(screen.getAllByTestId('effective-line')).toHaveLength(
      RC_MOUNTED.length - 1
    )
    unmount()

    renderRisk({} as RiskControlConfig, null)
    expect(chips()).toHaveLength(0)
    expect(screen.queryAllByTestId('effective-line')).toHaveLength(0)
    expect(screen.queryByTestId('effective-saved-note')).toBeNull()
  })

  it('says the chips read the SAVED strategy (note + every chip title)', () => {
    renderRisk()
    expect(screen.getByTestId('effective-saved-note')).toHaveTextContent(
      /SAVED strategy/
    )
    for (const c of chips())
      expect(c.getAttribute('title')).toContain('saved strategy config')
  })

  it('write semantics are unchanged with the chips mounted: breaker OFF still writes 0', () => {
    const config = {} as RiskControlConfig
    const { onChange } = renderRisk(config)
    fireEvent.click(screen.getByTestId('breaker-toggle'))
    expect(onChange).toHaveBeenCalledWith({
      ...config,
      consecutive_loss_halt: 0,
    })
  })
})

// ── DayPlanEditor ─────────────────────────────────────────────────────────

const DP_MOUNTED = [
  'structural_stop.buffer_points',
  'plan_enabled',
  'plan_mode',
  'planner_timeframes',
  'proximity_filter_atr',
  'max_levels',
  'htf_seats',
  'flip_reread',
  'death_reread',
  't1_currencies',
  'replan_cap',
  'approval_required',
  'wake_on_level_events',
  'min_scenario_quality',
  'one_setup_enabled',
  'one_setup_min_grade',
  'picture_htf.enabled',
  'picture_htf.tick_size',
  'picture_htf.pivot_window',
  'picture_htf.swing_lookback',
  'picture_htf.entry_window_sec',
  'picture_htf.freshness_sec',
  'picture_htf.min_rr',
].map((l) => 'day_plan.' + l)
const SESSION_BODY = [
  'min_grade',
  'min_scenario_quality',
  'max_trades',
  'plan_mode',
  'replan_cap',
].map((l) => 'day_plan.sessions.' + l)

// The stored day_plan the fixture was resolved from — with the picture knobs
// switched on in the FORM so their rows render.
const DP_CONFIG: DayPlanConfig = {
  ...(fixture.stored_config.day_plan as DayPlanConfig),
  picture_htf: { enabled: true },
}

function renderPlan(
  config: DayPlanConfig = DP_CONFIG,
  effective: StudioEffective | null = lookup()
) {
  const onChange = vi.fn()
  const utils = render(
    <DayPlanEditor
      config={config}
      onChange={onChange}
      language="en"
      effective={effective ?? undefined}
    />
  )
  return { onChange, ...utils }
}

function sessionBox(s: string): HTMLElement {
  // Toggle → header row → the session's bordered box
  return screen.getByTestId(`session-enable-${s}`).parentElement!
    .parentElement as HTMLElement
}

function openSession(s: string) {
  fireEvent.click(
    sessionBox(s).querySelector('button[aria-expanded]') as HTMLElement
  )
}

describe('DayPlanEditor — effective chips (W1 g)', () => {
  it('mounts one chip per covered strategy row + the per-session rows (NY open by default)', () => {
    renderPlan()
    const paths = chips().map((c) => c.getAttribute('data-path') as string)
    const strategyLevel = paths.filter(
      (p) => !p.startsWith('day_plan.sessions.')
    )
    expect([...strategyLevel].sort()).toEqual([...DP_MOUNTED].sort())
    // enable chip in each of the 3 session headers + NY's 5 body rows
    expect(paths.filter((p) => p === 'day_plan.sessions.enable')).toHaveLength(
      3
    )
    for (const p of SESSION_BODY)
      expect(paths.filter((x) => x === p)).toHaveLength(1)
  })

  it('opening ASIA and LONDON mounts their five body rows from THEIR reads', () => {
    renderPlan()
    openSession('ASIA') // accordion: one open at a time
    const asia = sessionBox('ASIA')
    for (const p of SESSION_BODY) expect(chipAt(p, asia)).not.toBeNull()
    expect(chipAt('day_plan.sessions.replan_cap', asia)).toHaveTextContent(
      'eff 1 · session override · session:ASIA'
    )
    expect(chipAt('day_plan.sessions.max_trades', asia)).toHaveTextContent(
      'eff no per-session cap · shipped default · strategy'
    )
    openSession('LONDON')
    const london = sessionBox('LONDON')
    for (const p of SESSION_BODY) expect(chipAt(p, london)).not.toBeNull()
    expect(chipAt('day_plan.sessions.enable', london)).toHaveTextContent(
      'eff off · shipped default (sessions_enabled [NY]) · strategy'
    )
  })

  it('a session override on a session row; the strategy row reads the session-less answer', () => {
    renderPlan()
    const ny = sessionBox('NY')
    expect(
      rowChip(screen.getByTestId('session-plan-mode-NY'))
    ).toHaveTextContent('eff strict · session override · session:NY')
    expect(chipAt('day_plan.sessions.max_trades', ny)).toHaveTextContent(
      'eff 3 · session override · session:NY'
    )
    expect(chipAt('day_plan.sessions.enable', ny)).toHaveTextContent(
      'eff on · shipped default (sessions_enabled [NY]) · strategy'
    )
    // The global plan-mode row is answered by the read WITHOUT a session —
    // never by the NY read's "strict · session override".
    const global = chips().filter(
      (c) => c.getAttribute('data-path') === 'day_plan.plan_mode'
    )
    expect(global).toHaveLength(1)
    expect(global[0]).toHaveTextContent(
      'eff advisory · strategy value · strategy'
    )
  })

  it('saved values, shipped defaults and a clamp render as sent', () => {
    renderPlan()
    expect(chipAt('day_plan.max_levels')).toHaveTextContent(
      'eff 10 · saved value · strategy'
    )
    expect(chipAt('day_plan.t1_currencies')).toHaveTextContent(
      'eff USD, EUR · saved value · strategy'
    )
    expect(chipAt('day_plan.planner_timeframes')).toHaveTextContent(
      'eff D, 4h, 1h, 15m · shipped default · strategy'
    )
    expect(chipAt('day_plan.replan_cap')).toHaveTextContent(
      'eff 2 · shipped default · strategy'
    )
    expect(chipAt('day_plan.picture_htf.min_rr')).toHaveTextContent(
      'eff 2.5 · clamp (trader.pictureMinRR: min_risk_reward_ratio floor) · venue:ninjatrader'
    )
    expect(chipAt('day_plan.structural_stop.buffer_points')).toHaveTextContent(
      'eff 4.5 · shipped default (ResolveStructuralStop:C5_MNQ_default[I]) · venue:ninjatrader'
    )
  })

  it('an n/a row renders the n/a sentence — never blank, never 0', () => {
    const eff = lookup()
    // As sent: the venue is unknown, so the master switch is n/a.
    // Synthetic (a row the server has no resolver for, in the server's form):
    const t1 = eff.byPath['day_plan.t1_currencies'] as EffectiveKnob
    eff.byPath['day_plan.t1_currencies'] = {
      ...t1,
      effective: 'n/a — no resolver registered',
      origin: 'saved value',
      resolver: 'none',
      resolved: false,
    }
    renderPlan(DP_CONFIG, eff)
    const master = chipAt('day_plan.plan_enabled') as HTMLElement
    expect(master.textContent).toBe(
      'n/a — venue unknown (the day plan runs on ninjatrader only)'
    )
    const none = chipAt('day_plan.t1_currencies') as HTMLElement
    expect(none.textContent).toBe('n/a — no resolver registered')
    for (const c of [master, none]) {
      expect(c.textContent).not.toMatch(/\b0\b/)
      expect(c.textContent).not.toContain('eff')
    }
  })

  it('a row absent from the map renders no chip; a session with no read renders none', () => {
    const eff = lookup()
    delete eff.byPath['day_plan.max_levels']
    delete eff.bySession.NY
    renderPlan(DP_CONFIG, eff)
    expect(chipAt('day_plan.max_levels')).toBeNull()
    expect(screen.getByDisplayValue('10')).toBeInTheDocument() // row still there
    const ny = sessionBox('NY')
    expect(chips(ny)).toHaveLength(0)
    // every line that did render carries a chip
    expect(screen.getAllByTestId('effective-line')).toHaveLength(chips().length)
  })

  it('no map → no chips, no note, rows as before', () => {
    renderPlan(DP_CONFIG, null)
    expect(chips()).toHaveLength(0)
    expect(screen.queryAllByTestId('effective-line')).toHaveLength(0)
    expect(screen.queryByTestId('effective-saved-note')).toBeNull()
  })

  it('write semantics are unchanged with the chips mounted: clearing the replan cap writes null', () => {
    const { onChange } = renderPlan({ ...DP_CONFIG, replan_cap: 3 })
    fireEvent.change(screen.getByTestId('replan-cap-strategy'), {
      target: { value: '' },
    })
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ replan_cap: null })
    )
  })
})
