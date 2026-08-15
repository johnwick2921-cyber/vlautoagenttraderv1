// P4.3 — SessionPlanCard + sub-component tests. The card is a pure view; these
// assert the states (loading/night/no-plan/error/active) and the level/scenario
// rendering. The mini chart is skipped here (facts empty or non-ninjatrader) so
// jsdom's missing canvas never matters.

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SessionPlanCard } from './SessionPlanCard'
import { ZoneTable } from './ZoneTable'
import { ScenarioList } from './ScenarioList'
import { BiasBlock } from './BiasBlock'
import { classifyLevel } from './LevelOverlayPrimitive'
import { levelFresh, levelNear, fmtDistance } from './levelState'
import type { PlanToday, PlanLevelFact, PlanScenario } from '../../lib/api/plan'

const fact = (over: Partial<PlanLevelFact>): PlanLevelFact => ({
  price: 30000,
  label: 'PDH',
  grade: 'A',
  instruction: 'fade',
  distance: 57,
  sweep: false,
  closes_beyond: 0,
  accept_have: 0,
  accept_need: 2,
  still_valid: true,
  ...over,
})

const scenario: PlanScenario = {
  id: 'S1',
  trigger: 'sweep the PDH then reclaim',
  condition: 'sweep_reclaim',
  direction: 'short',
  target_chain: [29950, 29900],
  invalid: '2x5m > 30020',
  quality: 'A+',
}

const activePlan: PlanToday = {
  found: true,
  trade_date: '2026-08-15',
  session: 'NY',
  night: false,
  mode: 'advisory',
  version: 2,
  lifecycle: 'active',
  model_id: 'deepseek-reasoner',
  replans_left: 1,
  warming: '',
  doc: {
    reasoning: 'range day',
    bias: {
      direction: 'short',
      conviction: 'medium',
      flip_condition: 'flips long on 2x5m > 30100',
    },
    levels: [],
    scenarios: [scenario],
    no_trade: ['11:00–13:00 CT lunch'],
    death_condition: '2x5m acceptance > 30100',
    day_type: 'range',
  },
  level_facts: [
    fact({ price: 30057, label: 'PDH', distance: 5 }),
    fact({
      price: 29900,
      label: 'nPOC·Tue',
      grade: 'B',
      distance: -100,
      still_valid: false,
    }),
  ],
}

describe('levelState', () => {
  it('derives fresh/tested/consumed', () => {
    expect(levelFresh(fact({}))).toBe('fresh')
    expect(levelFresh(fact({ sweep: true }))).toBe('tested')
    expect(levelFresh(fact({ still_valid: false }))).toBe('consumed')
  })
  it('near uses the distance band', () => {
    expect(levelNear(fact({ distance: 5 }))).toBe(true)
    expect(levelNear(fact({ distance: 57 }))).toBe(false)
  })
  it('fmtDistance signs correctly', () => {
    expect(fmtDistance(57)).toBe('+57')
    expect(fmtDistance(-12)).toBe('−12')
    expect(fmtDistance(0)).toBe('0')
  })
})

describe('classifyLevel', () => {
  it('routes provenance to a line kind', () => {
    expect(classifyLevel('PDH')).toBe('structural')
    expect(classifyLevel('nPOC·Tue')).toBe('volume')
    expect(classifyLevel('RN 30000')).toBe('minor')
    expect(classifyLevel('EQH')).toBe('minor')
  })
})

describe('BiasBlock', () => {
  it('renders the direction word + flip', () => {
    render(
      <BiasBlock
        bias={{
          direction: 'short',
          conviction: 'medium',
          flip_condition: 'flips long on 2x5m > 30100',
        }}
        language="en"
      />
    )
    expect(screen.getByText('SHORT')).toBeInTheDocument()
    expect(screen.getByText(/flips long/i)).toBeInTheDocument()
  })
})

describe('ZoneTable', () => {
  it('renders one row per level with provenance + distance', () => {
    render(<ZoneTable facts={activePlan.level_facts!} language="en" />)
    expect(screen.getByText('PDH')).toBeInTheDocument()
    expect(screen.getByText('nPOC·Tue')).toBeInTheDocument()
    expect(screen.getByText('+5')).toBeInTheDocument()
  })
  it('shows an empty state when there are no levels', () => {
    render(<ZoneTable facts={[]} language="en" />)
    expect(screen.getByText(/No levels/i)).toBeInTheDocument()
  })
})

describe('ScenarioList', () => {
  it('renders scenario id + targets + defaults to armed status', () => {
    render(<ScenarioList scenarios={[scenario]} language="en" />)
    expect(screen.getByText('S1')).toBeInTheDocument()
    expect(screen.getByText(/29950 → 29900/)).toBeInTheDocument()
    expect(screen.getAllByText(/armed/i).length).toBeGreaterThan(0)
  })
  it('honors a backend status map', () => {
    render(
      <ScenarioList
        scenarios={[scenario]}
        statusMap={{ S1: 'triggered' }}
        language="en"
      />
    )
    expect(screen.getAllByText(/triggered/i).length).toBeGreaterThan(0)
  })
})

describe('SessionPlanCard states', () => {
  it('renders the loading state', () => {
    render(
      <SessionPlanCard
        plan={null}
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
        isLoading
      />
    )
    expect(screen.getByText(/Loading plan/i)).toBeInTheDocument()
  })
  it('renders the night state', () => {
    render(
      <SessionPlanCard
        plan={{
          found: false,
          trade_date: '2026-08-15',
          session: '',
          night: true,
          mode: 'advisory',
        }}
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByText(/markets quiet/i)).toBeInTheDocument()
  })
  it('renders the no-plan-yet state for an enabled session with no plan', () => {
    render(
      <SessionPlanCard
        plan={{
          found: false,
          trade_date: '2026-08-15',
          session: 'NY',
          night: false,
          mode: 'advisory',
        }}
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getByText(/No plan yet/i)).toBeInTheDocument()
  })
  it('renders the fail-closed banner', () => {
    const failed: PlanToday = {
      ...activePlan,
      doc: {
        ...activePlan.doc!,
        day_type: 'no-trade',
        reasoning: 'FAIL-CLOSED: read error',
      },
    }
    render(
      <SessionPlanCard
        plan={failed}
        symbol="MNQ"
        exchange="ninjatrader"
        language="en"
      />
    )
    expect(screen.getAllByText(/sitting out/i).length).toBeGreaterThan(0)
  })
  it('renders an active plan (bias + levels + scenarios), no chart when non-ninjatrader', () => {
    render(
      <SessionPlanCard
        plan={activePlan}
        symbol="MNQ"
        exchange="binance"
        language="en"
      />
    )
    expect(screen.getByText('SHORT')).toBeInTheDocument()
    expect(screen.getByText('S1')).toBeInTheDocument()
    expect(screen.getByText(/lunch/i)).toBeInTheDocument() // no-trade window
    expect(screen.getByText('deepseek-reasoner')).toBeInTheDocument()
  })
})
