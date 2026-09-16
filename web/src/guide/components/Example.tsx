// LIVE-COMPONENT RULE (guide dispatch 2026-08-27): every chip/badge shown in
// the guide is the REAL React component rendered with mock props inside this
// wrapper — dashed border + "example" tag. Never screenshots.
import type { ReactNode } from 'react'

export function Example({
  label,
  children,
}: {
  label: string
  children: ReactNode
}) {
  return (
    <figure
      data-testid="guide-example"
      className="my-3 rounded-lg p-3"
      style={{
        border: '1px dashed var(--vl-gold-line)',
        background: 'var(--vl-card)',
      }}
    >
      <figcaption
        className="mb-2 flex items-center justify-between text-[10px] uppercase tracking-widest"
        style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
      >
        <span>example</span>
        <span style={{ color: 'var(--vl-muted)' }}>{label}</span>
      </figcaption>
      <div className="flex flex-wrap items-center gap-2">{children}</div>
    </figure>
  )
}
