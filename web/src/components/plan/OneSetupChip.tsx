// ONE SETUP (dispatch 102, 2026-09-10) — the arm seam's RECORDED verdict for a
// scenario: the book arms ONE play (the fade) at the best level near price,
// only on a permitted day. Three legs, all three shown on a decline. This chip
// renders what the seam DECIDED (the record it wrote), never a re-evaluation.
//
// An absent verdict reads `one-setup: not evaluated`, NEVER `allowed` — the
// plausible-zero the record forbids (A24). The switch OFF reads `off`.
export interface OneSetupScenarioView {
  allowed: boolean
  level: string
  play: string
  permission: string
  reason?: string
  best_price?: number
  best_names?: string
  best_grade?: string
  target?: string
  waiting?: boolean
  rank?: number
  evaluated_ms?: number
}

export interface OneSetupView {
  enabled: boolean
  min_grade?: string
  evaluated_ms?: number
  scenarios?: Record<string, OneSetupScenarioView>
}

export function oneSetupChipText(
  os: OneSetupView | undefined,
  id: string
): string {
  if (os && os.enabled === false) return 'one-setup: off'
  const v = os?.scenarios?.[id]
  if (!v) return 'one-setup: not evaluated'
  if (v.allowed && v.waiting) return 'one-setup: allowed · second_setup_waiting'
  if (v.allowed) return `one-setup: ALLOWED · target=${v.target ?? 'authored'}`
  return `one-setup: declined — level=${v.level} · play=${v.play} · permission=${v.permission}`
}

export function oneSetupState(
  os: OneSetupView | undefined,
  id: string
): 'off' | 'not evaluated' | 'allowed' | 'waiting' | 'declined' {
  if (os && os.enabled === false) return 'off'
  const v = os?.scenarios?.[id]
  if (!v) return 'not evaluated'
  if (v.allowed && v.waiting) return 'waiting'
  return v.allowed ? 'allowed' : 'declined'
}

export function OneSetupChip({
  id,
  os,
}: {
  id: string
  os: OneSetupView | undefined
}) {
  const state = oneSetupState(os, id)
  const color =
    state === 'allowed'
      ? 'var(--vl-green, #3a9d5d)'
      : state === 'waiting'
        ? 'var(--vl-gold)'
        : state === 'declined'
          ? 'var(--vl-red, #b04a4a)'
          : 'var(--vl-muted)'
  return (
    <span
      data-testid={`one-setup-${id}`}
      data-one-setup={state}
      className="text-[10px] uppercase"
      title="One setup [O]: the book arms only the fade (reject) at the best level near price, only on a permitted day. The follow side is RECORDED, never armed (round 17). This is the seam's recorded verdict, not a re-evaluation."
      style={{ color, fontFamily: 'var(--vl-font-ui)' }}
    >
      {oneSetupChipText(os, id)}
    </span>
  )
}
