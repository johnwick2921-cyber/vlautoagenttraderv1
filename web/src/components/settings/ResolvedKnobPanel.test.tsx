// The Settings panel that renders the knob registry.
//
// Each test pins a way the panel could mislead the operator: collapsing the two
// "does not matter" statuses into one word, printing a saved value it borrowed
// from the resolved side, or claiming "nothing resolves here" when the truth is
// that nothing was asked.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { ResolvedKnobPanel } from './ResolvedKnobPanel'

const payload = {
  summary: {
    schema: 57,
    classified: 167,
    live: 144,
    ineffective: 7,
    candidate_unverified: 16,
    suspended: 0,
    advisory: 0,
    display_only: 0,
    infra: 0,
    env_shadows: 0,
    env_shadow_paths: [],
  },
  knobs: [
    {
      path: 'risk_control.max_margin_usage',
      status: 'ineffective',
      ui_label:
        'read; does not take effect (prompt text only — advisory, never a gate (engine_prompt.go))',
      consumers: ['kernel/engine_prompt.go:412'],
      dual_level: false,
    },
    {
      path: 'day_plan.wake_on_ifvg',
      status: 'candidate-unverified',
      ui_label: 'no known reader — pending verification',
      consumers: [],
      dual_level: false,
    },
    {
      path: 'risk_control.min_risk_reward_ratio',
      status: 'live',
      ui_label: '',
      consumers: ['trader/armed_executor.go:68'],
      dual_level: false,
    },
  ],
  resolved: [
    {
      path: 'risk_control.min_risk_reward_ratio',
      saved: '2',
      resolved: '2',
      source: 'saved value',
      line: '2 → 2 · saved value',
    },
    {
      path: 'day_plan.plan_mode',
      saved: '(unset)',
      resolved: 'advisory',
      source: 'shipped default',
      line: '(unset) → advisory · shipped default',
    },
    {
      path: 'regime.htf_veto',
      saved: '(unset)',
      resolved: 'true',
      source: 'shipped default',
      line: '(unset) → true · shipped default',
    },
  ],
}

function stubFetch(body: unknown) {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(body) })
  )
}

describe('ResolvedKnobPanel', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('renders one saved → resolved · source line per resolved field', async () => {
    stubFetch(payload)
    render(<ResolvedKnobPanel traderId="t1" session="NY" />)

    const lines = await screen.findAllByTestId('resolved-line')
    expect(lines).toHaveLength(3)
    expect(lines[0].textContent).toContain('risk_control.min_risk_reward_ratio')
    expect(lines[0].textContent).toContain('2 → 2 · saved value')
    expect(lines[1].textContent).toContain(
      '(unset) → advisory · shipped default'
    )
    expect(lines[2].textContent).toContain('(unset) → true · shipped default')
  })

  // The whole point of the two labels: they must not read the same, and neither
  // may read as "dead".
  it('keeps the two ineffective/candidate labels distinct and never says dead', async () => {
    stubFetch(payload)
    render(<ResolvedKnobPanel traderId="t1" session="NY" />)

    // Only knobs that need explaining are listed — a live knob's behaviour is
    // the knob itself, and listing all 167 would bury the 23 that mislead.
    await waitFor(() =>
      expect(screen.getAllByTestId('knob-row').length).toBe(2)
    )
    const body = screen.getByTestId('resolved-panel').textContent ?? ''
    expect(body).not.toContain('risk_control.min_risk_reward_ratio\n')

    expect(body).toContain('read; does not take effect')
    expect(body).toContain('no known reader — pending verification')
    expect(body).not.toMatch(/\bdead\b/i)
    expect(body).not.toMatch(/\bunused\b/i)
  })

  // Absent is not empty. A trader-less answer must not render as "no fields
  // resolve", which the operator would read as a finding.
  it('omits the resolved section entirely when the payload has none', async () => {
    const { resolved: _drop, ...withoutResolved } = payload
    stubFetch(withoutResolved)
    render(<ResolvedKnobPanel />)

    await waitFor(() =>
      expect(screen.getByTestId('resolved-panel')).toBeTruthy()
    )
    expect(screen.queryAllByTestId('resolved-line')).toHaveLength(0)
    expect(screen.queryByTestId('resolved-section')).toBeNull()
  })

  it('sends the bearer token when one is stored', async () => {
    localStorage.setItem('auth_token', 'tok123')
    stubFetch(payload)
    render(<ResolvedKnobPanel traderId="t1" session="NY" />)

    await waitFor(() => expect(fetch).toHaveBeenCalled())
    const [, init] = (fetch as unknown as ReturnType<typeof vi.fn>).mock
      .calls[0]
    expect(init.headers.Authorization).toBe('Bearer tok123')
  })
})
