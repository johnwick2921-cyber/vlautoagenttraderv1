// W-ASKPLANNER-SCROLL (2026-09-17) — the thread is the ONE scroll container,
// bounded (min-height 0, flex child of a clipped shell), auto-scrolls to the
// newest message only while the owner is at the bottom, and wheel/keys that
// land outside the list (header, chips, input row) drive the list instead of
// the dashboard behind the sheet. jsdom does no layout, so geometry is stubbed
// per element and the assertions are about the contract, not pixels.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { AskPlannerPanel } from './AskPlannerPanel'
import type { PlanQAMessage } from '../../lib/api/plan'

const askPlanner = vi.fn()
const getPlanThread = vi.fn()

vi.mock('../../lib/api', () => ({
  api: {
    askPlanner: (...a: unknown[]) => askPlanner(...a),
    getPlanThread: (...a: unknown[]) => getPlanThread(...a),
    applyAsk: vi.fn(),
    declineAsk: vi.fn(),
  },
}))
vi.mock('sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

function msg(i: number): PlanQAMessage {
  return i % 2 === 0
    ? {
        id: i,
        role: 'owner',
        content: `question ${i}`,
        evidence: '',
        point_class: '',
        verdict: '',
        patch: '',
        applied: false,
        created_at: i,
      }
    : {
        id: i,
        role: 'planner',
        content: `answer ${i}`,
        evidence: 'evidence',
        point_class: 'BARE',
        verdict: 'DEFEND',
        patch: '',
        applied: false,
        created_at: i,
      }
}
const thread = (n: number) => Array.from({ length: n }, (_, i) => msg(i))

function mount() {
  return render(
    <AskPlannerPanel
      open
      traderId="t1"
      symbol="MNQ"
      planVersion={1}
      language="en"
      onClose={() => {}}
      onApplied={() => {}}
    />
  )
}

/** Give the jsdom list a scrollable geometry: 30 rows in a 500px viewport. */
function geometry(el: HTMLElement, scrollHeight = 2000, clientHeight = 500) {
  Object.defineProperty(el, 'scrollHeight', {
    configurable: true,
    get: () => scrollHeight,
  })
  Object.defineProperty(el, 'clientHeight', {
    configurable: true,
    get: () => clientHeight,
  })
}

const scrollTo = vi.fn(function (this: HTMLElement, _x: number, y: number) {
  this.scrollTop = y
})

beforeEach(() => {
  askPlanner.mockReset()
  getPlanThread.mockReset()
  scrollTo.mockClear()
  Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
    configurable: true,
    value: scrollTo,
  })
})
afterEach(() => {
  delete (HTMLElement.prototype as any).scrollTo
})

describe('AskPlannerPanel thread scrolling', () => {
  it('renders 30 messages inside ONE bounded scroll container, input still rendered', async () => {
    getPlanThread.mockResolvedValue({ thread: thread(30), kpi: {} })
    mount()
    await waitFor(() => expect(screen.getByText('question 28')).toBeTruthy())
    const list = screen.getByTestId('ask-thread')
    expect(list.style.overflowY).toBe('auto')
    expect(['0', '0px']).toContain(list.style.minHeight) // NOT 200 — that pushed the input row off short viewports
    expect(list.style.flex).toBe('1 1 auto')
    expect(list.style.overscrollBehavior).toBe('contain')
    expect(list.getAttribute('tabindex')).toBe('0') // keyboard-scrollable
    expect(list.querySelectorAll(':scope > div').length).toBe(30)
    const dialog = screen.getByRole('dialog')
    expect(dialog.style.overflow).toBe('hidden') // the shell clips; the list scrolls
    expect(dialog.contains(list)).toBe(true)
    expect(screen.getByRole('textbox')).toBeTruthy()
    expect(screen.getByRole('button', { name: /send/i })).toBeTruthy()
  })

  it('auto-scrolls to the newest message on open and on append while at the bottom', async () => {
    getPlanThread.mockResolvedValue({ thread: thread(30), kpi: {} })
    mount()
    await waitFor(() => expect(screen.getByText('question 28')).toBeTruthy())
    const list = screen.getByTestId('ask-thread')
    geometry(list)
    expect(scrollTo).toHaveBeenCalled() // open → bottom
    scrollTo.mockClear()

    // at the bottom (scrollTop == scrollHeight - clientHeight)
    list.scrollTop = 1500
    fireEvent.scroll(list)
    getPlanThread.mockResolvedValue({ thread: thread(32), kpi: {} })
    askPlanner.mockResolvedValue({ ok: true, data: {} })
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'q' } })
    fireEvent.click(screen.getByRole('button', { name: /send/i }))
    await waitFor(() => expect(screen.getByText('answer 31')).toBeTruthy())
    expect(scrollTo).toHaveBeenCalled()
    expect(scrollTo.mock.calls.at(-1)?.[1]).toBe(2000) // to scrollHeight
  })

  it('keeps the reading position when the owner scrolled up while waiting and the reply lands', async () => {
    getPlanThread.mockResolvedValue({ thread: thread(30), kpi: {} })
    mount()
    await waitFor(() => expect(screen.getByText('question 28')).toBeTruthy())
    const list = screen.getByTestId('ask-thread')
    geometry(list)

    // send → stuck to bottom (thinking row appended, scrolled)
    let release: (v: { ok: boolean }) => void = () => {}
    askPlanner.mockImplementation(() => new Promise((r) => (release = r)))
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'q' } })
    fireEvent.click(screen.getByRole('button', { name: /send/i }))
    await waitFor(() => expect(askPlanner).toHaveBeenCalledTimes(1))
    scrollTo.mockClear()

    // …then the owner scrolls UP to re-read while the planner thinks
    list.scrollTop = 200
    fireEvent.scroll(list)

    // the reply lands (refresh → 32 rows) — position must be KEPT
    getPlanThread.mockResolvedValue({ thread: thread(32), kpi: {} })
    await act(async () => {
      release({ ok: true })
    })
    await waitFor(() => expect(screen.getByText('answer 31')).toBeTruthy())
    expect(scrollTo).not.toHaveBeenCalled()
    expect(list.scrollTop).toBe(200)
  })

  it('a wheel over the header / input row (outside the list) scrolls the list, not the page', async () => {
    getPlanThread.mockResolvedValue({ thread: thread(30), kpi: {} })
    mount()
    await waitFor(() => expect(screen.getByText('question 28')).toBeTruthy())
    const list = screen.getByTestId('ask-thread')
    geometry(list)
    list.scrollTop = 0
    const input = screen.getByRole('textbox')
    const ev = new WheelEvent('wheel', {
      deltaY: 120,
      bubbles: true,
      cancelable: true,
    })
    input.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(true) // the dashboard behind must not move
    expect(list.scrollTop).toBe(120)
    // clamped at the end
    input.dispatchEvent(
      new WheelEvent('wheel', { deltaY: 99999, bubbles: true })
    )
    expect(list.scrollTop).toBe(1500)
    // a wheel INSIDE the list is left to native scrolling (not prevented)
    const inner = new WheelEvent('wheel', {
      deltaY: 10,
      bubbles: true,
      cancelable: true,
    })
    list.dispatchEvent(inner)
    expect(inner.defaultPrevented).toBe(false)
  })

  it('PageDown / ArrowDown with the input focused scroll the list, not the page', async () => {
    getPlanThread.mockResolvedValue({ thread: thread(30), kpi: {} })
    mount()
    await waitFor(() => expect(screen.getByText('question 28')).toBeTruthy())
    const list = screen.getByTestId('ask-thread')
    geometry(list)
    list.scrollTop = 0
    const input = screen.getByRole('textbox')
    input.focus()
    fireEvent.keyDown(input, { key: 'ArrowDown' })
    expect(list.scrollTop).toBe(40)
    fireEvent.keyDown(input, { key: 'PageDown' })
    expect(list.scrollTop).toBe(40 + 450) // 0.9 × clientHeight
    fireEvent.keyDown(input, { key: 'PageUp' })
    expect(list.scrollTop).toBe(40)
    // Enter still sends (existing contract) — no key hijack beyond scrolling
    askPlanner.mockResolvedValue({ ok: true, data: {} })
    fireEvent.change(input, { target: { value: 'why?' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => expect(askPlanner).toHaveBeenCalledTimes(1))
  })
})
