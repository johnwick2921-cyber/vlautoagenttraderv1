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
//
// PR #200 fold F1 (CTO 1790252194343): the same capture pins the install
// BODY. The server parses expires_at from the raw bytes as a canonical JSON
// NUMBER (internal/updateauth/strict.go rawUnixSeconds) and answers a quoted
// one 400; the client typed it as a string. The ONE byte string both sides
// are pinned to is testdata/updates-install-body.wire.txt (no trailing
// newline — the file IS the body): the Go side feeds it to the production
// parser and router (api/handler_updates_web_body_test.go) and pins
// `updater-bootstrap authorize`'s printed line to it
// (internal/updaterbootstrap/web_wire_test.go).

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import type { InternalAxiosRequestConfig } from 'axios'

import { httpClient } from '../httpClient'
import { updatesApi } from './updates'

type Sent = {
  url: string
  method: string
  headers: Record<string, unknown>
  data: unknown
}

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
      // after axios's transformRequest: the bytes that go on the wire
      data: config.data,
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
          expires_at: 1790244000,
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

describe('the install body on the wire is the grant `updater-bootstrap authorize` prints', () => {
  const wire = readFileSync(
    resolve(__dirname, 'testdata/updates-install-body.wire.txt'),
    'utf-8'
  )

  it('a typed body puts the shared wire fixture on the wire, expires_at a bare JSON number', async () => {
    const body: Parameters<typeof updatesApi.install>[0] = {
      release_id: 'v2026.09.24-1',
      job_id: '0123456789abcdef0123456789abcdef',
      expires_at: 1800000300,
      hmac: 'c'.repeat(64),
    }
    await updatesApi.install(body)
    expect(sent).toHaveLength(1)
    expect(sent[0].data).toBe(wire)
    expect(String(sent[0].data)).toContain('"expires_at":1800000300,')
  })

  it('a pasted authorize line (JSON.parse) goes back on the wire byte for byte', async () => {
    await updatesApi.install(JSON.parse(wire))
    expect(sent).toHaveLength(1)
    expect(sent[0].data).toBe(wire)
  })
})
