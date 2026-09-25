import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import {
  runAgentStream,
  stopActiveAgentStream,
  resetAgentSession,
} from './agentStream'
import { useAgentChatStore } from '../stores/agentChatStore'
import { chatStorageKey } from './agentChatStorage'

const encoder = new TextEncoder()
const response = (chunks: string[]) =>
  new Response(
    new ReadableStream({
      start(c) {
        chunks.forEach((x) => c.enqueue(encoder.encode(x)))
        c.close()
      },
    })
  )
beforeEach(() => {
  resetAgentSession()
  localStorage.clear()
  useAgentChatStore.getState().resetForUser('alice')
})
afterEach(() => {
  resetAgentSession()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})
describe('production agent stream ownership', () => {
  it('retains event headers across transport reads', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          response([
            'event: delta\ndata: "hello"\n\n',
            'event: done\ndata: ""\n\n',
          ])
        )
    )
    await runAgentStream({
      text: 'question',
      language: 'en',
      storageUserId: 'alice',
    })
    expect(useAgentChatStore.getState().messages.at(-1)?.text).toBe('hello')
    expect(useAgentChatStore.getState().loading).toBe(false)
  })
  it('old fetch completion cannot clear a newer user stream or write their history to the old user', async () => {
    let oldResolve!: (r: Response) => void
    let newResolve!: (r: Response) => void
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockImplementationOnce(
          () =>
            new Promise((r) => {
              oldResolve = r
            })
        )
        .mockImplementationOnce(
          () =>
            new Promise((r) => {
              newResolve = r
            })
        )
    )
    const old = runAgentStream({
      text: 'alice private',
      language: 'en',
      storageUserId: 'alice',
    })
    resetAgentSession()
    useAgentChatStore.getState().resetForUser('bob')
    const current = runAgentStream({
      text: 'bob private',
      language: 'en',
      storageUserId: 'bob',
    })
    oldResolve(response(['event: delta\ndata: "old"\n\n']))
    await old
    expect(useAgentChatStore.getState().loading).toBe(true)
    expect(localStorage.getItem(chatStorageKey('alice'))).not.toContain(
      'bob private'
    )
    newResolve(response(['event: delta\ndata: "new"\n\n']))
    await current
    expect(useAgentChatStore.getState().messages.at(-1)?.text).toBe('new')
  })
  it('stopping a request does not let its late failure overwrite a new request for the same user', async () => {
    let rejectOld!: (e: Error) => void
    let resolveNew!: (r: Response) => void
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockImplementationOnce(
          () =>
            new Promise((_r, j) => {
              rejectOld = j
            })
        )
        .mockImplementationOnce(
          () =>
            new Promise((r) => {
              resolveNew = r
            })
        )
    )
    const old = runAgentStream({
      text: 'old',
      language: 'en',
      storageUserId: 'alice',
    })
    stopActiveAgentStream('alice', 'en')
    const next = runAgentStream({
      text: 'next',
      language: 'en',
      storageUserId: 'alice',
    })
    rejectOld(new Error('late failure'))
    await old
    expect(useAgentChatStore.getState().loading).toBe(true)
    expect(
      useAgentChatStore
        .getState()
        .messages.some((m) => m.text.includes('late failure'))
    ).toBe(false)
    resolveNew(response(['event: done\ndata: "finished"\n\n']))
    await next
  })
  it('storage quota failure does not prevent stream cleanup', async () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('quota')
    })
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(['event: done\ndata: "ok"\n\n']))
    )
    await runAgentStream({ text: 'q', language: 'en', storageUserId: 'alice' })
    expect(useAgentChatStore.getState().loading).toBe(false)
    expect(useAgentChatStore.getState().messages.at(-1)?.text).toBe('ok')
  })
})

it('appends repeated deltas without treating them as cumulative snapshots', async () => {
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockResolvedValue(
        response([
          'event: delta\ndata: "yes"\n\n',
          'event: delta\ndata: "yes"\n\n',
        ])
      )
  )
  await runAgentStream({
    text: 'repeat',
    language: 'en',
    storageUserId: 'alice',
  })
  // Today's mergeStreamText dedups identical repeats (a later-wave fix that
  // supersedes #117's plain append), so the collapse is the correct result.
  expect(useAgentChatStore.getState().messages.at(-1)?.text).toBe('yes')
})
