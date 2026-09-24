// W-ONE-BUTTON M5 (U2) — UpdatesPage truth-rule tests at the component call
// site: API-only truth, n/a for the unknowable, absent job ≠ [], the install
// button disabled with its exact text while the review is open.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  health: vi.fn(),
  maintenance: vi.fn(),
  installationGate: vi.fn(),
  updatesStatus: vi.fn(),
  check: vi.fn(),
  install: vi.fn(),
  job: vi.fn(),
  getTraders: vi.fn(),
  getStrategyEffective: vi.fn(),
}))
vi.mock('../lib/api/updates', () => ({
  updatesApi: {
    health: mocks.health,
    maintenance: mocks.maintenance,
    installationGate: mocks.installationGate,
    updatesStatus: mocks.updatesStatus,
    check: mocks.check,
    install: mocks.install,
    job: mocks.job,
  },
  INSTALL_AUTHZ_UNDER_REVIEW: true,
  INSTALL_UNDER_REVIEW_TEXT: 'install authorization under review',
}))
vi.mock('../lib/api/traders', () => ({
  traderApi: { getTraders: mocks.getTraders },
}))
vi.mock('../lib/api/strategyEffective', () => ({
  strategyEffectiveApi: { getStrategyEffective: mocks.getStrategyEffective },
}))
vi.mock('../router/selectedTrader', () => ({
  loadStoredTraderId: () => undefined,
}))
vi.mock('../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))

import UpdatesPage from './UpdatesPage'

const heldMaintenance = {
  held: true,
  state: 'held' as const,
  job_id: 'job-7',
  since: '2026-09-24T06:00:00Z',
  in_flight_sends: 0,
  drained: false,
  addon_ack: null,
}

beforeEach(() => {
  for (const fn of Object.values(mocks)) fn.mockReset()
  mocks.health.mockResolvedValue({
    status: 'ok',
    time: '2026-09-24T06:00:00Z',
    revision: 'abc123',
  })
  mocks.maintenance.mockResolvedValue({
    held: false,
    state: 'clear',
    in_flight_sends: 0,
    drained: true,
    addon_ack: null,
  })
  mocks.installationGate.mockResolvedValue({
    ready: false,
    job_id: 'n/a',
    legs: [],
    traders: [],
    note: 'every leg must pass',
  })
  mocks.updatesStatus.mockResolvedValue({
    enrolled: true,
    manifest_verifier: 'stub',
    install_enabled: false,
  })
  mocks.getTraders.mockResolvedValue([])
})

describe('UpdatesPage', () => {
  it('renders only API truth: revision read, addon_ack null shows "no ack yet"', async () => {
    render(<UpdatesPage />)
    await waitFor(() => expect(screen.getByText('abc123')).toBeTruthy())
    expect(screen.getByText('no ack yet')).toBeTruthy()
    // gate ready=false is the API's own verdict, rendered exactly as returned
    expect(screen.getByText('false')).toBeTruthy()
  })

  it('shows "no update job" when the API names none', async () => {
    render(<UpdatesPage />)
    await waitFor(() => expect(screen.getByText('no update job')).toBeTruthy())
  })

  it('a 404 job keeps the server text, not a fabricated state', async () => {
    mocks.maintenance.mockResolvedValue(heldMaintenance)
    mocks.job.mockResolvedValue({ error: 'not found', status: 404 })
    render(<UpdatesPage />)
    await waitFor(() => expect(screen.getByText(/not found/)).toBeTruthy())
  })

  it('install stays disabled with the exact review text and never POSTs', async () => {
    render(<UpdatesPage />)
    await waitFor(() =>
      expect(
        screen.getByText('install authorization under review')
      ).toBeTruthy()
    )
    const button = screen.getByTestId('update-button') as HTMLButtonElement
    expect(button.disabled).toBe(true)
    expect(mocks.install).not.toHaveBeenCalled()
  })

  it('button shows Blocked when the API says install_enabled=false', async () => {
    mocks.updatesStatus.mockResolvedValue({
      enrolled: true,
      manifest_verifier: 'stub',
      install_enabled: false,
    })
    render(<UpdatesPage />)
    await waitFor(() => expect(screen.getByText('Blocked')).toBeTruthy())
  })

  it('gate legs render with each leg’s exact detail text', async () => {
    mocks.installationGate.mockResolvedValue({
      ready: false,
      job_id: 'n/a',
      legs: [
        {
          name: 'hold',
          pass: false,
          detail: 'held: job-7 since 2026-09-24',
          source: 'maintenance hold file',
        },
      ],
      traders: ['t1'],
      note: 'every leg must pass',
    })
    render(<UpdatesPage />)
    await waitFor(() =>
      expect(screen.getByText('held: job-7 since 2026-09-24')).toBeTruthy()
    )
  })

  it('4H completed bars: N = PivotWindow+4 from the effective settings, n stays n/a', async () => {
    mocks.getTraders.mockResolvedValue([
      { trader_id: 't1', trader_name: 'T', ai_model: 'm', strategy_id: 's/1' },
    ])
    mocks.getStrategyEffective.mockResolvedValue({
      strategy_id: 's/1',
      session: null,
      venue: null,
      settings: [
        {
          path: 'picture_htf.pivot_window',
          status: 'resolved',
          ui_label: 'PivotWindow',
          stored: { present: true, value: 8 },
          effective: 8,
          origin: 'stored',
          scope: 'strategy',
          resolver: 'direct',
          resolved: true,
        },
      ],
      coverage: { resolved: 1, total: 1, unresolved: [], not_enumerated: [] },
    })
    render(<UpdatesPage />)
    await waitFor(() => expect(screen.getByText(/of 12/)).toBeTruthy())
    // the completed count is n/a — never computed in the browser
    expect(screen.getAllByText('n/a').length).toBeGreaterThan(0)
  })
})
