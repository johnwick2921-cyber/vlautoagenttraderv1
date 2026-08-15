// P4.2 — session windows mirrored from kernel/session_registry.go
// (DefaultSessionRegistry). The backend is the authority for the LIVE active
// session (the /api/plan/today `session` field); this table only drives the 24h
// visual strip + killzone ticks. All times are America/Chicago "HH:MM".
//
// If the Go registry changes, update this table in lockstep (it is a visual
// mirror, not a second source of truth for trading state).

export type SessionName = 'ASIA' | 'LONDON' | 'NY'

export interface KillzoneCT {
  name: string
  startMin: number
  endMin: number
}

export interface SessionBand {
  name: SessionName
  startMin: number // CT minutes-of-day
  endMin: number // CT minutes-of-day (may be < startMin ⇒ wraps midnight)
  readMin: number
  flatMin: number
  killzones: KillzoneCT[]
  enabled: boolean
  hue: string // CSS var for the session hue
}

// "HH:MM" → minutes-of-day.
export function ctToMinutes(hhmm: string): number {
  const [h, m] = hhmm.split(':').map((n) => parseInt(n, 10))
  return h * 60 + m
}

const kz = (name: string, s: string, e: string): KillzoneCT => ({
  name,
  startMin: ctToMinutes(s),
  endMin: ctToMinutes(e),
})

// Mirror of DefaultSessionRegistry() — only NY enabled by default.
export const SESSION_BANDS: SessionBand[] = [
  {
    name: 'ASIA',
    startMin: ctToMinutes('17:00'),
    endMin: ctToMinutes('02:00'), // wraps midnight
    readMin: ctToMinutes('16:55'),
    flatMin: ctToMinutes('02:00'),
    killzones: [kz('asia_kz', '19:00', '23:00')],
    enabled: false,
    hue: 'var(--vl-session-asia)',
  },
  {
    name: 'LONDON',
    startMin: ctToMinutes('02:00'),
    endMin: ctToMinutes('08:30'),
    readMin: ctToMinutes('01:55'),
    flatMin: ctToMinutes('08:30'),
    killzones: [kz('london_kz', '02:00', '05:00')],
    enabled: false,
    hue: 'var(--vl-session-london)',
  },
  {
    name: 'NY',
    startMin: ctToMinutes('08:30'),
    endMin: ctToMinutes('15:00'),
    readMin: ctToMinutes('08:25'),
    flatMin: ctToMinutes('14:45'),
    killzones: [kz('ny_am', '08:30', '11:00'), kz('ny_pm', '13:00', '14:45')],
    enabled: true,
    hue: 'var(--vl-session-ny)',
  },
]

export const DAY_MINUTES = 24 * 60

// Current America/Chicago minutes-of-day, from the client clock. Used only for
// the "now" marker on the strip — never for trading decisions.
export function nowCTMinutes(now: Date = new Date()): number {
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: 'America/Chicago',
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
  }).formatToParts(now)
  const h = parseInt(parts.find((p) => p.type === 'hour')?.value ?? '0', 10)
  const m = parseInt(parts.find((p) => p.type === 'minute')?.value ?? '0', 10)
  // Intl can emit "24" for midnight hour in some engines — normalize.
  return ((h % 24) * 60 + m) % DAY_MINUTES
}

// Split a possibly-wrapping [startMin,endMin) window into 1–2 non-wrapping
// segments in [0,DAY_MINUTES), so the strip can render each as a plain band.
export function toSegments(
  startMin: number,
  endMin: number
): Array<[number, number]> {
  if (endMin > startMin) return [[startMin, endMin]]
  if (endMin === startMin) return [] // zero-width
  return [
    [startMin, DAY_MINUTES],
    [0, endMin],
  ]
}

// True if `min` is inside a possibly-wrapping [startMin,endMin) window.
export function inWindow(
  min: number,
  startMin: number,
  endMin: number
): boolean {
  if (endMin > startMin) return min >= startMin && min < endMin
  return min >= startMin || min < endMin // wraps midnight
}

// The DST caveat: London's CT window drifts by 1h across the ~2 weeks/year when
// US and EU DST transitions are out of phase. We surface a warning rather than
// silently shifting the visual (backend authority handles the real timing).
export function londonDSTWarning(now: Date = new Date()): boolean {
  // US and EU change DST on different dates in March & Oct/Nov. Approximate the
  // out-of-phase windows: mid-March and late-Oct→early-Nov.
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: 'America/Chicago',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(now)
  const mo = parseInt(parts.find((p) => p.type === 'month')?.value ?? '0', 10)
  const day = parseInt(parts.find((p) => p.type === 'day')?.value ?? '0', 10)
  const midMarch = mo === 3 && day >= 8 && day <= 30 // US springs forward ~2nd Sun; EU last Sun
  const lateOct = (mo === 10 && day >= 25) || (mo === 11 && day <= 7) // EU falls back ~last Sun Oct; US 1st Sun Nov
  return midMarch || lateOct
}
