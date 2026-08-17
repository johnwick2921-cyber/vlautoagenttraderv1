// P4.3 — the dashboard panel. Self-fetches the plan (usePlanToday), and composes
// the SessionTimelineStrip + SessionTabs + HandoverBanner + SessionPlanCard. This
// is what mounts top-left in TraderDashboardPage. Additive + dormant: it only
// renders when a trader is present; every no-plan state degrades gracefully.

import { useEffect, useState } from 'react'
import { useLanguage } from '../../contexts/LanguageContext'
import { usePlanToday, usePlanVersions } from './usePlan'
import { SessionTimelineStrip } from './SessionTimelineStrip'
import { SessionTabs, type SessionTab, type TabState } from './SessionTabs'
import { HandoverBanner } from './HandoverBanner'
import { SessionPlanCard } from './SessionPlanCard'
import { RereadButton } from './RereadButton'
import { AlertCenter } from './AlertCenter'
import { GateBlocksPanel } from './GateBlocksPanel'
import { SESSION_BANDS, type SessionName } from './sessionConfig'

interface Props {
  traderId?: string
  symbol?: string
  exchange?: string
}

const ALL_SESSIONS: SessionName[] = ['ASIA', 'LONDON', 'NY']

export function PlanCard({
  traderId,
  symbol = 'MNQ',
  exchange = 'ninjatrader',
}: Props) {
  const { language } = useLanguage()
  // W15.B — `selected` now drives the FETCH, not just the highlight. Before this
  // the tabs were a lie: clicking ASIA moved the chip while the card kept showing
  // the live session's plan, because the request had no session dimension.
  const [selected, setSelected] = useState<SessionName | null>(null)
  // ITEM 15 — which VERSION is being viewed. null = the live/latest one, which
  // is the only state that existed before: the chips were inert spans, so a past
  // version could not be opened at all.
  const [viewVersion, setViewVersion] = useState<number | null>(null)
  const { plan, isLoading, error, mutate } = usePlanToday(
    traderId,
    symbol,
    selected ?? undefined,
    viewVersion ?? undefined
  )
  const { versions, latestVersion } = usePlanVersions(
    traderId,
    selected ?? undefined
  )

  // Which session is LIVE right now (server-told), independent of what tab the
  // owner is looking at.
  const activeSession = (plan?.active_session as SessionName) || null

  // auto-advance: follow the backend active session until the owner picks a tab.
  useEffect(() => {
    if (activeSession && selected === null) setSelected(activeSession)
  }, [activeSession, selected])

  // Switching sessions must drop the version pin — v3 of NY is not v3 of ASIA.
  useEffect(() => {
    setViewVersion(null)
  }, [selected])

  const tabs = computeSessionTabs(activeSession, plan?.runnable_sessions)

  // handover banner: only the data-detectable transition (expired) is shown from
  // a poll; reading/born/read-failed are event-driven (P4.4 alerts).
  const showExpired =
    plan?.found &&
    (plan.lifecycle === 'expired' ||
      plan.lifecycle === 'died' ||
      plan.lifecycle === 'superseded')

  return (
    <div
      className="p-5 flex flex-col gap-3 animate-slide-in"
      style={{
        background: 'var(--vl-card)',
        border: '1px solid var(--vl-hair)',
        borderRadius: 'var(--vl-radius-card)',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      <AlertCenter traderId={traderId} language={language} />
      <SessionTimelineStrip activeSession={activeSession} language={language} />
      <SessionTabs
        tabs={tabs}
        selected={selected ?? activeSession ?? 'NY'}
        onSelect={setSelected}
        language={language}
      />
      {showExpired && activeSession && (
        <HandoverBanner
          phase="expired"
          session={activeSession}
          language={language}
        />
      )}
      {/* W16/R3 — refusals, where the owner already looks for plan state. */}
      <GateBlocksPanel traderId={traderId} language={language} />
      {/* ITEM 3 — the owner's manual re-read, next to the plan it acts on. */}
      <div className="flex justify-end">
        <RereadButton
          traderId={traderId}
          language={language}
          onDone={() => mutate()}
        />
      </div>
      <SessionPlanCard
        plan={plan}
        traderId={traderId}
        symbol={symbol}
        exchange={exchange}
        language={language}
        isLoading={isLoading}
        errored={!!error && !plan}
        onChanged={() => mutate()}
        versions={versions}
        latestVersion={latestVersion}
        onSelectVersion={(v) => setViewVersion(v === latestVersion ? null : v)}
      />
    </div>
  )
}

// tab states: the live active session is 'active'; runnable-but-inactive is
// 'inactive'; everything else is 'disabled' (SessionTabs refuses the click).
//
// W15.B — enablement comes from the SERVER (`runnable_sessions`, resolved by the
// same gate the bot runs). It used to come from the hardcoded SESSION_BANDS
// constant, so a session the owner switched on stayed greyed out forever. No
// server list (older payload / still loading) → fall back to the constant, which
// keeps the pre-W15 rendering.
export function computeSessionTabs(
  activeSession: SessionName | null,
  runnable?: string[]
): SessionTab[] {
  return ALL_SESSIONS.map((name) => {
    const on = runnable
      ? runnable.includes(name)
      : !!SESSION_BANDS.find((b) => b.name === name)?.enabled
    let state: TabState = 'disabled'
    if (activeSession === name) state = 'active'
    else if (on) state = 'inactive'
    return { name, state }
  })
}

export default PlanCard
