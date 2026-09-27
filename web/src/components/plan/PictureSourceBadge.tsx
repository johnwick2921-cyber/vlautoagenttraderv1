import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type { PictureEvidenceView, PlanScenario } from '../../lib/api/plan'
import { fmtCtClock, fmtLedgerPrice } from './EntryPolicyLine'

// W-EXEC-TRUTH W5 — the 📷 PICTURE badge on a MACHINE-authored scenario (a
// Picture HTF setup recorded as a Day Plan scenario, P<n>). Everything in the
// tooltip is READ from the scenario's machine record and the evidence frozen at
// hand-off (kernel.PlanMachineSource.evidence = trader PictureEvidence); the
// card computes nothing. A missing number reads "n/a", never 0. Evidence that
// cannot be read renders "evidence unavailable"; no evidence at all renders
// "evidence not recorded" (absent is not the same as unreadable).

export const SCENARIO_SOURCE_PICTURE = 'picture'

const isNum = (v: unknown): v is number =>
  typeof v === 'number' && Number.isFinite(v)

const isObject = (v: unknown): v is Record<string, unknown> =>
  typeof v === 'object' && v !== null && !Array.isArray(v)

/**
 * Reads machine.evidence. undefined → nothing was recorded; null → something
 * was recorded but it is not an evidence object (unreadable). A JSON string
 * holding an object is read too (a raw record stored as text).
 */
export function readPictureEvidence(
  raw: unknown
): PictureEvidenceView | null | undefined {
  if (raw === undefined) return undefined
  let v = raw
  if (typeof v === 'string') {
    try {
      v = JSON.parse(v)
    } catch {
      return null
    }
  }
  return isObject(v) ? (v as PictureEvidenceView) : null
}

// An R:R floor or estimate: a ratio > 0; anything else is unknown.
const ratio = (v: unknown): string =>
  isNum(v) && v > 0
    ? v.toLocaleString('en-US', { maximumFractionDigits: 2 })
    : 'n/a'

const priceOf = (v: unknown): string => fmtLedgerPrice(isNum(v) ? v : null)

const text = (v: unknown): string =>
  typeof v === 'string' && v.trim() ? v.trim() : 'n/a'

/** The badge's tooltip, one line: rule · H1 close vs the 4H body · window ·
 * stop source · R:R floor. Trading vocabulary (rule, H1, 4H, R:R) stays as
 * the ledger writes it; the card's own words translate. */
export function pictureEvidenceTitle(
  scenario: PlanScenario,
  language: Language = 'en'
): string {
  const head = `${tp('pictureSourceBadge', language)} — ${tp('pictureSourceTitle', language)}`
  const m = scenario.machine
  if (!m) return `${head} · ${tp('pictureMachineAbsent', language)}`
  const ev = readPictureEvidence(m.evidence)
  const ruleVer = isNum(m.rule_ver) && m.rule_ver > 0 ? `v${m.rule_ver}` : 'n/a'
  const parts = [head, `rule ${text(m.rule)} (version ${ruleVer})`]
  if (ev === undefined) {
    parts.push(tp('pictureEvidenceNotRecorded', language))
  } else if (ev === null) {
    parts.push(tp('pictureEvidenceUnavailable', language))
  } else {
    const role = typeof ev.level_role === 'string' && ev.level_role.trim()
    parts.push(
      `H1 close ${priceOf(ev.h1_new_close)} vs the 4H body ${priceOf(ev.body_bot)}–${priceOf(ev.body_top)}${role ? ` (${role})` : ''}`
    )
  }
  parts.push(
    tp('pictureWindowUntil', language, {
      t: fmtCtClock(isNum(m.eligible_until_ms) ? m.eligible_until_ms : null),
    })
  )
  if (ev) {
    parts.push(
      `stop ${priceOf(ev.stop)} · stop source ${text(ev.stop_source)}`,
      `R:R floor ${ratio(ev.rr_floor)}`
    )
  }
  return parts.join(' · ')
}

/** The badge itself; renders nothing on a planner scenario. */
export function PictureSourceBadge({
  scenario,
  language = 'en',
}: {
  scenario: PlanScenario
  language?: Language
}) {
  if (scenario.source !== SCENARIO_SOURCE_PICTURE) return null
  const title = pictureEvidenceTitle(scenario, language)
  return (
    <span
      data-testid={`picture-source-badge-${scenario.id}`}
      className="text-[9px] font-bold px-1.5 py-0.5 rounded"
      title={title}
      aria-label={title}
      style={{
        color: 'var(--vl-ivory)',
        border: '1px solid var(--vl-hair)',
        letterSpacing: '.08em',
        fontFamily: 'var(--vl-font-ui)',
      }}
    >
      {tp('pictureSourceBadge', language)}
    </span>
  )
}
