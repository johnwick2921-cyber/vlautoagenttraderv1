import { useAgentChatStore } from '../stores/agentChatStore'
import type { AgentMessage as Message, AgentStep } from '../types/agent'
import {
  clearAgentMessages,
  prepareAgentMessagesForPersistence,
  persistAgentMessages,
} from './agentChatStorage'

let msgIdCounter = 0
let activeStreamAbortController: AbortController | null = null
let activeStreamReader: ReadableStreamDefaultReader<Uint8Array> | null = null

function nextId() {
  return `msg-${Date.now()}-${++msgIdCounter}`
}

export function cleanupActiveAgentStream() {
  activeStreamAbortController?.abort()
  activeStreamAbortController = null
  void activeStreamReader?.cancel().catch(() => {
    // Ignore stream cancellation races during teardown.
  })
  activeStreamReader = null
}

export function stopActiveAgentStream(userId?: string, language = 'zh') {
  if (!activeStreamAbortController && !activeStreamReader) return
  const stoppedText =
    language === 'zh' ? '已中止当前回复。' : 'Stopped the current response.'
  const now = new Date().toLocaleTimeString([], {
    timeZone: 'America/Chicago', // owner contract: Houston time for every viewer
    hour: '2-digit',
    minute: '2-digit',
  })
  patchMessagesInStore(
    (prev) =>
      prev.map((m) => {
        if (m.role !== 'bot' || !m.streaming) return m
        const text = m.text?.trim()
          ? `${m.text.trimEnd()}\n\n${stoppedText}`
          : stoppedText
        return {
          ...m,
          text,
          streaming: false,
          time: m.time || now,
        }
      }),
    userId
  )
  cleanupActiveAgentStream()
  useAgentChatStore.getState().setLoading(false)
}

function persistMessagesSnapshotForUser(userId?: string) {
  const { hydrated, messages } = useAgentChatStore.getState()
  if (!hydrated || useAgentChatStore.getState().activeUserId !== userId) return
  const persistable = prepareAgentMessagesForPersistence(messages).slice(-100)
  try {
    persistAgentMessages(window.localStorage, userId, persistable)
  } catch {
    /* Storage unavailable; keep the session usable. */
  }
}

function replaceMessagesInStore(nextMessages: Message[], userId?: string) {
  if (useAgentChatStore.getState().activeUserId !== userId) return
  useAgentChatStore.getState().setMessages(nextMessages)
  persistMessagesSnapshotForUser(userId)
}

function patchMessagesInStore(
  updater: (prev: Message[]) => Message[],
  userId?: string
) {
  if (useAgentChatStore.getState().activeUserId !== userId) return
  const nextMessages = updater(useAgentChatStore.getState().messages)
  useAgentChatStore.getState().updateMessages(() => nextMessages)
  persistMessagesSnapshotForUser(userId)
}

export async function runAgentStream(params: {
  text: string
  token?: string | null
  language: string
  storageUserId?: string
  onDone?: () => void
}) {
  const { text, token, language, storageUserId, onDone } = params
  if (
    !text ||
    useAgentChatStore.getState().loading ||
    useAgentChatStore.getState().activeUserId !== storageUserId
  )
    return

  const time = new Date().toLocaleTimeString([], {
    timeZone: 'America/Chicago', // owner contract: Houston time for every viewer
    hour: '2-digit',
    minute: '2-digit',
  })
  const userMsg: Message = { id: nextId(), role: 'user', text, time }
  const botId = nextId()
  const nextConversation: Message[] = [
    userMsg,
    {
      id: botId,
      role: 'bot',
      text: '',
      time: '',
      streaming: true,
    },
  ]

  replaceMessagesInStore(
    text.trim() === '/clear'
      ? nextConversation
      : [...useAgentChatStore.getState().messages, ...nextConversation],
    storageUserId
  )
  useAgentChatStore.getState().setLoading(true)

  if (text.trim() === '/clear') {
    try {
      clearAgentMessages(window.localStorage, storageUserId)
      useAgentChatStore.getState().setDraftText('')
    } catch {
      // Ignore storage cleanup failure.
    }
  }

  let controller: AbortController | null = null
  let reader: ReadableStreamDefaultReader<Uint8Array> | undefined
  const ownsStream = () =>
    !!controller &&
    activeStreamAbortController === controller &&
    !controller.signal.aborted &&
    useAgentChatStore.getState().activeUserId === storageUserId
  try {
    activeStreamAbortController?.abort()
    controller = new AbortController()
    activeStreamAbortController = controller

    const res = await fetch('/api/agent/chat/stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify({ message: text, lang: language }),
      signal: controller.signal,
    })
    if (!ownsStream()) {
      await res.body?.cancel()
      return
    }
    if (!res.ok) {
      const errData = await res.json().catch(() => ({}))
      throw new Error(errData.error || `Server error (${res.status})`)
    }

    reader = res.body?.getReader()
    const decoder = new TextDecoder()
    if (!reader) throw new Error('No response body')
    const activeReader = reader
    activeStreamReader = activeReader
    controller.signal.addEventListener(
      'abort',
      () => {
        void activeReader.cancel().catch(() => {
          // Ignore double-cancel races.
        })
      },
      { once: true }
    )

    let buffer = ''
    let finalText = ''
    let stepCounter = 0
    const now = () =>
      new Date().toLocaleTimeString([], {
        timeZone: 'America/Chicago', // owner contract: Houston time for every viewer
        hour: '2-digit',
        minute: '2-digit',
      })
    const mergeStreamText = (current: string, incoming: string) => {
      if (!incoming) return current
      if (!current) return incoming
      if (incoming === current) return current
      if (incoming.startsWith(current)) return incoming
      if (current.startsWith(incoming)) return current
      return current + incoming
    }

    while (true) {
      const { done, value } = await activeReader.read()
      if (done) break
      if (!ownsStream()) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      let eventType = ''
      for (const line of lines) {
        if (line.startsWith('event: ')) {
          eventType = line.slice(7).trim()
        } else if (line.startsWith('data: ') && eventType) {
          const rawData = line.slice(6)
          let data: string
          try {
            data = JSON.parse(rawData)
          } catch {
            eventType = ''
            continue
          }
          if (eventType === 'delta') {
            finalText = mergeStreamText(finalText, data)
            patchMessagesInStore(
              (prev) =>
                prev.map((m) =>
                  m.id === botId ? { ...m, text: finalText, time: now() } : m
                ),
              storageUserId
            )
          } else if (eventType === 'plan') {
            const parsedSteps = parsePlanSteps(data)
            patchMessagesInStore(
              (prev) =>
                prev.map((m) =>
                  m.id === botId
                    ? {
                        ...m,
                        steps: parsedSteps.length > 0 ? parsedSteps : m.steps,
                        time: now(),
                      }
                    : m
                ),
              storageUserId
            )
          } else if (eventType === 'step_start') {
            stepCounter += 1
            const nextStep = parseStepEvent(data, stepCounter)
            patchMessagesInStore(
              (prev) =>
                prev.map((m) =>
                  m.id === botId
                    ? {
                        ...m,
                        steps: appendStep(m.steps, nextStep),
                        time: now(),
                      }
                    : m
                ),
              storageUserId
            )
          } else if (eventType === 'step_complete') {
            patchMessagesInStore(
              (prev) =>
                prev.map((m) =>
                  m.id === botId
                    ? {
                        ...m,
                        steps: markLatestRunningCompleted(m.steps, data),
                        time: now(),
                      }
                    : m
                ),
              storageUserId
            )
          } else if (eventType === 'replan') {
            patchMessagesInStore(
              (prev) =>
                prev.map((m) =>
                  m.id === botId
                    ? {
                        ...m,
                        steps: appendStep(m.steps, {
                          id: `replan-${Date.now()}`,
                          label: data,
                          status: 'replanned',
                          detail: data,
                        }),
                        time: now(),
                      }
                    : m
                ),
              storageUserId
            )
          } else if (eventType === 'done') {
            patchMessagesInStore(
              (prev) =>
                prev.map((m) =>
                  m.id === botId
                    ? {
                        ...m,
                        text: finalText || m.text || data,
                        time: now(),
                        streaming: false,
                      }
                    : m
                ),
              storageUserId
            )
          } else if (eventType === 'error') {
            throw new Error(data)
          }
          eventType = ''
        }
      }
    }
    if (!ownsStream()) return

    patchMessagesInStore(
      (prev) =>
        prev.map((m) =>
          m.id === botId && m.streaming
            ? {
                ...m,
                text: finalText || m.text || 'No response',
                streaming: false,
                time: now(),
              }
            : m
        ),
      storageUserId
    )
    window.dispatchEvent(new CustomEvent('agent-preferences-refresh'))
    window.dispatchEvent(new CustomEvent('agent-config-refresh'))
  } catch (e: any) {
    if (!ownsStream()) return
    if (e.name === 'AbortError') {
      patchMessagesInStore(
        (prev) =>
          prev.map((m) =>
            m.id === botId
              ? {
                  ...m,
                  streaming: false,
                  time:
                    m.time ||
                    new Date().toLocaleTimeString([], {
                      timeZone: 'America/Chicago', // owner contract: Houston time for every viewer
                      hour: '2-digit',
                      minute: '2-digit',
                    }),
                }
              : m
          ),
        storageUserId
      )
    } else {
      patchMessagesInStore(
        (prev) =>
          prev.map((m) =>
            m.id === botId
              ? {
                  ...m,
                  text: '⚠️ Error: ' + e.message,
                  time: new Date().toLocaleTimeString([], {
                    timeZone: 'America/Chicago', // owner contract: Houston time for every viewer
                    hour: '2-digit',
                    minute: '2-digit',
                  }),
                  streaming: false,
                }
              : m
          ),
        storageUserId
      )
    }
  }

  try {
    reader?.releaseLock()
  } catch {
    /* canceled reader */
  }
  if (controller && activeStreamAbortController === controller) {
    activeStreamAbortController = null
    activeStreamReader = null
    useAgentChatStore.getState().setLoading(false)
    onDone?.()
  }
}

function appendStep(
  existing: AgentStep[] | undefined,
  step: AgentStep
): AgentStep[] {
  const prev = existing ?? []
  const index = prev.findIndex((item) => item.id === step.id)
  if (index === -1) return [...prev, step]
  return prev.map((item, i) => (i === index ? { ...item, ...step } : item))
}

function parsePlanSteps(data: string): AgentStep[] {
  const text = data.replace(/^🗺️\s*(Plan|计划):\s*/i, '').trim()
  if (!text) return []
  return text.split(/\s*->\s*/).map((part, index) => {
    const cleaned = part.replace(/^\d+\./, '').trim()
    return {
      id: `action-${index + 1}`,
      label: cleaned || `Step ${index + 1}`,
      status: 'pending',
    }
  })
}

function parseStepEvent(data: string, fallbackIndex: number): AgentStep {
  const match =
    data.match(/Step\s+(\d+)\/(\d+):\s+(.+)$/i) ||
    data.match(/步骤\s+(\d+)\/(\d+):\s+(.+)$/)
  if (match) {
    const id = `action-${match[1]}`
    return {
      id,
      label: match[3].trim(),
      status: 'running',
      detail: data,
    }
  }
  return {
    id: `step-${fallbackIndex}`,
    label: data,
    status: 'running',
    detail: data,
  }
}

function markLatestRunningCompleted(
  existing: AgentStep[] | undefined,
  detail: string
): AgentStep[] {
  const prev = existing ?? []
  for (let i = prev.length - 1; i >= 0; i--) {
    if (prev[i].status === 'running') {
      return prev.map((step, index) =>
        index === i ? { ...step, status: 'completed', detail } : step
      )
    }
  }
  return prev
}
export function resetAgentSession() {
  cleanupActiveAgentStream()
  useAgentChatStore.getState().resetForUser(undefined)
  useAgentChatStore.getState().setHydrated(false)
}
