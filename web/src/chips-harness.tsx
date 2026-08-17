// COMPONENT HARNESS (dev-only) — see chips-harness.html.
//
// Mounts the REAL VersionChips at n=1, 2 and 6 inside a card-width container, so
// the layout can be measured in a browser rather than asserted. Not referenced by
// index.html, so it never enters the production bundle.

import { createRoot } from 'react-dom/client'
import { VersionChips } from './components/plan/chips'
import './theme/vl-tokens.css'

function Case({ n, current }: { n: number; current: number }) {
  return (
    <div
      data-testid={`case-${n}`}
      style={{
        background: 'var(--vl-card)',
        border: '1px solid var(--vl-hair)',
        borderRadius: 'var(--vl-radius-card)',
        padding: 20,
        margin: 12,
        // The real card interior at 390px viewport, where the header used to
        // overflow by ~41px and push the ✎/💬 buttons off the card.
        maxWidth: 316,
      }}
    >
      <div
        style={{
          color: 'var(--vl-faint)',
          font: '10px system-ui',
          marginBottom: 6,
        }}
      >
        n={n}
      </div>
      {/* Mirrors the card header: title + chips + a lifecycle badge competing
          for the same row. */}
      <div
        className="flex flex-wrap items-center justify-between gap-2"
        style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}
      >
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            gap: 8,
            minWidth: 0,
          }}
        >
          <span
            style={{
              font: '600 20px system-ui',
              color: 'var(--vl-ivory)',
            }}
          >
            Today's Plan
          </span>
          <VersionChips
            version={current}
            latest={n}
            count={n}
            onSelect={(v) => {
              document
                .getElementById('clicked')!
                .setAttribute('data-value', String(v))
            }}
            titleFor={(v) => `v${v} — death condition (all levels consumed)`}
          />
          <span
            style={{
              font: '700 9px system-ui',
              letterSpacing: '.1em',
              color: 'var(--vl-short)',
              border: '1px solid rgba(224,108,108,.4)',
              borderRadius: 5,
              padding: '2px 6px',
            }}
          >
            SUPERSEDED
          </span>
        </div>
        <div style={{ display: 'flex', gap: 4 }}>
          <button style={{ background: 'none', border: 0, padding: '4px 8px' }}>
            💬
          </button>
          <button style={{ background: 'none', border: 0, padding: '4px 8px' }}>
            ✎
          </button>
        </div>
      </div>
    </div>
  )
}

createRoot(document.getElementById('root')!).render(
  <>
    <span id="clicked" data-value="" />
    <Case n={1} current={1} />
    <Case n={2} current={2} />
    <Case n={6} current={3} />
  </>
)
