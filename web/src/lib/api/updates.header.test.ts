// W-ONE-BUTTON M3 × M5 wire pin: the M3 gate (api/handler_updates.go,
// updatesRefusal) refuses every /api/updates* request that does not carry
// X-NOFX-Update: 1 — before it reads the JWT — with 403 "update header
// missing or wrong". The M5 readers never sent it, so the enrolled admin's
// own page read 403 forever while every mocked shape pin stayed green.
//
// This pin runs the REAL httpClient (its request interceptor included) and
// captures the config axios would put on the wire, through a stand-in
// adapter: what is asserted is the outgoing request, not a mock's arguments.
// The Go side pins the header name against api.UpdateHeader
// (api/handler_updates_web_header_test.go).

import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import type { InternalAxiosRequestConfig } from 'axios'

import { httpClient } from '../httpClient'
import { updatesApi } from './updates'

type Sent = { url: string; method: string; headers: Record<string, unknown> }

const sent: Sent[] = []
// The instance is private in TS only; the adapter swap is the one seam that
// keeps the real interceptors in the path.
const instance = (
  httpClient as unknown as { axiosInstance: { defaults: { adapter: unknown } } }
).axiosInstance
let savedAdapter: unknown

beforeEach(() => {
  sent.length = 0
  savedAdapter = instance.defaults.adapter
  instance.defaults.adapter = async (config: InternalAxiosRequestConfig) => {
    const h = config.headers
    sent.push({
      url: String(config.url),
      method: String(config.method).toUpperCase(),
      headers:
        typeof h?.toJSON === 'function'
          ? (h.toJSON() as Record<string, unknown>)
          : { ...h },
    })
    return { data: {}, status: 200, statusText: 'OK', headers: {}, config }
  }
})

afterEach(() => {
  instance.defaults.adapter = savedAdapter
})

// Header names are case-insensitive on the wire; count every spelling.
function updateHeaderValues(h: Record<string, unknown>): unknown[] {
  return Object.entries(h)
    .filter(([k]) => k.toLowerCase() === 'x-nofx-update')
    .map(([, v]) => v)
}

describe('every /api/updates* request carries X-NOFX-Update: 1 (the M3 gate refuses it otherwise)', () => {
  const updateCalls: Array<[string, () => Promise<unknown>, string, string]> = [
    ['updatesStatus', () => updatesApi.updatesStatus(), 'GET', '/api/updates'],
    ['check', () => updatesApi.check(), 'POST', '/api/updates/check'],
    [
      'install',
      () =>
        updatesApi.install({
          release_id: 'r1',
          job_id: 'j1',
          expires_at: '2026-09-24T10:00:00Z',
          hmac: 'fixture-not-a-mac',
        }),
      'POST',
      '/api/updates/install',
    ],
    ['job', () => updatesApi.job('job-1'), 'GET', '/api/updates/jobs/job-1'],
  ]

  it.each(updateCalls)(
    '%s sends the header exactly once, value "1"',
    async (_name, call, method, url) => {
      await call()
      expect(sent).toHaveLength(1)
      expect(sent[0].method).toBe(method)
      expect(sent[0].url).toBe(url)
      expect(updateHeaderValues(sent[0].headers)).toEqual(['1'])
    }
  )

  const otherCalls: Array<[string, () => Promise<unknown>, string]> = [
    ['health', () => updatesApi.health(), '/api/health'],
    ['maintenance', () => updatesApi.maintenance(), '/api/maintenance'],
    [
      'installationGate',
      () => updatesApi.installationGate(),
      '/api/installation-gate',
    ],
  ]

  it.each(otherCalls)(
    '%s (not an /updates route) does NOT send it',
    async (_name, call, url) => {
      await call()
      expect(sent).toHaveLength(1)
      expect(sent[0].url).toBe(url)
      expect(updateHeaderValues(sent[0].headers)).toEqual([])
    }
  )
})
