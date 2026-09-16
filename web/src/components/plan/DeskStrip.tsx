import { useState } from 'react'
import useSWR from 'swr'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { api } from '../../lib/api'
import type { DeskLine, DeskStrip as DeskStripData } from '../../lib/api/plan'

// ── THE DESK STRIP ───────────────────────────────────────────────────────────
//
// The first thing on screen: one row per fact the owner needs during a session.
//
// THE STRIP'S LAW — every rule is here because a surface in this system already
// broke it:
//
//  1. NOTHING RENDERS UNDATED. Every row shows the age of the newest input it
//     used. A number with no time is a number you cannot act on.
//  2. UNKNOWN IS A VALUE, and it shows its reason. Never 0, never "—", never
//     the last known number silently. The dashboard once held a green status
//     through 113 minutes of silence; that is what a missing value looks like
//     when you render it as nothing.
//  3. NO GREEN WORD FOR A COMPOUND QUESTION. Row 1 never collapses to "OK": it
//     names process, feed, link and book separately.
//  4. A STALE SOURCE IS AMBER WITH ITS AGE — not the last value, quietly.
//  5. THE LEDGER'S PRICE IS NEVER THE BROKER'S. PROTECTION shows the accepted
//     stop or it shows UNKNOWN.
//
// THE BROWSER COMPUTES NOTHING. Every row arrives rendered from /api/desk, so
// this component cannot invent, round or stale-cache a number. It chooses
// colours and lays out text; that is all it is allowed to do.

const COLOURS: Record<DeskLine['state'], string> = {
  ok: 'var(--vl-text)',
  flat: 'var(--vl-faint)',
  stale: '#F0B90B', // amber — the source is older than its own bound
  unknown: '#F6465D', // red — we could not compute it, and we say why
}

function ageText(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return 'age UNKNOWN'
  if (ms === 0) return 'now'
  const s = Math.round(ms / 1000)
  if (s < 60) return `${s}s ago`
  const m = Math.round(s / 60)
  if (m < 60) return `${m}m ago`
  return `${Math.round(m / 60)}h ago`
}

function Row({ line }: { line: DeskLine }) {
  const colour = COLOURS[line.state] ?? 'var(--vl-text)'
  // RULE 1 + 2: an UNKNOWN shows its reason in place of a value, and every
  // dated row shows its age. Neither is optional.
  const body =
    line.state === 'unknown' && (!line.text || line.text === 'UNKNOWN')
      ? `UNKNOWN — ${line.reason ?? ''}`
      : line.text
  return (
    <div
      data-testid={`desk-line-${line.key}`}
      data-state={line.state}
      style={{
        display: 'flex',
        gap: 8,
        alignItems: 'baseline',
        padding: '2px 0',
        borderBottom: '1px solid var(--vl-hair)',
        fontFamily: 'var(--vl-font-mono, monospace)',
        fontSize: 11,
        lineHeight: 1.45,
      }}
    >
      <span
        style={{
          width: 82,
          flex: '0 0 82px',
          color: 'var(--vl-faint)',
          letterSpacing: '.06em',
          fontWeight: 700,
        }}
      >
        {line.label}
      </span>
      <span style={{ color: colour, flex: 1, wordBreak: 'break-word' }}>
        {body}
      </span>
      <span
        data-testid={`desk-age-${line.key}`}
        title={`source: ${line.source}`}
        style={{ color: 'var(--vl-faint)', flex: '0 0 auto', fontSize: 10 }}
      >
        {line.as_of_ms > 0 ? ageText(line.age_ms) : 'undated'}
      </span>
    </div>
  )
}

export function DeskStrip({ traderId }: { traderId?: string }) {
  // D5 — expanded by default; the collapse lives in COMPONENT STATE, never in
  // browser storage.
  const { language } = useLanguage()
  const deskLabels = {
    en: { expand: 'Expand Desk', collapse: 'Collapse Desk' },
    zh: { expand: '展开交易台', collapse: '收起交易台' },
    id: { expand: 'Perluas Desk', collapse: 'Ciutkan Desk' },
  }[language]
  const [open, setOpen] = useState(true)

  const { data, error } = useSWR<DeskStripData | null>(
    traderId ? `desk-${traderId}` : null,
    () => api.getDeskStrip(traderId as string),
    {
      // D4 — the cadence is the SERVER's resolved value: 5s while a position or
      // an arm is live, 15s otherwise. The browser does not decide how urgent
      // the desk is.
      refreshInterval: (latest) => latest?.cadence_ms ?? 15000,
      revalidateOnFocus: false,
    }
  )

  if (!traderId) return null

  // RULE 2 at the whole-strip level: a dead endpoint is stated, not blanked.
  // A blank strip and a strip full of UNKNOWNs mean different things, and only
  // the second one is honest.
  if (error || data === null) {
    return (
      <div
        data-testid="desk-strip"
        data-state="unreachable"
        style={{
          border: '1px solid #F6465D',
          borderRadius: 6,
          padding: '6px 10px',
          color: '#F6465D',
          fontFamily: 'var(--vl-font-mono, monospace)',
          fontSize: 11,
        }}
      >
        DESK — UNKNOWN: /api/desk could not be read. No value on this strip is
        current. This is not a quiet desk.
      </div>
    )
  }
  if (!data) {
    return (
      <div
        data-testid="desk-strip"
        data-state="loading"
        role="status"
        style={{
          padding: 12,
          border: '1px solid var(--vl-hair)',
          borderRadius: 6,
        }}
      >
        DESK — {t('loading', language)}
      </div>
    )
  }

  return (
    <div
      data-testid="desk-strip"
      style={{
        border: '1px solid var(--vl-hair)',
        borderRadius: 6,
        background: 'var(--vl-card-2, transparent)',
        padding: '4px 10px 6px',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          fontFamily: 'var(--vl-font-ui)',
          fontSize: 10,
          letterSpacing: '.08em',
          color: 'var(--vl-faint)',
          fontWeight: 700,
          padding: '2px 0 4px',
        }}
      >
        <button
          data-testid="desk-toggle"
          aria-label={open ? deskLabels.collapse : deskLabels.expand}
          aria-expanded={open}
          onClick={() => setOpen((v) => !v)}
          style={{
            background: 'none',
            border: '1px solid var(--vl-hair)',
            borderRadius: 4,
            color: 'var(--vl-faint)',
            cursor: 'pointer',
            fontSize: 10,
            padding: '0 5px',
          }}
        >
          {open ? '−' : '+'}
        </button>
        <span>DESK</span>
        <span data-testid="desk-unknown-count" style={{ fontWeight: 400 }}>
          {data.unknown_count} UNKNOWN
          {data.stale_count > 0 ? ` · ${data.stale_count} stale` : ''} ·{' '}
          {Math.round(data.cadence_ms / 1000)}s
        </span>
      </div>
      {open && data.lines.map((l) => <Row key={l.key} line={l} />)}
    </div>
  )
}
