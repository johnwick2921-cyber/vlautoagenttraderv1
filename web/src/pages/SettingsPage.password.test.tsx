// M3 red-team H1 — CTO ruling 1790231205208 item (1)+(5): the server requires
// current_password on PUT /api/user/password, so the Settings → Account form
// must SEND it (the server and this form land together — a server that
// required the field before the form sent it would lock the owner out of
// changing the password) and must SHOW the server's refusal text.
// Driven at the component call site: the real SettingsPage, the real form,
// the real fetch call; only the network and unrelated panels are stubbed.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
  logout: vi.fn(),
}))

vi.mock('sonner', () => ({
  toast: { success: mocks.toastSuccess, error: mocks.toastError },
}))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 'u-1', email: 'owner@example.test' },
    token: 'tok',
    logout: mocks.logout,
  }),
}))
vi.mock('../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))
vi.mock('../lib/api', () => ({
  api: {
    getModelConfigs: vi.fn().mockResolvedValue([]),
    getSupportedModels: vi.fn().mockResolvedValue([]),
    getExchangeConfigs: vi.fn().mockResolvedValue([]),
  },
}))
vi.mock('../components/settings/ResolvedKnobPanel', () => ({
  ResolvedKnobPanel: () => null,
}))
vi.mock('../components/trader/ExchangeConfigModal', () => ({
  ExchangeConfigModal: () => null,
}))
vi.mock('../components/trader/TelegramConfigModal', () => ({
  TelegramConfigModal: () => null,
}))
vi.mock('../components/trader/ModelConfigModal', () => ({
  ModelConfigModal: () => null,
}))
vi.mock('./UpdatesPage', () => ({ default: () => null }))

import { SettingsPage } from './SettingsPage'

type Reply = { status: number; body: unknown }
let passwordReply: Reply
let passwordCalls: { init: RequestInit | undefined }[]

beforeEach(() => {
  passwordCalls = []
  passwordReply = { status: 200, body: { message: 'Password updated' } }
  for (const fn of Object.values(mocks)) fn.mockReset()
  localStorage.setItem('auth_token', 'owner-session-token')
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string, init?: RequestInit) => {
      if (url === '/api/user/password') {
        passwordCalls.push({ init })
        return Promise.resolve(
          new Response(JSON.stringify(passwordReply.body), {
            status: passwordReply.status,
            headers: { 'Content-Type': 'application/json' },
          })
        )
      }
      return Promise.resolve(
        new Response('{}', {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      )
    })
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
  localStorage.clear()
})

function fillAndSubmit(current: string, next: string) {
  fireEvent.change(screen.getByLabelText('Current Password'), {
    target: { value: current },
  })
  fireEvent.change(screen.getByLabelText('New Password'), {
    target: { value: next },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Update Password' }))
}

describe('Settings → Account → Change password', () => {
  it('sends current_password with the new password, as the server requires', async () => {
    render(<SettingsPage />)
    fillAndSubmit('the-current-pass', 'the-new-pass-01')
    await waitFor(() => expect(passwordCalls).toHaveLength(1))
    const init = passwordCalls[0].init!
    expect(init.method).toBe('PUT')
    expect(JSON.parse(String(init.body))).toEqual({
      current_password: 'the-current-pass',
      new_password: 'the-new-pass-01',
    })
    expect((init.headers as Record<string, string>).Authorization).toBe(
      'Bearer owner-session-token'
    )
  })

  it('shows the server 403 refusal text (wrong current password)', async () => {
    passwordReply = {
      status: 403,
      body: { error: 'current password is incorrect' },
    }
    render(<SettingsPage />)
    fillAndSubmit('a-wrong-guess', 'the-new-pass-01')
    const alert = await screen.findByTestId('password-change-error')
    expect(alert.textContent).toBe('current password is incorrect')
    expect(mocks.toastError).toHaveBeenCalledWith(
      'current password is incorrect'
    )
    expect(mocks.toastSuccess).not.toHaveBeenCalled()
  })

  it('shows the server 400 refusal text', async () => {
    passwordReply = {
      status: 400,
      body: {
        error: 'current_password and new_password (min 8 chars) are required',
      },
    }
    render(<SettingsPage />)
    fillAndSubmit('x', 'the-new-pass-01')
    const alert = await screen.findByTestId('password-change-error')
    expect(alert.textContent).toBe(
      'current_password and new_password (min 8 chars) are required'
    )
  })

  it('a successful change signs this session out (H2: the server retires every earlier session)', async () => {
    render(<SettingsPage />)
    fillAndSubmit('the-current-pass', 'the-new-pass-01')
    await waitFor(() => expect(mocks.logout).toHaveBeenCalledTimes(1))
    expect(mocks.toastSuccess).toHaveBeenCalledWith(
      'Password updated — sign in again with the new password'
    )
  })

  it('a refused change does NOT sign the session out', async () => {
    passwordReply = {
      status: 403,
      body: { error: 'current password is incorrect' },
    }
    render(<SettingsPage />)
    fillAndSubmit('a-wrong-guess', 'the-new-pass-01')
    await screen.findByTestId('password-change-error')
    expect(mocks.logout).not.toHaveBeenCalled()
  })

  it('cannot submit without a current password (the button stays disabled)', () => {
    render(<SettingsPage />)
    fireEvent.change(screen.getByLabelText('New Password'), {
      target: { value: 'the-new-pass-01' },
    })
    expect(
      (
        screen.getByRole('button', {
          name: 'Update Password',
        }) as HTMLButtonElement
      ).disabled
    ).toBe(true)
    expect(passwordCalls).toHaveLength(0)
  })
})
