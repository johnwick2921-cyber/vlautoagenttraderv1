// P4.3 — the dashboard panel. Self-fetches the plan (usePlanToday), and composes
// the SessionTimelineStrip + SessionTabs + HandoverBanner + SessionPlanCard. This
// is what mounts top-left in TraderDashboardPage. Additive + dormant: it only
// renders when a trader is present; every no-plan state degrades gracefully.

import { useEffect, useState } from 'react'
import { useLanguage } from '../../contexts/LanguageContext'
import { usePlanToday } from './usePlan'
import { SessionTimelineStrip } from './SessionTimelineStrip'
import { SessionTabs, type SessionTab, type TabState } from './SessionTabs'
import { HandoverBanner } from './HandoverBanner'
import { SessionPlanCard } from './SessionPlanCard'
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
  const { plan, isLoading, error } = usePlanToday(traderId, symbol)

  const activeSession =
    (plan?.found ? (plan.session as SessionName) : null) || null
  const [selected, setSelected] = useState<SessionName>('NY')

  // auto-advance: follow the backend active session when it changes.
  useEffect(() => {
    if (activeSession) setSelected(activeSession)
  }, [activeSession])

  // tab states: the live active session is 'active'; enabled-but-inactive is
  // 'inactive'; disabled sessions (not enabled in the registry) are 'disabled'.
  const tabs: SessionTab[] = ALL_SESSIONS.map((name) => {
    const band = SESSION_BANDS.find((b) => b.name === name)
    let state: TabState = 'disabled'
    if (activeSession === name) state = 'active'
    else if (band?.enabled) state = 'inactive'
    return { name, state }
  })

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
      <SessionTimelineStrip activeSession={activeSession} language={language} />
      <SessionTabs
        tabs={tabs}
        selected={selected}
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
      <SessionPlanCard
        plan={plan}
        symbol={symbol}
        exchange={exchange}
        language={language}
        isLoading={isLoading}
        errored={!!error && !plan}
      />
    </div>
  )
}

export default PlanCard
