// W2 FADE PERMISSION (2026-09-10) — a LABEL beside the scenario, never a gate.
//
// Three states, and the third is the one that matters: a scenario with no
// evaluation reads `fade: not evaluated`, NEVER `permitted`. An absent label is
// an absence of a reading, and rendering it as permission would be the
// plausible-zero the record forbids (A24).
export interface FadeLabelView {
  label: 'permitted' | 'excluded' | 'not evaluated'
  exclusions?: string[]
  unknown?: string[]
  detail?: Record<string, string>
  at_ms?: number
}

export function fadeChipText(v: FadeLabelView | undefined): string {
  if (!v) return 'fade: not evaluated'
  if (v.label === 'permitted') return 'fade: permitted'
  if (v.label === 'excluded') {
    const parts = (v.exclusions ?? []).map((n) =>
      v.detail?.[n] ? `${n} ${v.detail[n]}` : n
    )
    return `fade: excluded — ${parts.join('; ')}`
  }
  return 'fade: not evaluated'
}

export function FadePermissionChip({
  id,
  v,
}: {
  id: string
  v: FadeLabelView | undefined
}) {
  const label = v?.label ?? 'not evaluated'
  const color =
    label === 'permitted'
      ? 'var(--vl-green, #3a9d5d)'
      : label === 'excluded'
        ? 'var(--vl-gold)'
        : 'var(--vl-muted)'
  const unknown =
    v?.unknown && v.unknown.length > 0
      ? ` · unknown: ${v.unknown.join(', ')}`
      : ''
  return (
    <span
      data-testid={`fade-${id}`}
      data-fade={label}
      className="text-[10px] uppercase"
      title="Label only — nothing refuses on it. E3 decides whether any exclusion becomes a rule."
      style={{ color, fontFamily: 'var(--vl-font-ui)' }}
    >
      {fadeChipText(v)}
      {unknown}
    </span>
  )
}
