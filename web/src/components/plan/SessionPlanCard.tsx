// P4.3 / P5 — the SessionPlanCard: composes bias / mini-chart / levels /
// scenarios / rules / footer, and renders every lifecycle state. It is a pure
// VIEW of plan_final; the OWNER DOOR (P5) hangs off the header: ✎ opens the edit
// sheet, 💬 opens Ask-Planner. Edits/applies bump the version via onChanged.

import { useState, type ReactNode } from 'react'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { PlanToday, PlanLevelFact } from '../../lib/api/plan'
import { LifecycleChip, VersionChips } from './chips'
import { BiasBlock } from './BiasBlock'
import { ZoneTable } from './ZoneTable'
import { ScenarioList } from './ScenarioList'
import { RulesBlock } from './RulesBlock'
import { PlanFooter } from './PlanFooter'
import { PlanMiniChart } from './PlanMiniChart'
import { EditSheet } from './EditSheet'
import { BulkAddSheet } from './BulkAddSheet'
import { AskPlannerPanel } from './AskPlannerPanel'
import { RealignPanel, RealignButton, type RealignState } from './RealignPanel'
import { api } from '../../lib/api'
import type { RealignChange } from '../../lib/api/plan'

interface Props {
  plan: PlanToday | null
  traderId?: string
  symbol: string
  exchange: string
  language: Language
  isLoading?: boolean
  errored?: boolean
  onChanged?: () => void // re-fetch after an edit / apply so the card version-bumps
}

// A centered state panel (loading / night / no-plan / error).
function StatePanel({
  icon,
  title,
  hint,
  tone,
}: {
  icon: string
  title: string
  hint?: string
  tone?: string
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-10 text-center">
      <span aria-hidden style={{ fontSize: 26, opacity: 0.7 }}>
        {icon}
      </span>
      <span
        className="text-sm font-semibold"
        style={{
          color: tone ?? 'var(--vl-ivory)',
          fontFamily: 'var(--vl-font-ui)',
        }}
      >
        {title}
      </span>
      {hint && (
        <span
          className="text-[11px] max-w-[260px]"
          style={{ color: 'var(--vl-faint)', fontFamily: 'var(--vl-font-ui)' }}
        >
          {hint}
        </span>
      )}
    </div>
  )
}

function Badge({ children, color }: { children: ReactNode; color: string }) {
  return (
    <span
      className="text-[9px] font-bold uppercase tracking-wider"
      style={{
        color,
        border: `1px solid ${color}`,
        borderRadius: 'var(--vl-radius-chip)',
        padding: '1px 5px',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      {children}
    </span>
  )
}

export function SessionPlanCard({
  plan,
  traderId,
  symbol,
  exchange,
  language,
  isLoading,
  errored,
  onChanged,
}: Props) {
  // P5 owner-door state (hooks before the early state-returns, unconditional).
  const [edit, setEdit] = useState<{
    open: boolean
    level?: PlanLevelFact
    index?: number
  }>({ open: false })
  const [askOpen, setAskOpen] = useState(false)
  const [bulkOpen, setBulkOpen] = useState(false)
  // W13 — plan re-alignment after an owner edit.
  const [realign, setRealign] = useState<RealignState>({ phase: 'idle' })
  const [flash, setFlash] = useState(0) // bump → changed rows flash gold once

  const runRealign = async (change: RealignChange, manual = false) => {
    if (!traderId) return
    setRealign({ phase: 'reviewing' })
    const res = await api.realignPlan(traderId, change, symbol, manual)
    switch (res.status) {
      case 'proposal':
        setRealign({ phase: 'proposal', res })
        break
      case 'no-change':
        setRealign({ phase: 'no-change' })
        break
      case 'capped':
        setRealign({ phase: 'capped' })
        break
      case 'failed':
        setRealign({ phase: 'failed' })
        break
      default: // skipped / debounced → stay quiet, the batch is already covered
        setRealign({ phase: 'idle' })
    }
  }
  // The owner door (edit / add / bulk / delete / Ask-Planner / re-align) may only
  // be opened while we are looking at the LIVE session. Every mutating endpoint
  // resolves reg.ActiveSession(now) SERVER-SIDE and writes the active session's
  // plan — it takes no session argument. Once the tabs became real (W15.C) the
  // card could display a sibling session, so leaving the door enabled there would
  // let an edit made against ASIA's levels land silently on NY's plan.
  // is_active comes from /api/plan/today; an older payload omits it (undefined),
  // which keeps the pre-W15.C behavior rather than locking the door.
  const viewingLiveSession = plan?.is_active !== false
  const doorEnabled = !!traderId && viewingLiveSession

  // ── non-plan states ──
  if (errored) {
    return (
      <StatePanel
        icon="⛔"
        title={tp('errorFailClosed', language)}
        hint={tp('errorHint', language)}
        tone="var(--vl-short)"
      />
    )
  }
  if (isLoading && !plan) {
    return <StatePanel icon="◌" title={tp('loading', language)} />
  }
  if (!plan || !plan.found) {
    if (plan?.night || !plan?.session) {
      return (
        <StatePanel
          icon="🌙"
          title={tp('night', language)}
          hint={tp('nightHint', language)}
        />
      )
    }
    // enabled session, plan not armed yet (pre-★2 graceful state)
    return (
      <StatePanel
        icon="🕗"
        title={tp('noPlanYet', language)}
        hint={tp('noPlanYetHint', language)}
      />
    )
  }

  const doc = plan.doc
  if (!doc) {
    return (
      <StatePanel
        icon="🕗"
        title={tp('noPlanYet', language)}
        hint={tp('noPlanYetHint', language)}
      />
    )
  }

  const failClosed =
    doc.day_type === 'no-trade' ||
    (doc.reasoning || '').startsWith('FAIL-CLOSED')
  const expired =
    plan.lifecycle === 'expired' ||
    plan.lifecycle === 'died' ||
    plan.lifecycle === 'superseded'
  const warming = plan.warming && plan.warming !== ''
  const facts = plan.level_facts ?? []

  return (
    <div
      className="flex flex-col gap-3"
      style={{ opacity: expired ? 0.7 : 1 }}
      role="region"
      aria-label={`${tp('title', language)}, v${plan.version ?? 1}, ${plan.lifecycle ?? 'active'}`}
    >
      {/* header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span
            style={{
              fontFamily: 'var(--vl-font-display)',
              fontSize: 20,
              fontWeight: 600,
              color: 'var(--vl-ivory)',
            }}
          >
            {tp('title', language)}
          </span>
          <VersionChips version={plan.version ?? 1} />
          <LifecycleChip
            lifecycle={plan.lifecycle ?? 'active'}
            language={language}
          />
          {plan.degraded && (
            <span
              data-testid="degraded-badge"
              title={`${plan.dark_regime_count ?? 0}/7 regime fields were unavailable at the read`}
              style={{
                fontSize: 9,
                fontWeight: 700,
                letterSpacing: '.1em',
                color: 'var(--vl-short)',
                border: '1px solid rgba(224,108,108,.4)',
                background: 'rgba(224,108,108,.08)',
                borderRadius: 5,
                padding: '2px 6px',
              }}
            >
              ⚠ DEGRADED {plan.dark_regime_count ?? 0}/7
            </span>
          )}
        </div>
        <div className="flex items-center gap-1">
          {/* the OWNER DOOR (P5): 💬 Ask-Planner + ✎ add level. Enabled once a
              trader is present; the level rows themselves are tap-to-edit. */}
          <button
            onClick={() => doorEnabled && setAskOpen(true)}
            disabled={!doorEnabled}
            className="text-[13px] px-2 py-1"
            style={{
              color: doorEnabled ? 'var(--vl-gold)' : 'var(--vl-faint)',
              opacity: doorEnabled ? 1 : 0.4,
              fontFamily: 'var(--vl-font-ui)',
              cursor: doorEnabled ? 'pointer' : 'not-allowed',
            }}
            title={tp('askPlannerTitle', language)}
          >
            💬
          </button>
          <button
            onClick={() => doorEnabled && setEdit({ open: true })}
            disabled={!doorEnabled}
            className="text-[13px] px-2 py-1"
            style={{
              color: doorEnabled ? 'var(--vl-gold)' : 'var(--vl-faint)',
              opacity: doorEnabled ? 1 : 0.4,
              fontFamily: 'var(--vl-font-ui)',
              cursor: doorEnabled ? 'pointer' : 'not-allowed',
            }}
            title={tp('edit', language)}
          >
            ✎
          </button>
        </div>
      </div>

      {/* badges + banners */}
      <div className="flex flex-wrap items-center gap-2">
        {plan.mode === 'advisory' && (
          <Badge color="var(--vl-muted)">{tp('modeAdvisory', language)}</Badge>
        )}
        {warming && (
          <Badge color="var(--vl-gold)">
            {tp('warming', language, { n: plan.warming as string })}
          </Badge>
        )}
        {warming && (
          <Badge color="var(--vl-faint)">{tp('uncalibrated', language)}</Badge>
        )}
      </div>
      {plan.mode === 'advisory' && (
        <div
          className="text-[11px] px-2.5 py-1.5"
          style={{
            background: 'var(--vl-gold-dim)',
            border: '1px solid var(--vl-gold-line)',
            borderRadius: 'var(--vl-radius-inner)',
            color: 'var(--vl-muted)',
            fontFamily: 'var(--vl-font-ui)',
          }}
        >
          {tp('advisoryMode', language)}
        </div>
      )}
      {failClosed && (
        <div
          className="text-[11px] px-2.5 py-1.5"
          style={{
            background: 'rgba(224,108,108,0.08)',
            border: '1px solid rgba(224,108,108,0.3)',
            borderRadius: 'var(--vl-radius-inner)',
            color: 'var(--vl-short)',
            fontFamily: 'var(--vl-font-ui)',
          }}
        >
          ⛔ {tp('errorFailClosed', language)} — {tp('errorHint', language)}
        </div>
      )}

      {/* bias */}
      <BiasBlock bias={doc.bias} language={language} />

      {/* mini chart (levels shared with the table) */}
      {facts.length > 0 && exchange === 'ninjatrader' && (
        <PlanMiniChart
          symbol={symbol}
          exchange={exchange}
          facts={facts}
          language={language}
        />
      )}

      {/* levels — tap a row to edit (P5), ＋ to add an owner level */}
      <ZoneTable
        facts={facts}
        language={language}
        onEdit={
          doorEnabled
            ? (level, index) => setEdit({ open: true, level, index })
            : undefined
        }
        onAdd={doorEnabled ? () => setEdit({ open: true }) : undefined}
        onBulkAdd={doorEnabled ? () => setBulkOpen(true) : undefined}
        flashKey={flash}
      />

      {/* W13 — re-align status / proposal, in place under the levels */}
      {doorEnabled && (
        <>
          <RealignPanel
            state={realign}
            traderId={traderId}
            symbol={symbol}
            language={language}
            onDismiss={() => setRealign({ phase: 'idle' })}
            onApplied={() => {
              setFlash((n) => n + 1)
              onChanged?.()
            }}
          />
          {realign.phase === 'idle' && (
            <div className="px-3 pb-1">
              <RealignButton
                language={language}
                onClick={() =>
                  runRealign(
                    { kind: 'edit-level', summary: 'manual re-align' },
                    true
                  )
                }
              />
            </div>
          )}
        </>
      )}

      {/* Why the door is shut — never leave a disabled surface unexplained. Sits
          OUTSIDE the doorEnabled block, which is exactly what is false here. */}
      {!!traderId && !viewingLiveSession && (
        <div
          className="px-3 pb-2 text-[10px]"
          style={{ color: 'var(--vl-faint)' }}
          data-testid="sibling-session-readonly"
        >
          {tp('siblingReadOnly', language)}
        </div>
      )}

      {/* scenarios */}
      {doc.scenarios && doc.scenarios.length > 0 && (
        <ScenarioList
          scenarios={doc.scenarios}
          statusMap={plan.scenario_status}
          language={language}
        />
      )}

      {/* rules */}
      <RulesBlock
        noTrade={doc.no_trade ?? []}
        deathCondition={doc.death_condition}
        language={language}
      />

      {/* footer */}
      <PlanFooter
        version={plan.version ?? 1}
        dayType={doc.day_type}
        modelId={plan.model_id}
        replansLeft={plan.replans_left}
        language={language}
      />

      {/* P5 owner door — portaled sheets */}
      {doorEnabled && (
        <>
          <EditSheet
            open={edit.open}
            traderId={traderId!}
            symbol={symbol}
            language={language}
            level={edit.level}
            levelIndex={edit.index}
            scenarioIds={(doc.scenarios ?? []).map((s) => s.id)}
            onClose={() => setEdit({ open: false })}
            onSaved={(change) => {
              onChanged?.()
              if (change) void runRealign(change)
            }}
          />
          <BulkAddSheet
            open={bulkOpen}
            traderId={traderId!}
            symbol={symbol}
            language={language}
            onClose={() => setBulkOpen(false)}
            onSaved={(change) => {
              onChanged?.()
              if (change) void runRealign(change)
            }}
          />
          <AskPlannerPanel
            open={askOpen}
            traderId={traderId!}
            symbol={symbol}
            planVersion={plan.version}
            language={language}
            onClose={() => setAskOpen(false)}
            onApplied={() => onChanged?.()}
          />
        </>
      )}
    </div>
  )
}
