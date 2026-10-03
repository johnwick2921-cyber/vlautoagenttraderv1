// UPDATER-USABLE-V1 — the review-constant pin at the component call site.
//
// INSTALL_AUTHZ_UNDER_REVIEW is a SINGLE constant in
// web/src/lib/api/updates.ts. While it is true, the install control is
// disabled with the exact review text and no install POST can fire — even
// with a valid authorization pasted and the server answering
// install_enabled=true / worker_listening=true. The CTO flips the constant
// in the separate final commit after his own adversarial pass; this file
// stays green before and after the flip.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  health: vi.fn(),
  maintenance: vi.fn(),
  installationGate: vi.fn(),
  updatesStatus: vi.fn(),
  check: vi.fn(),
  install: vi.fn(),
  job: vi.fn(),
  receipt: vi.fn(),
  getTraders: vi.fn(),
  getStrategyEffective: vi.fn(),
  getExchangeConfigs: vi.fn(),
  request: vi.fn(),
}))
vi.mock('../lib/api/updates', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api/updates')>()
  return {
    ...actual,
    updatesApi: {
      health: mocks.health,
      maintenance: mocks.maintenance,
      installationGate: mocks.installationGate,
      updatesStatus: mocks.updatesStatus,
      check: mocks.check,
      install: mocks.install,
      job: mocks.job,
      receipt: mocks.receipt,
    },
    // the constant, ON — as it ships until the CTO's separate flip commit.
    INSTALL_AUTHZ_UNDER_REVIEW: true,
    INSTALL_UNDER_REVIEW_TEXT: 'install authorization under review',
  }
})
vi.mock('../lib/api/traders', () => ({
  traderApi: { getTraders: mocks.getTraders },
}))
vi.mock('../lib/api/strategyEffective', () => ({
  strategyEffectiveApi: { getStrategyEffective: mocks.getStrategyEffective },
}))
vi.mock('../lib/api/config', () => ({
  configApi: { getExchangeConfigs: mocks.getExchangeConfigs },
}))
vi.mock('../lib/httpClient', () => ({
  httpClient: { request: mocks.request },
}))
vi.mock('../router/selectedTrader', () => ({
  loadStoredTraderId: () => undefined,
}))
vi.mock('../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))

import UpdatesPage from './UpdatesPage'

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
    status: {
      enrolled: true,
      manifest_verifier: 'configured',
      install_enabled: true,
      worker_listening: true,
    },
  })
  mocks.getTraders.mockResolvedValue([])
  mocks.getExchangeConfigs.mockResolvedValue([])
})

describe('UpdatesPage — review constant ON', () => {
  it('shows the exact review text and keeps the install button disabled', async () => {
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

  it('a valid paste cannot fire the install POST while the constant is true', async () => {
    render(<UpdatesPage />)
    await waitFor(() =>
      expect(
        screen.getByText('install authorization under review')
      ).toBeTruthy()
    )
    fireEvent.click(screen.getByTestId('advanced-toggle'))
    fireEvent.change(screen.getByTestId('authz-paste'), {
      target: {
        value: JSON.stringify({
          release_id: 'v9.9.9',
          job_id: 'job-202',
          expires_at: 1893456000,
          hmac: 'deadbeef',
        }),
      },
    })
    const button = screen.getByTestId(
      'update-button-advanced'
    ) as HTMLButtonElement
    // server says everything is ready, the paste parses — and the constant
    // still wins, exactly as ruled
    expect(button.disabled).toBe(true)
    fireEvent.click(button)
    expect(mocks.install).not.toHaveBeenCalled()
  })
})
