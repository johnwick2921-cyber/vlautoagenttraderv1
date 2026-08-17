// COMPONENT HARNESS (dev-only) — see chips-harness.html.
//
// Mounts the REAL VersionChips at n=1, 2 and 6 inside a card-width container, so
// the layout can be measured in a browser rather than asserted. Not referenced by
// index.html, so it never enters the production bundle.

import { createRoot } from 'react-dom/client'
import { VersionChips } from './components/plan/chips'
import { RereadButton } from './components/plan/RereadButton'
import { api } from './lib/api'
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

// ITEM 3 — the re-read control in both states, without a backend: stub the gate
// so the harness can render enabled and refused side by side.
function RereadCases() {
  return (
    <div
      style={{
        display: 'flex',
        gap: 24,
        margin: 12,
        padding: 20,
        background: 'var(--vl-card)',
        border: '1px solid var(--vl-hair)',
        borderRadius: 'var(--vl-radius-card)',
      }}
    >
      <div data-testid="reread-enabled">
        <div
          style={{
            color: 'var(--vl-faint)',
            font: '10px system-ui',
            marginBottom: 6,
          }}
        >
          enabled
        </div>
        <RereadButton traderId="t-enabled" language="en" />
      </div>
      <div data-testid="reread-refused">
        <div
          style={{
            color: 'var(--vl-faint)',
            font: '10px system-ui',
            marginBottom: 6,
          }}
        >
          refused
        </div>
        <RereadButton traderId="t-refused" language="en" />
      </div>
    </div>
  )
}

api.getRereadGate = async (traderId: string) =>
  traderId === 't-enabled'
    ? {
        allowed: true,
        session: 'NY',
        replans_left: 3,
        replan_cap: 4,
        version: 2,
      }
    : {
        allowed: false,
        reason: 'the re-read budget for NY is spent (4 of 4 used)',
        session: 'NY',
        replans_left: 0,
        replan_cap: 4,
        version: 5,
      }

createRoot(document.getElementById('root')!).render(
  <>
    <span id="clicked" data-value="" />
    <Case n={1} current={1} />
    <Case n={2} current={2} />
    <Case n={6} current={3} />
    <RereadCases />
  </>
)
