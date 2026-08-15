// P4.3 — the SessionPlanCard: composes bias / mini-chart / levels / scenarios /
// rules / footer, and renders every lifecycle state. It is a pure VIEW of
// plan_final from the API — computes nothing tradable. The edit/re-read/approve
// door is P5, so those buttons are present-but-disabled here (dormant).

import type { ReactNode } from 'react'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { PlanToday } from '../../lib/api/plan'
import { LifecycleChip, VersionChips } from './chips'
import { BiasBlock } from './BiasBlock'
import { ZoneTable } from './ZoneTable'
import { ScenarioList } from './ScenarioList'
import { RulesBlock } from './RulesBlock'
import { PlanFooter } from './PlanFooter'
import { PlanMiniChart } from './PlanMiniChart'

interface Props {
  plan: PlanToday | null
  symbol: string
  exchange: string
  language: Language
  isLoading?: boolean
  errored?: boolean
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
  symbol,
  exchange,
  language,
  isLoading,
  errored,
}: Props) {
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
        </div>
        <div className="flex items-center gap-1">
          {/* the owner door — P5; present but disabled/dormant here */}
          <button
            disabled
            className="text-[11px] px-2 py-1 opacity-40 cursor-not-allowed"
            style={{
              color: 'var(--vl-muted)',
              fontFamily: 'var(--vl-font-ui)',
            }}
            title={tp('edit', language)}
          >
            ✎
          </button>
          <button
            disabled
            className="text-[11px] px-2 py-1 opacity-40 cursor-not-allowed"
            style={{
              color: 'var(--vl-muted)',
              fontFamily: 'var(--vl-font-ui)',
            }}
            title={tp('reread', language)}
          >
            ↻
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

      {/* levels */}
      <ZoneTable facts={facts} language={language} />

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
    </div>
  )
}
