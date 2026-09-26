// W-EXEC-TRUTH W1 (g) — one settings row's EFFECTIVE value, ORIGIN and SCOPE,
// e.g. "eff 3 · saved value · strategy" / "eff 8 · shipped default · strategy".
//
// It renders the server's row verbatim (GET /api/strategies/:id/effective). It
// never computes a value, never borrows the form's value, and never turns
// "unknown" into 0: no row → nothing rendered; a value the server does not know
// arrives as "n/a — …" and is shown as that sentence.

import type { EffectiveKnob } from '../../lib/api/strategyEffective'

const REDACTED = 'redacted'

/** The server resolves the SAVED strategy row, never the form. Every chip says
 *  so in its title; EffectiveSavedNote says it once per editor. */
export const SAVED_CONFIG_TITLE =
  'saved strategy config — unsaved edits apply after Save'

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
  // W1 (CTO R2): an explicit 0 no Studio save confirmed refuses its trader.
  if (knob.origin.includes('UNCONFIRMED')) return 'text-amber-400'
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
      title={`${SAVED_CONFIG_TITLE} · ${storedTitle(knob)} · resolver: ${knob.resolver}`}
    >
      {text}
    </span>
  )
}

/** The chip on its own line under a settings row, with an optional short
 *  label when one card carries several rows (e.g. a guardrail's switch and its
 *  value). No row → renders NOTHING, not an empty line (L7: absent is not a
 *  value). */
export function EffectiveLine({
  knob,
  label,
  className = 'mt-1',
}: {
  knob?: EffectiveKnob | null
  label?: string
  className?: string
}) {
  if (!knob) return null
  return (
    <div
      data-testid="effective-line"
      className={`flex flex-wrap items-baseline gap-1 ${className}`}
    >
      {label && <span className="text-[10px] text-slate-500">{label}:</span>}
      <EffectiveChip knob={knob} />
    </div>
  )
}

/** The bare effective value the server resolved, for a summary that prints a
 *  number inside its own sentence (e.g. the futures risk panel). null when the
 *  row is absent, not known ("n/a …") or redacted — the caller prints n/a,
 *  never a literal fallback. */
export function effectiveValueText(knob?: EffectiveKnob | null): string | null {
  if (!knob || isNA(knob.effective) || knob.effective === REDACTED) return null
  return formatEffective(knob.effective)
}

const SAVED_NOTE: Record<string, string> = {
  en: 'eff = the value the running bot uses, read from the SAVED strategy (value · origin · scope). Unsaved edits show here after Save.',
  zh: 'eff = 运行中的机器人实际使用的值，读自【已保存】的策略（值 · 来源 · 作用域）。未保存的修改在保存后显示。',
  id: 'eff = nilai yang dipakai bot yang berjalan, dibaca dari strategi TERSIMPAN (nilai · asal · cakupan). Perubahan yang belum disimpan tampil setelah Simpan.',
}

/** One line per editor: the chips read the saved strategy, not the form. */
export function EffectiveSavedNote({ language }: { language: string }) {
  return (
    <p
      data-testid="effective-saved-note"
      className="text-[10px] font-mono text-slate-500"
      title={SAVED_CONFIG_TITLE}
    >
      {SAVED_NOTE[language] ?? SAVED_NOTE.en}
    </p>
  )
}
