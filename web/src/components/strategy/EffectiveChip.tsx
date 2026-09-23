// W-EXEC-TRUTH W1 (g) — one settings row's EFFECTIVE value, ORIGIN and SCOPE,
// e.g. "eff 3 · saved value · strategy" / "eff 8 · shipped default · strategy".
//
// It renders the server's row verbatim (GET /api/strategies/:id/effective). It
// never computes a value, never borrows the form's value, and never turns
// "unknown" into 0: no row → nothing rendered; a value the server does not know
// arrives as "n/a — …" and is shown as that sentence.

import type { EffectiveKnob } from '../../lib/api/strategyEffective'

const REDACTED = 'redacted'

function isNA(v: unknown): v is string {
  return typeof v === 'string' && v.startsWith('n/a')
}

/** Format a resolved value for the chip. null/undefined are "n/a" — never 0. */
export function formatEffective(v: unknown): string {
  if (v === null || v === undefined) return 'n/a'
  if (typeof v === 'boolean') return v ? 'on' : 'off'
  if (typeof v === 'number') return String(v)
  if (typeof v === 'string') return v === '' ? 'none' : v
  if (Array.isArray(v))
    return v.length === 0 ? 'none' : v.map(formatEffective).join(', ')
  if (typeof v === 'object') {
    const entries = Object.entries(v as Record<string, unknown>)
    return entries.length === 0
      ? 'none'
      : entries.map(([k, val]) => `${k}=${formatEffective(val)}`).join(', ')
  }
  return String(v)
}

function tone(knob: EffectiveKnob): string {
  if (isNA(knob.effective) || knob.effective === REDACTED)
    return 'text-slate-500'
  if (knob.origin.includes('not used')) return 'text-amber-400'
  if (
    knob.origin.startsWith('saved value') ||
    knob.origin.startsWith('session override')
  )
    return 'text-emerald-400'
  if (knob.origin.startsWith('env ')) return 'text-sky-400'
  if (knob.origin.startsWith('clamp') || knob.origin.startsWith('suspended'))
    return 'text-amber-400'
  return 'text-slate-400'
}

function storedTitle(knob: EffectiveKnob): string {
  if (!knob.stored.present) return 'stored: absent'
  if (knob.effective === REDACTED) return 'stored: redacted'
  return `stored: ${JSON.stringify(knob.stored.value)}`
}

export function EffectiveChip({ knob }: { knob?: EffectiveKnob | null }) {
  if (!knob) return null

  let text: string
  if (knob.effective === REDACTED) {
    text = REDACTED
  } else if (isNA(knob.effective)) {
    text = knob.effective
  } else {
    text = `eff ${formatEffective(knob.effective)} · ${knob.origin} · ${knob.scope}`
  }

  return (
    <span
      data-testid="effective-chip"
      data-path={knob.path}
      className={`text-[10px] font-mono ${tone(knob)}`}
      title={`${storedTitle(knob)} · resolver: ${knob.resolver}`}
    >
      {text}
    </span>
  )
}
